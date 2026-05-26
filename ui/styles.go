package ui

import "github.com/charmbracelet/lipgloss"

// ── Color palette (dark theme, accessible) ──────────────────────────────────
var (
	bg            = lipgloss.Color("#0f0f1a")
	bgSecondary   = lipgloss.Color("#16162a")
	surface       = lipgloss.Color("#1e1e3a")
	surfaceAlt    = lipgloss.Color("#262648")
	border        = lipgloss.Color("#3a3a5c")
	borderFocus   = lipgloss.Color("#6c5ce7")
	accentGold    = lipgloss.Color("#ffd700")
	accentCyan    = lipgloss.Color("#00cec9")
	textPrimary   = lipgloss.Color("#e0e0e0")
	textSecondary = lipgloss.Color("#9090a0")
	textDimmed    = lipgloss.Color("#5a5a70")
	textError     = lipgloss.Color("#ff6b6b")
	textSuccess   = lipgloss.Color("#00d2a0")
	textWarning   = lipgloss.Color("#feca57")
	textInfo      = lipgloss.Color("#74b9ff")
	textMuted     = lipgloss.Color("#6c6c84")

	// Status pill backgrounds
	pillInProgress = lipgloss.Color("#1a3a5c")
	pillPending    = lipgloss.Color("#3a3a1a")
	pillSuccess    = lipgloss.Color("#1a3a2a")
	pillFailed     = lipgloss.Color("#3a1a1a")
	pillStopped    = lipgloss.Color("#2a2a3a")
)

// ── Base ────────────────────────────────────────────────────────────────────
var (
	AppStyle = lipgloss.NewStyle().Padding(0).Margin(0)

	ContentStyle = lipgloss.NewStyle().
			Padding(0, 2).
			Background(bg)
)

// ── Header ──────────────────────────────────────────────────────────────────
var (
	HeaderStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(accentGold).
			Background(surface).
			Padding(0, 2).
			BorderBottom(true).
			BorderStyle(lipgloss.NormalBorder()).
			BorderForeground(borderFocus)

	TitleStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(accentGold).
			Padding(0, 1)

	TabStyle = lipgloss.NewStyle().
			Foreground(textDimmed).
			Padding(0, 1)

	ActiveTabStyle = lipgloss.NewStyle().
			Foreground(textPrimary).
			Background(borderFocus).
			Bold(true).
			Padding(0, 2)
)

// ── Status badges (colored pills) ───────────────────────────────────────────
var (
	StatusInProgress = lipgloss.NewStyle().
				Foreground(textInfo).
				Background(pillInProgress).
				Bold(true).
				Padding(0, 1)

	StatusPending = lipgloss.NewStyle().
			Foreground(textWarning).
			Background(pillPending).
			Bold(true).
			Padding(0, 1)

	StatusSuccessful = lipgloss.NewStyle().
				Foreground(textSuccess).
				Background(pillSuccess).
				Bold(true).
				Padding(0, 1)

	StatusFailed = lipgloss.NewStyle().
			Foreground(textError).
			Background(pillFailed).
			Bold(true).
			Padding(0, 1)

	StatusStopped = lipgloss.NewStyle().
			Foreground(textDimmed).
			Background(pillStopped).
			Bold(true).
			Padding(0, 1)
)

// ── List styles (zebra striping) ────────────────────────────────────────────
var (
	ListCursorStyle = lipgloss.NewStyle().
			Foreground(textPrimary).
			Background(borderFocus).
			Bold(true)

	ListNormalStyle = lipgloss.NewStyle().
			Foreground(textPrimary).
			Background(bg)

	ListAltStyle = lipgloss.NewStyle().
			Foreground(textPrimary).
			Background(bgSecondary)

	ListHeaderStyle = lipgloss.NewStyle().
			Foreground(textDimmed).
			Background(surface).
			Bold(true).
			Padding(0, 1)
)

// ── Detail / Card styles ────────────────────────────────────────────────────
var (
	SectionTitleStyle = lipgloss.NewStyle().
				Foreground(accentCyan).
				Bold(true).
				Padding(0, 1).
				BorderBottom(true).
				BorderStyle(lipgloss.NormalBorder()).
				BorderForeground(border)

	CardStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(border).
			Padding(1, 2).
			Background(surface)

	CardTitleStyle = lipgloss.NewStyle().
			Foreground(accentGold).
			Bold(true)

	KeyStyle = lipgloss.NewStyle().
			Foreground(textDimmed)

	ValueStyle = lipgloss.NewStyle().
			Foreground(textPrimary)

	StepCursorStyle = lipgloss.NewStyle().
			Foreground(accentCyan).
			Bold(true)

	StepNormalStyle = lipgloss.NewStyle().
			Foreground(textPrimary)
)

// ── Form / Input styles ─────────────────────────────────────────────────────
var (
	InputStyle = lipgloss.NewStyle().
			Foreground(textPrimary).
			Background(surface).
			Border(lipgloss.RoundedBorder()).
			BorderForeground(border).
			Padding(0, 1)

	FocusedInputStyle = lipgloss.NewStyle().
				Foreground(textPrimary).
				Background(surface).
				Border(lipgloss.RoundedBorder()).
				BorderForeground(borderFocus).
				Padding(0, 1)

	RequiredMarkerStyle = lipgloss.NewStyle().
				Foreground(textError).
				Bold(true)
)

// ── Log styles ──────────────────────────────────────────────────────────────
var (
	LogHighlightStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("#000000")).
				Background(lipgloss.Color("#ffd700"))

	LogLineNumStyle = lipgloss.NewStyle().
			Foreground(textDimmed).
			Width(5).
			Align(lipgloss.Right)

	LogMatchCounterStyle = lipgloss.NewStyle().
				Foreground(accentCyan).
				Bold(true)
)

// ── Feedback styles ─────────────────────────────────────────────────────────
var (
	HelpStyle = lipgloss.NewStyle().
			Foreground(textDimmed).
			Background(surface).
			Padding(0, 1).
			BorderTop(true).
			BorderStyle(lipgloss.NormalBorder()).
			BorderForeground(border)

	HelpGroupStyle = lipgloss.NewStyle().
			Foreground(textDimmed)

	HelpKeyStyle = lipgloss.NewStyle().
			Foreground(accentGold).
			Bold(true)

	ErrorStyle = lipgloss.NewStyle().
			Foreground(textError).
			Bold(true)

	SuccessStyle = lipgloss.NewStyle().
			Foreground(textSuccess).
			Bold(true)

	WarningStyle = lipgloss.NewStyle().
			Foreground(textWarning)

	LoadingStyle = lipgloss.NewStyle().
			Foreground(textInfo)

	FilterStyle = lipgloss.NewStyle().
			Foreground(accentGold).
			Background(surface).
			Padding(0, 1)

	// Empty state
	EmptyStateStyle = lipgloss.NewStyle().
			Foreground(textDimmed).
			Align(lipgloss.Center).
			Padding(2, 0)

	// Badge (small rounded pill for counts)
	BadgeStyle = lipgloss.NewStyle().
			Foreground(textPrimary).
			Background(borderFocus).
			Padding(0, 1)

	// Scrollbar
	ScrollThumbStyle = lipgloss.NewStyle().
				Foreground(borderFocus)

	ScrollTrackStyle = lipgloss.NewStyle().
				Foreground(border)
)

// ── Utility ─────────────────────────────────────────────────────────────────
var (
	DimmedStyle = lipgloss.NewStyle().
			Foreground(textDimmed)

	DividerStyle = lipgloss.NewStyle().
			Foreground(border)
)
