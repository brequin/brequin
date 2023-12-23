package db

import (
	"context"
	"fmt"
	"strings"
)

const listQuarters = `SELECT code, name FROM quarters ORDER BY quarter_rank(code)`
const insertQuarters = `INSERT INTO quarters (code, name) VALUES`

const listSubjectAreas = `SELECT code, name FROM subject_areas ORDER BY code`
const insertSubjectAreas = `INSERT INTO subject_areas (code, name) VALUES`

const listQuarterSubjectAreas = `SELECT subject_areas.code, subject_areas.name FROM quarter_subject_areas JOIN subject_areas ON quarter_subject_areas.subject_area_code = subject_areas.code WHERE quarter_code = $1 ORDER BY subject_areas.code`
const insertQuarterSubjectAreas = `INSERT INTO quarter_subject_areas (quarter_code, subject_area_code) VALUES`

const conflictDoNothing = `ON CONFLICT DO NOTHING`

func (d *Database) ListQuarters() ([]Quarter, error) {
	sql := listQuarters
	rows, err := d.Pool.Query(context.Background(), sql)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var quarters []Quarter
	for rows.Next() {
		var quarter Quarter
		if err := rows.Scan(&quarter.Code, &quarter.Name); err != nil {
			return nil, err
		}
		quarters = append(quarters, quarter)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return quarters, nil
}

func (d *Database) InsertQuarters(quarters []Quarter) error {
	if len(quarters) == 0 {
		return nil
	}

	var b strings.Builder
	separator := ""
	for _, quarter := range quarters {
		fmt.Fprintf(&b, "%v('%v', '%v')", separator, quarter.Code, quarter.Name)
		separator = ", "
	}
	sql := strings.Join([]string{insertQuarters, b.String(), conflictDoNothing}, " ")

	if _, err := d.Pool.Exec(context.Background(), sql); err != nil {
		return err
	}

	return nil
}

func (d *Database) ListSubjectAreas() ([]SubjectArea, error) {
	sql := listSubjectAreas
	rows, err := d.Pool.Query(context.Background(), sql)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var subjectAreas []SubjectArea
	for rows.Next() {
		var subjectArea SubjectArea
		if err := rows.Scan(&subjectArea.Code, &subjectArea.Name); err != nil {
			return nil, err
		}
		subjectAreas = append(subjectAreas, subjectArea)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return subjectAreas, nil
}

func (d *Database) InsertSubjectAreas(subjectAreas []SubjectArea) error {
	if len(subjectAreas) == 0 {
		return nil
	}

	var b strings.Builder
	separator := ""
	for _, subjectArea := range subjectAreas {
		fmt.Fprintf(&b, "%v ('%v', '%v')", separator, subjectArea.Code, subjectArea.Name)
		separator = ","
	}
	sql := strings.Join([]string{insertSubjectAreas, b.String(), conflictDoNothing}, " ")

	if _, err := d.Pool.Exec(context.Background(), sql); err != nil {
		return err
	}

	return nil
}

func (d *Database) ListQuarterSubjectAreas(quarter Quarter) ([]SubjectArea, error) {
	sql := listQuarterSubjectAreas
	rows, err := d.Pool.Query(context.Background(), sql, quarter.Code)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var subjectAreas []SubjectArea
	for rows.Next() {
		var subjectArea SubjectArea
		if err := rows.Scan(&subjectArea.Code, &subjectArea.Name); err != nil {
			return nil, err
		}
		subjectAreas = append(subjectAreas, subjectArea)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return subjectAreas, nil
}

func (d *Database) InsertQuarterSubjectAreas(quarter Quarter, subjectAreas []SubjectArea) error {
	if len(subjectAreas) == 0 {
		return nil
	}

	var b strings.Builder
	separator := ""
	for _, subjectArea := range subjectAreas {
		fmt.Fprintf(&b, "%v ('%v', '%v')", separator, quarter.Code, subjectArea.Code)
		separator = ","
	}
	sql := strings.Join([]string{insertQuarterSubjectAreas, b.String(), conflictDoNothing}, " ")

	if _, err := d.Pool.Exec(context.Background(), sql); err != nil {
		return err
	}

	return nil
}
