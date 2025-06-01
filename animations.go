package main

import (
	"math"
	"time"

	"github.com/charmbracelet/harmonica"
	"github.com/charmbracelet/lipgloss"
)

type AnimationState struct {
	SpringAnimation harmonica.Spring
	Target          float64
	Current         float64
	FadeAlpha       float64
	ScrollOffset    float64
	GradientPhase   float64
	ParticleSystem  []Particle
	LastUpdate      time.Time
}

type Particle struct {
	X, Y     float64
	VX, VY   float64
	Life     float64
	MaxLife  float64
	Color    string
	Char     string
}

func NewAnimationState() *AnimationState {
	return &AnimationState{
		SpringAnimation: harmonica.NewSpring(harmonica.FPS(60), 3.0, 0.8),
		Target:          0.0,
		Current:         0.0,
		FadeAlpha:       1.0,
		ScrollOffset:    0.0,
		GradientPhase:   0.0,
		ParticleSystem:  []Particle{},
		LastUpdate:      time.Now(),
	}
}

func (a *AnimationState) Update() {
	now := time.Now()
	dt := now.Sub(a.LastUpdate).Seconds()
	a.LastUpdate = now

	// Update spring animation for smooth scrolling
	velocity, position := a.SpringAnimation.Update(dt, a.Target, a.Current)
	a.Current = position
	_ = velocity

	// Update gradient phase for rainbow effects
	a.GradientPhase += dt * 2.0 // 2 cycles per second
	if a.GradientPhase > 2*math.Pi {
		a.GradientPhase -= 2 * math.Pi
	}

	// Update particles
	for i := len(a.ParticleSystem) - 1; i >= 0; i-- {
		p := &a.ParticleSystem[i]
		p.Life -= dt
		p.X += p.VX * dt
		p.Y += p.VY * dt
		p.VY += 50 * dt // gravity

		if p.Life <= 0 {
			// Remove dead particle
			a.ParticleSystem = append(a.ParticleSystem[:i], a.ParticleSystem[i+1:]...)
		}
	}
}

func (a *AnimationState) AddStatusChangeParticles(x, y int, status string) {
	colors := map[string][]string{
		"✓": {"46", "82", "118"},      // greens
		"✗": {"196", "160", "124"},    // reds
		"↑": {"220", "226", "190"},    // yellows
		"↓": {"33", "39", "45"},       // blues
		"↕": {"165", "171", "177"},    // purples
	}

	chars := []string{"·", "✦", "✧", "⋆", "★"}
	particleColors, exists := colors[status]
	if !exists {
		particleColors = []string{"15"} // white fallback
	}

	// Create burst of particles
	for i := 0; i < 8; i++ {
		angle := float64(i) * math.Pi / 4
		speed := 20.0 + math.Mod(float64(i)*17, 10) // varied speeds
		
		particle := Particle{
			X:       float64(x),
			Y:       float64(y),
			VX:      math.Cos(angle) * speed,
			VY:      math.Sin(angle)*speed - 20, // upward bias
			Life:    0.8 + math.Mod(float64(i)*13, 0.4),
			MaxLife: 1.2,
			Color:   particleColors[i%len(particleColors)],
			Char:    chars[i%len(chars)],
		}
		a.ParticleSystem = append(a.ParticleSystem, particle)
	}
}

func (a *AnimationState) AnimateToPosition(target float64) {
	a.Target = target
}

func (a *AnimationState) GetCurrentPosition() float64 {
	return a.Current
}

func (a *AnimationState) CreateRainbowGradient(text string, baseHue float64) string {
	if len(text) == 0 {
		return text
	}

	var result string
	for i, char := range text {
		// Create rainbow effect
		hue := baseHue + float64(i)*30 + a.GradientPhase*57.3 // convert radians to degrees
		hue = math.Mod(hue, 360)
		
		// Convert HSV to RGB (simplified)
		c := 1.0
		x := c * (1 - math.Abs(math.Mod(hue/60, 2)-1))
		var r, g, b float64
		
		switch {
		case hue < 60:
			r, g, b = c, x, 0
		case hue < 120:
			r, g, b = x, c, 0
		case hue < 180:
			r, g, b = 0, c, x
		case hue < 240:
			r, g, b = 0, x, c
		case hue < 300:
			r, g, b = x, 0, c
		default:
			r, g, b = c, 0, x
		}
		
		// Convert to 0-255 and then to color code
		colorCode := int(r*5)*36 + int(g*5)*6 + int(b*5) + 16
		
		style := lipgloss.NewStyle().Foreground(lipgloss.Color(string(rune(colorCode))))
		result += style.Render(string(char))
	}
	
	return result
}

func (a *AnimationState) CreatePulsingEffect(text string, intensity float64) string {
	// Create pulsing brightness effect
	phase := math.Sin(a.GradientPhase * 3) // 3 pulses per gradient cycle
	alpha := 0.3 + 0.7*(phase+1)/2 // pulse between 0.3 and 1.0
	alpha *= intensity
	
	// Convert alpha to brightness
	brightness := int(alpha * 255)
	if brightness > 255 {
		brightness = 255
	}
	
	style := lipgloss.NewStyle().
		Foreground(lipgloss.Color("15")).
		Background(lipgloss.Color("0")).
		Bold(alpha > 0.7) // bold when bright
	
	return style.Render(text)
}

func (a *AnimationState) RenderParticles(width, height int) string {
	if len(a.ParticleSystem) == 0 {
		return ""
	}
	
	// Create 2D grid for particle rendering
	grid := make([][]string, height)
	for i := range grid {
		grid[i] = make([]string, width)
		for j := range grid[i] {
			grid[i][j] = " "
		}
	}
	
	// Place particles in grid
	for _, p := range a.ParticleSystem {
		x, y := int(p.X), int(p.Y)
		if x >= 0 && x < width && y >= 0 && y < height {
			alpha := p.Life / p.MaxLife
			style := lipgloss.NewStyle().
				Foreground(lipgloss.Color(p.Color)).
				Background(lipgloss.Color("0"))
			
			if alpha < 0.3 {
				// Fade out
				style = style.Foreground(lipgloss.Color("8"))
			}
			
			grid[y][x] = style.Render(p.Char)
		}
	}
	
	// Convert grid to string
	var result string
	for _, row := range grid {
		for _, cell := range row {
			result += cell
		}
		result += "\n"
	}
	
	return result
}

func (a *AnimationState) CreateProgressBar(progress float64, width int, style string) string {
	filled := int(progress * float64(width))
	if filled > width {
		filled = width
	}
	
	var bar string
	
	switch style {
	case "smooth":
		// Smooth gradient bar
		for i := 0; i < width; i++ {
			if i < filled {
				intensity := float64(i) / float64(width)
				hue := 120 * intensity // green to red
				color := lipgloss.Color(string(rune(int(hue/360*255))))
				bar += lipgloss.NewStyle().Background(color).Render(" ")
			} else {
				bar += lipgloss.NewStyle().Background(lipgloss.Color("8")).Render(" ")
			}
		}
	case "blocks":
		// Block style with animation
		chars := []string{"▁", "▂", "▃", "▄", "▅", "▆", "▇", "█"}
		for i := 0; i < width; i++ {
			if i < filled {
				charIndex := int((a.GradientPhase + float64(i)*0.5)) % len(chars)
				style := lipgloss.NewStyle().Foreground(lipgloss.Color("46"))
				bar += style.Render(chars[charIndex])
			} else {
				bar += "·"
			}
		}
	default:
		// Default style
		for i := 0; i < filled; i++ {
			bar += "█"
		}
		for i := filled; i < width; i++ {
			bar += "░"
		}
	}
	
	return bar
}