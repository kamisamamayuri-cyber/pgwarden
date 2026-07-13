package databases

import (
	"context"
	"database/sql"

	"github.com/kamisamamayuri-cyber/pgwarden/internal/database/dbgen"
	"github.com/kamisamamayuri-cyber/pgwarden/internal/util/paginateutil"
)

type PaginateDatabasesParams struct {
	Page       int
	Limit      int
	HostFilter string
	NamesFilter []string
}

func (s *Service) PaginateDatabases(
	ctx context.Context, params PaginateDatabasesParams,
) (paginateutil.PaginateResponse, []dbgen.DatabasesServicePaginateDatabasesRow, error) {
	page := max(params.Page, 1)
	limit := min(max(params.Limit, 1), 100)

	hostFilter := sql.NullString{}
	if params.HostFilter != "" {
		hostFilter = sql.NullString{Valid: true, String: params.HostFilter}
	}

	if params.NamesFilter != nil && len(params.NamesFilter) == 0 {
		return paginateutil.CreatePaginateResponse(
			paginateutil.PaginateParams{Page: max(params.Page, 1), Limit: min(max(params.Limit, 1), 100)},
			0,
		), []dbgen.DatabasesServicePaginateDatabasesRow{}, nil
	}

	countParams := dbgen.DatabasesServicePaginateDatabasesCountParams{
		Host:          hostFilter,
		EncryptionKey: s.env.PBW_ENCRYPTION_KEY,
		Names:         params.NamesFilter,
	}

	count, err := s.dbgen.DatabasesServicePaginateDatabasesCount(ctx, countParams)
	if err != nil {
		return paginateutil.PaginateResponse{}, nil, err
	}

	paginateParams := paginateutil.PaginateParams{
		Page:  page,
		Limit: limit,
	}
	offset := paginateutil.CreateOffsetFromParams(paginateParams)
	paginateResponse := paginateutil.CreatePaginateResponse(paginateParams, int(count))

	databases, err := s.dbgen.DatabasesServicePaginateDatabases(
		ctx, dbgen.DatabasesServicePaginateDatabasesParams{
			Host:          hostFilter,
			EncryptionKey: s.env.PBW_ENCRYPTION_KEY,
			Names:         params.NamesFilter,
			Limit:         int32(params.Limit),
			Offset:        int32(offset),
		},
	)
	if err != nil {
		return paginateutil.PaginateResponse{}, nil, err
	}

	return paginateResponse, databases, nil
}
