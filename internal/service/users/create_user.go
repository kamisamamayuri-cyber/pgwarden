package users

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/kamisamamayuri-cyber/pgwarden/internal/database/dbgen"
	"github.com/kamisamamayuri-cyber/pgwarden/internal/util/cryptoutil"
)

func (s *Service) CreateUser(
	ctx context.Context, params dbgen.UsersServiceCreateUserParams,
) (dbgen.User, error) {
	if !params.Password.Valid || params.Password.String == "" {
		return dbgen.User{}, fmt.Errorf("password is required")
	}

	hash, err := cryptoutil.CreateBcryptHash(params.Password.String)
	if err != nil {
		return dbgen.User{}, err
	}
	params.Password = sql.NullString{String: hash, Valid: true}

	return s.dbgen.UsersServiceCreateUser(ctx, params)
}
