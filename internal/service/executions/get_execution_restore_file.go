package executions

import (
	"context"
	"fmt"

	"github.com/kamisamamayuri-cyber/pgwarden/internal/database/dbgen"
	"github.com/google/uuid"
)

// RestoreFile describes where the backup artifact for an execution is stored.
type RestoreFile struct {
	IsLocal  bool
	Path     string
	Bucket   string
	Region   string
	Endpoint string
	AccessKey string
	SecretKey string
}

// GetExecutionRestoreFile returns storage coordinates for streaming restore.
func (s *Service) GetExecutionRestoreFile(
	ctx context.Context, executionID uuid.UUID,
) (RestoreFile, error) {
	data, err := s.dbgen.ExecutionsServiceGetDownloadLinkOrPathData(
		ctx, dbgen.ExecutionsServiceGetDownloadLinkOrPathDataParams{
			ExecutionID:   executionID,
			DecryptionKey: s.env.PBW_ENCRYPTION_KEY,
		},
	)
	if err != nil {
		return RestoreFile{}, err
	}

	if !data.Path.Valid {
		return RestoreFile{}, fmt.Errorf("execution has no file associated")
	}

	file := RestoreFile{
		IsLocal: data.IsLocal,
		Path:    data.Path.String,
	}

	if data.IsLocal {
		file.Path = s.ints.StorageClient.LocalGetFullPath(data.Path.String)
		return file, nil
	}

	if !data.BucketName.Valid || !data.Region.Valid || !data.Endpoint.Valid {
		return RestoreFile{}, fmt.Errorf("execution S3 destination is incomplete")
	}

	file.Bucket = data.BucketName.String
	file.Region = data.Region.String
	file.Endpoint = data.Endpoint.String
	file.AccessKey = data.DecryptedAccessKey
	file.SecretKey = data.DecryptedSecretKey

	return file, nil
}
