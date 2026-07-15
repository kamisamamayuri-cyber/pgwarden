package restorations

import (
	"context"
	"database/sql"

	"github.com/kamisamamayuri-cyber/pgwarden/internal/database/dbgen"
	"github.com/kamisamamayuri-cyber/pgwarden/internal/util/paginateutil"
	"github.com/google/uuid"
)

type PaginateRestorationsParams struct {
	Page            int
	Limit           int
	ExecutionFilter uuid.NullUUID
	DatabaseFilter  uuid.NullUUID
	StatusFilter    sql.NullString
	NamesFilter     []string
}

func (s *Service) PaginateRestorations(
	ctx context.Context, params PaginateRestorationsParams,
) (paginateutil.PaginateResponse, []dbgen.RestorationsServicePaginateRestorationsRow, error) {
	page := max(params.Page, 1)
	limit := min(max(params.Limit, 1), 100)

	if params.NamesFilter != nil && len(params.NamesFilter) == 0 {
		return paginateutil.CreatePaginateResponse(
			paginateutil.PaginateParams{Page: page, Limit: limit},
			0,
		), []dbgen.RestorationsServicePaginateRestorationsRow{}, nil
	}

	count, err := s.dbgen.RestorationsServicePaginateRestorationsCount(
		ctx, dbgen.RestorationsServicePaginateRestorationsCountParams{
			ExecutionID: params.ExecutionFilter,
			DatabaseID:  params.DatabaseFilter,
			Status:      params.StatusFilter,
			Names:       params.NamesFilter,
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

	restorations, err := s.dbgen.RestorationsServicePaginateRestorations(
		ctx, dbgen.RestorationsServicePaginateRestorationsParams{
			ExecutionID: params.ExecutionFilter,
			DatabaseID:  params.DatabaseFilter,
			Status:      params.StatusFilter,
			Names:       params.NamesFilter,
			Limit:       int32(params.Limit),
			Offset:      int32(offset),
		},
	)
	if err != nil {
		return paginateutil.PaginateResponse{}, nil, err
	}

	return paginateResponse, restorations, nil
}
