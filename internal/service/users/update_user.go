package users

import (
	"context"

	"github.com/kamisamamayuri-cyber/pgwarden/internal/database/dbgen"
	"github.com/kamisamamayuri-cyber/pgwarden/internal/util/cryptoutil"
)

func (s *Service) UpdateUser(
	ctx context.Context, params dbgen.UsersServiceUpdateUserParams,
) (dbgen.User, error) {
	if params.Password.Valid {
		hashedPassword, err := cryptoutil.CreateBcryptHash(params.Password.String)
		if err != nil {
			return dbgen.User{}, err
		}
		params.Password.String = hashedPassword
	}

	return s.dbgen.UsersServiceUpdateUser(ctx, params)
}
