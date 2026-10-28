package vo

import "errors"

const (
	GradeSourceManual     = "manual"
	GradeSourceBackground = "background"
)

var ErrGradeSourceInvalid = errors.New("來源不正確")

type GradeSourceVo struct {
	Value string
}

func NewGradeSourceVo(rawSource string) (GradeSourceVo, error) {
	switch rawSource {
	case GradeSourceManual, GradeSourceBackground:
		return GradeSourceVo{Value: rawSource}, nil
	default:
		return GradeSourceVo{}, ErrGradeSourceInvalid
	}
}
