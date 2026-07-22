package restorations

import (
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/kamisamamayuri-cyber/pgwarden/internal/util/pathutil"
)

const (
	backupModeLatest = "latest"
	backupModeDated  = "dated"
)

func restoreEnvironmentTitleLatest(preset RestorePreset, target RestoreTarget) string {
	return fmt.Sprintf(
		"%s → %s (latest backup from prod)",
		preset.Title,
		target.Environment,
	)
}

func restoreEnvironmentTitleDated(
	preset RestorePreset, target RestoreTarget, finishedAt time.Time,
) string {
	return fmt.Sprintf(
		"%s → %s (backup %s UTC from prod)",
		preset.Title,
		target.Environment,
		finishedAt.UTC().Format("2006-01-02 15:04"),
	)
}

func restoreBackupsListRequest(id string) string {
	return "GET " + pathutil.BuildPath("/api/v1/restores/"+id+"/backups")
}

func restoreTargetsListRequest(id string) string {
	return "GET " + pathutil.BuildPath("/api/v1/restores/"+id+"/restore")
}

func restorePostRequest(id string) string {
	return "POST " + pathutil.BuildPath("/api/v1/restores/"+id+"/restore")
}

func restoreBackupDownloadRequest(databaseID, executionID string) string {
	return "GET " + pathutil.BuildPath("/api/v1/restores/"+databaseID+"/backups/"+executionID)
}

func restorationStatusRequest(restorationID uuid.UUID) string {
	return "GET " + pathutil.BuildPath("/api/v1/restorations/"+restorationID.String())
}

func restoreRequestBody(environment string, executionID string) map[string]string {
	body := map[string]string{
		"environment": environment,
	}
	if executionID != "" {
		body["execution_id"] = executionID
	}
	return body
}

func restoreRequestBodyWithFinishedAtDate(environment, finishedAt string) map[string]string {
	return map[string]string{
		"environment": environment,
		"finished_at": finishedAt,
	}
}

func restoreActionRequestBodyNote(presetID string) string {
	return restorePostRequest(presetID) +
		", Content-Type: application/x-www-form-urlencoded. " +
		"Form fields without execution_id and finished_at — restore the latest successful dump from prod. " +
		"List of dumps: " + restoreBackupsListRequest(presetID) + ". " +
		"For a specific dump, add execution_id (UUID from .../backups) " +
		"or finished_at (YYYY-MM-DD or RFC3339)."
}
