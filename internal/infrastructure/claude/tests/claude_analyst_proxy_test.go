package claude_test

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/CodeMachine0121/go-news-symbol-hunter/internal/domain/models/vo"
	"github.com/CodeMachine0121/go-news-symbol-hunter/internal/infrastructure/claude"
	"github.com/anthropics/anthropic-sdk-go"
	"github.com/anthropics/anthropic-sdk-go/option"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const searchReply = `{"id":"msg_1","type":"message","role":"assistant","model":"claude-opus-5-5","content":[{"type":"text","text":"Searching."},{"type":"tool_use","id":"toolu_search","name":"search_symbol_news","input":{"symbol":"BTC","category":"crypto"}}],"stop_reason":"tool_use","stop_sequence":null,"usage":{"input_tokens":100,"output_tokens":10}}`

const conclusionReply = `{"id":"msg_2","type":"message","role":"assistant","model":"claude-opus-5-5","content":[{"type":"tool_use","id":"toolu_submit","name":"submit_analysis","input":{"grade":"bullish","confidence":72,"timeHorizon":"short","reason":"ETF 資金流入","keyEventLinks":["https://news/1"],"riskFactors":["礦工賣壓"]}}],"stop_reason":"tool_use","stop_sequence":null,"usage":{"input_tokens":200,"output_tokens":20}}`

const refusalReply = `{"id":"msg_3","type":"message","role":"assistant","model":"claude-opus-5-5","content":[],"stop_reason":"refusal","stop_sequence":null,"stop_details":{"type":"refusal","category":"cyber","explanation":"x"},"usage":{"input_tokens":5,"output_tokens":0}}`

type capturedRequest struct {
	betaHeader string
	body       map[string]json.RawMessage
}

func startMessagesServer(t *testing.T, statusCode int, reply string) (*claude.ClaudeAnalystProxy, *capturedRequest) {
	captured := &capturedRequest{}
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		captured.betaHeader = request.Header.Get("anthropic-beta")
		requestBody, _ := io.ReadAll(request.Body)
		_ = json.Unmarshal(requestBody, &captured.body)
		writer.Header().Set("Content-Type", "application/json")
		writer.WriteHeader(statusCode)
		_, _ = writer.Write([]byte(reply))
	}))
	t.Cleanup(server.Close)
	client := anthropic.NewClient(option.WithBaseURL(server.URL), option.WithAPIKey("test-key"), option.WithMaxRetries(0))
	return claude.NewClaudeAnalystProxy(client, "claude-opus-5-5", "high"), captured
}

var bitcoinRequest = vo.AnalystRequestVo{Symbol: "BTC", Category: "crypto", SearchKeyword: "Bitcoin"}

func TestClaudeAnalystProxy_FirstTurnSendsTheConfiguredRequest(t *testing.T) {
	claudeAnalystProxy, captured := startMessagesServer(t, http.StatusOK, searchReply)

	analystTurn, err := claudeAnalystProxy.Respond(context.Background(), bitcoinRequest, nil)

	require.NoError(t, err)
	assert.Equal(t, "claude-opus-5-5", claudeAnalystProxy.ModelName())
	assert.Contains(t, captured.betaHeader, "server-side-fallback-2026-07-01")
	assert.JSONEq(t, `"claude-opus-5-5"`, string(captured.body["model"]))
	assert.JSONEq(t, `16000`, string(captured.body["max_tokens"]))
	assert.JSONEq(t, `{"effort":"high"}`, string(captured.body["output_config"]))
	assert.JSONEq(t, `"default"`, string(captured.body["fallbacks"]))
	assert.NotContains(t, captured.body, "tool_choice")
	assert.JSONEq(t, `{"type":"ephemeral"}`, string(captured.body["cache_control"]))
	var tools []struct {
		Name   string `json:"name"`
		Strict bool   `json:"strict"`
	}
	require.NoError(t, json.Unmarshal(captured.body["tools"], &tools))
	assert.Equal(t, []struct {
		Name   string `json:"name"`
		Strict bool   `json:"strict"`
	}{{Name: "search_symbol_news", Strict: true}, {Name: "submit_analysis", Strict: true}}, tools)
	var system []struct {
		Text string `json:"text"`
	}
	require.NoError(t, json.Unmarshal(captured.body["system"], &system))
	assert.Contains(t, system[0].Text, "submit_analysis exactly once")
	var messages []struct {
		Role    string `json:"role"`
		Content []struct {
			Text string `json:"text"`
		} `json:"content"`
	}
	require.NoError(t, json.Unmarshal(captured.body["messages"], &messages))
	require.Len(t, messages, 1)
	assert.Equal(t, "Analyze symbol BTC in market category crypto (search keyword: Bitcoin).", messages[0].Content[0].Text)
	assert.Equal(t, vo.AnalystTurnVo{
		Reply:        searchReply,
		NewsSearches: []vo.AnalystNewsSearchVo{{ToolCallID: "toolu_search", Symbol: "BTC", Category: "crypto"}},
		Usage:        vo.AnalystUsageVo{InputTokens: 100, OutputTokens: 10},
		ModelName:    "claude-opus-5-5",
	}, analystTurn)
}

func TestClaudeAnalystProxy_ReplaysPreviousExchangesAndReadsTheConclusion(t *testing.T) {
	claudeAnalystProxy, captured := startMessagesServer(t, http.StatusOK, conclusionReply)
	exchanges := []vo.AnalystExchangeVo{{Reply: searchReply, ToolResults: []vo.AnalystToolResultVo{
		{ToolCallID: "toolu_search", Content: `{"symbol":"BTC"}`},
		{ToolCallID: "toolu_other", Content: "找不到此標的", IsError: true},
	}}}

	analystTurn, err := claudeAnalystProxy.Respond(context.Background(), bitcoinRequest, exchanges)

	require.NoError(t, err)
	var messages []struct {
		Role    string `json:"role"`
		Content []struct {
			Type      string `json:"type"`
			ID        string `json:"id"`
			ToolUseID string `json:"tool_use_id"`
			IsError   bool   `json:"is_error"`
			Content   []struct {
				Text string `json:"text"`
			} `json:"content"`
		} `json:"content"`
	}
	require.NoError(t, json.Unmarshal(captured.body["messages"], &messages))
	require.Len(t, messages, 3)
	assert.Equal(t, "assistant", messages[1].Role)
	assert.Equal(t, "tool_use", messages[1].Content[1].Type)
	assert.Equal(t, "toolu_search", messages[1].Content[1].ID)
	assert.Equal(t, "user", messages[2].Role)
	assert.Equal(t, "toolu_search", messages[2].Content[0].ToolUseID)
	assert.Equal(t, `{"symbol":"BTC"}`, messages[2].Content[0].Content[0].Text)
	assert.False(t, messages[2].Content[0].IsError)
	assert.Equal(t, "toolu_other", messages[2].Content[1].ToolUseID)
	assert.True(t, messages[2].Content[1].IsError)
	assert.Equal(t, &vo.RawAnalysisConclusionVo{Grade: "bullish", Confidence: 72, TimeHorizon: "short", Reason: "ETF 資金流入", KeyEventLinks: []string{"https://news/1"}, RiskFactors: []string{"礦工賣壓"}}, analystTurn.Conclusion)
	assert.Empty(t, analystTurn.NewsSearches)
	assert.False(t, analystTurn.IsRefused)
	assert.Equal(t, vo.AnalystUsageVo{InputTokens: 200, OutputTokens: 20}, analystTurn.Usage)
}

func TestClaudeAnalystProxy_ReportsARefusal(t *testing.T) {
	claudeAnalystProxy, _ := startMessagesServer(t, http.StatusOK, refusalReply)

	analystTurn, err := claudeAnalystProxy.Respond(context.Background(), bitcoinRequest, nil)

	require.NoError(t, err)
	assert.True(t, analystTurn.IsRefused)
	assert.Nil(t, analystTurn.Conclusion)
}

func TestClaudeAnalystProxy_FailsWhenTheServiceErrors(t *testing.T) {
	claudeAnalystProxy, _ := startMessagesServer(t, http.StatusInternalServerError, `{"type":"error","error":{"type":"api_error","message":"boom"}}`)

	_, err := claudeAnalystProxy.Respond(context.Background(), bitcoinRequest, nil)

	assert.Error(t, err)
}

func TestClaudeAnalystProxy_FailsOnAnUnrestorableExchange(t *testing.T) {
	claudeAnalystProxy, _ := startMessagesServer(t, http.StatusOK, conclusionReply)

	_, err := claudeAnalystProxy.Respond(context.Background(), bitcoinRequest, []vo.AnalystExchangeVo{{Reply: "not json"}})

	assert.ErrorContains(t, err, "restore analyst reply")
}

func TestClaudeAnalystProxy_ReportsTheAnsweringModelAndCachedInput(t *testing.T) {
	claudeAnalystProxy, _ := startMessagesServer(t, http.StatusOK, `{"id":"msg_4","type":"message","role":"assistant","model":"claude-opus-4-8","content":[],"stop_reason":"end_turn","stop_sequence":null,"usage":{"input_tokens":10,"cache_creation_input_tokens":20,"cache_read_input_tokens":30,"output_tokens":5}}`)

	analystTurn, err := claudeAnalystProxy.Respond(context.Background(), bitcoinRequest, nil)

	require.NoError(t, err)
	assert.Equal(t, "claude-opus-4-8", analystTurn.ModelName)
	assert.Equal(t, vo.AnalystUsageVo{InputTokens: 60, OutputTokens: 5}, analystTurn.Usage)
}

func TestClaudeAnalystProxy_TellsTheAnalystWhenNewsIsLimitedToAPublishingWindow(t *testing.T) {
	taipei := time.FixedZone("Asia/Taipei", 8*60*60)
	testCases := []struct {
		name                string
		newsPublishedWindow vo.NewsPublishedWindowVo
		expectedInstruction string
	}{
		{name: "open-ended window", newsPublishedWindow: vo.NewsPublishedWindowVo{Since: time.Date(2026, 10, 7, 9, 0, 0, 0, taipei)}, expectedInstruction: "Analyze symbol 2330 in market category twStock (search keyword: 台積電). News searches only return news published since 2026-10-07T09:00:00+08:00; judge the symbol on that news alone."},
		{name: "closed window", newsPublishedWindow: vo.NewsPublishedWindowVo{Since: time.Date(2026, 10, 7, 9, 0, 0, 0, taipei), Before: time.Date(2026, 10, 7, 13, 30, 0, 0, taipei)}, expectedInstruction: "Analyze symbol 2330 in market category twStock (search keyword: 台積電). News searches only return news published from 2026-10-07T09:00:00+08:00 up to 2026-10-07T13:30:00+08:00; judge the symbol on that news alone."},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			claudeAnalystProxy, captured := startMessagesServer(t, http.StatusOK, searchReply)
			request := vo.AnalystRequestVo{Symbol: "2330", Category: "twStock", SearchKeyword: "台積電", NewsPublishedWindow: testCase.newsPublishedWindow}

			_, err := claudeAnalystProxy.Respond(context.Background(), request, nil)

			require.NoError(t, err)
			var messages []struct {
				Content []struct {
					Text string `json:"text"`
				} `json:"content"`
			}
			require.NoError(t, json.Unmarshal(captured.body["messages"], &messages))
			assert.Equal(t, testCase.expectedInstruction, messages[0].Content[0].Text)
		})
	}
}
