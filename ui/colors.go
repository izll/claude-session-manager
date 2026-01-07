package ui

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

// gradients defines available gradient color schemes
var gradients = map[string][]string{
	"gradient-rainbow":  {"#FF0000", "#FF7F00", "#FFFF00", "#00FF00", "#00FFFF", "#0000FF", "#8B00FF"},
	"gradient-sunset":   {"#FF512F", "#F09819", "#FF8C00", "#DD2476", "#FF416C"},
	"gradient-ocean":    {"#00D2FF", "#3A7BD5", "#00D2D3", "#54A0FF", "#2E86DE"},
	"gradient-forest":   {"#134E5E", "#11998E", "#38EF7D", "#A8E063", "#56AB2F"},
	"gradient-fire":     {"#FF0000", "#FF4500", "#FF6347", "#FF8C00", "#FFD700"},
	"gradient-ice":      {"#E0FFFF", "#B0E0E6", "#87CEEB", "#00CED1", "#4682B4"},
	"gradient-neon":     {"#FF00FF", "#00FFFF", "#39FF14", "#FF6600", "#BF00FF"},
	"gradient-galaxy":   {"#0F0C29", "#302B63", "#8E2DE2", "#4A00E0", "#24243E"},
	"gradient-pastel":   {"#FFB6C1", "#FFDAB9", "#FFFACD", "#98FB98", "#ADD8E6", "#E6E6FA"},
	"gradient-pink":     {"#FF69B4", "#FF1493", "#DB7093", "#FF69B4"},
	"gradient-blue":     {"#00BFFF", "#1E90FF", "#4169E1", "#0000FF", "#4169E1", "#1E90FF"},
	"gradient-green":    {"#00FF00", "#32CD32", "#228B22", "#006400", "#228B22", "#32CD32"},
	"gradient-gold":     {"#FFD700", "#FFA500", "#FF8C00", "#FFA500", "#FFD700"},
	"gradient-purple":   {"#9400D3", "#8A2BE2", "#9932CC", "#BA55D3", "#9932CC", "#8A2BE2"},
	"gradient-cyber":    {"#00FF00", "#00FFFF", "#FF00FF", "#00FFFF", "#00FF00"},
}

// ColorOption represents a color choice for session styling
type ColorOption struct {
	Name  string
	Color string
}

// colorOptions defines available colors for foreground/background
var colorOptions = []ColorOption{
	{"none", ""},
	{"auto", "auto"},
	{"black", "#000000"},
	{"white", "#FFFFFF"},
	{"red", "#FF6B6B"},
	{"orange", "#FFA500"},
	{"yellow", "#FFD93D"},
	{"lime", "#ADFF2F"},
	{"green", "#6BCB77"},
	{"teal", "#20B2AA"},
	{"cyan", "#4DD0E1"},
	{"sky", "#87CEEB"},
	{"blue", "#6C9EFF"},
	{"indigo", "#7B68EE"},
	{"purple", "#B388FF"},
	{"magenta", "#FF00FF"},
	{"pink", "#FF8FAB"},
	{"rose", "#FF69B4"},
	{"coral", "#FF7F50"},
	{"gold", "#FFD700"},
	{"silver", "#C0C0C0"},
	{"gray", "#888888"},
	{"dark-red", "#8B0000"},
	{"dark-green", "#006400"},
	{"dark-blue", "#00008B"},
	{"dark-purple", "#4B0082"},
	// Gradients at the end (for foreground only)
	{"gradient-rainbow", "gradient-rainbow"},
	{"gradient-sunset", "gradient-sunset"},
	{"gradient-ocean", "gradient-ocean"},
	{"gradient-forest", "gradient-forest"},
	{"gradient-fire", "gradient-fire"},
	{"gradient-ice", "gradient-ice"},
	{"gradient-neon", "gradient-neon"},
	{"gradient-galaxy", "gradient-galaxy"},
	{"gradient-pastel", "gradient-pastel"},
	{"gradient-pink", "gradient-pink"},
	{"gradient-blue", "gradient-blue"},
	{"gradient-green", "gradient-green"},
	{"gradient-gold", "gradient-gold"},
	{"gradient-purple", "gradient-purple"},
	{"gradient-cyber", "gradient-cyber"},
}

// sessionColors is an alias for backward compatibility
var sessionColors = colorOptions

// hexToRGB converts hex color to RGB values
func hexToRGB(hex string) (int, int, int) {
	hex = strings.TrimPrefix(hex, "#")
	if len(hex) != 6 {
		return 255, 255, 255
	}
	r, _ := strconv.ParseInt(hex[0:2], 16, 64)
	g, _ := strconv.ParseInt(hex[2:4], 16, 64)
	b, _ := strconv.ParseInt(hex[4:6], 16, 64)
	return int(r), int(g), int(b)
}

// interpolateColor interpolates between colors in a gradient
func interpolateColor(colors []string, position float64) string {
	if len(colors) == 0 {
		return "#FFFFFF"
	}
	if len(colors) == 1 {
		return colors[0]
	}

	// Clamp position
	if position <= 0 {
		return colors[0]
	}
	if position >= 1 {
		return colors[len(colors)-1]
	}

	// Find which segment we're in
	segment := position * float64(len(colors)-1)
	idx := int(segment)
	if idx >= len(colors)-1 {
		idx = len(colors) - 2
	}
	t := segment - float64(idx)

	r1, g1, b1 := hexToRGB(colors[idx])
	r2, g2, b2 := hexToRGB(colors[idx+1])

	r := int(float64(r1) + t*(float64(r2)-float64(r1)))
	g := int(float64(g1) + t*(float64(g2)-float64(g1)))
	b := int(float64(b1) + t*(float64(b2)-float64(b1)))

	return fmt.Sprintf("#%02X%02X%02X", r, g, b)
}

// applyGradientText applies a gradient to text with optional background color and bold
func applyGradientText(text, gradientName, bgColor string, bold bool) string {
	colors, ok := gradients[gradientName]
	if !ok || len(text) == 0 {
		return text
	}

	runes := []rune(text)
	var result strings.Builder

	for i, r := range runes {
		position := float64(i) / float64(len(runes)-1)
		if len(runes) == 1 {
			position = 0.5
		}
		color := interpolateColor(colors, position)
		style := lipgloss.NewStyle().Foreground(lipgloss.Color(color))
		if bgColor != "" {
			style = style.Background(lipgloss.Color(bgColor))
		}
		if bold {
			style = style.Bold(true)
		}
		result.WriteString(style.Render(string(r)))
	}

	return result.String()
}

// applyGradient applies a gradient to text without background
func applyGradient(text, gradientName string) string {
	return applyGradientText(text, gradientName, "", false)
}

// applyGradientWithBg applies a gradient to text with background color
func applyGradientWithBg(text, gradientName, bgColor string) string {
	return applyGradientText(text, gradientName, bgColor, false)
}

// applyGradientWithBgBold applies a gradient to text with background color and bold
func applyGradientWithBgBold(text, gradientName, bgColor string) string {
	return applyGradientText(text, gradientName, bgColor, true)
}

// getContrastColor returns black or white based on background luminance
func getContrastColor(bgColor string) string {
	r, g, b := hexToRGB(bgColor)
	// Calculate luminance
	luminance := (0.299*float64(r) + 0.587*float64(g) + 0.114*float64(b)) / 255.0
	if luminance > 0.5 {
		return "#000000" // Dark text for light background
	}
	return "#FFFFFF" // Light text for dark background
}

// applyTmuxGradient generates tmux format string with per-character gradient colors
func applyTmuxGradient(text, gradientName string, bold bool) string {
	colors, ok := gradients[gradientName]
	if !ok || len(text) == 0 {
		return text
	}

	runes := []rune(text)
	var result strings.Builder

	for i, r := range runes {
		position := float64(i) / float64(len(runes)-1)
		if len(runes) == 1 {
			position = 0.5
		}
		color := interpolateColor(colors, position)
		if bold {
			result.WriteString(fmt.Sprintf("#[fg=%s,bold]%c", color, r))
		} else {
			result.WriteString(fmt.Sprintf("#[fg=%s]%c", color, r))
		}
	}

	return result.String()
}

// formatTmuxSessionName formats session name for tmux status bar with appropriate colors
func formatTmuxSessionName(name, fgColor, bgColor string) string {
	// Check if foreground is gradient
	if _, isGradient := gradients[fgColor]; isGradient {
		// Gradient text + reset color after
		return applyTmuxGradient(name, fgColor, true) + "#[default]"
	}

	// Handle "auto" foreground - use contrast color based on background
	if fgColor == "auto" && bgColor != "" && bgColor != "auto" {
		bgCol := bgColor
		if colors, isGrad := gradients[bgColor]; isGrad && len(colors) > 0 {
			bgCol = colors[0]
		}
		textColor := getContrastColor(bgCol)
		return fmt.Sprintf("#[fg=%s,bg=%s,bold]%s#[default]", textColor, bgCol, name)
	}

	// Plain hex foreground color
	if fgColor != "" && fgColor != "auto" && len(fgColor) > 0 && fgColor[0] == '#' {
		// With background color
		if bgColor != "" && bgColor != "auto" {
			bgCol := bgColor
			if colors, isGrad := gradients[bgColor]; isGrad && len(colors) > 0 {
				bgCol = colors[0]
			}
			return fmt.Sprintf("#[fg=%s,bg=%s,bold]%s#[default]", fgColor, bgCol, name)
		}
		// Foreground only
		return fmt.Sprintf("#[fg=%s,bold]%s#[default]", fgColor, name)
	}

	// Background only (no foreground set) - use white text
	if bgColor != "" && bgColor != "auto" {
		bgCol := bgColor
		if colors, isGrad := gradients[bgColor]; isGrad && len(colors) > 0 {
			bgCol = colors[0]
		}
		return fmt.Sprintf("#[fg=#FAFAFA,bg=%s,bold]%s#[default]", bgCol, name)
	}

	// Default: white on purple
	return fmt.Sprintf("#[fg=#FAFAFA,bg=%s,bold]%s#[default]", ColorPurple, name)
}

// formatSessionNameLipgloss formats session name for UI display with lipgloss
func formatSessionNameLipgloss(name, fgColor, bgColor string) string {
	style := lipgloss.NewStyle().Bold(true)

	// Check if foreground is gradient
	if colors, isGradient := gradients[fgColor]; isGradient {
		// For gradients, apply colors to characters
		return applyLipglossGradient(name, colors)
	}

	// Handle "auto" foreground - use contrast color based on background
	if fgColor == "auto" && bgColor != "" && bgColor != "auto" {
		bgCol := bgColor
		if colors, isGrad := gradients[bgColor]; isGrad && len(colors) > 0 {
			bgCol = colors[0]
		}
		textColor := getContrastColor(bgCol)
		style = style.Foreground(lipgloss.Color(textColor)).Background(lipgloss.Color(bgCol))
		return style.Render(name)
	}

	// Plain hex foreground color
	if fgColor != "" && fgColor != "auto" && len(fgColor) > 0 && fgColor[0] == '#' {
		style = style.Foreground(lipgloss.Color(fgColor))
		// With background color
		if bgColor != "" && bgColor != "auto" {
			bgCol := bgColor
			if colors, isGrad := gradients[bgColor]; isGrad && len(colors) > 0 {
				bgCol = colors[0]
			}
			style = style.Background(lipgloss.Color(bgCol))
		}
		return style.Render(name)
	}

	// Background only (no foreground set) - use white text
	if bgColor != "" && bgColor != "auto" {
		bgCol := bgColor
		if colors, isGrad := gradients[bgColor]; isGrad && len(colors) > 0 {
			bgCol = colors[0]
		}
		style = style.Foreground(lipgloss.Color("#FAFAFA")).Background(lipgloss.Color(bgCol))
		return style.Render(name)
	}

	// Default: white on purple
	style = style.Foreground(lipgloss.Color("#FAFAFA")).Background(lipgloss.Color(ColorPurple))
	return style.Render(name)
}

// applyLipglossGradient applies gradient colors to text using lipgloss
func applyLipglossGradient(text string, colors []string) string {
	if len(colors) == 0 || len(text) == 0 {
		return text
	}

	runes := []rune(text)
	var result strings.Builder

	for i, r := range runes {
		colorIdx := i * len(colors) / len(runes)
		if colorIdx >= len(colors) {
			colorIdx = len(colors) - 1
		}
		style := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color(colors[colorIdx]))
		result.WriteString(style.Render(string(r)))
	}

	return result.String()
}
