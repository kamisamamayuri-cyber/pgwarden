package postgres

import (
	"archive/zip"
	"bytes"
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"

	"github.com/kamisamamayuri-cyber/pgwarden/internal/integration/storage"
)

const backupZipDumpFile = "dump.sql"

// RestoreFromLocalZip streams a local backup ZIP into psql without a temp copy.
func (Client) RestoreFromLocalZip(
	ctx context.Context, version PGVersion, connString, zipPath string, log io.Writer,
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

	return restoreFromZipReaderAt(ctx, version, connString, file, stat.Size(), log)
}

// RestoreFromS3Zip streams a backup ZIP from S3 into psql without writing it to disk.
func (Client) RestoreFromS3Zip(
	ctx context.Context,
	storageClient *storage.Client,
	version PGVersion,
	connString string,
	accessKey, secretKey, region, endpoint, bucketName, key string,
	log io.Writer,
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

	return restoreFromZipReaderAt(ctx, version, connString, readerAt, size, log)
}

func restoreFromZipReaderAt(
	ctx context.Context, version PGVersion, connString string,
	r io.ReaderAt, size int64, log io.Writer,
) error {
	zr, err := zip.NewReader(r, size)
	if err != nil {
		return fmt.Errorf("error opening ZIP reader: %w", err)
	}

	var dumpFile *zip.File
	for _, f := range zr.File {
		if f.Name == backupZipDumpFile {
			dumpFile = f
			break
		}
	}
	if dumpFile == nil {
		return fmt.Errorf("%s not found in backup ZIP", backupZipDumpFile)
	}

	rc, err := dumpFile.Open()
	if err != nil {
		return fmt.Errorf("error opening %s in ZIP: %w", backupZipDumpFile, err)
	}
	defer rc.Close()

	return runPSQLRestore(ctx, version, connString, sanitizeDumpReader(rc), log)
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
