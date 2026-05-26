package ui

import (
	"github.com/charmbracelet/lipgloss"
)

var (
	// Base styles
	AppStyle = lipgloss.NewStyle().
			Padding(0).
			Margin(0)

	// Header
	HeaderStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("#FFD700")).
			Background(lipgloss.Color("#1a1a2e")).
			Padding(0, 1)

	TabStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#888888")).
			Padding(0, 1)

	ActiveTabStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#FFFFFF")).
			Background(lipgloss.Color("#352b7b")).
			Bold(true).
			Padding(0, 1)

	// Status badges
	StatusInProgress = lipgloss.NewStyle().
				Foreground(lipgloss.Color("#3498db")).
				Bold(true)

	StatusPending = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#f1c40f")).
			Bold(true)

	StatusSuccessful = lipgloss.NewStyle().
				Foreground(lipgloss.Color("#2ecc71")).
				Bold(true)

	StatusFailed = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#e74c3c")).
			Bold(true)

	StatusStopped = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#95a5a6")).
			Bold(true)

	// List styles
	ListCursorStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#FFFFFF")).
			Background(lipgloss.Color("#352b7b")).
			Bold(true).
			Padding(0, 1)

	ListNormalStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#cccccc")).
			Padding(0, 1)

	// Detail styles
	DetailSectionStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("#FFD700")).
				Bold(true).
				Underline(true)

	DetailKeyStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#888888"))

	DetailValueStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("#ffffff"))

	// Help text
	HelpStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#555555")).
			Italic(true)

	// Error
	ErrorStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#e74c3c")).
			Bold(true)

	// Success
	SuccessStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#2ecc71")).
			Bold(true)

	// Loading
	LoadingStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#3498db"))

	// Input fields
	InputStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#ffffff")).
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("#352b7b")).
			Padding(0, 1)

	FocusedInputStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("#ffffff")).
				Border(lipgloss.RoundedBorder()).
				BorderForeground(lipgloss.Color("#FFD700")).
				Padding(0, 1)

	// Filter
	FilterStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#FFD700"))

	// Title
	TitleStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("#FFD700")).
			Padding(0, 1)

	// Border for panels
	PanelStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("#352b7b")).
			Padding(0, 1)

	// Log search highlight
	LogHighlightStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("#000000")).
				Background(lipgloss.Color("#FFD700"))

	// Dimmed
	DimmedStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#666666"))

	// Content area style that fills available space
	ContentStyle = lipgloss.NewStyle().
			Padding(0, 1)
)
