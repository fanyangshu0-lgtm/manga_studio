package domain

import "time"

type Position struct {
	X float64 `json:"x"`
	Y float64 `json:"y"`
}

type Viewport struct {
	X    float64 `json:"x"`
	Y    float64 `json:"y"`
	Zoom float64 `json:"zoom"`
}

type Node struct {
	ID       string         `json:"id"`
	Type     string         `json:"type"`
	Name     string         `json:"name"`
	Position Position       `json:"position"`
	Config   map[string]any `json:"config"`
}

type Edge struct {
	ID           string `json:"id"`
	Source       string `json:"source"`
	SourceHandle string `json:"sourceHandle"`
	Target       string `json:"target"`
	TargetHandle string `json:"targetHandle"`
}

type Workflow struct {
	ID        string    `json:"id"`
	ProjectID string    `json:"projectId"`
	Revision  int       `json:"revision"`
	Nodes     []Node    `json:"nodes"`
	Edges     []Edge    `json:"edges"`
	Viewport  Viewport  `json:"viewport"`
	UpdatedAt time.Time `json:"updatedAt"`
}

type Project struct {
	ID          string    `json:"id"`
	Name        string    `json:"name"`
	Description string    `json:"description"`
	CreatedAt   time.Time `json:"createdAt"`
	UpdatedAt   time.Time `json:"updatedAt"`
}

type Provider struct {
	ID               string         `json:"id"`
	Name             string         `json:"name"`
	Kind             string         `json:"kind"`
	BaseURL          string         `json:"baseUrl"`
	Capabilities     []string       `json:"capabilities"`
	Models           ProviderModels `json:"models"`
	Weight           int            `json:"weight"`
	Enabled          bool           `json:"enabled"`
	SecretCiphertext string         `json:"-"`
	SecretHint       string         `json:"secretHint"`
	CreatedAt        time.Time      `json:"createdAt"`
	UpdatedAt        time.Time      `json:"updatedAt"`
}

type ProviderModels struct {
	Script       string `json:"script"`
	VideoFast    string `json:"videoFast"`
	VideoQuality string `json:"videoQuality"`
	VideoDefault string `json:"videoDefault"`
}

type RunStatus string

const (
	RunQueued      RunStatus = "queued"
	RunRunning     RunStatus = "running"
	RunSucceeded   RunStatus = "succeeded"
	RunFailed      RunStatus = "failed"
	RunCancelled   RunStatus = "cancelled"
	RunInterrupted RunStatus = "interrupted"
)

type NodeRun struct {
	NodeID     string         `json:"nodeId"`
	NodeName   string         `json:"nodeName"`
	Status     RunStatus      `json:"status"`
	Progress   int            `json:"progress"`
	Message    string         `json:"message"`
	Outputs    map[string]any `json:"outputs,omitempty"`
	StartedAt  *time.Time     `json:"startedAt,omitempty"`
	FinishedAt *time.Time     `json:"finishedAt,omitempty"`
}

type Run struct {
	ID            string                    `json:"id"`
	ProjectID     string                    `json:"projectId"`
	Status        RunStatus                 `json:"status"`
	Progress      int                       `json:"progress"`
	Message       string                    `json:"message"`
	Workflow      Workflow                  `json:"workflow"`
	Nodes         map[string]NodeRun        `json:"nodes"`
	VideoTasks    map[string]VideoTaskState `json:"videoTasks,omitempty"`
	CreatedAt     time.Time                 `json:"createdAt"`
	StartedAt     *time.Time                `json:"startedAt,omitempty"`
	FinishedAt    *time.Time                `json:"finishedAt,omitempty"`
	InterruptedAt *time.Time                `json:"interruptedAt,omitempty"`
}

type VideoTaskState struct {
	ShotID  string `json:"shotId"`
	TaskID  string `json:"taskId,omitempty"`
	Status  string `json:"status"`
	AssetID string `json:"assetId,omitempty"`
	Error   string `json:"error,omitempty"`
}

type Asset struct {
	ID        string    `json:"id"`
	ProjectID string    `json:"projectId"`
	RunID     string    `json:"runId"`
	NodeID    string    `json:"nodeId"`
	Kind      string    `json:"kind"`
	Path      string    `json:"-"`
	MIME      string    `json:"mime"`
	Size      int64     `json:"size"`
	SHA256    string    `json:"sha256"`
	CreatedAt time.Time `json:"createdAt"`
}

type RunEvent struct {
	Type      string    `json:"type"`
	RunID     string    `json:"runId"`
	NodeID    string    `json:"nodeId,omitempty"`
	Status    RunStatus `json:"status"`
	Progress  int       `json:"progress"`
	Message   string    `json:"message"`
	Timestamp time.Time `json:"timestamp"`
}
