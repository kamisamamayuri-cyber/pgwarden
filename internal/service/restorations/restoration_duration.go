package restorations

import (
	"database/sql"
	"time"
)

// RestorationDuration returns elapsed time since start, or total time until finish.
func RestorationDuration(startedAt time.Time, finishedAt sql.NullTime) string {
	end := time.Now()
	if finishedAt.Valid {
		end = finishedAt.Time
	}
	if !end.After(startedAt) {
		return "0s"
	}
	return end.Sub(startedAt).Round(time.Second).String()
}
