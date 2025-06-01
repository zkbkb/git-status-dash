package main

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
)

// Popular theme definitions that can be auto-imported
var popularThemes = map[string]ThemeConfig{
	"ayu-light": {
		Name: "ayu-light",
		Colors: map[string]string{
			"success": "34",   // green
			"warning": "172",  // orange
			"error":   "167",  // red
			"info":    "37",   // cyan
			"dim":     "8",    // gray
		},
		Symbols: map[string]string{
			"success":  "✓",
			"ahead":    "↑", 
			"behind":   "↓",
			"diverged": "↕",
			"dirty":    "●",
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
	"ayu-dark": {
		Name: "ayu-dark",
		Colors: map[string]string{
			"success": "150",  // green
			"warning": "179",  // yellow
			"error":   "204",  // red
			"info":    "109",  // blue
			"dim":     "59",   // gray
		},
		Symbols: map[string]string{
			"success":  "✓",
			"ahead":    "↑",
			"behind":   "↓", 
			"diverged": "↕",
			"dirty":    "●",
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
	"catppuccin-latte": {
		Name: "catppuccin-latte",
		Colors: map[string]string{
			"success": "64",   // green
			"warning": "179",  // yellow
			"error":   "167",  // red
			"info":    "117",  // blue
			"dim":     "8",    // gray
		},
		Symbols: map[string]string{
			"success":  "✓",
			"ahead":    "↑",
			"behind":   "↓",
			"diverged": "↕", 
			"dirty":    "●",
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
	"catppuccin-mocha": {
		Name: "catppuccin-mocha",
		Colors: map[string]string{
			"success": "113",  // green
			"warning": "179",  // yellow
			"error":   "204",  // red
			"info":    "117",  // blue
			"dim":     "59",   // gray
		},
		Symbols: map[string]string{
			"success":  "✓",
			"ahead":    "↑",
			"behind":   "↓",
			"diverged": "↕",
			"dirty":    "●", 
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
	"nord": {
		Name: "nord",
		Colors: map[string]string{
			"success": "108",  // nord14 green
			"warning": "179",  // nord13 yellow
			"error":   "167",  // nord11 red
			"info":    "109",  // nord10 blue
			"dim":     "59",   // nord3 gray
		},
		Symbols: map[string]string{
			"success":  "✓",
			"ahead":    "↑",
			"behind":   "↓",
			"diverged": "↕",
			"dirty":    "●",
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
	"dracula": {
		Name: "dracula",
		Colors: map[string]string{
			"success": "84",   // green
			"warning": "228",  // yellow
			"error":   "212",  // red
			"info":    "141",  // purple
			"dim":     "59",   // gray
		},
		Symbols: map[string]string{
			"success":  "✓",
			"ahead":    "↑",
			"behind":   "↓",
			"diverged": "↕",
			"dirty":    "●",
			"error":    "⚠",
		},
		Effects: EffectsConfig{
			Matrix:     false,
			Glitch:     false,
			Typewriter: false,
			Particles:  false,
			Scanlines:  false,
		},
	},
}

func detectSystemTheme() (string, error) {
	switch runtime.GOOS {
	case "darwin":
		return detectMacOSTheme()
	case "linux":
		return detectLinuxTheme()
	case "windows":
		return detectWindowsTheme()
	default:
		return "auto", nil
	}
}

func detectMacOSTheme() (string, error) {
	cmd := exec.Command("defaults", "read", "-g", "AppleInterfaceStyle")
	output, err := cmd.Output()
	
	if err != nil {
		// If command fails, system is probably in light mode
		return "light", nil
	}
	
	if strings.TrimSpace(string(output)) == "Dark" {
		return "dark", nil
	}
	
	return "light", nil
}

func detectLinuxTheme() (string, error) {
	// Try GNOME
	if output, err := exec.Command("gsettings", "get", "org.gnome.desktop.interface", "gtk-theme").Output(); err == nil {
		theme := strings.TrimSpace(strings.Trim(string(output), "'\""))
		if strings.Contains(strings.ToLower(theme), "dark") {
			return "dark", nil
		}
		return "light", nil
	}
	
	// Try KDE
	if output, err := exec.Command("kreadconfig5", "--group", "Colors:Window", "--key", "BackgroundNormal").Output(); err == nil {
		// Parse RGB values to determine if dark
		color := strings.TrimSpace(string(output))
		if strings.HasPrefix(color, "0,0,0") || strings.HasPrefix(color, "32,") {
			return "dark", nil
		}
		return "light", nil
	}
	
	// Fallback to checking environment
	if term := os.Getenv("TERM"); strings.Contains(term, "dark") {
		return "dark", nil
	}
	
	return "auto", nil
}

func detectWindowsTheme() (string, error) {
	// Check Windows registry for dark mode
	cmd := exec.Command("reg", "query", "HKEY_CURRENT_USER\\SOFTWARE\\Microsoft\\Windows\\CurrentVersion\\Themes\\Personalize", "/v", "AppsUseLightTheme")
	output, err := cmd.Output()
	
	if err != nil {
		return "auto", nil
	}
	
	if strings.Contains(string(output), "0x0") {
		return "dark", nil
	}
	
	return "light", nil
}

func getAutoTheme() (string, error) {
	systemTheme, err := detectSystemTheme()
	if err != nil {
		return "matrix", err
	}
	
	switch systemTheme {
	case "dark":
		// Prefer dark themes
		if hasTheme("ayu-dark") {
			return "ayu-dark", nil
		}
		if hasTheme("catppuccin-mocha") {
			return "catppuccin-mocha", nil
		}
		if hasTheme("dracula") {
			return "dracula", nil
		}
		return "matrix", nil
		
	case "light":
		// Prefer light themes
		if hasTheme("ayu-light") {
			return "ayu-light", nil
		}
		if hasTheme("catppuccin-latte") {
			return "catppuccin-latte", nil
		}
		if hasTheme("minimal") {
			return "minimal", nil
		}
		return "minimal", nil
		
	default:
		return "matrix", nil
	}
}

func hasTheme(name string) bool {
	// Check built-in themes
	if _, exists := defaultThemes[name]; exists {
		return true
	}
	
	// Check popular themes
	if _, exists := popularThemes[name]; exists {
		return true
	}
	
	// Check user themes
	configDir, err := getConfigDir()
	if err != nil {
		return false
	}
	
	themeFile := filepath.Join(configDir, "themes", name+".json")
	_, err = os.Stat(themeFile)
	return err == nil
}

func importPopularTheme(name string) error {
	theme, exists := popularThemes[name]
	if !exists {
		return fmt.Errorf("popular theme '%s' not found", name)
	}
	
	configDir, err := getConfigDir()
	if err != nil {
		return err
	}
	
	themesDir := filepath.Join(configDir, "themes")
	if err := os.MkdirAll(themesDir, 0755); err != nil {
		return err
	}
	
	themeFile := filepath.Join(themesDir, name+".json")
	data, err := json.MarshalIndent(theme, "", "  ")
	if err != nil {
		return err
	}
	
	return os.WriteFile(themeFile, data, 0644)
}

func listPopularThemes() {
	fmt.Println("Popular themes available for import:")
	for name := range popularThemes {
		installed := ""
		if hasTheme(name) {
			installed = " (installed)"
		}
		fmt.Printf("  • %s%s\n", name, installed)
	}
	
	systemTheme, err := detectSystemTheme()
	if err == nil && systemTheme != "auto" {
		fmt.Printf("\nDetected system theme: %s\n", systemTheme)
		
		autoTheme, err := getAutoTheme()
		if err == nil {
			fmt.Printf("Recommended theme: %s\n", autoTheme)
		}
	}
}

func importTheme(name string) error {
	if err := importPopularTheme(name); err != nil {
		return err
	}
	
	fmt.Printf("✓ Imported theme '%s'\n", name)
	return nil
}

func setAutoTheme() error {
	autoTheme, err := getAutoTheme()
	if err != nil {
		return err
	}
	
	// Import the theme if it's a popular one and not installed
	if _, exists := popularThemes[autoTheme]; exists && !hasTheme(autoTheme) {
		if err := importPopularTheme(autoTheme); err != nil {
			return err
		}
		fmt.Printf("✓ Imported theme '%s'\n", autoTheme)
	}
	
	// Set the theme
	theme, err := loadTheme(autoTheme)
	if err != nil {
		return err
	}
	
	config, err := loadConfig()
	if err != nil {
		return err
	}
	
	config.Theme = *theme
	
	if err := saveConfig(config); err != nil {
		return err
	}
	
	systemTheme, _ := detectSystemTheme()
	fmt.Printf("✓ Auto-detected %s mode, set theme to '%s'\n", systemTheme, autoTheme)
	
	return nil
}

// Theme sources - where to find themes from other apps
type ThemeSource struct {
	Name     string
	Type     string // "vscode", "alacritty", "kitty", "tmux", "vim"
	URL      string
	Parser   func([]byte) (*ThemeConfig, error)
}

var themeSources = map[string]ThemeSource{
	"ayu-vscode": {
		Name: "ayu-vscode",
		Type: "vscode",
		URL:  "https://raw.githubusercontent.com/ayu-theme/vscode-ayu/master/themes/ayu-dark.json",
		Parser: parseVSCodeTheme,
	},
	"catppuccin-alacritty": {
		Name: "catppuccin-alacritty", 
		Type: "alacritty",
		URL:  "https://raw.githubusercontent.com/catppuccin/alacritty/main/catppuccin-mocha.yml",
		Parser: parseAlacrittyTheme,
	},
	"nord-kitty": {
		Name: "nord-kitty",
		Type: "kitty",
		URL:  "https://raw.githubusercontent.com/connorholyday/nord-kitty/master/nord.conf",
		Parser: parseKittyTheme,
	},
}

func parseVSCodeTheme(data []byte) (*ThemeConfig, error) {
	var vsTheme struct {
		Colors map[string]string `json:"colors"`
		Name   string            `json:"name"`
	}
	
	if err := json.Unmarshal(data, &vsTheme); err != nil {
		return nil, err
	}
	
	// Convert VS Code colors to our format
	theme := &ThemeConfig{
		Name: strings.ToLower(strings.ReplaceAll(vsTheme.Name, " ", "-")),
		Colors: map[string]string{
			"success": convertVSCodeColor(vsTheme.Colors["gitDecoration.addedResourceForeground"]),
			"warning": convertVSCodeColor(vsTheme.Colors["gitDecoration.modifiedResourceForeground"]),
			"error":   convertVSCodeColor(vsTheme.Colors["gitDecoration.deletedResourceForeground"]),
			"info":    convertVSCodeColor(vsTheme.Colors["foreground"]),
			"dim":     convertVSCodeColor(vsTheme.Colors["descriptionForeground"]),
		},
		Symbols: map[string]string{
			"success":  "✓",
			"ahead":    "↑",
			"behind":   "↓",
			"diverged": "↕",
			"dirty":    "●",
			"error":    "!",
		},
		Effects: EffectsConfig{
			Matrix:     false,
			Glitch:     false,
			Typewriter: false,
			Particles:  false,
			Scanlines:  false,
		},
	}
	
	return theme, nil
}

func parseAlacrittyTheme(data []byte) (*ThemeConfig, error) {
	// Parse YAML-like structure manually (simple approach)
	lines := strings.Split(string(data), "\n")
	colors := make(map[string]string)
	
	for _, line := range lines {
		if strings.Contains(line, ":") && strings.Contains(line, "#") {
			parts := strings.Split(line, ":")
			if len(parts) == 2 {
				key := strings.TrimSpace(parts[0])
				value := strings.TrimSpace(parts[1])
				if strings.HasPrefix(value, "'#") || strings.HasPrefix(value, "\"#") {
					// Extract hex color
					value = strings.Trim(value, "'\"")
					colors[key] = convertHexToTerminal(value)
				}
			}
		}
	}
	
	theme := &ThemeConfig{
		Name: "catppuccin-mocha",
		Colors: map[string]string{
			"success": colors["green"],
			"warning": colors["yellow"],
			"error":   colors["red"],
			"info":    colors["blue"],
			"dim":     colors["bright_black"],
		},
		Symbols: map[string]string{
			"success":  "✓",
			"ahead":    "↑",
			"behind":   "↓",
			"diverged": "↕",
			"dirty":    "●",
			"error":    "!",
		},
		Effects: EffectsConfig{
			Matrix:     false,
			Glitch:     false,
			Typewriter: false,
			Particles:  false,
			Scanlines:  false,
		},
	}
	
	return theme, nil
}

func parseKittyTheme(data []byte) (*ThemeConfig, error) {
	lines := strings.Split(string(data), "\n")
	colors := make(map[string]string)
	
	for _, line := range lines {
		if strings.Contains(line, "#") && !strings.HasPrefix(strings.TrimSpace(line), "#") {
			parts := strings.Fields(line)
			if len(parts) >= 2 {
				key := parts[0]
				value := parts[1]
				if strings.HasPrefix(value, "#") {
					colors[key] = convertHexToTerminal(value)
				}
			}
		}
	}
	
	theme := &ThemeConfig{
		Name: "nord",
		Colors: map[string]string{
			"success": colors["color2"],  // green
			"warning": colors["color3"],  // yellow
			"error":   colors["color1"],  // red
			"info":    colors["color4"],  // blue
			"dim":     colors["color8"],  // bright black
		},
		Symbols: map[string]string{
			"success":  "✓",
			"ahead":    "↑",
			"behind":   "↓",
			"diverged": "↕",
			"dirty":    "●",
			"error":    "!",
		},
		Effects: EffectsConfig{
			Matrix:     false,
			Glitch:     false,
			Typewriter: false,
			Particles:  false,
			Scanlines:  false,
		},
	}
	
	return theme, nil
}

func convertVSCodeColor(color string) string {
	if color == "" {
		return "white"
	}
	// Simple hex to terminal color conversion
	return convertHexToTerminal(color)
}

func convertHexToTerminal(hex string) string {
	if hex == "" {
		return "white"
	}
	
	// Remove # if present
	hex = strings.TrimPrefix(hex, "#")
	
	// Simple mapping of common colors to terminal codes
	colorMap := map[string]string{
		"ff0000": "red",
		"00ff00": "green", 
		"0000ff": "blue",
		"ffff00": "yellow",
		"ff00ff": "magenta",
		"00ffff": "cyan",
		"ffffff": "white",
		"000000": "black",
		// Add more mappings as needed
	}
	
	if termColor, exists := colorMap[strings.ToLower(hex)]; exists {
		return termColor
	}
	
	// Fallback to a reasonable default
	return "white"
}

func downloadTheme(sourceName string) error {
	source, exists := themeSources[sourceName]
	if !exists {
		return fmt.Errorf("theme source '%s' not found", sourceName)
	}
	
	fmt.Printf("Downloading theme from %s...\n", source.Name)
	
	resp, err := http.Get(source.URL)
	if err != nil {
		return fmt.Errorf("failed to download theme: %v", err)
	}
	defer resp.Body.Close()
	
	if resp.StatusCode != 200 {
		return fmt.Errorf("failed to download theme: HTTP %d", resp.StatusCode)
	}
	
	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("failed to read theme data: %v", err)
	}
	
	theme, err := source.Parser(data)
	if err != nil {
		return fmt.Errorf("failed to parse theme: %v", err)
	}
	
	// Save theme
	configDir, err := getConfigDir()
	if err != nil {
		return err
	}
	
	themesDir := filepath.Join(configDir, "themes")
	if err := os.MkdirAll(themesDir, 0755); err != nil {
		return err
	}
	
	themeFile := filepath.Join(themesDir, theme.Name+".json")
	themeData, err := json.MarshalIndent(theme, "", "  ")
	if err != nil {
		return err
	}
	
	if err := os.WriteFile(themeFile, themeData, 0644); err != nil {
		return err
	}
	
	fmt.Printf("✓ Downloaded and imported theme '%s'\n", theme.Name)
	return nil
}

func listThemeSources() {
	fmt.Println("Available theme sources:")
	for name, source := range themeSources {
		fmt.Printf("  • %s (%s)\n", name, source.Type)
	}
	fmt.Println("\nUsage: git-status-dash config download <source-name>")
}

func importLocalTheme(appType, filePath string) error {
	data, err := os.ReadFile(filePath)
	if err != nil {
		return fmt.Errorf("failed to read file: %v", err)
	}
	
	var parser func([]byte) (*ThemeConfig, error)
	
	switch appType {
	case "vscode":
		parser = parseVSCodeTheme
	case "alacritty":
		parser = parseAlacrittyTheme
	case "kitty":
		parser = parseKittyTheme
	default:
		return fmt.Errorf("unsupported app type: %s", appType)
	}
	
	theme, err := parser(data)
	if err != nil {
		return fmt.Errorf("failed to parse theme: %v", err)
	}
	
	// Save theme
	configDir, err := getConfigDir()
	if err != nil {
		return err
	}
	
	themesDir := filepath.Join(configDir, "themes")
	if err := os.MkdirAll(themesDir, 0755); err != nil {
		return err
	}
	
	themeFile := filepath.Join(themesDir, theme.Name+".json")
	themeData, err := json.MarshalIndent(theme, "", "  ")
	if err != nil {
		return err
	}
	
	if err := os.WriteFile(themeFile, themeData, 0644); err != nil {
		return err
	}
	
	fmt.Printf("✓ Imported theme '%s' from %s\n", theme.Name, filePath)
	return nil
}