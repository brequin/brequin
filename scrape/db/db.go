package db

import (
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"
)

type Database struct {
	Pool *pgxpool.Pool
}

func ValueNodeId(subjectAreaCode, catalogNumber string) string {
	const idTemplate = "%v#%v"
	return fmt.Sprintf(idTemplate, subjectAreaCode, catalogNumber)
}
