package service

import (
	"errors"

	"github.com/CodeMachine0121/go-news-symbol-hunter/internal/domain/models/vo"
)

const (
	FailureReasonSessionRunInterrupted     = "服務重新啟動，執行中斷"
	FailureReasonStaleGradesNotPurged      = "前一天結果刪除失敗"
	FailureReasonTrackedSymbolsUnavailable = "追蹤標的讀取失敗"
	FailureReasonSessionGradeNotSaved      = "時段評等保存失敗"
	FailureReasonCombinedGradeNotSaved     = "綜合評等保存失敗"
)

var (
	ErrSymbolAndCategoryRequiredTogether = vo.ErrSymbolAndCategoryRequiredTogether
	ErrSessionRunAlreadyExists           = errors.New("this trading session already ran")
	ErrSessionGradeStorageUnavailable    = errors.New("session grade storage unavailable")
)
