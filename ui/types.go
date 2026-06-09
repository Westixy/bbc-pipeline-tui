package ui

import (
	"fmt"

	"github.com/bbc/infra-pipeline-ui/bitbucket"
	"github.com/bbc/infra-pipeline-ui/config"
	"github.com/charmbracelet/bubbles/textinput"
)

// Screen constants
const (
	ScreenList = iota
	ScreenDetail
	ScreenLogs
	ScreenRun
	ScreenProjects
	ScreenManageProjects
)

// State constants
const (
	StateLoading = iota
	StateReady
	StateError
)

// Model is the main Bubble Tea model for the TUI.
type Model struct {
	// Config & client
	Config   *config.Config
	Client   *bitbucket.CachedClient
	Projects []config.Project

	// Current project tab
	ActiveProject int

	// Pipeline list
	Pipelines     []bitbucket.Pipeline
	NextPageURL   string
	ListCursor    int
	ListFilter    string
	ListState     int // StateLoading, StateReady, StateError
	ListError     string
	ListScrollOff int // scroll offset for list viewport

	// Pipeline detail
	SelectedPipeline *bitbucket.Pipeline
	Steps            []bitbucket.PipelineStep
	StepCursor       int    // cursor for step selection in detail view
	DetailScrollOff  int    // scroll offset for detail/steps viewport
	DetailMessage    string // transient message (e.g. browser URL opened)
	ConfigVars       []bitbucket.PipelineVariable
	ParsedLogVars    []bitbucket.PipelineVariable
	DetailState      int
	DetailError      string

	// Pipeline preview (split panel with list)
	PreviewPipeline    *bitbucket.Pipeline
	PreviewSteps       []bitbucket.PipelineStep
	PreviewConfigVars  []bitbucket.PipelineVariable
	PreviewUUID        string
	PreviewState       int
	PreviewError       string
	PreviewScrollOff   int
	FocusPanel         int // 0 = list, 1 = preview
	SplitLeftWidth     int

	// Logs
	LogContent    string
	LogStepName   string
	LogSearchTerm string
	LogSearchMode bool  // true when the user is typing a search term
	LogMatchIndex int   // current match index (-1 if no match selected)
	LogMatchLines []int // line numbers of all matches
	LogShowVars   bool  // show parsed variables instead of log content
	LogState      int
	LogError      string
	LogScrollOff  int // vertical scroll offset for log viewport
	LogHScroll    int // horizontal scroll offset (columns)

	// Manage projects
	ManageWorkspaceInput textinput.Model
	WorkspaceRepos       []bitbucket.Repository
	WorkspaceReposState  int // StateLoading, StateReady, StateError
	WorkspaceReposError  string
	ManageFocus          int // 0 = favorites pane, 1 = browse pane
	ManageFavCursor      int // cursor in favorites list
	ManageRepoCursor     int // cursor in workspace repos list
	ManageFavScrollOff   int // scroll offset in favorites pane
	ManageRepoScrollOff  int // scroll offset in workspace repos pane

	// Run form
	RunBranch      textinput.Model
	RunSelector    textinput.Model // custom pipeline selector pattern
	RunEditorInput textinput.Model // used for inline variable editing
	RunVars        []bitbucket.PipelineVariable
	RunVarCursor   int  // cursor position in variable list
	RunEditMode    bool // true when editing a variable inline
	RunFocus       int
	RunState       int
	RunError       string
	RunSuccessMsg  string

	// Global
	Width  int
	Height int
	Screen int
	Ready  bool
}

// ScreenTitle returns the title for the current screen.
func (m Model) ScreenTitle() string {
	p := m.Projects[m.ActiveProject]
	prefix := p.Workspace + "/" + p.RepoSlug
	switch m.Screen {
	case ScreenList:
		return prefix + " - Pipelines"
	case ScreenDetail:
		pipeline := m.SelectedPipeline
		bn := pipeline.BuildNumber
		state := pipeline.State.Name
		if pipeline.State.Result != nil {
			state += "/" + pipeline.State.Result.Name
		}
		branch := pipeline.Target.RefName
		return fmt.Sprintf(prefix+" - Pipeline #%d (%s) [%s]", bn, branch, state)
	case ScreenLogs:
		return prefix + " - Step Log: " + m.LogStepName
	case ScreenRun:
		return prefix + " - Trigger Pipeline"
	case ScreenProjects:
		return "Select Project"
	case ScreenManageProjects:
		return "Manage Projects"
	}
	return ""
}

// FullRepoName returns the full workspace/repo_slug name for the active project.
func (m Model) FullRepoName() string {
	p := m.Projects[m.ActiveProject]
	return p.Workspace + "/" + p.RepoSlug
}
