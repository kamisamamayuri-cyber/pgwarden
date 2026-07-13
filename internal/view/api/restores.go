package api

import (
	"database/sql"
	"errors"
	"mime"
	"net/http"
	"strconv"
	"strings"

	"github.com/kamisamamayuri-cyber/pgwarden/internal/service/restorations"
	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
)

func (h *handlers) listRestorePresetsHandler(c echo.Context) error {
	ctx := c.Request().Context()
	access := h.accessFrom(c)

	catalog, err := h.servs.RestorationsService.ListRestoreCatalog(ctx)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{
			"error": err.Error(),
		})
	}

	if h.servs.RbacService.Enabled() {
		catalog = filterRestoreCatalog(catalog, access.CanViewPreset)
	}

	return c.JSON(http.StatusOK, catalog)
}

func (h *handlers) getRestoreDatabaseHandler(c echo.Context) error {
	ctx := c.Request().Context()
	id := c.Param("id")
	access := h.accessFrom(c)

	if h.servs.RbacService.Enabled() && !access.CanViewPreset(id) {
		// Respond as if the preset does not exist so callers cannot probe
		// which presets are configured (IDOR oracle).
		return c.JSON(http.StatusNotFound, map[string]string{
			"error": "restore database not found",
		})
	}

	database, err := h.servs.RestorationsService.GetRestoreDatabase(ctx, id)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return c.JSON(http.StatusNotFound, map[string]string{
				"error": "restore database not found",
			})
		}
		return c.JSON(http.StatusInternalServerError, map[string]string{
			"error": err.Error(),
		})
	}

	return c.JSON(http.StatusOK, database)
}

func (h *handlers) listRestoreBackupsHandler(c echo.Context) error {
	ctx := c.Request().Context()
	id := c.Param("id")
	access := h.accessFrom(c)

	if h.servs.RbacService.Enabled() && !access.CanViewPreset(id) {
		// Respond as if the preset does not exist so callers cannot probe
		// which presets are configured (IDOR oracle).
		return c.JSON(http.StatusNotFound, map[string]string{
			"error": "restore database not found",
		})
	}

	limit := 100
	if raw := strings.TrimSpace(c.QueryParam("limit")); raw != "" {
		parsed, err := strconv.Atoi(raw)
		if err != nil || parsed < 1 {
			return c.JSON(http.StatusBadRequest, map[string]string{
				"error": "limit must be a positive integer",
			})
		}
		if parsed > 500 {
			return c.JSON(http.StatusBadRequest, map[string]string{
				"error": "limit must not exceed 500",
			})
		}
		limit = parsed
	}

	backups, err := h.servs.RestorationsService.ListRestoreBackups(ctx, id, limit)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return c.JSON(http.StatusNotFound, map[string]string{
				"error": "restore database not found",
			})
		}
		return c.JSON(http.StatusInternalServerError, map[string]string{
			"error": err.Error(),
		})
	}

	return c.JSON(http.StatusOK, backups)
}

func (h *handlers) getRestoreBackupDownloadHandler(c echo.Context) error {
	ctx := c.Request().Context()
	id := c.Param("id")
	access := h.accessFrom(c)

	if h.servs.RbacService.Enabled() && !access.CanViewPreset(id) {
		// Respond as if the preset does not exist so callers cannot probe
		// which presets are configured (IDOR oracle).
		return c.JSON(http.StatusNotFound, map[string]string{
			"error": "restore database not found",
		})
	}

	executionID, err := uuid.Parse(c.Param("execution_id"))
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": "execution_id must be a valid UUID",
		})
	}

	download, err := h.servs.RestorationsService.GetRestoreBackupDownload(ctx, id, executionID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return c.JSON(http.StatusNotFound, map[string]string{
				"error": "backup not found",
			})
		}
		return c.JSON(http.StatusInternalServerError, map[string]string{
			"error": err.Error(),
		})
	}

	return c.JSON(http.StatusOK, download)
}

func (h *handlers) getRestoreTargetsHandler(c echo.Context) error {
	ctx := c.Request().Context()
	id := c.Param("id")
	access := h.accessFrom(c)

	if h.servs.RbacService.Enabled() && !access.CanViewPreset(id) {
		// Respond as if the preset does not exist so callers cannot probe
		// which presets are configured (IDOR oracle).
		return c.JSON(http.StatusNotFound, map[string]string{
			"error": "restore database not found",
		})
	}

	targets, err := h.servs.RestorationsService.GetRestoreTargets(ctx, id)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return c.JSON(http.StatusNotFound, map[string]string{
				"error": "restore database not found",
			})
		}
		return c.JSON(http.StatusInternalServerError, map[string]string{
			"error": err.Error(),
		})
	}

	return c.JSON(http.StatusOK, targets)
}

func (h *handlers) runRestoreHandler(c echo.Context) error {
	ctx := c.Request().Context()
	id := c.Param("id")
	access := h.accessFrom(c)

	if h.servs.RbacService.Enabled() {
		if !access.CanViewPreset(id) {
			// Same response as a missing preset: no existence oracle.
			return c.JSON(http.StatusNotFound, map[string]string{
				"error": "restore database not found",
			})
		}
		if !access.CanExecutePreset(id) {
			return c.JSON(http.StatusForbidden, map[string]string{"error": "forbidden"})
		}
	}

	params, err := bindRestoreParams(c)
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": err.Error(),
		})
	}

	result, err := h.servs.RestorationsService.StartRestore(ctx, id, params)
	if err != nil {
		switch {
		case errors.Is(err, sql.ErrNoRows):
			return c.JSON(http.StatusNotFound, map[string]string{
				"error": "restore database not found",
			})
		case errors.Is(err, restorations.ErrRestoreEnvironmentRequired):
			return c.JSON(http.StatusBadRequest, map[string]string{
				"error": err.Error(),
			})
		case errors.Is(err, restorations.ErrRestoreEnvironmentNotFound):
			return c.JSON(http.StatusNotFound, map[string]string{
				"error": err.Error(),
			})
		case errors.Is(err, restorations.ErrRestoreActionNotReady):
			return c.JSON(http.StatusConflict, map[string]string{
				"error": err.Error(),
			})
		case errors.Is(err, restorations.ErrRestoreAlreadyRunning):
			return c.JSON(http.StatusConflict, map[string]string{
				"error": err.Error(),
			})
		case errors.Is(err, restorations.ErrRestoreBackupNotFound):
			return c.JSON(http.StatusNotFound, map[string]string{
				"error": err.Error(),
			})
		default:
			return c.JSON(http.StatusInternalServerError, map[string]string{
				"error": err.Error(),
			})
		}
	}

	return c.JSON(http.StatusAccepted, result)
}

type restoreBody struct {
	Environment string `form:"environment"`
	ExecutionID string `form:"execution_id"`
	FinishedAt  string `form:"finished_at"`
}

func bindRestoreParams(c echo.Context) (restorations.RestoreStartParams, error) {
	params := restorations.RestoreStartParams{}

	mediaType, _, err := mime.ParseMediaType(c.Request().Header.Get("Content-Type"))
	if err != nil || mediaType != "application/x-www-form-urlencoded" {
		return params, errors.New("Content-Type must be application/x-www-form-urlencoded")
	}

	if err := c.Request().ParseForm(); err != nil {
		return params, err
	}

	var body restoreBody
	if err := c.Bind(&body); err != nil {
		return params, err
	}

	body.Environment = strings.TrimSpace(body.Environment)
	body.ExecutionID = strings.TrimSpace(body.ExecutionID)
	body.FinishedAt = strings.TrimSpace(body.FinishedAt)

	if body.Environment == "" {
		return params, restorations.ErrRestoreEnvironmentRequired
	}
	params.Environment = body.Environment

	if body.ExecutionID != "" && body.FinishedAt != "" {
		return params, errors.New("use either execution_id or finished_at, not both")
	}

	if body.ExecutionID != "" {
		executionID, err := uuid.Parse(body.ExecutionID)
		if err != nil {
			return params, errors.New("execution_id must be a valid UUID")
		}
		params.ExecutionID = &executionID
		return params, nil
	}

	if body.FinishedAt != "" {
		finishedAt, err := restorations.ParseRestoreFinishedAt(body.FinishedAt)
		if err != nil {
			return params, err
		}
		params.FinishedAt = &finishedAt
	}

	return params, nil
}
