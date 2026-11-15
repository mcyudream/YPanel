package service

import "time"

type ChatMessage struct {
	ID          string           `json:"id"`
	Role        string           `json:"role"`
	Content     string           `json:"content"`
	Reasoning   string           `json:"reasoning,omitempty"`
	Steps       []ChatStep       `json:"steps"`
	Knowledge   []ChatKnowledge  `json:"knowledge,omitempty"`
	Segments    []ChatSegment    `json:"segments,omitempty"`
	Attachments []ChatAttachment `json:"attachments,omitempty"`
	Pending     bool             `json:"pending,omitempty"`
	Error       bool             `json:"error,omitempty"`
	Interrupted bool             `json:"interrupted,omitempty"`
	DurationMS  int64            `json:"durationMs,omitempty"`
	CreatedAt   time.Time        `json:"createdAt"`
}

type ChatSegment struct {
	Type      string           `json:"type"`
	ID        string           `json:"id,omitempty"`
	Text      string           `json:"text,omitempty"`
	Name      string           `json:"name,omitempty"`
	Args      string           `json:"args,omitempty"`
	Summary   string           `json:"summary,omitempty"`
	Output    string           `json:"output,omitempty"`
	OutputID  string           `json:"outputId,omitempty"`
	Status    string           `json:"status,omitempty"`
	Risk      string           `json:"risk,omitempty"`
	AskID     string           `json:"askId,omitempty"`
	Questions []map[string]any `json:"questions,omitempty"`
	Done      bool             `json:"done,omitempty"`
}

type ChatAttachment struct {
	ID   string `json:"id"`
	Name string `json:"name"`
	MIME string `json:"mime"`
	Size int64  `json:"size"`
	Kind string `json:"kind"`
}

// AIChatSettings 的默认值在解码前填充，避免 bool=false 被默认值覆盖。
type AIChatSettings struct {
	ProviderID        uint     `json:"providerId"`
	Model             string   `json:"model"`
	Mode              string   `json:"mode"`
	MaxRounds         int      `json:"maxRounds"`
	ContextLimit      int      `json:"contextLimit"`
	MaxOutputTokens   int      `json:"maxOutputTokens"`
	AutoCompact       bool     `json:"autoCompact"`
	CompactThreshold  float64  `json:"compactThreshold"`
	RetainTurns       int      `json:"retainTurns"`
	SummaryProviderID uint     `json:"summaryProviderId"`
	SummaryModel      string   `json:"summaryModel"`
	SystemPrompt      string   `json:"systemPrompt"`
	Temperature       *float64 `json:"temperature,omitempty"`
	Skills            []string `json:"skills"`
	Memory            bool     `json:"memory"`
	Knowledge         bool     `json:"knowledge"`
	Tools             bool     `json:"tools"`
}

type AIContextUsage struct {
	Tokens         int    `json:"tokens"`
	Limit          int    `json:"limit"`
	OutputReserve  int    `json:"outputReserve"`
	System         int    `json:"system"`
	History        int    `json:"history"`
	Tools          int    `json:"tools"`
	Attachments    int    `json:"attachments"`
	Estimated      bool   `json:"estimated"`
	InputTokens    int    `json:"inputTokens,omitempty"`
	OutputTokens   int    `json:"outputTokens,omitempty"`
	Summary        string `json:"summary,omitempty"`
	SummaryThrough int    `json:"summaryThrough,omitempty"`
}

type AIChatSnapshot struct {
	ConversationID uint           `json:"conversationId"`
	Title          string         `json:"title"`
	Version        uint           `json:"version"`
	RunID          string         `json:"runId"`
	Status         string         `json:"status"`
	LastEventID    int            `json:"lastEventId"`
	Messages       []ChatMessage  `json:"messages"`
	Total          int            `json:"total"`
	HasMore        bool           `json:"hasMore"`
	Settings       AIChatSettings `json:"settings"`
	Context        AIContextUsage `json:"context"`
}

type AIChatStart struct {
	ConversationID uint           `json:"conversationId"`
	RequestID      string         `json:"requestId"`
	Version        *uint          `json:"version,omitempty"`
	Content        string         `json:"content"`
	AttachmentIDs  []string       `json:"attachmentIds"`
	Settings       AIChatSettings `json:"settings"`
	Scene          string         `json:"scene"`
	Focus          string         `json:"focus"`
}

type aiRunContextKey struct{}
type aiRunIdentity struct {
	UserID, ConversationID uint
	RunID                  string
}

type aiToolCallContextKey struct{}
