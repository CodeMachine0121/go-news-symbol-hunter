package controller

import "github.com/CodeMachine0121/go-news-symbol-hunter/internal/domain/models/dto"

type ErrorResponseBody struct {
	Error ErrorDetail `json:"error"`
}

type ErrorDetail struct {
	Code       string                   `json:"code"`
	Message    string                   `json:"message"`
	Candidates []dto.SymbolCandidateDto `json:"candidates,omitempty"`
}
