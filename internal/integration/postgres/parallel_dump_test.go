package postgres

import (
	"archive/zip"
	"bytes"
	"context"
	"fmt"
	"hash/crc32"
	"io"
	"strings"
	"testing"

	"github.com/klauspost/compress/flate"
)

func TestWriteRawEntryRoundtrip(t *testing.T) {
	payloadSmall := strings.Repeat("INSERT INTO t VALUES (1, 'abc');\n", 1000)
	payloadBig := strings.Repeat("COPY line with some data\t42\t3.14\n", 50000)

	var buf bytes.Buffer
	zw := zip.NewWriter(&buf)
	zw.RegisterCompressor(zip.Deflate, func(out io.Writer) (io.WriteCloser, error) {
		return flate.NewWriter(out, 3)
	})

	w, err := zw.CreateHeader(&zip.FileHeader{Name: preDataFileName, Method: zip.Deflate})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := w.Write([]byte("CREATE TABLE t (a int);\n")); err != nil {
		t.Fatal(err)
	}

	for name, payload := range map[string]string{
		"20-data-00001-public.small.sql": payloadSmall,
		"20-data-00002-public.big.sql":   payloadBig,
	} {
		e := newRawZipEntry(context.Background(), name)
		go func(data string) {
			crcW, compCount := newTestCompressor(t, e, data)
			e.crc = crcW
			e.rawSize = uint64(len(data))
			e.compSize = compCount
			close(e.chunks)
			close(e.done)
		}(payload)
		if err := writeRawEntry(zw, e); err != nil {
			t.Fatalf("writeRawEntry(%s): %v", name, err)
		}
	}

	if err := writeParallelDumpMeta(zw, "00000003-test-1", 2, []string{
		preDataFileName,
		"20-data-00001-public.small.sql",
		"20-data-00002-public.big.sql",
	}, map[string][]string{
		"20-data-00001-public.small.sql": {"public.small"},
		"20-data-00002-public.big.sql":   {"public.big"},
	}); err != nil {
		t.Fatal(err)
	}
	if err := zw.Close(); err != nil {
		t.Fatal(err)
	}

	zr, err := zip.NewReader(bytes.NewReader(buf.Bytes()), int64(buf.Len()))
	if err != nil {
		t.Fatalf("reading archive back: %v", err)
	}

	want := map[string]string{
		"20-data-00001-public.small.sql": payloadSmall,
		"20-data-00002-public.big.sql":   payloadBig,
		preDataFileName:                  "CREATE TABLE t (a int);\n",
	}
	found := map[string]bool{}
	for _, f := range zr.File {
		expected, ok := want[f.Name]
		if !ok {
			continue
		}
		rc, err := f.Open()
		if err != nil {
			t.Fatalf("open %s: %v", f.Name, err)
		}
		got, err := io.ReadAll(rc)
		rc.Close()
		if err != nil {
			t.Fatalf("read %s: %v", f.Name, err)
		}
		if string(got) != expected {
			t.Fatalf("%s: content mismatch (got %d bytes, want %d)", f.Name, len(got), len(expected))
		}
		found[f.Name] = true
	}
	for name := range want {
		if !found[name] {
			t.Fatalf("%s missing from archive", name)
		}
	}
}

func newTestCompressor(t *testing.T, e *rawZipEntry, data string) (uint32, uint64) {
	t.Helper()
	crcW := testCRC(data)
	compCount := &countingChunkWriter{entry: e}
	comp, err := flate.NewWriter(compCount, 3)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := io.Copy(comp, strings.NewReader(data)); err != nil {
		t.Fatal(err)
	}
	if err := comp.Close(); err != nil {
		t.Fatal(err)
	}
	return crcW, compCount.count
}

func testCRC(data string) uint32 {
	h := crc32.NewIEEE()
	h.Write([]byte(data))
	return h.Sum32()
}

func TestPlanDataJobs(t *testing.T) {
	gb := int64(1024 * 1024 * 1024)
	mb := int64(1024 * 1024)

	tables := []parallelDumpTableInfo{
		{Pattern: "public.giant1", Bytes: 30 * gb},
		{Pattern: "public.giant2", Bytes: 400 * mb},
		{Pattern: "public.mid1", Bytes: 200 * mb},
		{Pattern: "public.mid2", Bytes: 100 * mb},
		{Pattern: "public.small1", Bytes: 10 * mb},
		{Pattern: "public.small2", Bytes: 1 * mb},
	}

	jobs := planDataJobs(tables)

	if len(jobs) != 4 {
		t.Fatalf("expected 4 jobs, got %d: %+v", len(jobs), jobs)
	}
	for i, want := range [][]string{
		{"public.giant1"},
		{"public.giant2"},
		{"public.mid1"},
		{"public.mid2", "public.small1", "public.small2"},
	} {
		if len(jobs[i].Patterns) != len(want) {
			t.Fatalf("job %d: expected %v, got %v", i, want, jobs[i].Patterns)
		}
		for k, p := range want {
			if jobs[i].Patterns[k] != p {
				t.Fatalf("job %d: expected %v, got %v", i, want, jobs[i].Patterns)
			}
		}
	}

	many := make([]parallelDumpTableInfo, 500)
	for i := range many {
		many[i] = parallelDumpTableInfo{
			Pattern: fmt.Sprintf("public.t%03d", i), Bytes: 1 * mb,
		}
	}
	jobs = planDataJobs(many)
	if len(jobs) != 3 {
		t.Fatalf("expected 3 batch jobs for 500 tiny tables (cap 200), got %d", len(jobs))
	}
	total := 0
	for _, j := range jobs {
		if len(j.Patterns) > parallelDumpBatchMaxCount {
			t.Fatalf("batch exceeds max count: %d", len(j.Patterns))
		}
		total += len(j.Patterns)
	}
	if total != 500 {
		t.Fatalf("tables lost in batching: %d != 500", total)
	}
}
