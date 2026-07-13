package api

import (
	"database/sql"
	"errors"
	"net/http"

	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
)

func (h *handlers) getRestorationStatusHandler(c echo.Context) error {
	ctx := c.Request().Context()

	restorationID, err := uuid.Parse(c.Param("restoration_id"))
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": "restoration_id must be a valid UUID",
		})
	}

	status, err := h.servs.RestorationsService.GetRestorationStatus(ctx, restorationID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return c.JSON(http.StatusNotFound, map[string]string{
				"error": "restoration not found",
			})
		}
		return c.JSON(http.StatusInternalServerError, map[string]string{
			"error": err.Error(),
		})
	}

	access := h.accessFrom(c)
	if h.servs.RbacService.Enabled() && !access.CanViewPbwName(status.BackupName) {
		// Same response as a missing restoration: no existence oracle.
		return c.JSON(http.StatusNotFound, map[string]string{
			"error": "restoration not found",
		})
	}

	return c.JSON(http.StatusOK, status)
}
