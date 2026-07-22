package postgres

import (
	"archive/zip"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"sync"

	"github.com/kamisamamayuri-cyber/pgwarden/internal/integration/storage"
)

const backupZipDumpFile = "dump.sql"

// RestoreFromLocalZip streams a local backup ZIP into psql without a temp copy.
func (Client) RestoreFromLocalZip(
	ctx context.Context, version PGVersion, connString, zipPath string,
	log io.Writer, jobs int,
) error {
	file, err := os.Open(zipPath)
	if err != nil {
		return fmt.Errorf("error opening ZIP file: %w", err)
	}
	defer file.Close()

	stat, err := file.Stat()
	if err != nil {
		return fmt.Errorf("error stating ZIP file: %w", err)
	}

	return restoreFromZipReaderAt(ctx, version, connString, file, stat.Size(), log, jobs)
}

// RestoreFromS3Zip streams a backup ZIP from S3 into psql without writing it to disk.
func (Client) RestoreFromS3Zip(
	ctx context.Context,
	storageClient *storage.Client,
	version PGVersion,
	connString string,
	accessKey, secretKey, region, endpoint, bucketName, key string,
	log io.Writer,
	jobs int,
) error {
	size, err := storageClient.S3ObjectSize(
		ctx, accessKey, secretKey, region, endpoint, bucketName, key,
	)
	if err != nil {
		return err
	}

	readerAt := storageClient.S3NewReaderAt(
		ctx, accessKey, secretKey, region, endpoint, bucketName, key, size,
	)

	return restoreFromZipReaderAt(ctx, version, connString, readerAt, size, log, jobs)
}

func restoreFromZipReaderAt(
	ctx context.Context, version PGVersion, connString string,
	r io.ReaderAt, size int64, log io.Writer, jobs int,
) error {
	zr, err := zip.NewReader(r, size)
	if err != nil {
		return fmt.Errorf("error opening ZIP reader: %w", err)
	}

	files := make(map[string]*zip.File, len(zr.File))
	for _, f := range zr.File {
		files[f.Name] = f
	}

	if dumpFile, ok := files[backupZipDumpFile]; ok {
		rc, err := dumpFile.Open()
		if err != nil {
			return fmt.Errorf("error opening %s in ZIP: %w", backupZipDumpFile, err)
		}
		defer rc.Close()
		return runPSQLRestore(ctx, version, connString, sanitizeDumpReader(rc), log)
	}

	if manifestFile, ok := files[manifestFileName]; ok {
		manifest, err := readRestoreManifest(manifestFile)
		if err != nil {
			return err
		}
		return restoreParallelDump(ctx, version, connString, files, manifest, log, jobs)
	}

	return fmt.Errorf("neither %s nor %s found in backup ZIP", backupZipDumpFile, manifestFileName)
}

func readRestoreManifest(manifestFile *zip.File) (parallelDumpManifest, error) {
	var manifest parallelDumpManifest

	rc, err := manifestFile.Open()
	if err != nil {
		return manifest, fmt.Errorf("error opening %s: %w", manifestFileName, err)
	}
	manifestJSON, err := io.ReadAll(rc)
	rc.Close()
	if err != nil {
		return manifest, fmt.Errorf("error reading %s: %w", manifestFileName, err)
	}

	if err := json.Unmarshal(manifestJSON, &manifest); err != nil {
		return manifest, fmt.Errorf("error parsing %s: %w", manifestFileName, err)
	}
	if manifest.Format != parallelDumpFormat {
		return manifest, fmt.Errorf("unsupported backup format %q", manifest.Format)
	}
	if len(manifest.Files) == 0 {
		return manifest, fmt.Errorf("empty file list in %s", manifestFileName)
	}
	return manifest, nil
}

func restoreParallelDump(
	ctx context.Context, version PGVersion, connString string,
	files map[string]*zip.File, manifest parallelDumpManifest,
	log io.Writer, jobs int,
) error {
	var preData, postData *zip.File
	var dataFiles []*zip.File
	for _, name := range manifest.Files {
		f, ok := files[name]
		if !ok {
			return fmt.Errorf("file %q listed in manifest is missing from the archive", name)
		}
		switch name {
		case preDataFileName:
			preData = f
		case postDataFileName:
			postData = f
		default:
			dataFiles = append(dataFiles, f)
		}
	}

	if log != nil {
		log = &syncWriter{w: log}
	}

	if preData != nil {
		if err := restoreZipFile(ctx, version, connString, preData, log); err != nil {
			return err
		}
	}

	if err := restoreDataFiles(ctx, version, connString, dataFiles, log, jobs); err != nil {
		return err
	}

	if postData != nil {
		if err := restoreZipFile(ctx, version, connString, postData, log); err != nil {
			return err
		}
	}
	return nil
}

func restoreDataFiles(
	ctx context.Context, version PGVersion, connString string,
	dataFiles []*zip.File, log io.Writer, jobs int,
) error {
	if len(dataFiles) == 0 {
		return nil
	}
	if jobs > len(dataFiles) {
		jobs = len(dataFiles)
	}
	if jobs < 1 {
		jobs = 1
	}

	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	queue := make(chan *zip.File, len(dataFiles))
	for _, f := range dataFiles {
		queue <- f
	}
	close(queue)

	errs := make(chan error, jobs)
	var wg sync.WaitGroup
	for range jobs {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for f := range queue {
				if ctx.Err() != nil {
					return
				}
				if err := restoreZipFile(ctx, version, connString, f, log); err != nil {
					errs <- err
					cancel()
					return
				}
			}
		}()
	}
	wg.Wait()
	close(errs)

	return <-errs
}

func restoreZipFile(
	ctx context.Context, version PGVersion, connString string,
	f *zip.File, log io.Writer,
) error {
	rc, err := f.Open()
	if err != nil {
		return fmt.Errorf("error opening %s in ZIP: %w", f.Name, err)
	}
	defer rc.Close()

	if err := runPSQLRestore(ctx, version, connString, sanitizeDumpReader(rc), log); err != nil {
		return fmt.Errorf("restoring %s: %w", f.Name, err)
	}
	return nil
}

func runPSQLRestore(
	ctx context.Context, version PGVersion, connString string, dump io.Reader, log io.Writer,
) error {
	if closer, ok := dump.(io.Closer); ok {
		defer closer.Close()
	}

	errorBuffer := &bytes.Buffer{}
	var stderr io.Writer = errorBuffer
	if log != nil {
		stderr = io.MultiWriter(errorBuffer, log)
	}

	cmd := exec.CommandContext(ctx, version.Value.PSQL, "-v", "ON_ERROR_STOP=1", connString)
	cmd.Stdin = dump
	cmd.Stderr = stderr

	if err := cmd.Run(); err != nil {
		return fmt.Errorf(
			"error running psql v%s restore: %s",
			version.Value.Version, errorBuffer.String(),
		)
	}

	return nil
}
