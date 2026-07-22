package executions

import (
	"context"
	"database/sql"

	"github.com/google/uuid"
	"github.com/kamisamamayuri-cyber/pgwarden/internal/database/dbgen"
	"github.com/kamisamamayuri-cyber/pgwarden/internal/util/paginateutil"
)

type PaginateExecutionsParams struct {
	Page              int
	Limit             int
	DatabaseFilter    uuid.NullUUID
	DestinationFilter uuid.NullUUID
	BackupFilter      uuid.NullUUID
	StatusFilter      string
	NamesFilter       []string
	HostFilter        string
}

func (s *Service) PaginateExecutions(
	ctx context.Context, params PaginateExecutionsParams,
) (paginateutil.PaginateResponse, []dbgen.ExecutionsServicePaginateExecutionsRow, error) {
	page := max(params.Page, 1)
	limit := min(max(params.Limit, 1), 100)

	statusFilter := sql.NullString{}
	if params.StatusFilter != "" {
		statusFilter = sql.NullString{Valid: true, String: params.StatusFilter}
	}

	hostFilter := sql.NullString{}
	if params.HostFilter != "" {
		hostFilter = sql.NullString{Valid: true, String: params.HostFilter}
	}

	if params.NamesFilter != nil && len(params.NamesFilter) == 0 {
		return paginateutil.CreatePaginateResponse(
			paginateutil.PaginateParams{Page: page, Limit: limit},
			0,
		), []dbgen.ExecutionsServicePaginateExecutionsRow{}, nil
	}

	count, err := s.dbgen.ExecutionsServicePaginateExecutionsCount(
		ctx, dbgen.ExecutionsServicePaginateExecutionsCountParams{
			BackupID:      params.BackupFilter,
			DatabaseID:    params.DatabaseFilter,
			DestinationID: params.DestinationFilter,
			Status:        statusFilter,
			Names:         params.NamesFilter,
			Host:          hostFilter,
		},
	)
	if err != nil {
		return paginateutil.PaginateResponse{}, nil, err
	}

	paginateParams := paginateutil.PaginateParams{
		Page:  page,
		Limit: limit,
	}
	offset := paginateutil.CreateOffsetFromParams(paginateParams)
	paginateResponse := paginateutil.CreatePaginateResponse(paginateParams, int(count))

	executions, err := s.dbgen.ExecutionsServicePaginateExecutions(
		ctx, dbgen.ExecutionsServicePaginateExecutionsParams{
			BackupID:      params.BackupFilter,
			DatabaseID:    params.DatabaseFilter,
			DestinationID: params.DestinationFilter,
			Status:        statusFilter,
			Names:         params.NamesFilter,
			Host:          hostFilter,
			Limit:         int32(params.Limit),
			Offset:        int32(offset),
		},
	)
	if err != nil {
		return paginateutil.PaginateResponse{}, nil, err
	}

	return paginateResponse, executions, nil
}
