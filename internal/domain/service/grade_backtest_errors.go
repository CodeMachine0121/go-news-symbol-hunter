package service

import (
	"errors"

	"github.com/CodeMachine0121/go-news-symbol-hunter/internal/domain/models/vo"
)

var (
	ErrBacktestPeriodInvalid         = vo.ErrAnalysisCompletedPeriodInvalid
	ErrBacktestAnalysisLimitExceeded = errors.New("涵蓋評等紀錄超過 500 筆，請縮小期間")
	ErrPriceServiceUnavailable       = errors.New("價格服務暫時無法使用")
	ErrGradeSourceInvalid            = vo.ErrGradeSourceInvalid
	ErrBacktestStorageUnavailable    = errors.New("backtest storage unavailable")
)
