package backups

import (
	"context"
	"database/sql"

	"github.com/kamisamamayuri-cyber/pgwarden/internal/database/dbgen"
	"github.com/kamisamamayuri-cyber/pgwarden/internal/util/paginateutil"
)

type PaginateBackupsParams struct {
	Page        int
	Limit       int
	HostFilter  string
	NamesFilter []string
}

func (s *Service) PaginateBackups(
	ctx context.Context, params PaginateBackupsParams,
) (paginateutil.PaginateResponse, []dbgen.BackupsServicePaginateBackupsRow, error) {
	page := max(params.Page, 1)
	limit := min(max(params.Limit, 1), 100)

	hostFilter := sql.NullString{}
	if params.HostFilter != "" {
		hostFilter = sql.NullString{Valid: true, String: params.HostFilter}
	}

	if params.NamesFilter != nil && len(params.NamesFilter) == 0 {
		return paginateutil.CreatePaginateResponse(
			paginateutil.PaginateParams{Page: page, Limit: limit},
			0,
		), []dbgen.BackupsServicePaginateBackupsRow{}, nil
	}

	countParams := dbgen.BackupsServicePaginateBackupsCountParams{
		Host:          hostFilter,
		EncryptionKey: s.env.PBW_ENCRYPTION_KEY,
		Names:         params.NamesFilter,
	}

	count, err := s.dbgen.BackupsServicePaginateBackupsCount(ctx, countParams)
	if err != nil {
		return paginateutil.PaginateResponse{}, nil, err
	}

	paginateParams := paginateutil.PaginateParams{
		Page:  page,
		Limit: limit,
	}
	offset := paginateutil.CreateOffsetFromParams(paginateParams)
	paginateResponse := paginateutil.CreatePaginateResponse(paginateParams, int(count))

	backups, err := s.dbgen.BackupsServicePaginateBackups(
		ctx, dbgen.BackupsServicePaginateBackupsParams{
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

	return paginateResponse, backups, nil
}
