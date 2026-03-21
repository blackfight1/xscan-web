package scanner

import (
	"bufio"
	"context"
	"database/sql"
	"errors"
	"fmt"
	"log"
	"net/url"
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
	executions    map[string]*taskExecution
}

type Config struct {
	XscanPath     string
	ToolsDir      string
	ResultsDir    string
	MaxConcurrent int
}

type taskExecution struct {
	cancel context.CancelFunc
	cmd    *exec.Cmd
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
		executions:    make(map[string]*taskExecution),
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
			ctx := s.beginExecution(taskID)
			s.mu.Unlock()

			s.executeTask(ctx, taskID)

			s.mu.Lock()
			s.running--
			s.endExecution(taskID)
			s.mu.Unlock()
		case <-s.quit:
			return
		}
	}
}

func (s *Scanner) Stop() {
	s.mu.Lock()
	var cancels []context.CancelFunc
	for _, e := range s.executions {
		if e != nil && e.cancel != nil {
			cancels = append(cancels, e.cancel)
		}
	}
	s.mu.Unlock()

	for _, cancel := range cancels {
		cancel()
	}
	close(s.quit)
}

// CreateTask creates a new task and enqueues it.
func (s *Scanner) CreateTask(scanMode, rootDomain, targetURL string) (*models.Task, error) {
	mode := strings.TrimSpace(strings.ToLower(scanMode))
	if mode == "" {
		mode = models.ScanModeDomain
	}

	rootDomain = strings.TrimSpace(rootDomain)
	targetURL = strings.TrimSpace(targetURL)
	displayTarget := rootDomain
	if mode == models.ScanModeURL {
		displayTarget = targetURL
	}

	id := uuid.New().String()[:8]
	now := time.Now()

	_, err := database.DB.Exec(
		`INSERT INTO tasks (id, scan_mode, root_domain, target_url, status, current_step, created_at, updated_at)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?)`,
		id, mode, displayTarget, targetURL, models.StatusPending, "waiting", now, now,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create task: %w", err)
	}

	task := &models.Task{
		ID:          id,
		ScanMode:    mode,
		RootDomain:  displayTarget,
		TargetURL:   targetURL,
		Status:      models.StatusPending,
		CurrentStep: "waiting",
		CreatedAt:   now,
		UpdatedAt:   now,
	}

	s.taskQueue <- id
	return task, nil
}

// GetTask returns a task by ID.
func (s *Scanner) GetTask(id string) (*models.Task, error) {
	task := &models.Task{}
	var finishedAt sql.NullTime

	err := database.DB.QueryRow(
		`SELECT id, scan_mode, root_domain, target_url, status, subdomain_count, alive_count, xss_count,
		        current_step, error_message, created_at, updated_at, finished_at
		 FROM tasks WHERE id = ?`,
		id,
	).Scan(
		&task.ID, &task.ScanMode, &task.RootDomain, &task.TargetURL, &task.Status,
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

// GetTasks returns all tasks.
func (s *Scanner) GetTasks() ([]models.Task, error) {
	rows, err := database.DB.Query(
		`SELECT id, scan_mode, root_domain, target_url, status, subdomain_count, alive_count, xss_count,
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
			&task.ID, &task.ScanMode, &task.RootDomain, &task.TargetURL, &task.Status,
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

// GetTaskDetail returns task with all related data.
func (s *Scanner) GetTaskDetail(id string) (*models.TaskDetailResponse, error) {
	task, err := s.GetTask(id)
	if err != nil {
		return nil, err
	}

	detail := &models.TaskDetailResponse{Task: *task}

	rows, err := database.DB.Query("SELECT id, task_id, domain, created_at FROM subdomains WHERE task_id = ?", id)
	if err == nil {
		defer rows.Close()
		for rows.Next() {
			var sub models.Subdomain
			rows.Scan(&sub.ID, &sub.TaskID, &sub.Domain, &sub.CreatedAt)
			detail.Subdomains = append(detail.Subdomains, sub)
		}
	}

	rows2, err := database.DB.Query("SELECT id, task_id, url, status_code, title, created_at FROM alive_urls WHERE task_id = ?", id)
	if err == nil {
		defer rows2.Close()
		for rows2.Next() {
			var alive models.AliveURL
			rows2.Scan(&alive.ID, &alive.TaskID, &alive.URL, &alive.StatusCode, &alive.Title, &alive.CreatedAt)
			detail.AliveURLs = append(detail.AliveURLs, alive)
		}
	}

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

// DeleteTask deletes a task and its results.
func (s *Scanner) DeleteTask(id string) error {
	s.cancelExecution(id)

	database.DB.Exec("DELETE FROM xss_results WHERE task_id = ?", id)
	database.DB.Exec("DELETE FROM alive_urls WHERE task_id = ?", id)
	database.DB.Exec("DELETE FROM subdomains WHERE task_id = ?", id)

	_, err := database.DB.Exec("DELETE FROM tasks WHERE id = ?", id)
	if err != nil {
		return err
	}

	taskDir := filepath.Join(s.resultsDir, id)
	os.RemoveAll(taskDir)
	return nil
}

// GetReport returns the XSS report markdown content.
func (s *Scanner) GetReport(taskID string) (string, error) {
	taskDir := filepath.Join(s.resultsDir, taskID)
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

// executeTask runs either domain workflow or URL workflow.
func (s *Scanner) executeTask(ctx context.Context, taskID string) {
	log.Printf("[Task %s] Starting execution", taskID)

	task, err := s.GetTask(taskID)
	if err != nil {
		log.Printf("[Task %s] Failed to get task: %v", taskID, err)
		return
	}

	if errors.Is(ctx.Err(), context.Canceled) {
		s.cancelTask(taskID, "task cancelled")
		return
	}

	taskDir := filepath.Join(s.resultsDir, taskID)
	os.MkdirAll(taskDir, 0755)

	mode := strings.TrimSpace(strings.ToLower(task.ScanMode))
	if mode == "" {
		mode = models.ScanModeDomain
	}

	if mode == models.ScanModeURL {
		rawTarget := strings.TrimSpace(task.TargetURL)
		if rawTarget == "" {
			rawTarget = strings.TrimSpace(task.RootDomain)
		}

		// Split by newlines to support batch URLs
		var urlList []string
		for _, line := range strings.Split(rawTarget, "\n") {
			line = strings.TrimSpace(line)
			if line != "" && isValidHTTPURL(line) {
				urlList = append(urlList, line)
			}
		}
		if len(urlList) == 0 {
			s.failTask(taskID, fmt.Sprintf("no valid URLs found in target: %s", rawTarget))
			return
		}

		// Store all URLs as alive_urls
		for _, u := range urlList {
			database.DB.Exec("INSERT INTO alive_urls (task_id, url) VALUES (?, ?)", taskID, u)
		}
		s.updateTaskCount(taskID, "alive_count", len(urlList))

		var xssCount int
		var err error

		if len(urlList) == 1 {
			// Single URL: use xscan spider -u
			s.updateTaskStatus(taskID, models.StatusScanning, fmt.Sprintf("running xscan spider -u (%s)", urlList[0]))
			xssCount, err = s.runXscanSingleURL(ctx, taskID, urlList[0], taskDir)
			if err != nil {
				if errors.Is(err, context.Canceled) {
					s.cancelTask(taskID, "task cancelled")
					return
				}
				s.failTask(taskID, fmt.Sprintf("xscan spider -u failed: %v", err))
				return
			}
		} else {
			// Multiple URLs: use xscan spider -f
			s.updateTaskStatus(taskID, models.StatusScanning, fmt.Sprintf("running xscan spider -f (%d URLs)", len(urlList)))
			xssCount, err = s.runXscan(ctx, taskID, urlList, taskDir)
			if err != nil {
				if errors.Is(err, context.Canceled) {
					s.cancelTask(taskID, "task cancelled")
					return
				}
				s.failTask(taskID, fmt.Sprintf("xscan spider -f failed: %v", err))
				return
			}
		}

		if errors.Is(ctx.Err(), context.Canceled) {
			s.cancelTask(taskID, "task cancelled")
			return
		}

		s.updateTaskCount(taskID, "xss_count", xssCount)
		s.completeTask(taskID)
		log.Printf("[Task %s] Completed URL mode (%d URLs). Found %d XSS vulnerabilities", taskID, len(urlList), xssCount)
		return
	}

	rawDomains := strings.TrimSpace(task.TargetURL)
	if rawDomains == "" {
		rawDomains = strings.TrimSpace(task.RootDomain)
	}
	rootDomains := splitUniqueLines(rawDomains)
	if len(rootDomains) == 0 {
		s.failTask(taskID, "no valid root domains found")
		return
	}

	s.updateTaskStatus(taskID, models.StatusSubdomain, "collecting subdomains")
	subdomains, err := s.collectSubdomains(ctx, taskID, rootDomains, taskDir)
	if err != nil {
		if errors.Is(err, context.Canceled) {
			s.cancelTask(taskID, "task cancelled")
			return
		}
		s.failTask(taskID, fmt.Sprintf("subdomain collection failed: %v", err))
		return
	}

	for _, sub := range subdomains {
		database.DB.Exec("INSERT INTO subdomains (task_id, domain) VALUES (?, ?)", taskID, sub)
	}
	s.updateTaskCount(taskID, "subdomain_count", len(subdomains))
	log.Printf("[Task %s] Found %d subdomains", taskID, len(subdomains))

	if len(subdomains) == 0 {
		subdomains = rootDomains
	}

	s.updateTaskStatus(taskID, models.StatusHttpx, "probing alive hosts")
	aliveURLs, err := s.probeAlive(ctx, taskID, subdomains, taskDir)
	if err != nil {
		if errors.Is(err, context.Canceled) {
			s.cancelTask(taskID, "task cancelled")
			return
		}
		s.failTask(taskID, fmt.Sprintf("httpx probe failed: %v", err))
		return
	}

	for _, u := range aliveURLs {
		database.DB.Exec("INSERT INTO alive_urls (task_id, url) VALUES (?, ?)", taskID, u)
	}
	s.updateTaskCount(taskID, "alive_count", len(aliveURLs))
	log.Printf("[Task %s] Found %d alive URLs", taskID, len(aliveURLs))

	if len(aliveURLs) == 0 {
		s.failTask(taskID, "no alive URL found")
		return
	}

	s.updateTaskStatus(taskID, models.StatusScanning, "running xscan spider -f")
	xssCount, err := s.runXscan(ctx, taskID, aliveURLs, taskDir)
	if err != nil {
		if errors.Is(err, context.Canceled) {
			s.cancelTask(taskID, "task cancelled")
			return
		}
		s.failTask(taskID, fmt.Sprintf("xscan spider -f failed: %v", err))
		return
	}

	if errors.Is(ctx.Err(), context.Canceled) {
		s.cancelTask(taskID, "task cancelled")
		return
	}

	s.updateTaskCount(taskID, "xss_count", xssCount)
	s.completeTask(taskID)
	log.Printf("[Task %s] Completed domain mode. Found %d XSS vulnerabilities", taskID, xssCount)
}

// collectSubdomains uses subfinder to collect subdomains.
func (s *Scanner) collectSubdomains(ctx context.Context, taskID string, domains []string, taskDir string) ([]string, error) {
	taskDirAbs, err := filepath.Abs(taskDir)
	if err != nil {
		return nil, fmt.Errorf("failed to resolve task dir: %w", err)
	}
	domains = splitUniqueLines(strings.Join(domains, "\n"))
	if len(domains) == 0 {
		return nil, nil
	}

	subfinderPath := filepath.Join(s.toolsDir, "subfinder")
	if _, err := os.Stat(subfinderPath); os.IsNotExist(err) {
		subfinderPath = "subfinder"
	} else if absPath, err := filepath.Abs(subfinderPath); err == nil {
		subfinderPath = absPath
	}

	if len(domains) == 1 {
		outputFile := filepath.Join(taskDirAbs, "subdomains.txt")
		cmd := exec.CommandContext(ctx, subfinderPath, "-d", domains[0], "-silent", "-all", "-o", outputFile)
		cmd.Dir = taskDirAbs
		output, err := s.runCommand(taskID, cmd)
		if err != nil {
			if errors.Is(err, context.Canceled) {
				return nil, context.Canceled
			}
			log.Printf("[Subfinder] Error: %v, Output: %s", err, string(output))
			if writeErr := os.WriteFile(outputFile, []byte(domains[0]+"\n"), 0644); writeErr != nil {
				return nil, writeErr
			}
		}
		return readLines(outputFile)
	}

	inputFile := filepath.Join(taskDirAbs, "root_domains.txt")
	outputDir := filepath.Join(taskDirAbs, "subfinder_output")
	if err := os.WriteFile(inputFile, []byte(strings.Join(domains, "\n")), 0644); err != nil {
		return nil, fmt.Errorf("failed to write subfinder input file: %w", err)
	}
	if err := os.MkdirAll(outputDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create subfinder output dir: %w", err)
	}

	cmd := exec.CommandContext(ctx, subfinderPath, "-dL", inputFile, "-silent", "-all", "-oD", outputDir)
	cmd.Dir = taskDirAbs
	output, err := s.runCommand(taskID, cmd)
	if err != nil {
		if errors.Is(err, context.Canceled) {
			return nil, context.Canceled
		}
		log.Printf("[Subfinder] Batch error: %v, Output: %s", err, string(output))
		return domains, nil
	}

	subdomains, err := readLinesFromDir(outputDir)
	if err != nil {
		return nil, err
	}
	if len(subdomains) == 0 {
		return domains, nil
	}

	return subdomains, nil
}

// probeAlive uses httpx to probe alive hosts.
func (s *Scanner) probeAlive(ctx context.Context, taskID string, subdomains []string, taskDir string) ([]string, error) {
	taskDirAbs, err := filepath.Abs(taskDir)
	if err != nil {
		return nil, fmt.Errorf("failed to resolve task dir: %w", err)
	}
	inputFile := filepath.Join(taskDirAbs, "subdomains_all.txt")
	outputFile := filepath.Join(taskDirAbs, "alive.txt")

	var content strings.Builder
	for _, sub := range subdomains {
		content.WriteString(sub + "\n")
	}
	if err := os.WriteFile(inputFile, []byte(content.String()), 0644); err != nil {
		return nil, fmt.Errorf("failed to write input file: %w", err)
	}

	httpxPath := filepath.Join(s.toolsDir, "httpx")
	if _, err := os.Stat(httpxPath); os.IsNotExist(err) {
		httpxPath = "httpx"
	} else if absPath, err := filepath.Abs(httpxPath); err == nil {
		httpxPath = absPath
	}

	cmd := exec.CommandContext(ctx, httpxPath, "-l", inputFile, "-silent", "-o", outputFile, "-no-color")
	cmd.Dir = taskDirAbs
	output, err := s.runCommand(taskID, cmd)
	if err != nil {
		if errors.Is(err, context.Canceled) {
			return nil, context.Canceled
		}
		log.Printf("[Httpx] Error: %v, Output: %s", err, string(output))
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

// runXscan runs xscan spider in batch mode using -f.
func (s *Scanner) runXscan(ctx context.Context, taskID string, urls []string, taskDir string) (int, error) {
	taskDirAbs, err := filepath.Abs(taskDir)
	if err != nil {
		return 0, fmt.Errorf("failed to resolve task dir: %w", err)
	}

	xssOutputDir := filepath.Join(taskDirAbs, "xscan_output")
	os.MkdirAll(xssOutputDir, 0755)

	if len(urls) == 0 {
		return 0, nil
	}

	targetsFile := filepath.Join(taskDirAbs, "xscan_targets.txt")
	if err := os.WriteFile(targetsFile, []byte(strings.Join(urls, "\n")), 0644); err != nil {
		return 0, fmt.Errorf("failed to write xscan targets file: %w", err)
	}

	xscanPathAbs, xscanDir, err := s.resolveXscanExec()
	if err != nil {
		return 0, err
	}

	log.Printf("[Task %s] Scanning %d URLs by file: %s", taskID, len(urls), targetsFile)
	cmd := exec.CommandContext(ctx, xscanPathAbs, "--output-dir", xssOutputDir, "spider", "-f", targetsFile)
	cmd.Dir = xscanDir
	output, err := s.runCommand(taskID, cmd)
	if err != nil {
		if errors.Is(err, context.Canceled) {
			return 0, context.Canceled
		}
		if strings.Contains(string(output), "flag provided but not defined: -f") {
			log.Printf("[Task %s] -f not supported, fallback to -file", taskID)
			cmd = exec.CommandContext(ctx, xscanPathAbs, "--output-dir", xssOutputDir, "spider", "-file", targetsFile)
			cmd.Dir = xscanDir
			output, err = s.runCommand(taskID, cmd)
		}
		if err != nil {
			if errors.Is(err, context.Canceled) {
				return 0, context.Canceled
			}
			return 0, fmt.Errorf("xscan spider file scan failed: %w, output: %s", err, string(output))
		}
	}

	if errors.Is(ctx.Err(), context.Canceled) {
		return 0, context.Canceled
	}

	return s.parseAndStoreXSSReports(ctx, taskID, xssOutputDir, taskDirAbs)
}

// runXscanSingleURL runs xscan spider directly on one URL using -u.
func (s *Scanner) runXscanSingleURL(ctx context.Context, taskID, targetURL, taskDir string) (int, error) {
	taskDirAbs, err := filepath.Abs(taskDir)
	if err != nil {
		return 0, fmt.Errorf("failed to resolve task dir: %w", err)
	}

	xssOutputDir := filepath.Join(taskDirAbs, "xscan_output")
	os.MkdirAll(xssOutputDir, 0755)

	xscanPathAbs, xscanDir, err := s.resolveXscanExec()
	if err != nil {
		return 0, err
	}

	log.Printf("[Task %s] Scanning single URL by -u: %s", taskID, targetURL)
	cmd := exec.CommandContext(ctx, xscanPathAbs, "--output-dir", xssOutputDir, "spider", "-u", targetURL)
	cmd.Dir = xscanDir
	output, err := s.runCommand(taskID, cmd)
	if err != nil {
		if errors.Is(err, context.Canceled) {
			return 0, context.Canceled
		}
		if strings.Contains(string(output), "flag provided but not defined: -u") {
			log.Printf("[Task %s] -u not supported, fallback to file mode", taskID)
			return s.runXscan(ctx, taskID, []string{targetURL}, taskDirAbs)
		}
		return 0, fmt.Errorf("xscan spider url scan failed: %w, output: %s", err, string(output))
	}

	if errors.Is(ctx.Err(), context.Canceled) {
		return 0, context.Canceled
	}

	return s.parseAndStoreXSSReports(ctx, taskID, xssOutputDir, taskDirAbs)
}

func (s *Scanner) resolveXscanExec() (string, string, error) {
	xscanPath := strings.TrimSpace(s.xscanPath)
	if xscanPath == "" {
		return "", "", fmt.Errorf("xscan path is empty")
	}

	xscanPathAbs, err := filepath.Abs(xscanPath)
	if err != nil {
		return "", "", fmt.Errorf("failed to resolve xscan path: %w", err)
	}

	info, err := os.Stat(xscanPathAbs)
	if err != nil {
		return "", "", fmt.Errorf("xscan binary not found at %s: %w", xscanPathAbs, err)
	}
	if info.IsDir() {
		return "", "", fmt.Errorf("xscan path is a directory, expected executable file: %s", xscanPathAbs)
	}

	return xscanPathAbs, filepath.Dir(xscanPathAbs), nil
}

func (s *Scanner) parseAndStoreXSSReports(ctx context.Context, taskID, xssOutputDir, taskDir string) (int, error) {
	totalXSS := 0
	matches, err := filepath.Glob(filepath.Join(xssOutputDir, "*_xss.md"))
	if err != nil {
		return 0, err
	}

	for _, match := range matches {
		if errors.Is(ctx.Err(), context.Canceled) {
			return 0, context.Canceled
		}

		content, err := os.ReadFile(match)
		if err != nil {
			continue
		}
		reportContent := string(content)
		if strings.TrimSpace(reportContent) == "" {
			continue
		}

		database.DB.Exec(
			`INSERT INTO xss_results (task_id, url, report_content) VALUES (?, ?, ?)`,
			taskID, match, reportContent,
		)
		totalXSS++

		destName := filepath.Base(match)
		destPath := filepath.Join(taskDir, destName)
		os.WriteFile(destPath, content, 0644)
	}

	return totalXSS, nil
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

func (s *Scanner) completeTask(taskID string) {
	now := time.Now()
	database.DB.Exec(
		`UPDATE tasks SET status = ?, current_step = ?, updated_at = ?, finished_at = ? WHERE id = ?`,
		models.StatusCompleted, "completed", now, now, taskID,
	)
}

func (s *Scanner) failTask(taskID, errMsg string) {
	log.Printf("[Task %s] Failed: %s", taskID, errMsg)
	now := time.Now()
	database.DB.Exec(
		`UPDATE tasks SET status = ?, current_step = ?, error_message = ?, updated_at = ?, finished_at = ? WHERE id = ?`,
		models.StatusFailed, "failed", errMsg, now, now, taskID,
	)
}

func (s *Scanner) cancelTask(taskID, reason string) {
	log.Printf("[Task %s] Cancelled: %s", taskID, reason)
	now := time.Now()
	database.DB.Exec(
		`UPDATE tasks SET status = ?, current_step = ?, error_message = ?, updated_at = ?, finished_at = ? WHERE id = ?`,
		models.StatusCancelled, "cancelled", reason, now, now, taskID,
	)
}

func (s *Scanner) beginExecution(taskID string) context.Context {
	ctx, cancel := context.WithCancel(context.Background())
	s.executions[taskID] = &taskExecution{cancel: cancel}
	return ctx
}

func (s *Scanner) endExecution(taskID string) {
	delete(s.executions, taskID)
}

func (s *Scanner) runCommand(taskID string, cmd *exec.Cmd) ([]byte, error) {
	s.mu.Lock()
	if state, ok := s.executions[taskID]; ok {
		state.cmd = cmd
	}
	s.mu.Unlock()

	defer func() {
		s.mu.Lock()
		if state, ok := s.executions[taskID]; ok && state.cmd == cmd {
			state.cmd = nil
		}
		s.mu.Unlock()
	}()

	return cmd.CombinedOutput()
}

func (s *Scanner) cancelExecution(taskID string) {
	s.mu.Lock()
	state, ok := s.executions[taskID]
	if !ok || state == nil {
		s.mu.Unlock()
		return
	}
	cancel := state.cancel
	cmd := state.cmd
	s.mu.Unlock()

	if cancel != nil {
		cancel()
	}
	if cmd != nil && cmd.Process != nil {
		_ = cmd.Process.Kill()
	}
}

func isValidHTTPURL(raw string) bool {
	parsed, err := url.ParseRequestURI(raw)
	if err != nil || parsed.Host == "" {
		return false
	}
	return parsed.Scheme == "http" || parsed.Scheme == "https"
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

func readLinesFromDir(dir string) ([]string, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, err
	}

	var all []string
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}

		lines, err := readLines(filepath.Join(dir, entry.Name()))
		if err != nil {
			return nil, err
		}
		all = append(all, lines...)
	}

	return splitUniqueLines(strings.Join(all, "\n")), nil
}

func splitUniqueLines(raw string) []string {
	var lines []string
	seen := make(map[string]struct{})

	for _, line := range strings.Split(raw, "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		if _, ok := seen[line]; ok {
			continue
		}
		seen[line] = struct{}{}
		lines = append(lines, line)
	}

	return lines
}
