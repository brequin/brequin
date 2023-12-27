package db

type Quarter struct {
	Code string
	Name string
}

type SubjectArea struct {
	Code string
	Name string
}

type NodeType string

const (
	NodeTypeValue NodeType = "value"
	NodeTypeAnd   NodeType = "and"
	NodeTypeOr    NodeType = "or"
)

type Node struct {
	Id   string
	Type NodeType
}

type Course struct {
	SubjectAreaCode string
	CatalogNumber   string
	Name            string
	NodeId          string
}

type CourseDescription struct {
	SubjectAreaCode string
	CatalogNumber   string
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
