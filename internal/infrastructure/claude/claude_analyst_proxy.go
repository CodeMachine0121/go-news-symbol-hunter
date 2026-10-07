package claude

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/CodeMachine0121/go-news-symbol-hunter/internal/domain/models/vo"
	"github.com/anthropics/anthropic-sdk-go"
	"github.com/anthropics/anthropic-sdk-go/shared/constant"
)

const (
	searchSymbolNewsToolName = "search_symbol_news"
	submitAnalysisToolName   = "submit_analysis"
	analystMaximumTokens     = 16000
)

const analystSystemPrompt = `You are an equity and crypto news analyst. You assess only the information side of a symbol: news, announcements and events. You do not use price charts, technical indicators or fund flows.

Work in this order:
1. Call search_symbol_news for the symbol you were given. You may also search closely related symbols (peers, suppliers, the underlying chain) when that changes the picture, but keep searches few and purposeful.
2. Weigh the news: what is material, what is noise, what conflicts. Prefer primary, recent and specific news over commentary.
3. Call submit_analysis exactly once with your conclusion. Do not answer in plain text.

Conclusion rules:
- grade: strongBullish, bullish, neutral, bearish or strongBearish, for the stated time horizon.
- confidence: an integer from 0 to 100 reflecting how strongly the evidence supports the grade. With little or conflicting news, stay neutral with low confidence.
- timeHorizon: short (within two weeks) or mid (within three months).
- reason: a concise explanation in Traditional Chinese (Taiwan) that names the decisive news.
- keyEventLinks: up to five links copied exactly from the news returned by search_symbol_news that drove the grade.
- riskFactors: up to five short Traditional Chinese statements of what would prove the conclusion wrong.
If no news was found, submit neutral with confidence 0 and say so in the reason.`

type searchSymbolNewsInput struct {
	Symbol   string `json:"symbol"`
	Category string `json:"category"`
}

type submitAnalysisInput struct {
	Grade         string   `json:"grade"`
	Confidence    int      `json:"confidence"`
	TimeHorizon   string   `json:"timeHorizon"`
	Reason        string   `json:"reason"`
	KeyEventLinks []string `json:"keyEventLinks"`
	RiskFactors   []string `json:"riskFactors"`
}

type ClaudeAnalystProxy struct {
	client    anthropic.Client
	modelName string
	effort    string
}

func NewClaudeAnalystProxy(client anthropic.Client, modelName string, effort string) *ClaudeAnalystProxy {
	return &ClaudeAnalystProxy{client: client, modelName: modelName, effort: effort}
}

func (claudeAnalystProxy *ClaudeAnalystProxy) ModelName() string {
	return claudeAnalystProxy.modelName
}

func (claudeAnalystProxy *ClaudeAnalystProxy) Respond(ctx context.Context, request vo.AnalystRequestVo, exchanges []vo.AnalystExchangeVo) (vo.AnalystTurnVo, error) {
	instruction := fmt.Sprintf("Analyze symbol %s in market category %s (search keyword: %s).", request.Symbol, request.Category, request.SearchKeyword)
	newsPublishedWindow := request.NewsPublishedWindow
	switch {
	case !newsPublishedWindow.Since.IsZero() && !newsPublishedWindow.Before.IsZero():
		instruction += fmt.Sprintf(" News searches only return news published from %s up to %s; judge the symbol on that news alone.", newsPublishedWindow.Since.Format(time.RFC3339), newsPublishedWindow.Before.Format(time.RFC3339))
	case !newsPublishedWindow.Since.IsZero():
		instruction += fmt.Sprintf(" News searches only return news published since %s; judge the symbol on that news alone.", newsPublishedWindow.Since.Format(time.RFC3339))
	}
	messages := []anthropic.BetaMessageParam{anthropic.NewBetaUserMessage(anthropic.NewBetaTextBlock(instruction))}
	for _, exchange := range exchanges {
		var reply anthropic.BetaMessage
		if err := json.Unmarshal([]byte(exchange.Reply), &reply); err != nil {
			return vo.AnalystTurnVo{}, fmt.Errorf("restore analyst reply: %w", err)
		}
		toolResultBlocks := make([]anthropic.BetaContentBlockParamUnion, 0, len(exchange.ToolResults))
		for _, toolResult := range exchange.ToolResults {
			toolResultBlocks = append(toolResultBlocks, anthropic.NewBetaToolResultBlock(toolResult.ToolCallID, toolResult.Content, toolResult.IsError))
		}
		messages = append(messages, reply.ToParam(), anthropic.NewBetaUserMessage(toolResultBlocks...))
	}
	response, err := claudeAnalystProxy.client.Beta.Messages.New(ctx, anthropic.BetaMessageNewParams{
		Model:        anthropic.Model(claudeAnalystProxy.modelName),
		MaxTokens:    analystMaximumTokens,
		System:       []anthropic.BetaTextBlockParam{{Text: analystSystemPrompt, CacheControl: anthropic.NewBetaCacheControlEphemeralParam()}},
		Tools:        analystTools,
		OutputConfig: anthropic.BetaOutputConfigParam{Effort: anthropic.BetaOutputConfigEffort(claudeAnalystProxy.effort)},
		Betas:        []anthropic.AnthropicBeta{anthropic.AnthropicBetaServerSideFallback2026_07_01},
		Fallbacks:    anthropic.BetaFallbacksParamUnion{OfDefault: constant.ValueOf[constant.Default]()},
		Messages:     messages,
		// caches the growing conversation so each round only pays full price for its new turn
		CacheControl: anthropic.NewBetaCacheControlEphemeralParam(),
	})
	if err != nil {
		return vo.AnalystTurnVo{}, err
	}
	analystTurn := vo.AnalystTurnVo{
		Reply:     response.RawJSON(),
		IsRefused: response.StopReason == anthropic.BetaStopReasonRefusal,
		Usage: vo.AnalystUsageVo{
			InputTokens:  response.Usage.InputTokens + response.Usage.CacheCreationInputTokens + response.Usage.CacheReadInputTokens,
			OutputTokens: response.Usage.OutputTokens,
		},
		// a server-side fallback may answer with a different model than the one requested
		ModelName: string(response.Model),
	}
	for _, block := range response.Content {
		toolUse, isToolUse := block.AsAny().(anthropic.BetaToolUseBlock)
		if !isToolUse {
			continue
		}
		switch toolUse.Name {
		case searchSymbolNewsToolName:
			var searchInput searchSymbolNewsInput
			// strict tools guarantee schema-valid input; a malformed one degrades to an empty search the domain rejects
			_ = json.Unmarshal([]byte(toolUse.JSON.Input.Raw()), &searchInput)
			analystTurn.NewsSearches = append(analystTurn.NewsSearches, vo.AnalystNewsSearchVo{ToolCallID: toolUse.ID, Symbol: searchInput.Symbol, Category: searchInput.Category})
		case submitAnalysisToolName:
			var conclusionInput submitAnalysisInput
			// a malformed conclusion degrades to an empty reason, which the domain treats as incomplete
			_ = json.Unmarshal([]byte(toolUse.JSON.Input.Raw()), &conclusionInput)
			analystTurn.Conclusion = &vo.RawAnalysisConclusionVo{
				Grade:         conclusionInput.Grade,
				Confidence:    conclusionInput.Confidence,
				TimeHorizon:   conclusionInput.TimeHorizon,
				Reason:        conclusionInput.Reason,
				KeyEventLinks: conclusionInput.KeyEventLinks,
				RiskFactors:   conclusionInput.RiskFactors,
			}
		}
	}
	return analystTurn, nil
}

var analystTools = []anthropic.BetaToolUnionParam{
	{OfTool: &anthropic.BetaToolParam{
		Name:        searchSymbolNewsToolName,
		Description: anthropic.String("Search the last seven days of news for a symbol. Returns up to 30 news items (title, link, publishedAt, providerName, summary) and the providers that failed. twStock symbols are TWSE listed stock codes such as 2330; usStock symbols are tickers such as AAPL; crypto symbols are coin tickers such as BTC."),
		Strict:      anthropic.Bool(true),
		InputSchema: anthropic.BetaToolInputSchemaParam{
			Properties: map[string]any{
				"symbol":   map[string]any{"type": "string", "description": "The symbol to search."},
				"category": map[string]any{"type": "string", "enum": []string{"crypto", "twStock", "usStock"}},
			},
			Required:    []string{"symbol", "category"},
			ExtraFields: map[string]any{"additionalProperties": false},
		},
	}},
	{OfTool: &anthropic.BetaToolParam{
		Name:        submitAnalysisToolName,
		Description: anthropic.String("Submit the final analysis. Call exactly once, after searching."),
		Strict:      anthropic.Bool(true),
		InputSchema: anthropic.BetaToolInputSchemaParam{
			Properties: map[string]any{
				"grade":         map[string]any{"type": "string", "enum": []string{"strongBullish", "bullish", "neutral", "bearish", "strongBearish"}},
				"confidence":    map[string]any{"type": "integer", "description": "0 to 100"},
				"timeHorizon":   map[string]any{"type": "string", "enum": []string{"short", "mid"}},
				"reason":        map[string]any{"type": "string"},
				"keyEventLinks": map[string]any{"type": "array", "items": map[string]any{"type": "string"}},
				"riskFactors":   map[string]any{"type": "array", "items": map[string]any{"type": "string"}},
			},
			Required:    []string{"grade", "confidence", "timeHorizon", "reason", "keyEventLinks", "riskFactors"},
			ExtraFields: map[string]any{"additionalProperties": false},
		},
	}},
}
