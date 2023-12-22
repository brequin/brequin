package db

type Quarter struct {
	Code string
	Name string
}

type SubjectArea struct {
	Code string
	Name string
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
