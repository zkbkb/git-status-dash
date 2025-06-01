package main

import (
	"math"
	"math/rand"
	"strings"
	"time"

	"github.com/briandowns/spinner"
	"github.com/charmbracelet/lipgloss"
)

type HackerEffects struct {
	MatrixRain     []MatrixColumn
	TypeWriter     TypeWriterEffect
	GlitchEffect   GlitchState
	ASCIISpinners  map[string]*spinner.Spinner
	LastUpdate     time.Time
}

type MatrixColumn struct {
	X      int
	Chars  []rune
	Head   int
	Speed  float64
	Length int
}

type TypeWriterEffect struct {
	Text       string
	Position   int
	Speed      time.Duration
	LastUpdate time.Time
	Complete   bool
}

type GlitchState struct {
	Text           string
	GlitchIntensity float64
	LastGlitch     time.Time
}

func NewHackerEffects(width, height int) *HackerEffects {
	h := &HackerEffects{
		MatrixRain:    make([]MatrixColumn, width/2), // Sparse columns
		ASCIISpinners: make(map[string]*spinner.Spinner),
		LastUpdate:    time.Now(),
	}

	// Initialize matrix columns
	for i := range h.MatrixRain {
		h.MatrixRain[i] = MatrixColumn{
			X:      i * 2,
			Chars:  make([]rune, height),
			Head:   rand.Intn(height),
			Speed:  0.5 + rand.Float64()*2, // 0.5-2.5 chars per second
			Length: 5 + rand.Intn(15),      // 5-20 chars long
		}
		h.generateMatrixChars(&h.MatrixRain[i], height)
	}

	// Initialize ASCII spinners (monochrome only)
	h.initSpinners()

	return h
}

func (h *HackerEffects) initSpinners() {
	// Hacker-friendly monochrome spinners
	spinnerSets := map[string][]string{
		"dots":    {"⠋", "⠙", "⠹", "⠸", "⠼", "⠴", "⠦", "⠧", "⠇", "⠏"},
		"line":    {"|", "/", "-", "\\"},
		"arrows":  {"←", "↖", "↑", "↗", "→", "↘", "↓", "↙"},
		"blocks":  {"▁", "▃", "▄", "▅", "▆", "▇", "█", "▇", "▆", "▅", "▄", "▃"},
		"bounce":  {"⠁", "⠂", "⠄", "⡀", "⢀", "⠠", "⠐", "⠈"},
		"pulse":   {"●", "◐", "◑", "◒", "◓", "◔", "◕", "○"},
		"binary":  {"0", "1"},
		"matrix":  {"日", "a", "Z", "3", "ﾊ", "ﾐ", "ﾋ", "ｰ", "ｳ", "ｼ", "ﾅ", "ﾓ", "ﾆ", "ｻ", "ﾜ", "ﾂ", "ｵ", "ﾘ", "ｱ", "ﾎ", "ﾃ", "ﾏ", "ｹ", "ﾒ", "ｴ", "ｶ", "ｷ", "ﾑ", "ﾕ", "ﾗ", "ｾ", "ﾈ", "ｽ", "ﾀ", "ﾇ", "ﾍ"},
		"scanner": {"[    ]", "[=   ]", "[==  ]", "[=== ]", "[====]", "[ ===]", "[  ==]", "[   =]"},
		"wave":    {"▁", "▂", "▃", "▄", "▅", "▆", "▇", "█", "▇", "▆", "▅", "▄", "▃", "▂"},
	}

	for name, charset := range spinnerSets {
		s := spinner.New(charset, 100*time.Millisecond)
		s.Color("white") // Monochrome
		h.ASCIISpinners[name] = s
	}
}

func (h *HackerEffects) generateMatrixChars(col *MatrixColumn, height int) {
	// Classic Matrix characters: katakana, latin, numbers
	matrixChars := []rune{
		'0', '1', '2', '3', '4', '5', '6', '7', '8', '9',
		'A', 'B', 'C', 'D', 'E', 'F', 'G', 'H', 'I', 'J', 'K', 'L', 'M',
		'N', 'O', 'P', 'Q', 'R', 'S', 'T', 'U', 'V', 'W', 'X', 'Y', 'Z',
		'a', 'b', 'c', 'd', 'e', 'f', 'g', 'h', 'i', 'j', 'k', 'l', 'm',
		'n', 'o', 'p', 'q', 'r', 's', 't', 'u', 'v', 'w', 'x', 'y', 'z',
		'!', '@', '#', '$', '%', '^', '&', '*', '(', ')', '-', '+', '=',
		'[', ']', '{', '}', '|', '\\', ':', ';', '"', '\'', '<', '>', ',', '.', '?', '/',
	}

	for i := range col.Chars {
		col.Chars[i] = matrixChars[rand.Intn(len(matrixChars))]
	}
}

func (h *HackerEffects) Update(width, height int) {
	now := time.Now()
	dt := now.Sub(h.LastUpdate).Seconds()
	h.LastUpdate = now

	// Update matrix rain
	for i := range h.MatrixRain {
		col := &h.MatrixRain[i]
		col.Head += int(col.Speed * dt * 10) // Scale for visibility

		if col.Head > height+col.Length {
			col.Head = -col.Length
			col.Speed = 0.5 + rand.Float64()*2
			h.generateMatrixChars(col, height)
		}
	}

	// Update typewriter
	if !h.TypeWriter.Complete && now.Sub(h.TypeWriter.LastUpdate) > h.TypeWriter.Speed {
		if h.TypeWriter.Position < len(h.TypeWriter.Text) {
			h.TypeWriter.Position++
			h.TypeWriter.LastUpdate = now
		} else {
			h.TypeWriter.Complete = true
		}
	}

	// Update glitch effect
	if now.Sub(h.GlitchEffect.LastGlitch) > 100*time.Millisecond {
		h.GlitchEffect.GlitchIntensity = rand.Float64() * 0.3 // Max 30% glitch
		h.GlitchEffect.LastGlitch = now
	}
}

func (h *HackerEffects) RenderMatrixRain(width, height int) string {
	if len(h.MatrixRain) == 0 {
		return ""
	}

	// Create grid
	grid := make([][]rune, height)
	for i := range grid {
		grid[i] = make([]rune, width)
		for j := range grid[i] {
			grid[i][j] = ' '
		}
	}

	// Render columns
	for _, col := range h.MatrixRain {
		if col.X >= width {
			continue
		}

		for i := 0; i < col.Length; i++ {
			y := col.Head - i
			if y >= 0 && y < height {
				charIndex := (y + i) % len(col.Chars)
				
				// Fade effect: head is brighter
				intensity := float64(col.Length-i) / float64(col.Length)
				if intensity > 0.3 {
					grid[y][col.X] = col.Chars[charIndex]
				} else if intensity > 0.1 && rand.Float64() < intensity {
					grid[y][col.X] = col.Chars[charIndex]
				}
			}
		}
	}

	// Convert grid to string
	var result strings.Builder
	style := lipgloss.NewStyle().Foreground(lipgloss.Color("8")) // Dark gray for background effect

	for _, row := range grid {
		line := string(row)
		if strings.TrimSpace(line) != "" {
			result.WriteString(style.Render(line) + "\n")
		} else {
			result.WriteString("\n")
		}
	}

	return result.String()
}

func (h *HackerEffects) StartTypeWriter(text string, speed time.Duration) {
	h.TypeWriter = TypeWriterEffect{
		Text:       text,
		Position:   0,
		Speed:      speed,
		LastUpdate: time.Now(),
		Complete:   false,
	}
}

func (h *HackerEffects) RenderTypeWriter() string {
	if h.TypeWriter.Position == 0 {
		return ""
	}

	displayed := h.TypeWriter.Text[:h.TypeWriter.Position]
	
	// Add cursor if not complete
	if !h.TypeWriter.Complete {
		cursor := "▋"
		if time.Now().UnixNano()/500000000%2 == 0 { // Blink every 500ms
			cursor = " "
		}
		displayed += cursor
	}

	return displayed
}

func (h *HackerEffects) ApplyGlitchEffect(text string) string {
	if h.GlitchEffect.GlitchIntensity == 0 {
		return text
	}

	glitchChars := []rune{'#', '@', '&', '%', '$', '!', '?', '*', '+', '=', '~'}
	result := []rune(text)

	for i := range result {
		if rand.Float64() < h.GlitchEffect.GlitchIntensity {
			result[i] = glitchChars[rand.Intn(len(glitchChars))]
		}
	}

	return string(result)
}

func (h *HackerEffects) GetSpinner(name string) string {
	if s, exists := h.ASCIISpinners[name]; exists {
		return s.Prefix
	}
	return "●"
}

func (h *HackerEffects) CreateASCIIChart(values []float64, width, height int) string {
	if len(values) == 0 {
		return ""
	}

	// Find min/max for scaling
	min, max := values[0], values[0]
	for _, v := range values {
		if v < min {
			min = v
		}
		if v > max {
			max = v
		}
	}

	var result strings.Builder
	chartChars := []rune{' ', '▁', '▂', '▃', '▄', '▅', '▆', '▇', '█'}

	// Render from top to bottom
	for y := height - 1; y >= 0; y-- {
		for x := 0; x < width && x < len(values); x++ {
			// Normalize value to chart height
			normalized := (values[x] - min) / (max - min)
			valueHeight := int(normalized * float64(height))
			
			if valueHeight > y {
				// Calculate which character to use based on partial fill
				charIndex := len(chartChars) - 1
				if valueHeight == y+1 {
					// Partial character for smooth effect
					partial := (normalized*float64(height) - float64(y)) * float64(len(chartChars)-1)
					charIndex = int(partial)
					if charIndex >= len(chartChars) {
						charIndex = len(chartChars) - 1
					}
				}
				result.WriteRune(chartChars[charIndex])
			} else {
				result.WriteRune(' ')
			}
		}
		result.WriteRune('\n')
	}

	return result.String()
}

func (h *HackerEffects) CreateWaveEffect(text string, phase float64) string {
	result := []rune(text)
	
	for i, char := range result {
		if char == ' ' {
			continue
		}
		
		// Create wave displacement
		wave := math.Sin(float64(i)*0.5 + phase)
		if wave > 0.3 {
			// Make character "float up"
			result[i] = char
		} else if wave > -0.3 {
			// Normal position
			result[i] = char
		} else {
			// Make character "sink down" 
			result[i] = rune(strings.ToLower(string(char))[0])
		}
	}
	
	return string(result)
}

func (h *HackerEffects) CreateScanlineEffect(text string, position int) string {
	lines := strings.Split(text, "\n")
	
	for i, line := range lines {
		if i == position%len(lines) {
			// Highlight current scanline
			style := lipgloss.NewStyle().Reverse(true)
			lines[i] = style.Render(line)
		} else if abs(i-position%len(lines)) == 1 {
			// Dim adjacent lines
			style := lipgloss.NewStyle().Foreground(lipgloss.Color("8"))
			lines[i] = style.Render(line)
		}
	}
	
	return strings.Join(lines, "\n")
}

func abs(x int) int {
	if x < 0 {
		return -x
	}
	return x
}