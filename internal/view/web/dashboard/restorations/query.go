package restorations

import (
	"fmt"

	"github.com/google/uuid"
	"github.com/kamisamamayuri-cyber/pgwarden/internal/util/pathutil"
)

func buildRestorationDetailsURL(restorationID uuid.UUID) string {
	return pathutil.BuildPath(
		fmt.Sprintf("/dashboard/restorations/%s/details", restorationID),
	)
}
