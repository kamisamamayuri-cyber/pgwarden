package auth

import (
	"time"

	"github.com/kamisamamayuri-cyber/pgwarden/internal/config"
	"github.com/kamisamamayuri-cyber/pgwarden/internal/database/dbgen"
)

const (
	maxSessionAge = time.Hour * 12
)

type Service struct {
	env   config.Env
	dbgen *dbgen.Queries
}

func New(env config.Env, dbgen *dbgen.Queries) *Service {
	return &Service{
		env:   env,
		dbgen: dbgen,
	}
}
