package storage

import (
	"context"
	"fmt"
	"io"

	"github.com/kamisamamayuri-cyber/pgwarden/internal/util/strutil"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/s3"
)

// S3ObjectSize returns the object size in bytes.
func (Client) S3ObjectSize(
	ctx context.Context,
	accessKey, secretKey, region, endpoint, bucketName, key string,
) (int64, error) {
	s3Client, err := createS3Client(ctx, accessKey, secretKey, region, endpoint)
	if err != nil {
		return 0, err
	}

	key = strutil.RemoveLeadingSlash(key)
	head, err := s3Client.HeadObject(ctx, &s3.HeadObjectInput{
		Bucket: aws.String(bucketName),
		Key:    aws.String(key),
	})
	if err != nil {
		return 0, fmt.Errorf("failed to head S3 object: %w", err)
	}

	if head.ContentLength == nil {
		return 0, fmt.Errorf("s3 object has no content length")
	}

	return *head.ContentLength, nil
}

// S3NewReaderAt returns a random-access reader for an S3 object.
// size must come from S3ObjectSize (or equivalent HeadObject).
func (c Client) S3NewReaderAt(
	ctx context.Context,
	accessKey, secretKey, region, endpoint, bucketName, key string,
	size int64,
) io.ReaderAt {
	s3Client, err := createS3Client(ctx, accessKey, secretKey, region, endpoint)
	if err != nil {
		return &readerAtError{err: err}
	}

	key = strutil.RemoveLeadingSlash(key)
	return newS3ReaderAt(ctx, s3Client, bucketName, key, size)
}

type readerAtError struct {
	err error
}

func (r *readerAtError) ReadAt(_ []byte, _ int64) (int, error) {
	return 0, r.err
}
