package interfaces

import (
	"context"

	"github.com/CodeMachine0121/go-news-symbol-hunter/internal/domain/models/vo"
)

type IAnalystProxy interface {
	ModelName() string
	Respond(ctx context.Context, request vo.AnalystRequestVo, exchanges []vo.AnalystExchangeVo) (vo.AnalystTurnVo, error)
}
