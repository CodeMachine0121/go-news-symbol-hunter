package service

import "errors"

const (
	FailureReasonAnalystUnavailable    = "AI 服務暫時無法使用"
	FailureReasonAnalystExceededRounds = "AI 未在限制內完成分析"
	FailureReasonAnalystIncomplete     = "AI 未提供完整分析"
	FailureReasonAnalystRefused        = "AI 拒絕分析此標的"
	FailureReasonResultNotSaved        = "分析結果保存失敗"
	FailureReasonInterruptedByRestart  = "服務重新啟動，分析中斷"
)

var (
	ErrAnalysisEventNotFound      = errors.New("找不到此分析事件")
	ErrAnalysisAlreadyRunning     = errors.New("an analysis for this symbol is already running")
	ErrAnalysisStorageUnavailable = errors.New("analysis storage unavailable")
)
