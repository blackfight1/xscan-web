package scanner

import (
	"bufio"
	"database/sql"
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"xscan-web/internal/database"
	"xscan-web/internal/models"

	"github.com/google/uuid"
)

type Scanner struct {
	xscanPath     string
	toolsDir      string
	resultsDir    string
	maxConcurrent int
	running       int
	mu            sync.Mutex
	taskQueue     chan string
	quit          chan struct{}
}

type Config struct {
	XscanPath     string
	ToolsDir      string
	ResultsDir    string
	MaxConcurrent int
}

func New(cfg Config) *Scanner {
	if cfg.MaxConcurrent <= 0 {
		cfg.MaxConcurrent = 2
	}

	os.MkdirAll(cfg.ResultsDir, 0755)
	os.MkdirAll(cfg.ToolsDir, 0755)

	s := &Scanner{
		xscanPath:     cfg.XscanPath,
		toolsDir:      cfg.ToolsDir,
		resultsDir:    cfg.ResultsDir,
		maxConcurrent: cfg.MaxConcurrent,
		taskQueue:     make(chan string, 100),
		quit:          make(chan struct{}),
	}

	go s.worker()
	return s
}

func (s *Scanner) worker() {
	for {
		select {
		case taskID := <-s.taskQueue:
			s.mu.Lock()
			s.running++
			s.mu.Unlock()

			s.executeTask(taskID)

			s.mu.Lock()
			s.running--
			s.mu.Unlock()
		case <-s.quit:
			return
		}
	}
}

func (s *Scanner) Stop() {
	close(s.quit)
}

// CreateTask creates a new task and enqueues it
func (s *Scanner) CreateTask(rootDomain string) (*models.Task, error) {
	id := uuid.New().String()[:8]
	now := time.Now()

	_, err := database.DB.Exec(
		`INSERT INTO tasks (id, root_domain, status, current_step, created_at, updated_at)
		 VALUES (?, ?, ?, ?, ?, ?)`,
		id, rootDomain, models.StatusPending, "等待执行", now, now,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create task: %w", err)
	}

	task := &models.Task{
		ID:          id,
		RootDomain:  rootDomain,
		Status:      models.StatusPending,
		CurrentStep: "等待执行",
		CreatedAt:   now,
		UpdatedAt:   now,
	}

	// Enqueue task
	s.taskQueue <- id

	return task, nil
}

// GetTask returns a task by ID
func (s *Scanner) GetTask(id string) (*models.Task, error) {
	task := &models.Task{}
	var finishedAt sql.NullTime

	err := database.DB.QueryRow(
		`SELECT id, root_domain, status, subdomain_count, alive_count, xss_count,
		        current_step, error_message, created_at, updated_at, finished_at
		 FROM tasks WHERE id = ?`, id,
	).Scan(
		&task.ID, &task.RootDomain, &task.Status,
		&task.SubdomainCount, &task.AliveCount, &task.XssCount,
		&task.CurrentStep, &task.ErrorMessage,
		&task.CreatedAt, &task.UpdatedAt, &finishedAt,
	)
	if err != nil {
		return nil, err
	}

	if finishedAt.Valid {
		task.FinishedAt = &finishedAt.Time
	}
	return task, nil
}

// GetTasks returns all tasks
func (s *Scanner) GetTasks() ([]models.Task, error) {
	rows, err := database.DB.Query(
		`SELECT id, root_domain, status, subdomain_count, alive_count, xss_count,
		        current_step, error_message, created_at, updated_at, finished_at
		 FROM tasks ORDER BY created_at DESC`,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var tasks []models.Task
	for rows.Next() {
		var task models.Task
		var finishedAt sql.NullTime
		err := rows.Scan(
			&task.ID, &task.RootDomain, &task.Status,
			&task.SubdomainCount, &task.AliveCount, &task.XssCount,
			&task.CurrentStep, &task.ErrorMessage,
			&task.CreatedAt, &task.UpdatedAt, &finishedAt,
		)
		if err != nil {
			continue
		}
		if finishedAt.Valid {
			task.FinishedAt = &finishedAt.Time
		}
		tasks = append(tasks, task)
	}
	return tasks, nil
}

// GetTaskDetail returns task with all related data
func (s *Scanner) GetTaskDetail(id string) (*models.TaskDetailResponse, error) {
	task, err := s.GetTask(id)
	if err != nil {
		return nil, err
	}

	detail := &models.TaskDetailResponse{
		Task: *task,
	}

	// Get subdomains
	rows, err := database.DB.Query("SELECT id, task_id, domain, created_at FROM subdomains WHERE task_id = ?", id)
	if err == nil {
		defer rows.Close()
		for rows.Next() {
			var sub models.Subdomain
			rows.Scan(&sub.ID, &sub.TaskID, &sub.Domain, &sub.CreatedAt)
			detail.Subdomains = append(detail.Subdomains, sub)
		}
	}

	// Get alive URLs
	rows2, err := database.DB.Query("SELECT id, task_id, url, status_code, title, created_at FROM alive_urls WHERE task_id = ?", id)
	if err == nil {
		defer rows2.Close()
		for rows2.Next() {
			var alive models.AliveURL
			rows2.Scan(&alive.ID, &alive.TaskID, &alive.URL, &alive.StatusCode, &alive.Title, &alive.CreatedAt)
			detail.AliveURLs = append(detail.AliveURLs, alive)
		}
	}

	// Get XSS results
	rows3, err := database.DB.Query("SELECT id, task_id, url, payload, param, position, report_content, created_at FROM xss_results WHERE task_id = ?", id)
	if err == nil {
		defer rows3.Close()
		for rows3.Next() {
			var xss models.XssResult
			rows3.Scan(&xss.ID, &xss.TaskID, &xss.URL, &xss.Payload, &xss.Param, &xss.Position, &xss.ReportContent, &xss.CreatedAt)
			detail.XssResults = append(detail.XssResults, xss)
		}
	}

	return detail, nil
}

// DeleteTask deletes a task and its results
func (s *Scanner) DeleteTask(id string) error {
	// Delete related data first
	database.DB.Exec("DELETE FROM xss_results WHERE task_id = ?", id)
	database.DB.Exec("DELETE FROM alive_urls WHERE task_id = ?", id)
	database.DB.Exec("DELETE FROM subdomains WHERE task_id = ?", id)

	_, err := database.DB.Exec("DELETE FROM tasks WHERE id = ?", id)
	if err != nil {
		return err
	}

	// Clean up result directory
	taskDir := filepath.Join(s.resultsDir, id)
	os.RemoveAll(taskDir)

	return nil
}

// GetReport returns the XSS report markdown content
func (s *Scanner) GetReport(taskID string) (string, error) {
	taskDir := filepath.Join(s.resultsDir, taskID)

	// Find *_xss.md files
	matches, err := filepath.Glob(filepath.Join(taskDir, "*_xss.md"))
	if err != nil || len(matches) == 0 {
		return "", fmt.Errorf("no report found for task %s", taskID)
	}

	var allContent strings.Builder
	for _, match := range matches {
		content, err := os.ReadFile(match)
		if err != nil {
			continue
		}
		allContent.WriteString(string(content))
		allContent.WriteString("\n\n---\n\n")
	}

	return allContent.String(), nil
}

// executeTask runs the full scan pipeline
func (s *Scanner) executeTask(taskID string) {
	log.Printf("[Task %s] Starting execution", taskID)

	task, err := s.GetTask(taskID)
	if err != nil {
		log.Printf("[Task %s] Failed to get task: %v", taskID, err)
		return
	}

	taskDir := filepath.Join(s.resultsDir, taskID)
	os.MkdirAll(taskDir, 0755)

	// Step 1: Subdomain collection
	s.updateTaskStatus(taskID, models.StatusSubdomain, "正在收集子域名...")
	subdomains, err := s.collectSubdomains(task.RootDomain, taskDir)
	if err != nil {
		s.failTask(taskID, fmt.Sprintf("子域名收集失败: %v", err))
		return
	}

	// Save subdomains to DB
	for _, sub := range subdomains {
		database.DB.Exec("INSERT INTO subdomains (task_id, domain) VALUES (?, ?)", taskID, sub)
	}
	s.updateTaskCount(taskID, "subdomain_count", len(subdomains))
	log.Printf("[Task %s] Found %d subdomains", taskID, len(subdomains))

	if len(subdomains) == 0 {
		// If no subdomains found, use the root domain directly
		subdomains = []string{task.RootDomain}
	}

	// Step 2: HTTP alive detection
	s.updateTaskStatus(taskID, models.StatusHttpx, "正在探测存活主机...")
	aliveURLs, err := s.probeAlive(subdomains, taskDir)
	if err != nil {
		s.failTask(taskID, fmt.Sprintf("存活探测失败: %v", err))
		return
	}

	// Save alive URLs to DB
	for _, u := range aliveURLs {
		database.DB.Exec("INSERT INTO alive_urls (task_id, url) VALUES (?, ?)", taskID, u)
	}
	s.updateTaskCount(taskID, "alive_count", len(aliveURLs))
	log.Printf("[Task %s] Found %d alive URLs", taskID, len(aliveURLs))

	if len(aliveURLs) == 0 {
		s.failTask(taskID, "未发现存活的URL")
		return
	}

	// Step 3: XSS scanning with xscan
	s.updateTaskStatus(taskID, models.StatusScanning, "正在进行XSS扫描...")
	xssCount, err := s.runXscan(taskID, aliveURLs, taskDir)
	if err != nil {
		s.failTask(taskID, fmt.Sprintf("XSS扫描失败: %v", err))
		return
	}

	s.updateTaskCount(taskID, "xss_count", xssCount)

	// Step 4: Complete
	now := time.Now()
	database.DB.Exec(
		`UPDATE tasks SET status = ?, current_step = ?, updated_at = ?, finished_at = ? WHERE id = ?`,
		models.StatusCompleted, "扫描完成", now, now, taskID,
	)
	log.Printf("[Task %s] Completed. Found %d XSS vulnerabilities", taskID, xssCount)
}

// collectSubdomains uses subfinder to collect subdomains
func (s *Scanner) collectSubdomains(domain string, taskDir string) ([]string, error) {
	outputFile := filepath.Join(taskDir, "subdomains.txt")

	// Try subfinder first
	subfinderPath := filepath.Join(s.toolsDir, "subfinder")
	if _, err := os.Stat(subfinderPath); os.IsNotExist(err) {
		subfinderPath = "subfinder" // Try system PATH
	}

	cmd := exec.Command(subfinderPath, "-d", domain, "-silent", "-o", outputFile)
	cmd.Dir = taskDir
	output, err := cmd.CombinedOutput()
	if err != nil {
		log.Printf("[Subfinder] Error: %v, Output: %s", err, string(output))
		// If subfinder fails, create a file with just the root domain
		os.WriteFile(outputFile, []byte(domain+"\n"), 0644)
	}

	return readLines(outputFile)
}

// probeAlive uses httpx to probe alive hosts
func (s *Scanner) probeAlive(subdomains []string, taskDir string) ([]string, error) {
	inputFile := filepath.Join(taskDir, "subdomains_all.txt")
	outputFile := filepath.Join(taskDir, "alive.txt")

	// Write subdomains to input file
	var content strings.Builder
	for _, sub := range subdomains {
		content.WriteString(sub + "\n")
	}
	if err := os.WriteFile(inputFile, []byte(content.String()), 0644); err != nil {
		return nil, fmt.Errorf("failed to write input file: %w", err)
	}

	// Try httpx
	httpxPath := filepath.Join(s.toolsDir, "httpx")
	if _, err := os.Stat(httpxPath); os.IsNotExist(err) {
		httpxPath = "httpx" // Try system PATH
	}

	cmd := exec.Command(httpxPath, "-l", inputFile, "-silent", "-o", outputFile, "-no-color")
	cmd.Dir = taskDir
	output, err := cmd.CombinedOutput()
	if err != nil {
		log.Printf("[Httpx] Error: %v, Output: %s", err, string(output))
		// If httpx fails, prefix subdomains with https://
		var urls []string
		for _, sub := range subdomains {
			if !strings.HasPrefix(sub, "http") {
				urls = append(urls, "https://"+sub)
			} else {
				urls = append(urls, sub)
			}
		}
		urlContent := strings.Join(urls, "\n")
		os.WriteFile(outputFile, []byte(urlContent), 0644)
	}

	return readLines(outputFile)
}

// runXscan runs xscan spider on each alive URL
func (s *Scanner) runXscan(taskID string, urls []string, taskDir string) (int, error) {
	xssOutputDir := filepath.Join(taskDir, "xscan_output")
	os.MkdirAll(xssOutputDir, 0755)

	totalXss := 0

	for i, url := range urls {
		log.Printf("[Task %s] Scanning (%d/%d): %s", taskID, i+1, len(urls), url)
		s.updateTaskStatus(taskID, models.StatusScanning,
			fmt.Sprintf("正在扫描 (%d/%d): %s", i+1, len(urls), url))

		cmd := exec.Command(s.xscanPath, "spider", "-u", url, "--output-dir", xssOutputDir)
		cmd.Dir = filepath.Dir(s.xscanPath)
		output, err := cmd.CombinedOutput()
		if err != nil {
			log.Printf("[Task %s] xscan error for %s: %v, output: %s", taskID, url, err, string(output))
			continue
		}
	}

	// Parse XSS results from output directory
	matches, _ := filepath.Glob(filepath.Join(xssOutputDir, "*_xss.md"))
	for _, match := range matches {
		content, err := os.ReadFile(match)
		if err != nil {
			continue
		}
		reportContent := string(content)
		if strings.TrimSpace(reportContent) == "" {
			continue
		}

		// Save to database
		database.DB.Exec(
			`INSERT INTO xss_results (task_id, url, report_content) VALUES (?, ?, ?)`,
			taskID, match, reportContent,
		)
		totalXss++
	}

	// Also copy xss.md files to task directory for easy access
	for _, match := range matches {
		destName := filepath.Base(match)
		destPath := filepath.Join(taskDir, destName)
		content, _ := os.ReadFile(match)
		os.WriteFile(destPath, content, 0644)
	}

	return totalXss, nil
}

func (s *Scanner) updateTaskStatus(taskID, status, step string) {
	database.DB.Exec(
		`UPDATE tasks SET status = ?, current_step = ?, updated_at = ? WHERE id = ?`,
		status, step, time.Now(), taskID,
	)
}

func (s *Scanner) updateTaskCount(taskID, field string, count int) {
	database.DB.Exec(
		fmt.Sprintf(`UPDATE tasks SET %s = ?, updated_at = ? WHERE id = ?`, field),
		count, time.Now(), taskID,
	)
}

func (s *Scanner) failTask(taskID, errMsg string) {
	log.Printf("[Task %s] Failed: %s", taskID, errMsg)
	now := time.Now()
	database.DB.Exec(
		`UPDATE tasks SET status = ?, current_step = ?, error_message = ?, updated_at = ?, finished_at = ? WHERE id = ?`,
		models.StatusFailed, "执行失败", errMsg, now, now, taskID,
	)
}

func readLines(filename string) ([]string, error) {
	file, err := os.Open(filename)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	var lines []string
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line != "" {
			lines = append(lines, line)
		}
	}
	return lines, scanner.Err()
}
