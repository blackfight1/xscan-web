package models

import "time"

// Task status constants
const (
	ScanModeDomain = "domain"
	ScanModeURL    = "url"
)

const (
	StatusPending   = "pending"
	StatusSubdomain = "subdomain_collecting"
	StatusHttpx     = "httpx_probing"
	StatusScanning  = "xss_scanning"
	StatusCompleted = "completed"
	StatusFailed    = "failed"
	StatusCancelled = "cancelled"
)

type Task struct {
	ID             string     `json:"id"`
	ScanMode       string     `json:"scan_mode"`
	RootDomain     string     `json:"root_domain"`
	TargetURL      string     `json:"target_url"`
	Status         string     `json:"status"`
	SubdomainCount int        `json:"subdomain_count"`
	AliveCount     int        `json:"alive_count"`
	XssCount       int        `json:"xss_count"`
	CurrentStep    string     `json:"current_step"`
	ErrorMessage   string     `json:"error_message"`
	CreatedAt      time.Time  `json:"created_at"`
	UpdatedAt      time.Time  `json:"updated_at"`
	FinishedAt     *time.Time `json:"finished_at"`
}

type Subdomain struct {
	ID        int       `json:"id"`
	TaskID    string    `json:"task_id"`
	Domain    string    `json:"domain"`
	CreatedAt time.Time `json:"created_at"`
}

type AliveURL struct {
	ID         int       `json:"id"`
	TaskID     string    `json:"task_id"`
	URL        string    `json:"url"`
	StatusCode int       `json:"status_code"`
	Title      string    `json:"title"`
	CreatedAt  time.Time `json:"created_at"`
}

type XssResult struct {
	ID            int       `json:"id"`
	TaskID        string    `json:"task_id"`
	URL           string    `json:"url"`
	Payload       string    `json:"payload"`
	Param         string    `json:"param"`
	Position      string    `json:"position"`
	ReportContent string    `json:"report_content"`
	CreatedAt     time.Time `json:"created_at"`
}

// API request/response types
type CreateTaskRequest struct {
	Mode       string   `json:"mode"`
	RootDomain string   `json:"root_domain"`
	TargetURL  string   `json:"target_url"`
	Targets    []string `json:"targets"`
}

type TaskListResponse struct {
	Tasks      []Task `json:"tasks"`
	TotalCount int    `json:"total_count"`
}

type TaskDetailResponse struct {
	Task       Task        `json:"task"`
	Subdomains []Subdomain `json:"subdomains"`
	AliveURLs  []AliveURL  `json:"alive_urls"`
	XssResults []XssResult `json:"xss_results"`
}
