package jobs

import (
	"context"
	"database/sql"

	"github.com/google/uuid"
	"github.com/kamisamamayuri-cyber/pgwarden/internal/database/dbgen"
	"github.com/kamisamamayuri-cyber/pgwarden/internal/util/paginateutil"
)

const KindBackup = "backup"
const KindFixOwner = "fix_owner"
const KindRestore = "restore"

type PaginateJobsParams struct {
	Page              int
	Limit             int
	DatabaseFilter    uuid.NullUUID
	DestinationFilter uuid.NullUUID
	BackupFilter      uuid.NullUUID
	StatusFilter      string
	HostFilter        string
	KindFilter        string
	NamesFilter       []string
}

type JobRef struct {
	ID   uuid.UUID
	Kind string
}

func (s *Service) PaginateJobs(
	ctx context.Context, params PaginateJobsParams,
) (paginateutil.PaginateResponse, []JobRef, error) {
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

	kindFilter := sql.NullString{}
	if params.KindFilter != "" {
		kindFilter = sql.NullString{Valid: true, String: params.KindFilter}
	}

	if params.NamesFilter != nil && len(params.NamesFilter) == 0 {
		return paginateutil.CreatePaginateResponse(
			paginateutil.PaginateParams{Page: page, Limit: limit},
			0,
		), []JobRef{}, nil
	}

	count, err := s.dbgen.JobsServicePaginateCount(
		ctx, dbgen.JobsServicePaginateCountParams{
			BackupID:      params.BackupFilter,
			DatabaseID:    params.DatabaseFilter,
			DestinationID: params.DestinationFilter,
			Status:        statusFilter,
			Names:         params.NamesFilter,
			Host:          hostFilter,
			Kind:          kindFilter,
		},
	)
	if err != nil {
		return paginateutil.PaginateResponse{}, nil, err
	}

	paginateParams := paginateutil.PaginateParams{Page: page, Limit: limit}
	offset := paginateutil.CreateOffsetFromParams(paginateParams)
	paginateResponse := paginateutil.CreatePaginateResponse(paginateParams, int(count))

	rows, err := s.dbgen.JobsServicePaginateIDs(
		ctx, dbgen.JobsServicePaginateIDsParams{
			BackupID:      params.BackupFilter,
			DatabaseID:    params.DatabaseFilter,
			DestinationID: params.DestinationFilter,
			Status:        statusFilter,
			Names:         params.NamesFilter,
			Host:          hostFilter,
			Kind:          kindFilter,
			Limit:         int32(limit),
			Offset:        int32(offset),
		},
	)
	if err != nil {
		return paginateutil.PaginateResponse{}, nil, err
	}

	refs := make([]JobRef, len(rows))
	for i, r := range rows {
		refs[i] = JobRef{ID: r.ID, Kind: r.Kind}
	}
	return paginateResponse, refs, nil
}

func (s *Service) HasActiveJobs(ctx context.Context) (bool, error) {
	return s.dbgen.JobsServiceHasActive(ctx)
}
