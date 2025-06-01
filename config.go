package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
)

type UserConfig struct {
	Theme         ThemeConfig         `json:"theme"`
	Performance   PerformanceConfig   `json:"performance"`
	Display       DisplayConfig       `json:"display"`
	Filter        FilterConfig        `json:"filter"`
	Behavior      BehaviorConfig      `json:"behavior"`
	Notifications NotificationConfig  `json:"notifications"`
	SkipDirs      []string            `json:"skip_directories"`
}

type ThemeConfig struct {
	Name         string            `json:"name"`
	Colors       map[string]string `json:"colors"`
	Symbols      map[string]string `json:"symbols"`
	Effects      EffectsConfig     `json:"effects"`
}

type EffectsConfig struct {
	Matrix      bool `json:"matrix_rain"`
	Glitch      bool `json:"glitch_text"`
	Typewriter  bool `json:"typewriter"`
	Particles   bool `json:"particles"`
	Scanlines   bool `json:"scanlines"`
}

type PerformanceConfig struct {
	Workers    int `json:"workers"`
	Timeout    int `json:"timeout_seconds"`
	MaxDepth   int `json:"max_depth"`
	BatchSize  int `json:"batch_size"`
}

type DisplayConfig struct {
	ColumnWidth    int    `json:"column_width"`
	ShowBranch     bool   `json:"show_branch"`
	ShowCommit     bool   `json:"show_last_commit"`
	ShowTimestamp  bool   `json:"show_timestamp"`
	CompactMode    bool   `json:"compact_mode"`
	TreeView       bool   `json:"tree_view"`
	TimeFormat     string `json:"time_format"`
	FlashOnChange  bool   `json:"flash_on_change"`
	ShowIcons      bool   `json:"show_icons"`
	GroupByStatus  bool   `json:"group_by_status"`
}

type FilterConfig struct {
	ShowSynced    bool     `json:"show_synced"`
	ShowAhead     bool     `json:"show_ahead"`
	ShowBehind    bool     `json:"show_behind"`
	ShowDirty     bool     `json:"show_dirty"`
	ShowError     bool     `json:"show_error"`
	HiddenStates  []string `json:"hidden_states"`
	OnlyRecent    bool     `json:"only_recent"`
	RecentDays    int      `json:"recent_days"`
}

type BehaviorConfig struct {
	AutoRefresh     bool   `json:"auto_refresh"`
	RefreshInterval int    `json:"refresh_interval_ms"`
	DefaultMode     string `json:"default_mode"` // "tui", "report", "watch"
	WatchFiles      bool   `json:"watch_files"`
	TTLMode         bool   `json:"ttl_mode"`
	TTLSeconds      int    `json:"ttl_seconds"`
	SoundOnChange   bool   `json:"sound_on_change"`
	NotifyOnChange  bool   `json:"notify_on_change"`
	ExitOnComplete  bool   `json:"exit_on_complete"`
}

type NotificationConfig struct {
	Enabled      bool     `json:"enabled"`
	OnStates     []string `json:"on_states"`
	SoundFile    string   `json:"sound_file"`
	Title        string   `json:"title"`
	Message      string   `json:"message"`
}

// Default themes
var defaultThemes = map[string]ThemeConfig{
	"matrix": {
		Name: "matrix",
		Colors: map[string]string{
			"success": "green",
			"warning": "yellow", 
			"error":   "red",
			"info":    "white",
			"dim":     "8",
		},
		Symbols: map[string]string{
			"success":  "✓",
			"ahead":    "↑",
			"behind":   "↓",
			"diverged": "↕",
			"dirty":    "✗",
			"error":    "⚠",
		},
		Effects: EffectsConfig{
			Matrix:     true,
			Glitch:     true,
			Typewriter: true,
			Particles:  true,
			Scanlines:  true,
		},
	},
	"minimal": {
		Name: "minimal",
		Colors: map[string]string{
			"success": "white",
			"warning": "white",
			"error":   "white", 
			"info":    "white",
			"dim":     "8",
		},
		Symbols: map[string]string{
			"success":  "✓",
			"ahead":    "↑",
			"behind":   "↓", 
			"diverged": "↕",
			"dirty":    "✗",
			"error":    "!",
		},
		Effects: EffectsConfig{
			Matrix:     false,
			Glitch:     false,
			Typewriter: false,
			Particles:  false,
			Scanlines:  false,
		},
	},
	"hacker": {
		Name: "hacker",
		Colors: map[string]string{
			"success": "green",
			"warning": "yellow",
			"error":   "red",
			"info":    "green",
			"dim":     "8",
		},
		Symbols: map[string]string{
			"success":  "[OK]",
			"ahead":    "[>>]",
			"behind":   "[<<]",
			"diverged": "[<>]", 
			"dirty":    "[!!]",
			"error":    "[ER]",
		},
		Effects: EffectsConfig{
			Matrix:     false,
			Glitch:     true,
			Typewriter: true,
			Particles:  false,
			Scanlines:  true,
		},
	},
	"neon": {
		Name: "neon",
		Colors: map[string]string{
			"success": "46",   // bright green
			"warning": "226",  // bright yellow
			"error":   "196",  // bright red
			"info":    "51",   // bright cyan
			"dim":     "8",
		},
		Symbols: map[string]string{
			"success":  "●",
			"ahead":    "▲",
			"behind":   "▼",
			"diverged": "◆",
			"dirty":    "⬢",
			"error":    "⚡",
		},
		Effects: EffectsConfig{
			Matrix:     false,
			Glitch:     false,
			Typewriter: false,
			Particles:  true,
			Scanlines:  false,
		},
	},
}

func getConfigDir() (string, error) {
	var configDir string
	
	if runtime.GOOS == "windows" {
		configDir = os.Getenv("APPDATA")
		if configDir == "" {
			home, err := os.UserHomeDir()
			if err != nil {
				return "", err
			}
			configDir = filepath.Join(home, "AppData", "Roaming")
		}
	} else {
		configDir = os.Getenv("XDG_CONFIG_HOME")
		if configDir == "" {
			home, err := os.UserHomeDir()
			if err != nil {
				return "", err
			}
			configDir = filepath.Join(home, ".config")
		}
	}
	
	return filepath.Join(configDir, "git-status-dash"), nil
}

func loadConfig() (*UserConfig, error) {
	configDir, err := getConfigDir()
	if err != nil {
		return getDefaultConfig(), err
	}

	configFile := filepath.Join(configDir, "config.json")
	
	// If config doesn't exist, return defaults
	if _, err := os.Stat(configFile); os.IsNotExist(err) {
		return getDefaultConfig(), nil
	}

	data, err := os.ReadFile(configFile)
	if err != nil {
		return getDefaultConfig(), err
	}

	var config UserConfig
	if err := json.Unmarshal(data, &config); err != nil {
		return getDefaultConfig(), err
	}

	return &config, nil
}

func saveConfig(config *UserConfig) error {
	configDir, err := getConfigDir()
	if err != nil {
		return err
	}

	// Create config directory if it doesn't exist
	if err := os.MkdirAll(configDir, 0755); err != nil {
		return err
	}

	configFile := filepath.Join(configDir, "config.json")
	
	data, err := json.MarshalIndent(config, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(configFile, data, 0644)
}

func getDefaultConfig() *UserConfig {
	return &UserConfig{
		Theme: defaultThemes["matrix"],
		Performance: PerformanceConfig{
			Workers:   runtime.NumCPU() * 2,
			Timeout:   3,
			MaxDepth:  -1, // unlimited
			BatchSize: 10,
		},
		Display: DisplayConfig{
			ColumnWidth:    30,
			ShowBranch:     true,
			ShowCommit:     true,
			ShowTimestamp:  false,
			CompactMode:    false,
			TreeView:       false,
			TimeFormat:     "15:04:05",
			FlashOnChange:  true,
			ShowIcons:      true,
			GroupByStatus:  false,
		},
		Filter: FilterConfig{
			ShowSynced:   false,
			ShowAhead:    true,
			ShowBehind:   true,
			ShowDirty:    true,
			ShowError:    true,
			HiddenStates: []string{},
			OnlyRecent:   false,
			RecentDays:   7,
		},
		Behavior: BehaviorConfig{
			AutoRefresh:     true,
			RefreshInterval: 2000, // 2 seconds
			DefaultMode:     "tui",
			WatchFiles:      true,
			TTLMode:         false,
			TTLSeconds:      30,
			SoundOnChange:   false,
			NotifyOnChange:  false,
			ExitOnComplete:  false,
		},
		Notifications: NotificationConfig{
			Enabled:   false,
			OnStates:  []string{"dirty", "ahead", "behind", "error"},
			SoundFile: "",
			Title:     "Git Status Update",
			Message:   "Repository status changed",
		},
		SkipDirs: []string{
			"node_modules", ".cache", ".venv", "venv", "__pycache__",
			".tox", "build", "dist", ".npm", ".yarn", "vendor",
			"target", ".gradle", ".idea", ".vscode", "Pods", "DerivedData",
			".git", ".svn", ".hg", ".bzr", // VCS dirs
			"coverage", ".nyc_output", ".pytest_cache", // Test dirs
			"logs", "tmp", "temp", ".tmp", // Temp dirs
		},
	}
}

func initConfig() error {
	config := getDefaultConfig()
	
	configDir, err := getConfigDir()
	if err != nil {
		return err
	}

	fmt.Printf("Initializing config in: %s\n", configDir)
	
	if err := saveConfig(config); err != nil {
		return err
	}

	// Create themes directory
	themesDir := filepath.Join(configDir, "themes")
	if err := os.MkdirAll(themesDir, 0755); err != nil {
		return err
	}

	// Save default themes
	for name, theme := range defaultThemes {
		themeFile := filepath.Join(themesDir, name+".json")
		data, err := json.MarshalIndent(theme, "", "  ")
		if err != nil {
			continue
		}
		os.WriteFile(themeFile, data, 0644)
	}

	fmt.Println("✓ Config initialized with default themes:")
	fmt.Println("  - matrix (default)")
	fmt.Println("  - minimal")  
	fmt.Println("  - hacker")
	fmt.Println("  - neon")
	fmt.Printf("\nEdit config: %s/config.json\n", configDir)
	fmt.Printf("Edit themes: %s/themes/\n", configDir)

	return nil
}

func loadTheme(name string) (*ThemeConfig, error) {
	// Check built-in themes first
	if theme, exists := defaultThemes[name]; exists {
		return &theme, nil
	}

	// Check user themes
	configDir, err := getConfigDir()
	if err != nil {
		return nil, err
	}

	themeFile := filepath.Join(configDir, "themes", name+".json")
	data, err := os.ReadFile(themeFile)
	if err != nil {
		return nil, fmt.Errorf("theme '%s' not found", name)
	}

	var theme ThemeConfig
	if err := json.Unmarshal(data, &theme); err != nil {
		return nil, err
	}

	return &theme, nil
}

func listThemes() ([]string, error) {
	var themes []string
	
	// Add built-in themes
	for name := range defaultThemes {
		themes = append(themes, name+" (built-in)")
	}

	// Add user themes
	configDir, err := getConfigDir()
	if err != nil {
		return themes, nil
	}

	themesDir := filepath.Join(configDir, "themes")
	entries, err := os.ReadDir(themesDir)
	if err != nil {
		return themes, nil
	}

	for _, entry := range entries {
		if !entry.IsDir() && filepath.Ext(entry.Name()) == ".json" {
			name := strings.TrimSuffix(entry.Name(), ".json")
			// Skip if it's a built-in theme
			if _, exists := defaultThemes[name]; !exists {
				themes = append(themes, name+" (custom)")
			}
		}
	}

	return themes, nil
}

func showConfig() {
	config, err := loadConfig()
	if err != nil {
		fmt.Printf("Error loading config: %v\n", err)
		return
	}

	data, err := json.MarshalIndent(config, "", "  ")
	if err != nil {
		fmt.Printf("Error formatting config: %v\n", err)
		return
	}

	configDir, _ := getConfigDir()
	fmt.Printf("Config file: %s/config.json\n\n", configDir)
	fmt.Println(string(data))
}

func listAllThemes() {
	themes, err := listThemes()
	if err != nil {
		fmt.Printf("Error listing themes: %v\n", err)
		return
	}

	fmt.Println("Available themes:")
	for _, theme := range themes {
		fmt.Printf("  • %s\n", theme)
	}

	config, err := loadConfig()
	if err == nil {
		fmt.Printf("\nCurrent theme: %s\n", config.Theme.Name)
	}
}

func setTheme(themeName string) {
	// Load the theme to validate it exists
	theme, err := loadTheme(themeName)
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		return
	}

	// Load current config
	config, err := loadConfig()
	if err != nil {
		fmt.Printf("Error loading config: %v\n", err)
		return
	}

	// Update theme
	config.Theme = *theme

	// Save config
	if err := saveConfig(config); err != nil {
		fmt.Printf("Error saving config: %v\n", err)
		return
	}

	fmt.Printf("✓ Theme set to '%s'\n", themeName)
}

func setConfigValue(key, value string) {
	config, err := loadConfig()
	if err != nil {
		fmt.Printf("Error loading config: %v\n", err)
		return
	}

	// Parse key path and set value
	switch {
	case strings.HasPrefix(key, "display."):
		setDisplayConfig(config, strings.TrimPrefix(key, "display."), value)
	case strings.HasPrefix(key, "filter."):
		setFilterConfig(config, strings.TrimPrefix(key, "filter."), value)
	case strings.HasPrefix(key, "behavior."):
		setBehaviorConfig(config, strings.TrimPrefix(key, "behavior."), value)
	case strings.HasPrefix(key, "performance."):
		setPerformanceConfig(config, strings.TrimPrefix(key, "performance."), value)
	case strings.HasPrefix(key, "notifications."):
		setNotificationConfig(config, strings.TrimPrefix(key, "notifications."), value)
	default:
		fmt.Printf("Unknown config key: %s\n", key)
		fmt.Println("Available keys:")
		fmt.Println("  display.tree_view, display.flash_on_change, display.show_timestamp")
		fmt.Println("  filter.show_synced, filter.only_recent, filter.recent_days")
		fmt.Println("  behavior.refresh_interval, behavior.ttl_mode, behavior.ttl_seconds")
		fmt.Println("  performance.workers, performance.timeout")
		return
	}

	if err := saveConfig(config); err != nil {
		fmt.Printf("Error saving config: %v\n", err)
		return
	}

	fmt.Printf("✓ Set %s = %s\n", key, value)
}

func setDisplayConfig(config *UserConfig, key, value string) {
	switch key {
	case "tree_view":
		config.Display.TreeView = value == "true"
	case "flash_on_change":
		config.Display.FlashOnChange = value == "true"
	case "show_timestamp":
		config.Display.ShowTimestamp = value == "true"
	case "show_branch":
		config.Display.ShowBranch = value == "true"
	case "show_commit":
		config.Display.ShowCommit = value == "true"
	case "compact_mode":
		config.Display.CompactMode = value == "true"
	case "show_icons":
		config.Display.ShowIcons = value == "true"
	case "group_by_status":
		config.Display.GroupByStatus = value == "true"
	}
}

func setFilterConfig(config *UserConfig, key, value string) {
	switch key {
	case "show_synced":
		config.Filter.ShowSynced = value == "true"
	case "show_ahead":
		config.Filter.ShowAhead = value == "true"
	case "show_behind":
		config.Filter.ShowBehind = value == "true"
	case "show_dirty":
		config.Filter.ShowDirty = value == "true"
	case "show_error":
		config.Filter.ShowError = value == "true"
	case "only_recent":
		config.Filter.OnlyRecent = value == "true"
	case "recent_days":
		if days, err := strconv.Atoi(value); err == nil {
			config.Filter.RecentDays = days
		}
	}
}

func setBehaviorConfig(config *UserConfig, key, value string) {
	switch key {
	case "auto_refresh":
		config.Behavior.AutoRefresh = value == "true"
	case "refresh_interval":
		if interval, err := strconv.Atoi(value); err == nil {
			config.Behavior.RefreshInterval = interval
		}
	case "watch_files":
		config.Behavior.WatchFiles = value == "true"
	case "ttl_mode":
		config.Behavior.TTLMode = value == "true"
	case "ttl_seconds":
		if seconds, err := strconv.Atoi(value); err == nil {
			config.Behavior.TTLSeconds = seconds
		}
	case "sound_on_change":
		config.Behavior.SoundOnChange = value == "true"
	case "notify_on_change":
		config.Behavior.NotifyOnChange = value == "true"
	case "exit_on_complete":
		config.Behavior.ExitOnComplete = value == "true"
	}
}

func setPerformanceConfig(config *UserConfig, key, value string) {
	switch key {
	case "workers":
		if workers, err := strconv.Atoi(value); err == nil {
			config.Performance.Workers = workers
		}
	case "timeout":
		if timeout, err := strconv.Atoi(value); err == nil {
			config.Performance.Timeout = timeout
		}
	case "max_depth":
		if depth, err := strconv.Atoi(value); err == nil {
			config.Performance.MaxDepth = depth
		}
	case "batch_size":
		if size, err := strconv.Atoi(value); err == nil {
			config.Performance.BatchSize = size
		}
	}
}

func setNotificationConfig(config *UserConfig, key, value string) {
	switch key {
	case "enabled":
		config.Notifications.Enabled = value == "true"
	case "sound_file":
		config.Notifications.SoundFile = value
	case "title":
		config.Notifications.Title = value
	case "message":
		config.Notifications.Message = value
	}
}