package jobs

import (
	"github.com/kamisamamayuri-cyber/pgwarden/internal/database/dbgen"
)

type Service struct {
	dbgen *dbgen.Queries
}

func New(dbgen *dbgen.Queries) *Service {
	return &Service{dbgen: dbgen}
}
