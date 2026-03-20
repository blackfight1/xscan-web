package handler

import (
	"net/http"

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

// CreateTask POST /api/tasks
func (h *Handler) CreateTask(c *gin.Context) {
	var req models.CreateTaskRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "root_domain is required"})
		return
	}

	task, err := h.scanner.CreateTask(req.RootDomain)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"message": "任务创建成功",
		"task":    task,
	})
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

	c.JSON(http.StatusOK, gin.H{
		"tasks":       tasks,
		"total_count": len(tasks),
	})
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

	c.JSON(http.StatusOK, gin.H{"message": "任务删除成功"})
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
