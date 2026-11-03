package agent

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"charm.land/catwalk/pkg/catwalk"
	"charm.land/fantasy"
	"github.com/charmbracelet/crush/internal/agent/notify"
	"github.com/charmbracelet/crush/internal/message"
	"github.com/charmbracelet/crush/internal/pubsub"
	"github.com/stretchr/testify/require"
)

// summaryFailModel: lần 1 trả tool call với usage lớn (ép tự tóm tắt); request
// tóm tắt (system prompt của summarize) hỏng như provider im lặng quá hạn; các
// request khác trả "done". Ghi lại thứ tự để biết có thử tóm tắt lại không.
type summaryFailModel struct {
	n        atomic.Int64
	mu       sync.Mutex
	kinds    []string
	prompts  []string
	failWith error
}

func (m *summaryFailModel) Provider() string { return "fake" }
func (m *summaryFailModel) Model() string    { return "fake-model" }
func (m *summaryFailModel) Generate(context.Context, fantasy.Call) (*fantasy.Response, error) {
	return &fantasy.Response{Content: fantasy.ResponseContent{fantasy.TextContent{Text: "title"}}, FinishReason: fantasy.FinishReasonStop}, nil
}

func (m *summaryFailModel) GenerateObject(context.Context, fantasy.ObjectCall) (*fantasy.ObjectResponse, error) {
	return nil, errors.New("not supported")
}

func (m *summaryFailModel) StreamObject(context.Context, fantasy.ObjectCall) (fantasy.ObjectStreamResponse, error) {
	return nil, errors.New("not supported")
}

func (m *summaryFailModel) record(kind, prompt string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.kinds = append(m.kinds, kind)
	m.prompts = append(m.prompts, prompt)
}

func (m *summaryFailModel) seen() []string {
	m.mu.Lock()
	defer m.mu.Unlock()
	return append([]string(nil), m.kinds...)
}

func (m *summaryFailModel) Stream(_ context.Context, call fantasy.Call) (fantasy.StreamResponse, error) {
	n := m.n.Add(1)
	last := lastUserText(call.Prompt)
	if isSummaryCall(call) {
		m.record("summary", last)
		return nil, m.failWith
	}
	if n == 1 {
		m.record("work", last)
		return func(yield func(fantasy.StreamPart) bool) {
			if !yield(fantasy.StreamPart{Type: fantasy.StreamPartTypeToolInputStart, ID: "tc1", ToolCallName: "noop"}) {
				return
			}
			if !yield(fantasy.StreamPart{Type: fantasy.StreamPartTypeToolInputEnd, ID: "tc1"}) {
				return
			}
			if !yield(fantasy.StreamPart{Type: fantasy.StreamPartTypeToolCall, ID: "tc1", ToolCallName: "noop", ToolCallInput: "{}"}) {
				return
			}
			yield(fantasy.StreamPart{
				Type: fantasy.StreamPartTypeFinish, FinishReason: fantasy.FinishReasonToolCalls,
				Usage: fantasy.Usage{InputTokens: 900, OutputTokens: 60, TotalTokens: 960},
			})
		}, nil
	}
	m.record("work", last)
	return func(yield func(fantasy.StreamPart) bool) {
		if !yield(fantasy.StreamPart{Type: fantasy.StreamPartTypeTextStart, ID: "1"}) {
			return
		}
		if !yield(fantasy.StreamPart{Type: fantasy.StreamPartTypeTextDelta, ID: "1", Delta: "done"}) {
			return
		}
		if !yield(fantasy.StreamPart{Type: fantasy.StreamPartTypeTextEnd, ID: "1"}) {
			return
		}
		yield(fantasy.StreamPart{
			Type: fantasy.StreamPartTypeFinish, FinishReason: fantasy.FinishReasonStop,
			Usage: fantasy.Usage{InputTokens: 950, OutputTokens: 10, TotalTokens: 960},
		})
	}, nil
}

// Tự tóm tắt hỏng (provider im lặng quá hạn) không còn làm dừng việc: lượt
// làm tiếp chạy ngay với lịch sử đầy đủ, không thử tóm tắt lại trong lúc đó,
// và RunComplete báo xong bình thường. Trước đây lỗi tóm tắt trả thẳng ra
// ngoài và công việc dừng giữa chừng.
func TestAutoSummarizeFailureContinuesTheWork(t *testing.T) {
	env := testEnv(t)
	broker := pubsub.NewBroker[notify.RunComplete]()
	t.Cleanup(broker.Shutdown)
	m := &summaryFailModel{failWith: &requestTimeoutError{timeout: 6 * time.Minute, idle: true, cause: context.DeadlineExceeded}}
	cw := catwalk.Model{ContextWindow: 1000, DefaultMaxTokens: 100}
	sa := NewSessionAgent(SessionAgentOptions{
		LargeModel:  modelWith(m, "prov", "model", cw),
		SmallModel:  modelWith(&finishStreamModel{text: "title"}, "prov", "small", cw),
		IsYolo:      true,
		Sessions:    env.sessions,
		Messages:    env.messages,
		RunComplete: broker,
		Tools:       []fantasy.AgentTool{namedTool("noop")},
	}).(*sessionAgent)
	sess, err := env.sessions.Create(t.Context(), "session")
	require.NoError(t, err)
	ch := broker.Subscribe(t.Context())

	_, err = sa.Run(t.Context(), SessionAgentCall{SessionID: sess.ID, RunID: "x", Prompt: "refactor the scanner"})
	require.NoError(t, err, "a failed summary must not end the work")

	rc := waitRunComplete(t, ch, "x")
	require.False(t, rc.Cancelled)
	require.Empty(t, rc.Error)
	require.Equal(t, "done", rc.Text)
	require.Equal(t, []string{"work", "summary", "work"}, m.seen(), "one summary attempt, then the work continues without retrying it")
	require.Contains(t, m.prompts[2], "automatic context summary failed")
	require.Contains(t, m.prompts[2], "refactor the scanner")

	// Bản tóm tắt hỏng còn trong transcript (người dùng thấy lỗi) nhưng không
	// thành điểm cắt lịch sử.
	stored, err := env.sessions.Get(t.Context(), sess.ID)
	require.NoError(t, err)
	require.Empty(t, stored.SummaryMessageID)
	msgs, err := env.messages.List(t.Context(), sess.ID)
	require.NoError(t, err)
	var failedSummary bool
	for _, msg := range msgs {
		if msg.IsSummaryMessage {
			if f := msg.FinishPart(); f != nil && f.Reason == message.FinishReasonError {
				failedSummary = true
			}
		}
	}
	require.True(t, failedSummary)

	// Prompt mới của người dùng: tự tóm tắt được thử lại.
	_, err = sa.Run(t.Context(), SessionAgentCall{SessionID: sess.ID, RunID: "y", Prompt: "next"})
	require.NoError(t, err)
	waitRunComplete(t, ch, "y")
	require.Contains(t, m.seen()[3:], "summary", "a new user prompt tries summarizing again")
}

// Huỷ lúc đang tóm tắt vẫn dừng lượt (không bị coi là tóm tắt hỏng để làm tiếp).
func TestAutoSummarizeCancelStillStops(t *testing.T) {
	env := testEnv(t)
	broker := pubsub.NewBroker[notify.RunComplete]()
	t.Cleanup(broker.Shutdown)
	m := &summaryFailModel{failWith: context.Canceled}
	cw := catwalk.Model{ContextWindow: 1000, DefaultMaxTokens: 100}
	sa := NewSessionAgent(SessionAgentOptions{
		LargeModel:  modelWith(m, "prov", "model", cw),
		SmallModel:  modelWith(&finishStreamModel{text: "title"}, "prov", "small", cw),
		IsYolo:      true,
		Sessions:    env.sessions,
		Messages:    env.messages,
		RunComplete: broker,
		Tools:       []fantasy.AgentTool{namedTool("noop")},
	}).(*sessionAgent)
	sess, err := env.sessions.Create(t.Context(), "session")
	require.NoError(t, err)
	ch := broker.Subscribe(t.Context())

	_, err = sa.Run(t.Context(), SessionAgentCall{SessionID: sess.ID, RunID: "x", Prompt: "work"})
	require.ErrorIs(t, err, context.Canceled)
	require.True(t, waitRunComplete(t, ch, "x").Cancelled)
	require.Equal(t, []string{"work", "summary"}, m.seen())
}

// Request tóm tắt có hạn chờ gấp summaryRequestTimeoutScale lần hạn thường.
func TestSummaryRequestTimeoutIsScaled(t *testing.T) {
	t.Parallel()
	m := requestTimeoutModel{timeout: 2 * time.Minute}
	require.Equal(t, 2*time.Minute, m.budget(t.Context()))
	require.Equal(t, 6*time.Minute, m.budget(withRequestTimeoutScale(t.Context(), summaryRequestTimeoutScale)))
}

// isSummaryCall: request tóm tắt mang system prompt của summarize.
func isSummaryCall(call fantasy.Call) bool {
	for _, msg := range call.Prompt {
		if msg.Role != fantasy.MessageRoleSystem {
			continue
		}
		for _, part := range msg.Content {
			if tp, ok := fantasy.AsMessagePart[fantasy.TextPart](part); ok && strings.Contains(tp.Text, "summar") {
				return true
			}
		}
	}
	return false
}

// workStep là một request "work" của scriptedModel: lỗi trả ngay khi gửi, hoặc
// tool call / chữ "done" kèm usage báo về.
type workStep struct {
	err      error
	toolCall bool
	in, out  int64
}

// scriptedModel chạy steps theo thứ tự (hết thì trả "done" nhỏ); request tóm tắt
// thứ i trả summaryErrs[i] (nil hoặc hết danh sách là tóm tắt thành công).
type scriptedModel struct {
	summaryFailModel
	steps       []workStep
	summaryErrs []error
	work, sum   atomic.Int64
}

func (m *scriptedModel) Stream(_ context.Context, call fantasy.Call) (fantasy.StreamResponse, error) {
	last := lastUserText(call.Prompt)
	if isSummaryCall(call) {
		m.record("summary", last)
		if i := int(m.sum.Add(1)) - 1; i < len(m.summaryErrs) && m.summaryErrs[i] != nil {
			return nil, m.summaryErrs[i]
		}
		return textStream("summary of the work", fantasy.FinishReasonStop, fantasy.Usage{InputTokens: 50, OutputTokens: 20}), nil
	}
	i := int(m.work.Add(1)) - 1
	m.record("work", last)
	s := workStep{in: 300, out: 10}
	if i < len(m.steps) {
		s = m.steps[i]
	}
	if s.err != nil {
		return nil, s.err
	}
	if !s.toolCall {
		return textStream("done", fantasy.FinishReasonStop, fantasy.Usage{InputTokens: s.in, OutputTokens: s.out}), nil
	}
	id := fmt.Sprintf("tc%d", i)
	return func(yield func(fantasy.StreamPart) bool) {
		if !yield(fantasy.StreamPart{Type: fantasy.StreamPartTypeToolInputStart, ID: id, ToolCallName: "noop"}) {
			return
		}
		if !yield(fantasy.StreamPart{Type: fantasy.StreamPartTypeToolInputEnd, ID: id}) {
			return
		}
		if !yield(fantasy.StreamPart{Type: fantasy.StreamPartTypeToolCall, ID: id, ToolCallName: "noop", ToolCallInput: "{}"}) {
			return
		}
		yield(fantasy.StreamPart{
			Type: fantasy.StreamPartTypeFinish, FinishReason: fantasy.FinishReasonToolCalls,
			Usage: fantasy.Usage{InputTokens: s.in, OutputTokens: s.out, TotalTokens: s.in + s.out},
		})
	}, nil
}

// runScripted chạy một prompt trên session mới với window 1000 token (ngưỡng tự
// tóm tắt 200) và trả RunComplete của nó.
func runScripted(t *testing.T, m *scriptedModel, prompt string) (notify.RunComplete, string) {
	t.Helper()
	env := testEnv(t)
	broker := pubsub.NewBroker[notify.RunComplete]()
	t.Cleanup(broker.Shutdown)
	cw := catwalk.Model{ContextWindow: 1000, DefaultMaxTokens: 100}
	sa := NewSessionAgent(SessionAgentOptions{
		LargeModel:  modelWith(m, "prov", "model", cw),
		SmallModel:  modelWith(&finishStreamModel{text: "title"}, "prov", "small", cw),
		IsYolo:      true,
		Sessions:    env.sessions,
		Messages:    env.messages,
		RunComplete: broker,
		Tools:       []fantasy.AgentTool{namedTool("noop")},
	}).(*sessionAgent)
	sess, err := env.sessions.Create(t.Context(), "session")
	require.NoError(t, err)
	ch := broker.Subscribe(t.Context())
	_, _ = sa.Run(t.Context(), SessionAgentCall{SessionID: sess.ID, RunID: "x", Prompt: prompt})
	rc := waitRunComplete(t, ch, "x")
	stored, err := env.sessions.Get(t.Context(), sess.ID)
	require.NoError(t, err)
	return rc, stored.SummaryMessageID
}

// Câu lỗi thật của vLLM sau LiteLLM (03/11): mẫu openai của fantasy không nhận
// ra ("requested 8192 output tokens"), nên IsContextTooLarge() là false.
func vllmOverflowErr() error {
	return &fantasy.ProviderError{
		Title:      "bad request",
		StatusCode: 400,
		Message:    `litellm.ContextWindowExceededError: litellm.BadRequestError: ContextWindowExceededError: Hosted_vllmException - {"error":{"message":"This model's maximum context length is 200000 tokens. However, you requested 8192 output tokens and your prompt contains at least 191809 input tokens, for a total of at least 200001 tokens.","type":"BadRequestError","param":"input_tokens","code":400}}`,
	}
}

// Sự cố 03/11: tóm tắt hỏng một lần thì tự tóm tắt tắt suốt chuỗi làm tiếp, context
// chạy thẳng tới trần thật của model. Nay context lớn thêm nửa ngưỡng kể từ lần
// hỏng thì thử tóm tắt lại; chưa tới đó thì không thử (mỗi lần có thể mất cả hạn chờ).
func TestAutoSummarizeRetriesAfterContextGrows(t *testing.T) {
	m := &scriptedModel{
		steps: []workStep{
			{toolCall: true, in: 800, out: 20}, // 820: chạm ngưỡng, tóm tắt hỏng
			{toolCall: true, in: 850, out: 20}, // 870 < 820+100: chưa thử lại
			{toolCall: true, in: 910, out: 20}, // 930 ≥ 920: thử lại, thành công
		},
		summaryErrs: []error{&requestTimeoutError{timeout: 6 * time.Minute, idle: true, cause: context.DeadlineExceeded}},
	}
	rc, summaryID := runScripted(t, m, "refactor the scanner")
	require.Empty(t, rc.Error)
	require.Equal(t, "done", rc.Text)
	require.Equal(t, []string{"work", "summary", "work", "work", "summary", "work"}, m.seen())
	require.NotEmpty(t, summaryID, "the retried summary becomes the history cut point")
}

// Sự cố 03/11: session đã vượt trần thật thì mọi prompt mới đều bị provider từ chối
// ngay request đầu (ngưỡng chỉ được xét sau một bước thành công) và session kẹt luôn.
// Nay lỗi tràn context của provider dẫn tới tóm tắt rồi làm tiếp.
func TestProviderContextOverflowSummarizesAndContinues(t *testing.T) {
	m := &scriptedModel{steps: []workStep{{err: vllmOverflowErr()}}}
	rc, summaryID := runScripted(t, m, "tiếp tục code")
	require.Empty(t, rc.Error)
	require.Equal(t, "done", rc.Text)
	require.Equal(t, []string{"work", "summary", "work"}, m.seen())
	require.NotEmpty(t, summaryID)
	require.Contains(t, m.prompts[2], "tiếp tục code", "the continuation carries the user's request")
}

// Vẫn tràn ngay ở bước đầu sau khi tóm tắt: tóm tắt nữa không giúp, lượt dừng với
// lỗi của provider thay vì tóm tắt lặp mãi.
func TestProviderContextOverflowRightAfterSummaryStops(t *testing.T) {
	m := &scriptedModel{steps: []workStep{{err: vllmOverflowErr()}, {err: vllmOverflowErr()}}}
	rc, _ := runScripted(t, m, "tiếp tục code")
	require.Contains(t, rc.Error, "maximum context length")
	require.Equal(t, []string{"work", "summary", "work"}, m.seen())
}

// Tóm tắt để gỡ tràn cũng hỏng: lượt dừng với lỗi tràn của provider.
func TestProviderContextOverflowSummaryFailureStops(t *testing.T) {
	m := &scriptedModel{
		steps:       []workStep{{err: vllmOverflowErr()}},
		summaryErrs: []error{&requestTimeoutError{timeout: 6 * time.Minute, idle: true, cause: context.DeadlineExceeded}},
	}
	rc, summaryID := runScripted(t, m, "tiếp tục code")
	require.Contains(t, rc.Error, "maximum context length")
	require.Equal(t, []string{"work", "summary"}, m.seen())
	require.Empty(t, summaryID)
}

// Đã chạy xong tool sau lần tóm tắt rồi mới tràn: lịch sử đã lớn thêm, tóm tắt
// lại vẫn có ích (chỉ tràn ngay ở bước đầu sau tóm tắt mới là vô ích).
func TestProviderContextOverflowAfterProgressSummarizesAgain(t *testing.T) {
	m := &scriptedModel{steps: []workStep{
		{err: vllmOverflowErr()},
		{toolCall: true, in: 300, out: 10},
		{err: vllmOverflowErr()},
	}}
	rc, _ := runScripted(t, m, "tiếp tục code")
	require.Empty(t, rc.Error)
	require.Equal(t, "done", rc.Text)
	require.Equal(t, []string{"work", "summary", "work", "work", "summary", "work"}, m.seen())
}

// Huỷ lúc đang tóm tắt để gỡ tràn: lượt dừng như huỷ, không báo lỗi tràn.
func TestProviderContextOverflowCancelDuringSummaryStops(t *testing.T) {
	m := &scriptedModel{steps: []workStep{{err: vllmOverflowErr()}}, summaryErrs: []error{context.Canceled}}
	rc, _ := runScripted(t, m, "tiếp tục code")
	require.True(t, rc.Cancelled)
	require.Equal(t, []string{"work", "summary"}, m.seen())
}

func TestIsContextOverflow(t *testing.T) {
	t.Parallel()
	for name, tc := range map[string]struct {
		err  error
		want bool
	}{
		"vllm behind litellm":    {vllmOverflowErr(), true},
		"fantasy parsed":         {&fantasy.ProviderError{StatusCode: 400, ContextTooLargeErr: true}, true},
		"other bad request":      {&fantasy.ProviderError{Title: "bad request", StatusCode: 400, Message: "invalid tool schema"}, false},
		"not a provider error":   {errors.New("maximum context length is 200000 tokens"), false},
		"subagent tool overflow": {&fantasy.ToolExecutionError{ToolName: "agent", Err: vllmOverflowErr()}, false},
		"own overflow check":     {&contextOverflowError{used: 900, window: 1000, threshold: 200}, false},
		"nil":                    {nil, false},
	} {
		require.Equal(t, tc.want, isContextOverflow(tc.err), name)
	}
}
