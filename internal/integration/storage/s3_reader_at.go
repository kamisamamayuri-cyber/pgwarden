package storage

import (
	"context"
	"fmt"
	"io"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/s3"
)

// s3ReaderAt implements io.ReaderAt over a single S3 object using Range requests.
// archive/zip needs random access to read the central directory at the end of the file.
type s3ReaderAt struct {
	// ctx is stored because io.ReaderAt has no context parameter; cancelling
	// it aborts in-flight range requests of a running restore.
	ctx    context.Context
	client *s3.Client
	bucket string
	key    string
	size   int64
}

func newS3ReaderAt(
	ctx context.Context, client *s3.Client, bucketName, key string, size int64,
) *s3ReaderAt {
	return &s3ReaderAt{
		ctx:    ctx,
		client: client,
		bucket: bucketName,
		key:    key,
		size:   size,
	}
}

func (r *s3ReaderAt) ReadAt(p []byte, off int64) (int, error) {
	if off < 0 {
		return 0, fmt.Errorf("negative offset: %d", off)
	}
	if off >= r.size {
		return 0, io.EOF
	}

	toRead := int64(len(p))
	maxEnd := r.size - 1
	end := off + toRead - 1
	if end > maxEnd {
		end = maxEnd
	}

	rangeHeader := fmt.Sprintf("bytes=%d-%d", off, end)
	out, err := r.client.GetObject(r.ctx, &s3.GetObjectInput{
		Bucket: aws.String(r.bucket),
		Key:    aws.String(r.key),
		Range:  aws.String(rangeHeader),
	})
	if err != nil {
		return 0, fmt.Errorf("s3 GetObject range %s: %w", rangeHeader, err)
	}
	defer out.Body.Close()

	want := int(end - off + 1)
	n, err := io.ReadFull(out.Body, p[:want])
	if err == io.EOF || err == io.ErrUnexpectedEOF {
		return n, io.EOF
	}
	if err != nil {
		return n, fmt.Errorf("reading s3 object body: %w", err)
	}
	return n, nil
}
