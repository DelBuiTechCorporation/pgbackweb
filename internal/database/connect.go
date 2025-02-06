package database

import (
	"database/sql"

	"github.com/eduardolat/pgbackweb/internal/config"
	"github.com/eduardolat/pgbackweb/internal/logger"
	_ "github.com/lib/pq"
)

func Connect(env config.Env) *sql.DB {
	db, err := sql.Open("postgres", env.PBW_POSTGRES_CONN_STRING)
	if err != nil {
		logger.FatalError(
			"could not connect to DB",
			logger.KV{
				"error": err,
			},
		)
	}

	err = db.Ping()
	if err != nil {
		logger.FatalError(
			"could not ping DB",
			logger.KV{
				"error": err,
			},
		)
	}

	db.SetMaxOpenConns(10)
	logger.Info("connected to DB")

	return db
}
