package postgres

import (
	"archive/zip"
	"bytes"
	"context"
	"os"
	"path/filepath"
	"testing"
)

func TestRestoreFromLocalZip(t *testing.T) {
	t.Helper()

	if os.Getenv("PBW_INTEGRATION_TEST") == "" {
		t.Skip("set PBW_INTEGRATION_TEST=1 to run restore integration test")
	}

	connString := os.Getenv("PBW_TEST_CONN_STRING")
	if connString == "" {
		t.Skip("PBW_TEST_CONN_STRING is required")
	}

	zipPath := writeTestBackupZip(t)
	client := Client{}

	if err := client.RestoreFromLocalZip(context.Background(), PG17, connString, zipPath, nil); err != nil {
		t.Fatalf("RestoreFromLocalZip: %v", err)
	}
}

func TestRestoreFromZipReaderAt(t *testing.T) {
	t.Helper()

	zipPath := writeTestBackupZip(t)
	data, err := os.ReadFile(zipPath)
	if err != nil {
		t.Fatalf("read zip: %v", err)
	}

	readerAt := bytes.NewReader(data)
	if err := restoreFromZipReaderAt(context.Background(), PG17, "invalid", readerAt, int64(len(data)), nil); err == nil {
		t.Fatal("expected psql error for invalid conn string")
	}
}

func writeTestBackupZip(t *testing.T) string {
	t.Helper()

	dir := t.TempDir()
	zipPath := filepath.Join(dir, "backup.zip")

	file, err := os.Create(zipPath)
	if err != nil {
		t.Fatalf("create zip: %v", err)
	}
	defer file.Close()

	zw := zip.NewWriter(file)
	w, err := zw.Create(backupZipDumpFile)
	if err != nil {
		t.Fatalf("create zip entry: %v", err)
	}
	if _, err := w.Write([]byte("SELECT 1;\n")); err != nil {
		t.Fatalf("write dump: %v", err)
	}
	if err := zw.Close(); err != nil {
		t.Fatalf("close zip writer: %v", err)
	}

	return zipPath
}
