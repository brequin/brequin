package db

import "github.com/jackc/pgx/v5/pgxpool"

type Database struct {
	Pool *pgxpool.Pool
}
