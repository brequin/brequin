package db

type Quarter struct {
	Code string
	Name string
}

type SubjectArea struct {
	Code string
	Name string
}

type NodeType int

const (
	NodeTypeValue NodeType = iota
	NodeTypeAnd
	NodeTypeOr
)

type Node struct {
	Id   string
	Type NodeType
}

type Course struct {
	SubjectAreaCode string
	CatalogNumber   string
	NodeId          string
}

type CourseDetails struct {
	SubjectAreaCode string
	CatalogNumber   string
	Name            string
	Description     string
}

type Relation struct {
	SourceId     string
	TargetId     string
	Enforced     *bool
	Prereq       *bool
	Coreq        *bool
	MinimumGrade *string
}
