package vo_test

import (
	"testing"

	"github.com/CodeMachine0121/go-news-symbol-hunter/internal/domain/models/vo"
	"github.com/stretchr/testify/assert"
)

func TestGradeSourceVo(t *testing.T) {
	testCases := []struct {
		name                string
		rawSource           string
		expectedGradeSource vo.GradeSourceVo
		expectedError       error
	}{
		{name: "manual", rawSource: "manual", expectedGradeSource: vo.GradeSourceVo{Value: "manual"}},
		{name: "background", rawSource: "background", expectedGradeSource: vo.GradeSourceVo{Value: "background"}},
		{name: "anything else", rawSource: "other", expectedError: vo.ErrGradeSourceInvalid},
		{name: "blank", rawSource: "", expectedError: vo.ErrGradeSourceInvalid},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			gradeSource, err := vo.NewGradeSourceVo(testCase.rawSource)

			assert.ErrorIs(t, err, testCase.expectedError)
			assert.Equal(t, testCase.expectedGradeSource, gradeSource)
		})
	}
}
