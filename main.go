package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/fsnotify/fsnotify"
	"github.com/spf13/cobra"
)

// Version is set at build time via ldflags
var Version = "dev"

type GitStatus struct {
	Symbol       string
	Message      string
	Branch       string
	LastCommit   string
	RepoPath     string
	RelativePath string
	ModTime      time.Time
}

type Config struct {
	Directory string
	Report    bool
	All       bool
	TUI       bool
	Depth     int
	Theme     string
}

type model struct {
	repos        []GitStatus
	cursor       int
	loading      bool
	baseDir      string
	showDetail   bool
	config       Config
	cache        map[string]GitStatus
	animations   *AnimationState
	watcher      *fsnotify.Watcher
	lastUpdate   time.Time
	updateCount  int
	hackerFX     *HackerEffects
	matrixMode   bool
	termWidth    int
	termHeight   int
}

var config Config

func main() {
	rootCmd := &cobra.Command{
		Use:   "git-status-dash [directory]",
		Short: "Monitor git repository status in real-time",
		Long: `Git Status Dashboard recursively scans directories for git repositories
and displays their status with beautiful TUI or report output.`,
		Args:    cobra.MaximumNArgs(1),
		Version: Version,
		Run:     run,
	}

	// Config commands
	configCmd := &cobra.Command{
		Use:   "config",
		Short: "Manage configuration",
		Long:  "Manage themes, settings, and preferences for git-status-dash",
	}

	initCmd := &cobra.Command{
		Use:   "init",
		Short: "Initialize config with defaults",
		Run: func(cmd *cobra.Command, args []string) {
			if err := initConfig(); err != nil {
				log.Fatal(err)
			}
		},
	}

	showCmd := &cobra.Command{
		Use:   "show",
		Short: "Show current configuration",
		Run: func(cmd *cobra.Command, args []string) {
			showConfig()
		},
	}

	themesCmd := &cobra.Command{
		Use:   "themes",
		Short: "List available themes",
		Run: func(cmd *cobra.Command, args []string) {
			listAllThemes()
		},
	}

	setThemeCmd := &cobra.Command{
		Use:   "theme <name>",
		Short: "Set the active theme",
		Args:  cobra.ExactArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			setTheme(args[0])
		},
	}

	autoCmd := &cobra.Command{
		Use:   "auto",
		Short: "Auto-detect and set theme based on system",
		Run: func(cmd *cobra.Command, args []string) {
			if err := setAutoTheme(); err != nil {
				log.Fatal(err)
			}
		},
	}

	downloadCmd := &cobra.Command{
		Use:   "download <source>",
		Short: "Download theme from external source",
		Args:  cobra.ExactArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			if err := downloadTheme(args[0]); err != nil {
				log.Fatal(err)
			}
		},
	}

	sourcesCmd := &cobra.Command{
		Use:   "sources",
		Short: "List available theme sources",
		Run: func(cmd *cobra.Command, args []string) {
			listThemeSources()
		},
	}

	importCmd := &cobra.Command{
		Use:   "import <app-type> <file-path>",
		Short: "Import theme from local file (vscode/alacritty/kitty)",
		Args:  cobra.ExactArgs(2),
		Run: func(cmd *cobra.Command, args []string) {
			if err := importLocalTheme(args[0], args[1]); err != nil {
				log.Fatal(err)
			}
		},
	}

	setCmd := &cobra.Command{
		Use:   "set <key> <value>",
		Short: "Set a configuration value",
		Args:  cobra.ExactArgs(2),
		Run: func(cmd *cobra.Command, args []string) {
			setConfigValue(args[0], args[1])
		},
	}

	configCmd.AddCommand(initCmd, showCmd, themesCmd, setThemeCmd, autoCmd, downloadCmd, sourcesCmd, importCmd, setCmd)
	rootCmd.AddCommand(configCmd)

	rootCmd.Flags().BoolVarP(&config.Report, "report", "r", false, "Generate a brief report")
	rootCmd.Flags().StringVarP(&config.Directory, "directory", "d", "", "Specify the directory to scan")
	rootCmd.Flags().BoolVarP(&config.All, "all", "a", false, "Show all repositories, including synced ones")
	rootCmd.Flags().BoolVarP(&config.TUI, "tui", "t", false, "Interactive TUI interface")
	rootCmd.Flags().IntVar(&config.Depth, "depth", -1, "Limit recursion depth when scanning repos")
	rootCmd.Flags().StringVar(&config.Theme, "theme", "", "Override theme for this run")

	rootCmd.SetHelpTemplate(`Git Status Dashboard

Usage:
  {{.UseLine}}

Flags:
{{.LocalFlags.FlagUsages}}
Examples:
  git-status-dash --report
  git-status-dash -d ~/projects -a

Status Information:
  ✓ Synced and up to date
  ↑ Ahead of remote (local commits to push)
  ↓ Behind remote (commits to pull)
  ↕ Diverged (need to merge or rebase)
  ✗ Uncommitted changes
  ⚠ Error accessing repository
`)

	if err := rootCmd.Execute(); err != nil {
		log.Fatal(err)
	}
}

func run(cmd *cobra.Command, args []string) {
	if config.Directory == "" {
		if len(args) > 0 {
			config.Directory = args[0]
		} else {
			var err error
			config.Directory, err = os.Getwd()
			if err != nil {
				log.Fatal(err)
			}
		}
	}

	if config.Depth == -1 {
		config.Depth = -1 // unlimited
	}

	// Default to TUI unless --report is specified
	if !config.Report {
		config.TUI = true
	}

	if config.TUI {
		runTUI()
	} else {
		runReport()
	}
}

func runTUI() {
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		log.Printf("Warning: Could not create file watcher: %v", err)
	}

	m := model{
		repos:       []GitStatus{},
		loading:     true,
		baseDir:     config.Directory,
		showDetail:  false,
		config:      config,
		cache:       make(map[string]GitStatus),
		animations:  NewAnimationState(),
		watcher:     watcher,
		lastUpdate:  time.Now(),
		updateCount: 0,
		hackerFX:    NewHackerEffects(80, 24), // Default terminal size
		matrixMode:  false,
		termWidth:   80,
		termHeight:  24,
	}

	p := tea.NewProgram(m, tea.WithAltScreen())
	if _, err := p.Run(); err != nil {
		log.Fatal(err)
	}

	if watcher != nil {
		watcher.Close()
	}
}

func runReport() {
	repos := findGitReposOptimized(config.Directory, config.Depth)
	
	fmt.Printf("Found %d repositories, loading......\n", len(repos))

	reposToShow := repos
	if !config.All {
		var unsynced []GitStatus
		for _, repo := range repos {
			if repo.Symbol != "✓" {
				unsynced = append(unsynced, repo)
			}
		}
		reposToShow = unsynced
	}

	for _, repo := range reposToShow {
		repoName := repo.RelativePath
		if repoName == "" {
			repoName = "."
		}
		line := fmt.Sprintf("%s %-30s %s", repo.Symbol, repoName, repo.Message)
		
		switch repo.Symbol {
		case "✓":
			fmt.Printf("\033[32m%s\033[0m\n", line)
		case "✗", "⚠":
			fmt.Printf("\033[31m%s\033[0m\n", line)
		case "↑", "↓", "↕":
			fmt.Printf("\033[33m%s\033[0m\n", line)
		default:
			fmt.Println(line)
		}
	}
}

func (m model) Init() tea.Cmd {
	commands := []tea.Cmd{
		scanRepos(m.baseDir, m.config.Depth, m.cache),
		tea.Tick(time.Millisecond*100, func(t time.Time) tea.Msg {
			return tickMsg(t)
		}),
		tea.Tick(time.Millisecond*16, func(t time.Time) tea.Msg {
			return animationTickMsg(t)
		}),
	}

	// Set up file watching
	if m.watcher != nil {
		commands = append(commands, m.watchForChanges())
		go m.setupWatchers()
	}

	return tea.Batch(commands...)
}

func (m model) watchForChanges() tea.Cmd {
	return func() tea.Msg {
		if m.watcher == nil {
			return nil
		}

		select {
		case event, ok := <-m.watcher.Events:
			if !ok {
				return nil
			}
			// Trigger rescan on git-related file changes
			if strings.Contains(event.Name, ".git") || 
			   strings.HasSuffix(event.Name, ".go") ||
			   strings.HasSuffix(event.Name, ".js") ||
			   strings.HasSuffix(event.Name, ".py") {
				return fileChangeMsg(event.Name)
			}
		case err, ok := <-m.watcher.Errors:
			if !ok {
				return nil
			}
			log.Printf("Watcher error: %v", err)
		}
		return nil
	}
}

func (m model) setupWatchers() {
	if m.watcher == nil {
		return
	}

	// Watch all git repositories
	for _, repo := range m.repos {
		gitDir := filepath.Join(repo.RepoPath, ".git")
		m.watcher.Add(gitDir)
		m.watcher.Add(repo.RepoPath) // Watch the repo root too
	}
}

type reposFoundMsg []GitStatus
type tickMsg time.Time
type fileChangeMsg string
type animationTickMsg time.Time

func scanRepos(baseDir string, depth int, cache map[string]GitStatus) tea.Cmd {
	return tea.Cmd(func() tea.Msg {
		repos := findGitRepos(baseDir, depth, cache)
		return reposFoundMsg(repos)
	})
}

func findGitRepos(baseDir string, maxDepth int, cache map[string]GitStatus) []GitStatus {
	var repos []GitStatus
	var mu sync.Mutex
	var wg sync.WaitGroup

	walkWithDepth(baseDir, baseDir, 0, maxDepth, &repos, &mu, &wg, cache)
	wg.Wait()

	// Sort by modification time (newest first) like original
	sort.Slice(repos, func(i, j int) bool {
		return repos[i].ModTime.After(repos[j].ModTime)
	})

	return repos
}

func walkWithDepth(currentPath, baseDir string, currentDepth, maxDepth int, repos *[]GitStatus, mu *sync.Mutex, wg *sync.WaitGroup, cache map[string]GitStatus) {
	if maxDepth != -1 && currentDepth > maxDepth {
		return
	}

	entries, err := os.ReadDir(currentPath)
	if err != nil {
		return
	}

	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}

		path := filepath.Join(currentPath, entry.Name())

		if entry.Name() == ".git" {
			repoPath := currentPath
			wg.Add(1)
			go func(rp string) {
				defer wg.Done()
				status := getGitStatus(rp, baseDir, cache)
				mu.Lock()
				*repos = append(*repos, status)
				mu.Unlock()
			}(repoPath)
			return // Don't recurse into .git directory
		}

		// Skip heavy directories
		if entry.Name() == "node_modules" || entry.Name() == ".cache" || entry.Name() == ".venv" {
			continue
		}

		walkWithDepth(path, baseDir, currentDepth+1, maxDepth, repos, mu, wg, cache)
	}
}

func getGitStatus(repoPath, baseDir string, cache map[string]GitStatus) GitStatus {
	// Remove cache for now to fix race condition
	// TODO: Add proper mutex if we want caching

	relPath, _ := filepath.Rel(baseDir, repoPath)
	
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Get modification time
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

	statusCmd := exec.CommandContext(ctx, "git", "-C", repoPath, "status", "--porcelain")
	statusOut, err := statusCmd.Output()
	if err != nil {
		return status
	}

	aheadCmd := exec.CommandContext(ctx, "git", "-C", repoPath, "rev-list", "--count", "@{u}..HEAD")
	aheadOut, _ := aheadCmd.Output()
	ahead := strings.TrimSpace(string(aheadOut))

	behindCmd := exec.CommandContext(ctx, "git", "-C", repoPath, "rev-list", "--count", "HEAD..@{u}")
	behindOut, _ := behindCmd.Output()
	behind := strings.TrimSpace(string(behindOut))

	branchCmd := exec.CommandContext(ctx, "git", "-C", repoPath, "rev-parse", "--abbrev-ref", "HEAD")
	branchOut, _ := branchCmd.Output()
	status.Branch = strings.TrimSpace(string(branchOut))

	commitCmd := exec.CommandContext(ctx, "git", "-C", repoPath, "log", "-1", "--pretty=%h %cr %an")
	commitOut, _ := commitCmd.Output()
	status.LastCommit = strings.TrimSpace(string(commitOut))

	statusStr := strings.TrimSpace(string(statusOut))
	
	if statusStr == "" && ahead == "0" && behind == "0" {
		status.Symbol = "✓"
		status.Message = "Up to date"
	} else if ahead != "0" && behind != "0" {
		status.Symbol = "↕"
		status.Message = fmt.Sprintf("Diverged (%s ahead, %s behind)", ahead, behind)
	} else if ahead != "0" {
		status.Symbol = "↑"
		status.Message = fmt.Sprintf("%s commit(s) to push", ahead)
	} else if behind != "0" {
		status.Symbol = "↓"
		status.Message = fmt.Sprintf("%s commit(s) to pull", behind)
	} else {
		status.Symbol = "✗"
		status.Message = "Uncommitted changes"
	}

	return status
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c", "q":
			return m, tea.Quit
		case "up", "k":
			oldCursor := m.cursor
			if m.cursor > 0 {
				m.cursor--
				// Animate cursor movement
				m.animations.AnimateToPosition(float64(m.cursor))
				if oldCursor != m.cursor {
					m.animations.AddStatusChangeParticles(0, m.cursor, "nav")
				}
			}
		case "down", "j":
			oldCursor := m.cursor
			if m.cursor < len(m.repos)-1 {
				m.cursor++
				// Animate cursor movement
				m.animations.AnimateToPosition(float64(m.cursor))
				if oldCursor != m.cursor {
					m.animations.AddStatusChangeParticles(0, m.cursor, "nav")
				}
			}
		case "enter", " ":
			m.showDetail = !m.showDetail
			if m.showDetail && len(m.repos) > 0 {
				m.animations.AddStatusChangeParticles(15, 5, m.repos[m.cursor].Symbol)
			}
		case "esc":
			m.showDetail = false
		case "m":
			// Toggle matrix mode
			m.matrixMode = !m.matrixMode
		case "r":
			// Force refresh
			m.loading = true
			m.updateCount++
			m.lastUpdate = time.Now()
			// Clear cache to force fresh data
			m.cache = make(map[string]GitStatus)
			return m, scanRepos(m.baseDir, m.config.Depth, m.cache)
		}

	case reposFoundMsg:
		repos := []GitStatus(msg)
		
		// Check for status changes and trigger particles
		for i, newRepo := range repos {
			for j, oldRepo := range m.repos {
				if newRepo.RepoPath == oldRepo.RepoPath && newRepo.Symbol != oldRepo.Symbol {
					m.animations.AddStatusChangeParticles(30, j, newRepo.Symbol)
					break
				}
			}
			_ = i
		}
		
		if !m.config.All {
			var unsynced []GitStatus
			for _, repo := range repos {
				if repo.Symbol != "✓" {
					unsynced = append(unsynced, repo)
				}
			}
			m.repos = unsynced
		} else {
			m.repos = repos
		}
		m.loading = false
		m.lastUpdate = time.Now()
		m.updateCount++

		// Set up watchers for new repos
		if m.watcher != nil {
			go m.setupWatchers()
		}

	case fileChangeMsg:
		// File changed, trigger refresh
		if time.Since(m.lastUpdate) > 2*time.Second { // Debounce
			m.loading = true
			m.lastUpdate = time.Now()
			return m, tea.Batch(
				scanRepos(m.baseDir, m.config.Depth, m.cache),
				m.watchForChanges(),
			)
		}
		return m, m.watchForChanges()

	case animationTickMsg:
		m.animations.Update()
		m.hackerFX.Update(m.termWidth, m.termHeight)
		return m, tea.Tick(time.Millisecond*16, func(t time.Time) tea.Msg {
			return animationTickMsg(t)
		})

	case tickMsg:
		return m, tea.Tick(time.Millisecond*100, func(t time.Time) tea.Msg {
			return tickMsg(t)
		})
	}

	return m, nil
}

func (m model) View() string {
	var s strings.Builder

	titleStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("62")).
		Padding(1, 2)

	s.WriteString(titleStyle.Render("🚀 Git Status Dashboard"))
	s.WriteString("\n\n")

	if m.loading {
		spinner := []string{"⠋", "⠙", "⠹", "⠸", "⠼", "⠴", "⠦", "⠧", "⠇", "⠏"}
		loadingStyle := lipgloss.NewStyle().
			Foreground(lipgloss.Color("205")).
			Bold(true)
		s.WriteString(loadingStyle.Render(fmt.Sprintf("%s Scanning repositories...", spinner[int(time.Now().UnixNano()/100000000)%len(spinner)])))
		return s.String()
	}

	if len(m.repos) == 0 {
		s.WriteString("No git repositories found.")
		return s.String()
	}

	// Main repo list
	for i, repo := range m.repos {
		cursor := " "
		if m.cursor == i {
			cursor = ">"
		}

		symbolStyle := lipgloss.NewStyle()
		repoStyle := lipgloss.NewStyle()
		messageStyle := lipgloss.NewStyle()

		switch repo.Symbol {
		case "✓":
			symbolStyle = symbolStyle.Foreground(lipgloss.Color("46"))
			repoStyle = repoStyle.Foreground(lipgloss.Color("46"))
			messageStyle = messageStyle.Foreground(lipgloss.Color("46"))
		case "✗", "⚠":
			symbolStyle = symbolStyle.Foreground(lipgloss.Color("196"))
			repoStyle = repoStyle.Foreground(lipgloss.Color("196"))
			messageStyle = messageStyle.Foreground(lipgloss.Color("196"))
		case "↑", "↓", "↕":
			symbolStyle = symbolStyle.Foreground(lipgloss.Color("220"))
			repoStyle = repoStyle.Foreground(lipgloss.Color("220"))
			messageStyle = messageStyle.Foreground(lipgloss.Color("220"))
		}

		if m.cursor == i {
			symbolStyle = symbolStyle.Background(lipgloss.Color("238"))
			repoStyle = repoStyle.Background(lipgloss.Color("238"))
			messageStyle = messageStyle.Background(lipgloss.Color("238"))
		}

		repoName := repo.RelativePath
		if repoName == "" {
			repoName = "."
		}

		line := fmt.Sprintf("%s %s %-30s %s",
			cursor,
			symbolStyle.Render(repo.Symbol),
			repoStyle.Render(repoName),
			messageStyle.Render(repo.Message),
		)

		s.WriteString(line + "\n")
	}

	// Detail popup
	if m.showDetail && len(m.repos) > 0 && m.cursor < len(m.repos) {
		repo := m.repos[m.cursor]
		detailStyle := lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("62")).
			Padding(1, 2).
			Margin(1, 0).
			Background(lipgloss.Color("0"))

		detailContent := fmt.Sprintf(
			"Repository Details\n\n"+
				"Path: %s\n"+
				"Branch: %s\n"+
				"Status: %s\n"+
				"Last Commit: %s",
			repo.RepoPath,
			repo.Branch,
			repo.Message,
			repo.LastCommit,
		)

		s.WriteString("\n")
		s.WriteString(detailStyle.Render(detailContent))
	}

	s.WriteString("\n")
	helpStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("241")).
		Italic(true)

	helpText := "↑/↓: navigate • enter: details • q: quit"
	if m.showDetail {
		helpText = "↑/↓: navigate • esc: close details • q: quit"
	}
	s.WriteString(helpStyle.Render(helpText))

	return s.String()
}