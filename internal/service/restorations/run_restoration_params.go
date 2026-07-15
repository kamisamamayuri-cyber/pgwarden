package restorations

import (
	"github.com/google/uuid"
)

// RestoreTargetOptions enables preset restore preparation on the target PostgreSQL.
type RestoreTargetOptions struct {
	DatabaseName string
	Owner        string
}

// RunRestorationParams configures a restore job.
type RunRestorationParams struct {
	ExecutionID           uuid.UUID
	DatabaseID            uuid.NullUUID
	ConnString            string
	ExistingRestorationID uuid.NullUUID
	Target                *RestoreTargetOptions
}
