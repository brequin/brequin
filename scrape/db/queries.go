package db

import (
	"context"
	"strconv"
	"strings"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

const listQuarters = `SELECT code, name FROM quarters ORDER BY quarter_rank(code)`
const insertQuarter = `INSERT INTO quarters (code, name) VALUES ($1, $2) ON CONFLICT DO NOTHING`

const listSubjectAreas = `SELECT code, name FROM subject_areas ORDER BY code`
const insertSubjectArea = `INSERT INTO subject_areas (code, name) VALUES ($1, $2) ON CONFLICT DO NOTHING`

const listQuarterSubjectAreas = `SELECT subject_areas.code, subject_areas.name FROM quarter_subject_areas JOIN subject_areas ON quarter_subject_areas.subject_area_code = subject_areas.code WHERE quarter_code = $1 ORDER BY subject_areas.code`
const insertQuarterSubjectArea = `INSERT INTO quarter_subject_areas (quarter_code, subject_area_code) VALUES ($1, $2) ON CONFLICT DO NOTHING`

const insertNode = `INSERT INTO nodes (id, type) VALUES ($1, $2) ON CONFLICT id DO UPDATE type=EXCLUDED.type`
const insertCourse = `INSERT INTO courses (subject_area_code, catalog_number, node_id) VALUES ($1, $2) ON CONFLICT DO NOTHING`
const insertRelation = `INSERT INTO relations (source_id, target_id, enforced, prereq, coreq, minimum_grade) VALUES ($1, $2, $3, $4, $5, $6) ON CONFLICT DO NOTHING`

const insertCourseDetails = `INSERT INTO courses_details (subject_area_code, catalog_number, name, units, level, description) VALUES ($1, $2, $3, $4, $5, $6) ON CONFLICT (subject_area_code, catalog_number) DO UPDATE SET name=EXCLUDED.name, units=EXCLUDED.units, level=EXCLUDED.level, description=EXCLUDED.description`

func FormatOptionalBoolean(b *bool) string {
	if b != nil {
		return strconv.FormatBool(*b)
	}
	return "null"
}

func FormatOptionalString(s *string) string {
	if s != nil {
		return *s
	}
	return ""
}

func insertCallback(ct pgconn.CommandTag) error {
	return nil
}

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

	batch := pgx.Batch{}
	var queuedQueries []*pgx.QueuedQuery

	for _, quarter := range quarters {
		queuedQueries = append(queuedQueries, batch.Queue(insertQuarter, quarter.Code, quarter.Name))
	}

	for _, queuedQuery := range queuedQueries {
		queuedQuery.Exec(insertCallback)
	}

	if err := d.Pool.SendBatch(context.Background(), &batch).Close(); err != nil {
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

	batch := pgx.Batch{}
	var queuedQueries []*pgx.QueuedQuery

	for _, subjectArea := range subjectAreas {
		queuedQueries = append(queuedQueries, batch.Queue(insertSubjectArea, subjectArea.Code, subjectArea.Name))
	}

	for _, queuedQuery := range queuedQueries {
		queuedQuery.Exec(insertCallback)
	}

	if err := d.Pool.SendBatch(context.Background(), &batch).Close(); err != nil {
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

	batch := pgx.Batch{}
	var queuedQueries []*pgx.QueuedQuery

	for _, subjectArea := range subjectAreas {
		queuedQueries = append(queuedQueries, batch.Queue(insertQuarterSubjectArea, quarter.Code, subjectArea.Code))
	}

	for _, queuedQuery := range queuedQueries {
		queuedQuery.Exec(insertCallback)
	}

	if err := d.Pool.SendBatch(context.Background(), &batch).Close(); err != nil {
		return err
	}

	return nil
}

func (d *Database) InsertNodes(nodes []Node) error {
	if len(nodes) == 0 {
		return nil
	}

	batch := pgx.Batch{}
	var queuedQueries []*pgx.QueuedQuery

	for _, node := range nodes {
		queuedQueries = append(queuedQueries, batch.Queue(insertNode, node.Id, node.Type))
	}

	for _, queuedQuery := range queuedQueries {
		queuedQuery.Exec(insertCallback)
	}

	if err := d.Pool.SendBatch(context.Background(), &batch).Close(); err != nil {
		return err
	}

	return nil
}

func (d *Database) InsertCourses(courses []Course) error {
	if len(courses) == 0 {
		return nil
	}

	batch := pgx.Batch{}
	var queuedQueries []*pgx.QueuedQuery

	for _, course := range courses {
		queuedQueries = append(queuedQueries, batch.Queue(insertCourse, course.SubjectAreaCode, course.CatalogNumber, course.NodeId))
	}

	for _, queuedQuery := range queuedQueries {
		queuedQuery.Exec(insertCallback)
	}

	if err := d.Pool.SendBatch(context.Background(), &batch).Close(); err != nil {
		return err
	}

	return nil
}

func (d *Database) InsertRelations(relations []Relation) error {
	if len(relations) == 0 {
		return nil
	}

	batch := pgx.Batch{}
	var queuedQueries []*pgx.QueuedQuery

	for _, relation := range relations {
		enforced := FormatOptionalBoolean(relation.Enforced)
		prereq := FormatOptionalBoolean(relation.Prereq)
		coreq := FormatOptionalBoolean(relation.Coreq)
		minimumGrade := FormatOptionalString(relation.MinimumGrade)

		queuedQueries = append(queuedQueries, batch.Queue(insertRelation, relation.SourceId, relation.TargetId, enforced, prereq, coreq, minimumGrade))
	}

	for _, queuedQuery := range queuedQueries {
		queuedQuery.Exec(insertCallback)
	}

	if err := d.Pool.SendBatch(context.Background(), &batch).Close(); err != nil {
		return err
	}

	return nil
}

func (d *Database) InsertCoursesDetails(coursesDetails []CourseDetails) error {
	if len(coursesDetails) == 0 {
		return nil
	}

	batch := pgx.Batch{}
	var queuedQueries []*pgx.QueuedQuery

	for _, courseDetails := range coursesDetails {
		queuedQueries = append(
			queuedQueries,
			batch.Queue(
				insertCourseDetails,
				courseDetails.SubjectAreaCode,
				courseDetails.CatalogNumber,
				courseDetails.Name,
				courseDetails.Units,
				courseDetails.Level,
				strings.ReplaceAll(courseDetails.Description, "\x00", ""),
			),
		)
	}

	for _, queuedQuery := range queuedQueries {
		queuedQuery.Exec(insertCallback)
	}

	if err := d.Pool.SendBatch(context.Background(), &batch).Close(); err != nil {
		return err
	}

	return nil
}
