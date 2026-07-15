package users

import (
	"context"

	"github.com/kamisamamayuri-cyber/pgwarden/internal/database/dbgen"
)

func (s *Service) CreateSsoUser(
	ctx context.Context, params dbgen.UsersServiceCreateSsoUserParams,
) (dbgen.User, error) {
	return s.dbgen.UsersServiceCreateSsoUser(ctx, params)
}
