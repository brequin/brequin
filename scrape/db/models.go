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
	NodeTypeValue  NodeType = "value"
	NodeTypeSwitch NodeType = "switch"
)

type Node struct {
	Id   string
	Type NodeType
}

type Course struct {
	NodeId            string
	SubjectAreaCode   string
	CourseNumber      string
	CourseDescription *string
}

type Grade string

const (
	GradeAPlus  Grade = "A+"
	GradeA      Grade = "A"
	GradeAMinus Grade = "A-"
	GradeBPlus  Grade = "B+"
	GradeB      Grade = "B"
	GradeBMinus Grade = "B-"
	GradeCPlus  Grade = "C+"
	GradeC      Grade = "C"
	GradeCMinus Grade = "C-"
	GradeDPlus  Grade = "D+"
	GradeD      Grade = "D"
	GradeDMinus Grade = "D-"
	GradeF      Grade = "F"
)

type Relation struct {
	SourceId     string
	TargetId     string
	Enforced     *bool
	Prereq       *bool
	Coreq        *bool
	MinimumGrade *Grade
}
