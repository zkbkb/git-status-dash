package main

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
	"sync"
	"time"
)

// WorkerPool manages concurrent git status operations
type WorkerPool struct {
	workers    int
	jobs       chan RepoJob
	results    chan GitStatus
	wg         sync.WaitGroup
	ctx        context.Context
	cancel     context.CancelFunc
}

type RepoJob struct {
	RepoPath string
	BaseDir  string
}

// NewWorkerPool creates a pool with optimal worker count
func NewWorkerPool() *WorkerPool {
	// Use CPU count * 2 for I/O bound work, but cap at reasonable limit
	workers := runtime.NumCPU() * 2
	if workers > 16 {
		workers = 16 // Don't go crazy on high-core machines
	}
	
	ctx, cancel := context.WithCancel(context.Background())
	
	return &WorkerPool{
		workers: workers,
		jobs:    make(chan RepoJob, workers*2), // Buffer jobs
		results: make(chan GitStatus, workers*2),
		ctx:     ctx,
		cancel:  cancel,
	}
}

func (wp *WorkerPool) Start() {
	for i := 0; i < wp.workers; i++ {
		wp.wg.Add(1)
		go wp.worker(i)
	}
}

func (wp *WorkerPool) worker(id int) {
	defer wp.wg.Done()
	
	for {
		select {
		case job, ok := <-wp.jobs:
			if !ok {
				return
			}
			
			// Process the git status
			status := getGitStatusOptimized(job.RepoPath, job.BaseDir)
			
			select {
			case wp.results <- status:
			case <-wp.ctx.Done():
				return
			}
			
		case <-wp.ctx.Done():
			return
		}
	}
}

func (wp *WorkerPool) Submit(job RepoJob) {
	select {
	case wp.jobs <- job:
	case <-wp.ctx.Done():
	}
}

func (wp *WorkerPool) Stop() {
	close(wp.jobs)
	wp.wg.Wait()
	close(wp.results)
	wp.cancel()
}

// Enhanced git status with optimizations
func getGitStatusOptimized(repoPath, baseDir string) GitStatus {
	relPath, _ := filepath.Rel(baseDir, repoPath)
	
	// Quick file system checks first
	info, err := os.Stat(repoPath)
	var modTime time.Time
	if err == nil {
		modTime = info.ModTime()
	}

	status := GitStatus{
		RepoPath:     repoPath,
		RelativePath: relPath,
		Symbol:       "⚠",
		Message:      "Error accessing repository",
		ModTime:      modTime,
	}

	// Fast context with shorter timeout for batch processing
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	// Use faster git commands where possible
	statusCmd := exec.CommandContext(ctx, "git", "-C", repoPath, "status", "--porcelain", "--untracked-files=no")
	statusOut, err := statusCmd.Output()
	if err != nil {
		return status
	}

	// Parallel execution of git commands
	type gitResult struct {
		ahead   string
		behind  string
		branch  string
		commit  string
	}

	resultChan := make(chan gitResult, 1)
	
	go func() {
		var result gitResult
		var wg sync.WaitGroup
		
		// Execute git commands in parallel
		wg.Add(4)
		
		go func() {
			defer wg.Done()
			if out, err := exec.CommandContext(ctx, "git", "-C", repoPath, "rev-list", "--count", "@{u}..HEAD").Output(); err == nil {
				result.ahead = strings.TrimSpace(string(out))
			}
		}()
		
		go func() {
			defer wg.Done()
			if out, err := exec.CommandContext(ctx, "git", "-C", repoPath, "rev-list", "--count", "HEAD..@{u}").Output(); err == nil {
				result.behind = strings.TrimSpace(string(out))
			}
		}()
		
		go func() {
			defer wg.Done()
			if out, err := exec.CommandContext(ctx, "git", "-C", repoPath, "rev-parse", "--abbrev-ref", "HEAD").Output(); err == nil {
				result.branch = strings.TrimSpace(string(out))
			}
		}()
		
		go func() {
			defer wg.Done()
			if out, err := exec.CommandContext(ctx, "git", "-C", repoPath, "log", "-1", "--pretty=%h %cr %an").Output(); err == nil {
				result.commit = strings.TrimSpace(string(out))
			}
		}()
		
		wg.Wait()
		resultChan <- result
	}()

	// Wait for results or timeout
	select {
	case result := <-resultChan:
		status.Branch = result.branch
		status.LastCommit = result.commit
		
		statusStr := strings.TrimSpace(string(statusOut))
		
		if statusStr == "" && result.ahead == "0" && result.behind == "0" {
			status.Symbol = "✓"
			status.Message = "Up to date"
		} else if result.ahead != "0" && result.behind != "0" {
			status.Symbol = "↕"
			status.Message = fmt.Sprintf("Diverged (%s ahead, %s behind)", result.ahead, result.behind)
		} else if result.ahead != "0" {
			status.Symbol = "↑"
			status.Message = fmt.Sprintf("%s commit(s) to push", result.ahead)
		} else if result.behind != "0" {
			status.Symbol = "↓"
			status.Message = fmt.Sprintf("%s commit(s) to pull", result.behind)
		} else {
			status.Symbol = "✗"
			status.Message = "Uncommitted changes"
		}
		
	case <-ctx.Done():
		status.Symbol = "⚠"
		status.Message = "Timeout"
	}

	return status
}

// Enhanced repo discovery with smarter filtering
func findGitReposOptimized(baseDir string, maxDepth int) []GitStatus {
	// First pass: collect all repo paths
	var repoPaths []string
	repoPathsChan := make(chan string, 100)
	
	go func() {
		defer close(repoPathsChan)
		walkReposOptimized(baseDir, baseDir, 0, maxDepth, repoPathsChan)
	}()
	
	for repoPath := range repoPathsChan {
		repoPaths = append(repoPaths, repoPath)
	}

	if len(repoPaths) == 0 {
		return []GitStatus{}
	}

	// Second pass: process with worker pool
	workerPool := NewWorkerPool()
	workerPool.Start()
	defer workerPool.Stop()

	// Submit all jobs
	go func() {
		for _, repoPath := range repoPaths {
			workerPool.Submit(RepoJob{
				RepoPath: repoPath,
				BaseDir:  baseDir,
			})
		}
	}()

	// Collect results with timeout
	var repos []GitStatus
	timeout := time.After(30 * time.Second)
	
	for len(repos) < len(repoPaths) {
		select {
		case status := <-workerPool.results:
			repos = append(repos, status)
		case <-timeout:
			// Don't wait forever for slow repos
			break
		}
	}

	// Sort by modification time (newest first)
	sort.Slice(repos, func(i, j int) bool {
		return repos[i].ModTime.After(repos[j].ModTime)
	})

	return repos
}

func walkReposOptimized(currentPath, baseDir string, currentDepth, maxDepth int, repoPaths chan<- string) {
	if maxDepth != -1 && currentDepth > maxDepth {
		return
	}

	entries, err := os.ReadDir(currentPath)
	if err != nil {
		return
	}

	// Check if current directory is a git repo
	for _, entry := range entries {
		if entry.IsDir() && entry.Name() == ".git" {
			repoPaths <- currentPath
			return // Don't recurse into .git directory
		}
	}

	// Enhanced skip list with more patterns
	skipDirs := map[string]bool{
		"node_modules":    true,
		".cache":          true,
		".venv":          true,
		"venv":           true,
		"__pycache__":    true,
		".tox":           true,
		"build":          true,
		"dist":           true,
		".npm":           true,
		".yarn":          true,
		"vendor":         true,
		"target":         true, // Rust
		".gradle":        true, // Gradle
		".idea":          true, // IntelliJ
		".vscode":        true, // VS Code
		"Pods":           true, // iOS
		"DerivedData":    true, // Xcode
	}

	// Process directories in parallel batches
	var wg sync.WaitGroup
	semaphore := make(chan struct{}, 4) // Limit concurrent directory processing

	for _, entry := range entries {
		if !entry.IsDir() || skipDirs[entry.Name()] {
			continue
		}

		wg.Add(1)
		go func(entryName string) {
			defer wg.Done()
			
			semaphore <- struct{}{} // Acquire
			defer func() { <-semaphore }() // Release
			
			path := filepath.Join(currentPath, entryName)
			walkReposOptimized(path, baseDir, currentDepth+1, maxDepth, repoPaths)
		}(entry.Name())
	}

	wg.Wait()
}