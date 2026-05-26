package ui

import (
	"github.com/bbc/infra-pipeline-ui/config"
	"github.com/bbc/infra-pipeline-ui/bitbucket"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
)

// NewModel creates a new Model with the given config and client.
func NewModel(cfg *config.Config, client *bitbucket.Client) Model {
	runBranch := textinput.New()
	runBranch.Placeholder = "main"
	runBranch.CharLimit = 200
	runBranch.Width = 40

	runSelector := textinput.New()
	runSelector.Placeholder = "default"
	runSelector.CharLimit = 200
	runSelector.Width = 40

	runEditorInput := textinput.New()
	runEditorInput.Placeholder = ""
	runEditorInput.CharLimit = 500
	runEditorInput.Width = 30

	return Model{
		Config:         cfg,
		Client:         client,
		Projects:       cfg.Projects,
		ListState:      StateLoading,
		RunBranch:      runBranch,
		RunSelector:    runSelector,
		RunEditorInput: runEditorInput,
		RunVars:        []bitbucket.PipelineVariable{},
		RunEditMode:    false,
		Screen:         ScreenList,
	}
}

// Init initializes the model, returning the command to fetch pipelines.
func (m Model) Init() tea.Cmd {
	return tea.Batch(
		fetchPipelines(m.Client, m.Projects[m.ActiveProject].Workspace, m.Projects[m.ActiveProject].RepoSlug),
	)
}

// Update handles all messages and routes them to the appropriate handler.
func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd

	// Global key handlers
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.Width = msg.Width
		m.Height = msg.Height
		if !m.Ready {
			m.Ready = true
		}
		return m, nil

	case tea.KeyMsg:
		switch msg.String() {
		case "q", "ctrl+c":
			return m, tea.Quit

		case "esc":
			return m.handleEsc()

		case "p":
			// Open project selection screen (list and detail screens)
			if m.Screen == ScreenList || m.Screen == ScreenDetail {
				if len(m.Projects) > 1 {
					m.Screen = ScreenProjects
				}
			}
			return m, nil
		}
	}

	// Route to screen-specific handlers
	switch m.Screen {
	case ScreenList:
		return m.updateList(msg)
	case ScreenDetail:
		return m.updateDetail(msg)
	case ScreenLogs:
		return m.updateLogs(msg)
	case ScreenRun:
		return m.updateRun(msg)
	case ScreenProjects:
		return m.updateProjects(msg)
	}

	return m, tea.Batch(cmds...)
}

func (m Model) handleEsc() (tea.Model, tea.Cmd) {
	switch m.Screen {
	case ScreenDetail:
		m.Screen = ScreenList
		m.SelectedPipeline = nil
		m.Steps = nil
		m.ListCursor = 0
		// Refresh pipeline list
		m.ListState = StateLoading
		m.Pipelines = nil
		return m, fetchPipelines(m.Client, m.Projects[m.ActiveProject].Workspace, m.Projects[m.ActiveProject].RepoSlug)
	case ScreenLogs:
		m.Screen = ScreenDetail
		m.LogContent = ""
		m.LogSearchTerm = ""
		m.LogState = StateLoading
		return m, nil
	case ScreenRun:
		m.Screen = ScreenDetail
		m.RunError = ""
		m.RunSuccessMsg = ""
		m.RunBranch.SetValue("")
		m.RunEditorInput.SetValue("")
		m.RunEditorInput.Blur()
		m.RunVars = []bitbucket.PipelineVariable{}
		m.RunVarCursor = 0
		m.RunEditMode = false
		m.RunFocus = runFieldBranch
		m.RunState = StateReady
		return m, nil
	default:
		return m, nil
	}
}

// handleLoading checks if the current screen is in loading state.
func handleLoading(state int) tea.Cmd {
	if state == StateLoading {
		return nil
	}
	return nil
}

// handleErrorMsg handles error messages for any screen.
func (m *Model) handleErrorMsg(screen string, err error) {
	switch screen {
	case "list":
		m.ListState = StateError
		m.ListError = err.Error()
	case "detail":
		m.DetailState = StateError
		m.DetailError = err.Error()
	case "log":
		m.LogState = StateError
		m.LogError = err.Error()
	case "run":
		m.RunState = StateError
		m.RunError = err.Error()
	}
}