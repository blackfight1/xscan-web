package handler

import (
	"fmt"
	"net/http"
	"net/url"
	"strings"

	"xscan-web/internal/models"
	"xscan-web/internal/scanner"

	"github.com/gin-gonic/gin"
)

type Handler struct {
	scanner *scanner.Scanner
}

func New(s *scanner.Scanner) *Handler {
	return &Handler{scanner: s}
}

// CreateTask POST /api/tasks - supports batch targets
func (h *Handler) CreateTask(c *gin.Context) {
	var req models.CreateTaskRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request body"})
		return
	}

	mode := strings.ToLower(strings.TrimSpace(req.Mode))
	if mode == "" {
		mode = models.ScanModeDomain
	}

	// Collect targets: from targets array, or fallback to single root_domain / target_url
	var targets []string
	for _, t := range req.Targets {
		t = strings.TrimSpace(t)
		if t != "" {
			targets = append(targets, t)
		}
	}
	// Backward compat: if targets empty, use old fields
	if len(targets) == 0 {
		if mode == models.ScanModeDomain && strings.TrimSpace(req.RootDomain) != "" {
			targets = []string{strings.TrimSpace(req.RootDomain)}
		} else if mode == models.ScanModeURL && strings.TrimSpace(req.TargetURL) != "" {
			targets = []string{strings.TrimSpace(req.TargetURL)}
		}
	}

	if len(targets) == 0 {
		if mode == models.ScanModeDomain {
			c.JSON(http.StatusBadRequest, gin.H{"error": "at least one domain is required"})
		} else {
			c.JSON(http.StatusBadRequest, gin.H{"error": "at least one URL is required"})
		}
		return
	}

	switch mode {
	case models.ScanModeDomain:
		// Domain mode: each domain creates a separate task
		var createdTasks []*models.Task
		for _, domain := range targets {
			domain = strings.TrimSpace(domain)
			if domain == "" {
				continue
			}
			task, err := h.scanner.CreateTask(mode, domain, "")
			if err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("failed to create task for %s: %v", domain, err)})
				return
			}
			createdTasks = append(createdTasks, task)
		}
		if len(createdTasks) == 1 {
			c.JSON(http.StatusCreated, gin.H{"message": "Task created", "task": createdTasks[0]})
		} else {
			c.JSON(http.StatusCreated, gin.H{"message": fmt.Sprintf("%d tasks created", len(createdTasks)), "tasks": createdTasks})
		}

	case models.ScanModeURL:
		// URL mode: validate all URLs first
		var validURLs []string
		for _, u := range targets {
			u = strings.TrimSpace(u)
			if u == "" {
				continue
			}
			parsed, err := url.ParseRequestURI(u)
			if err != nil || parsed.Scheme == "" || parsed.Host == "" {
				c.JSON(http.StatusBadRequest, gin.H{"error": fmt.Sprintf("invalid URL: %s", u)})
				return
			}
			if parsed.Scheme != "http" && parsed.Scheme != "https" {
				c.JSON(http.StatusBadRequest, gin.H{"error": fmt.Sprintf("URL must start with http/https: %s", u)})
				return
			}
			validURLs = append(validURLs, u)
		}
		if len(validURLs) == 0 {
			c.JSON(http.StatusBadRequest, gin.H{"error": "no valid URLs provided"})
			return
		}

		// All URLs go into ONE task
		// For display, use first URL or count
		displayTarget := validURLs[0]
		if len(validURLs) > 1 {
			displayTarget = fmt.Sprintf("%s (+%d more)", validURLs[0], len(validURLs)-1)
		}
		targetURLsJoined := strings.Join(validURLs, "\n")

		task, err := h.scanner.CreateTask(mode, displayTarget, targetURLsJoined)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusCreated, gin.H{"message": "Task created", "task": task})

	default:
		c.JSON(http.StatusBadRequest, gin.H{"error": "mode must be domain or url"})
	}
}

// GetTasks GET /api/tasks
func (h *Handler) GetTasks(c *gin.Context) {
	tasks, err := h.scanner.GetTasks()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	if tasks == nil {
		tasks = []models.Task{}
	}
	c.JSON(http.StatusOK, gin.H{"tasks": tasks, "total_count": len(tasks)})
}

// GetTask GET /api/tasks/:id
func (h *Handler) GetTask(c *gin.Context) {
	id := c.Param("id")
	detail, err := h.scanner.GetTaskDetail(id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Task not found"})
		return
	}
	c.JSON(http.StatusOK, detail)
}

// DeleteTask DELETE /api/tasks/:id
func (h *Handler) DeleteTask(c *gin.Context) {
	id := c.Param("id")
	if err := h.scanner.DeleteTask(id); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "Task deleted"})
}

// GetReport GET /api/tasks/:id/report
func (h *Handler) GetReport(c *gin.Context) {
	id := c.Param("id")
	report, err := h.scanner.GetReport(id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"report": report})
}
