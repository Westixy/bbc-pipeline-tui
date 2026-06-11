/**
 * API client for the BBC Pipeline Manager backend.
 * All API calls go to relative /api/... endpoints.
 */

const BASE = '/api';

async function request(url, opts = {}) {
  const res = await fetch(`${BASE}${url}`, {
    ...opts,
    headers: { 'Content-Type': 'application/json', ...opts.headers },
  });
  if (!res.ok) {
    const data = await res.json().catch(() => ({ message: res.statusText }));
    throw new Error(data.message || `HTTP ${res.status}`);
  }
  return res.json();
}

// --- Projects ---
export function listProjects() {
  return request('/projects');
}

export function addProject(workspace, repoSlug) {
  return request('/projects', {
    method: 'POST',
    body: JSON.stringify({ workspace, repo_slug: repoSlug }),
  });
}

export function removeProject(id) {
  return request(`/projects/${id}`, { method: 'DELETE' });
}

// --- Pipelines ---
export function listPipelines(projectId, params = {}) {
  // Strip empty/falsy values to avoid sending e.g. filter=&sort=-created_on
  const clean = {};
  for (const [k, v] of Object.entries(params)) {
    if (v) clean[k] = v;
  }
  const q = new URLSearchParams(clean).toString();
  return request(`/projects/${projectId}/pipelines${q ? '?' + q : ''}`);
}

function cleanUuid(uuid) {
  return uuid ? uuid.replace(/^\{|\}$/g, '') : uuid;
}

export function getPipeline(projectId, uuid) {
  return request(`/projects/${projectId}/pipelines/${cleanUuid(uuid)}`);
}

export function listSteps(projectId, pipelineUuid) {
  return request(`/projects/${projectId}/pipelines/${cleanUuid(pipelineUuid)}/steps`);
}

export function getStepLog(projectId, pipelineUuid, stepUuid) {
  return request(`/projects/${projectId}/pipelines/${cleanUuid(pipelineUuid)}/steps/${cleanUuid(stepUuid)}/log`);
}

export function triggerPipeline(projectId, target, variables = []) {
  return request(`/projects/${projectId}/pipelines`, {
    method: 'POST',
    body: JSON.stringify({ target, variables }),
  });
}

export function stopPipeline(projectId, uuid) {
  return request(`/projects/${projectId}/pipelines/${cleanUuid(uuid)}/stop`, { method: 'POST' });
}

// --- Variables ---
export function listVariables(projectId) {
  return request(`/projects/${projectId}/variables`);
}

export function getLogVariables(projectId, pipelineUuid) {
  return request(`/projects/${projectId}/pipelines/${cleanUuid(pipelineUuid)}/log-vars`);
}

// --- Repositories ---
export function listRepositories(workspace) {
  return request(`/repositories/${workspace}`);
}