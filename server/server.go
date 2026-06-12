package server

import (
	"embed"
	"encoding/json"
	"fmt"
	"io"
	"io/fs"
	"log"
	"net/http"
	"regexp"
	"strconv"
	"strings"

	"github.com/bbc/infra-pipeline-ui/bitbucket"
	"github.com/bbc/infra-pipeline-ui/config"
)

//go:embed all:webapp-dist
var webappDist embed.FS

// Server holds the HTTP server state.
type Server struct {
	cfg    *config.Config
	client *bitbucket.Client
	mux    *http.ServeMux
}

// New creates a new HTTP server wrapping the Bitbucket client.
func New(cfg *config.Config, client *bitbucket.Client) *Server {
	s := &Server{
		cfg:    cfg,
		client: client,
		mux:    http.NewServeMux(),
	}
	s.registerRoutes()
	return s
}

// Handler returns the HTTP handler for the server.
func (s *Server) Handler() http.Handler {
	return corsMiddleware(s.mux)
}

// registerRoutes sets up all API routes.
func (s *Server) registerRoutes() {
	// Health
	s.mux.HandleFunc("GET /api/health", s.handleHealth)

	// Projects
	s.mux.HandleFunc("GET /api/projects", s.handleListProjects)
	s.mux.HandleFunc("POST /api/projects", s.handleAddProject)
	s.mux.HandleFunc("DELETE /api/projects/{id}", s.handleRemoveProject)

	// Project-specific operations (scoped by project index)
	s.mux.HandleFunc("GET /api/projects/{id}/pipelines", s.handleListPipelines)
	s.mux.HandleFunc("GET /api/projects/{id}/pipelines/{uuid}", s.handleGetPipeline)
	s.mux.HandleFunc("POST /api/projects/{id}/pipelines", s.handleTriggerPipeline)
	s.mux.HandleFunc("POST /api/projects/{id}/pipelines/{uuid}/stop", s.handleStopPipeline)
	s.mux.HandleFunc("GET /api/projects/{id}/pipelines/{uuid}/steps", s.handleListSteps)
	s.mux.HandleFunc("GET /api/projects/{id}/pipelines/{uuid}/steps/{stepUuid}", s.handleGetStep)
	s.mux.HandleFunc("GET /api/projects/{id}/pipelines/{uuid}/steps/{stepUuid}/log", s.handleGetStepLog)
	s.mux.HandleFunc("GET /api/projects/{id}/pipelines/{uuid}/log-vars", s.handleGetLogVariables)
	s.mux.HandleFunc("GET /api/projects/{id}/variables", s.handleListVariables)

	// Workspace repositories
	s.mux.HandleFunc("GET /api/repositories/{workspace}", s.handleListRepositories)

	// Serve embedded Svelte frontend
	distFS, err := fs.Sub(webappDist, "webapp-dist")
	if err != nil {
		log.Printf("WARNING: webapp-dist not embedded (run 'npm run build' in webapp/): %v", err)
		s.mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "text/plain")
			w.Write([]byte("Frontend not built. Run 'npm run build' in the webapp/ directory."))
		})
		return
	}
	fileServer := http.FileServer(http.FS(distFS))

	// Single SPA handler: serve static files if they exist, else fallback to index.html
	s.mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		cleanPath := strings.TrimPrefix(r.URL.Path, "/")
		if cleanPath == "" {
			cleanPath = "index.html"
		}

		// Try serving the exact file
		f, err := distFS.Open(cleanPath)
		if err == nil {
			f.Close()
			fileServer.ServeHTTP(w, r)
			return
		}

		// Try as a directory (serve index.html inside if present)
		f2, err2 := distFS.Open(strings.TrimSuffix(cleanPath, "/") + "/index.html")
		if err2 == nil {
			f2.Close()
		}

		// SPA fallback: serve index.html for any non-file route
		indexData, err3 := fs.ReadFile(distFS, "index.html")
		if err3 != nil {
			http.Error(w, "Not found", http.StatusNotFound)
			return
		}
		w.Header().Set("Content-Type", "text/html")
		w.Write(indexData)
	})
}

// parseProjectID extracts the project index from the URL path value.
func (s *Server) projectByID(idStr string) (*config.Project, int, error) {
	id, err := strconv.Atoi(idStr)
	if err != nil || id < 0 || id >= len(s.cfg.Projects) {
		return nil, -1, errProjectNotFound
	}
	return &s.cfg.Projects[id], id, nil
}

// corsMiddleware adds CORS headers for development.
func corsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
		if r.Method == "OPTIONS" {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		next.ServeHTTP(w, r)
	})
}

// --- Errors ---

var errProjectNotFound = &apiError{Code: 404, Message: "Project not found"}

type apiError struct {
	Code    int    `json:"-"`
	Message string `json:"message"`
}

func (e *apiError) Error() string { return e.Message }

func writeJSON(w http.ResponseWriter, status int, v interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(v)
}

// ensureBraces wraps UUID with curly braces — Go 1.22 mux strips { } from route params
// but Bitbucket API requires UUIDs wrapped in braces like {a29195fa-...}.
func ensureBraces(uuid string) string {
	uuid = strings.Trim(uuid, "{}")
	return "{" + uuid + "}"
}

func writeError(w http.ResponseWriter, err error) {
	if ae, ok := err.(*apiError); ok {
		writeJSON(w, ae.Code, ae)
	} else {
		writeJSON(w, http.StatusInternalServerError, &apiError{Code: 500, Message: err.Error()})
	}
}

// --- Handlers ---

func (s *Server) handleHealth(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, 200, map[string]string{"status": "ok"})
}

type projectResponse struct {
	ID        int    `json:"id"`
	Name      string `json:"name"`
	Workspace string `json:"workspace"`
	RepoSlug  string `json:"repo_slug"`
}

func (s *Server) handleListProjects(w http.ResponseWriter, r *http.Request) {
	s.writeProjectList(w)
}

func (s *Server) handleAddProject(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Workspace string `json:"workspace"`
		RepoSlug  string `json:"repo_slug"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, &apiError{Code: 400, Message: "Invalid JSON: " + err.Error()})
		return
	}
	if req.Workspace == "" || req.RepoSlug == "" {
		writeError(w, &apiError{Code: 400, Message: "workspace and repo_slug are required"})
		return
	}
	if s.cfg.AddProject(req.Workspace, req.RepoSlug) {
		if err := s.cfg.Save(); err != nil {
			writeError(w, err)
			return
		}
	}
	s.writeProjectList(w)
}

func (s *Server) handleRemoveProject(w http.ResponseWriter, r *http.Request) {
	idStr := r.PathValue("id")
	id, err := strconv.Atoi(idStr)
	if err != nil || id < 0 || id >= len(s.cfg.Projects) {
		writeError(w, errProjectNotFound)
		return
	}
	p := s.cfg.Projects[id]
	s.cfg.RemoveProject(p.Workspace, p.RepoSlug)
	if err := s.cfg.Save(); err != nil {
		writeError(w, err)
		return
	}
	s.writeProjectList(w)
}

func (s *Server) writeProjectList(w http.ResponseWriter) {
	projs := make([]projectResponse, len(s.cfg.Projects))
	for i, p := range s.cfg.Projects {
		projs[i] = projectResponse{
			ID:        i,
			Name:      p.Workspace + "/" + p.RepoSlug,
			Workspace: p.Workspace,
			RepoSlug:  p.RepoSlug,
		}
	}
	writeJSON(w, 200, map[string]interface{}{
		"projects": projs,
	})
}

func (s *Server) handleListPipelines(w http.ResponseWriter, r *http.Request) {
	project, _, err := s.projectByID(r.PathValue("id"))
	if err != nil {
		writeError(w, err)
		return
	}
	params := &bitbucket.ListPipelinesParams{
		Sort: r.URL.Query().Get("sort"),
	}
	if pl := r.URL.Query().Get("pagelen"); pl != "" {
		if n, e := strconv.Atoi(pl); e == nil {
			params.Pagelen = n
		}
	}
	if pg := r.URL.Query().Get("page"); pg != "" {
		if n, e := strconv.Atoi(pg); e == nil {
			params.Page = n
		}
	}
	if filter := r.URL.Query().Get("filter"); filter != "" {
		params.Filter = fmt.Sprintf("target.ref_name=\"%s\"", filter)
	}
	if params.Sort == "" {
		params.Sort = "-created_on"
	}

	result, err := s.client.ListPipelines(project.Workspace, project.RepoSlug, params)
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, 200, result)
}

func (s *Server) handleGetPipeline(w http.ResponseWriter, r *http.Request) {
	project, _, err := s.projectByID(r.PathValue("id"))
	if err != nil {
		writeError(w, err)
		return
	}
	uuid := ensureBraces(r.PathValue("uuid"))

	pipeline, err := s.client.GetPipeline(project.Workspace, project.RepoSlug, uuid)
	if err != nil {
		writeError(w, err)
		return
	}

	// Also fetch steps
	steps := []bitbucket.PipelineStep{}
	stepsResult, err := s.client.ListPipelineSteps(project.Workspace, project.RepoSlug, uuid)
	if err == nil {
		steps = stepsResult.Values
	}

	response := map[string]interface{}{
		"pipeline": pipeline,
		"steps":    steps,
	}
	writeJSON(w, 200, response)
}

func (s *Server) handleListSteps(w http.ResponseWriter, r *http.Request) {
	project, _, err := s.projectByID(r.PathValue("id"))
	if err != nil {
		writeError(w, err)
		return
	}
	uuid := ensureBraces(r.PathValue("uuid"))

	steps, err := s.client.ListPipelineSteps(project.Workspace, project.RepoSlug, uuid)
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, 200, steps)
}

func (s *Server) handleGetStep(w http.ResponseWriter, r *http.Request) {
	project, _, err := s.projectByID(r.PathValue("id"))
	if err != nil {
		writeError(w, err)
		return
	}

	step, err := s.client.GetPipelineStep(project.Workspace, project.RepoSlug, ensureBraces(r.PathValue("uuid")), ensureBraces(r.PathValue("stepUuid")))
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, 200, step)
}

func (s *Server) handleGetStepLog(w http.ResponseWriter, r *http.Request) {
	project, _, err := s.projectByID(r.PathValue("id"))
	if err != nil {
		writeError(w, err)
		return
	}

	logContent, err := s.client.GetStepLog(project.Workspace, project.RepoSlug, ensureBraces(r.PathValue("uuid")), ensureBraces(r.PathValue("stepUuid")))
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, 200, map[string]string{
		"log":      logContent,
		"stepUuid": r.PathValue("stepUuid"),
	})
}

func (s *Server) handleTriggerPipeline(w http.ResponseWriter, r *http.Request) {
	project, _, err := s.projectByID(r.PathValue("id"))
	if err != nil {
		writeError(w, err)
		return
	}

	var req bitbucket.TriggerPipelineRequest
	body, _ := io.ReadAll(r.Body)
	if len(body) > 0 {
		if err := json.Unmarshal(body, &req); err != nil {
			writeError(w, &apiError{Code: 400, Message: "Invalid JSON: " + err.Error()})
			return
		}
	}

	pipeline, err := s.client.TriggerPipeline(project.Workspace, project.RepoSlug, req)
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, 201, pipeline)
}

func (s *Server) handleStopPipeline(w http.ResponseWriter, r *http.Request) {
	project, _, err := s.projectByID(r.PathValue("id"))
	if err != nil {
		writeError(w, err)
		return
	}

	if err := s.client.StopPipeline(project.Workspace, project.RepoSlug, ensureBraces(r.PathValue("uuid"))); err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, 200, map[string]string{"status": "stopped"})
}

func (s *Server) handleListVariables(w http.ResponseWriter, r *http.Request) {
	project, _, err := s.projectByID(r.PathValue("id"))
	if err != nil {
		writeError(w, err)
		return
	}

	vars, err := s.client.ListPipelineVariables(project.Workspace, project.RepoSlug)
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, 200, map[string]interface{}{
		"variables": vars.Values,
	})
}

// handleGetLogVariables parses pipeline variables from the first step's log for a given pipeline run.
func (s *Server) handleGetLogVariables(w http.ResponseWriter, r *http.Request) {
	project, _, err := s.projectByID(r.PathValue("id"))
	if err != nil {
		writeError(w, err)
		return
	}
	uuid := ensureBraces(r.PathValue("uuid"))

	stepsResult, err := s.client.ListPipelineSteps(project.Workspace, project.RepoSlug, uuid)
	if err != nil || len(stepsResult.Values) == 0 {
		if err != nil {
			writeError(w, err)
			return
		}
		writeJSON(w, 200, map[string]interface{}{"variables": []bitbucket.PipelineVariable{}})
		return
	}

	firstStep := stepsResult.Values[0]
	logContent, err := s.client.GetStepLog(project.Workspace, project.RepoSlug, uuid, firstStep.UUID)
	if err != nil {
		writeError(w, err)
		return
	}

	vars := parsePipelineVariablesFromLog(logContent)
	writeJSON(w, 200, map[string]interface{}{
		"variables": vars,
	})
}

func (s *Server) handleListRepositories(w http.ResponseWriter, r *http.Request) {
	workspace := r.PathValue("workspace")

	repos, err := s.client.ListAllRepositories(workspace)
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, 200, map[string]interface{}{
		"repositories": repos,
	})
}

// --- Log variable parsing ---

var varBlockRe = regexp.MustCompile(`(?m)^Pipeline variables:\n((?:[ \t]+\w[\w.]*:[ \t]+[^\n]*\n)*)`)

// parsePipelineVariablesFromLog extracts pipeline variables from a step log
// that contains a "Pipeline variables:" block.
func parsePipelineVariablesFromLog(logContent string) []bitbucket.PipelineVariable {
	m := varBlockRe.FindStringSubmatch(logContent)
	if m == nil {
		return nil
	}
	block := m[1]
	lines := strings.Split(block, "\n")
	var vars []bitbucket.PipelineVariable
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		idx := strings.Index(line, ":")
		if idx < 0 {
			continue
		}
		key := strings.TrimSpace(line[:idx])
		value := strings.TrimSpace(line[idx+1:])
		if key != "" {
			vars = append(vars, bitbucket.PipelineVariable{Key: key, Value: value})
		}
	}
	return vars
}