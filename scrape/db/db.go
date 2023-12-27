package db

import (
	"fmt"
	"strconv"

	"github.com/jackc/pgx/v5/pgxpool"
)

type Database struct {
	Pool *pgxpool.Pool
}

func Flag(b bool) byte {
	return strconv.FormatBool(b)[0]
}

func ValueNodeId(subjectAreaCode, catalogNumber string) string {
	const idTemplate = "%v#%v"
	return fmt.Sprintf(idTemplate, subjectAreaCode, catalogNumber)
}
