package auth

import (
	"context"

	"github.com/google/uuid"
	"github.com/kamisamamayuri-cyber/pgwarden/internal/database/dbgen"
)

func (s *Service) CreateSession(
	ctx context.Context,
	userID uuid.UUID,
	ip, userAgent string,
	groups []string,
	fullAccess bool,
) (dbgen.AuthServiceCreateSessionRow, error) {
	if groups == nil {
		groups = []string{}
	}

	return s.dbgen.AuthServiceCreateSession(
		ctx, dbgen.AuthServiceCreateSessionParams{
			UserID:        userID,
			Ip:            ip,
			UserAgent:     userAgent,
			Token:         uuid.NewString(),
			EncryptionKey: s.env.PBW_ENCRYPTION_KEY,
			Groups:        groups,
			FullAccess:    fullAccess,
		},
	)
}
