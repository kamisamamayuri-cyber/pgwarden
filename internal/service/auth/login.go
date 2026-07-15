package auth

import (
	"context"
	"fmt"

	"github.com/kamisamamayuri-cyber/pgwarden/internal/database/dbgen"
	"github.com/kamisamamayuri-cyber/pgwarden/internal/util/cryptoutil"
)

func (s *Service) Login(
	ctx context.Context, email, password, ip, userAgent string,
) (dbgen.AuthServiceCreateSessionRow, error) {
	user, err := s.dbgen.AuthServiceLoginGetUserByEmail(ctx, email)
	if err != nil {
		return dbgen.AuthServiceCreateSessionRow{}, err
	}

	if !user.Password.Valid || user.Password.String == "" {
		return dbgen.AuthServiceCreateSessionRow{}, fmt.Errorf("invalid password")
	}

	if err := cryptoutil.VerifyBcryptHash(password, user.Password.String); err != nil {
		return dbgen.AuthServiceCreateSessionRow{}, fmt.Errorf("invalid password")
	}

	return s.CreateSession(ctx, user.ID, ip, userAgent, nil, true)
}
