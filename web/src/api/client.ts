import type { NodeDefinition, Project, Provider, Run, Workflow } from '../types'

const base = import.meta.env.VITE_API_BASE ?? '/api/v1'

async function request<T>(path: string, init?: RequestInit): Promise<T> {
  const response = await fetch(`${base}${path}`, {
	...init,
	credentials: 'same-origin',
    headers: { 'Content-Type': 'application/json', ...init?.headers },
  })
  if (!response.ok) {
    const payload = await response.json().catch(() => ({ error: { message: response.statusText } })) as { error?: { message?: string } }
    throw new Error(payload.error?.message ?? `请求失败 (${response.status})`)
  }
  if (response.status === 204) return undefined as T
  return response.json() as Promise<T>
}

export const api = {
	session: () => request<{ authenticated: boolean; username: string }>('/auth/session'),
	login: (username: string, password: string) => request<{ authenticated: boolean; username: string }>('/auth/login', { method: 'POST', body: JSON.stringify({ username, password }) }),
	logout: () => request<{ authenticated: boolean }>('/auth/logout', { method: 'POST' }),
  nodeCatalog: () => request<NodeDefinition[]>('/catalog/nodes'),
  projects: () => request<Project[]>('/projects'),
  createProject: (input: { name: string; description: string }) => request<{ project: Project; workflow: Workflow }>('/projects', { method: 'POST', body: JSON.stringify(input) }),
  workflow: (projectId: string) => request<Workflow>(`/projects/${projectId}/workflow`),
  saveWorkflow: (projectId: string, workflow: Workflow) => request<Workflow>(`/projects/${projectId}/workflow`, { method: 'PUT', body: JSON.stringify(workflow) }),
  createRun: (projectId: string) => request<Run>('/runs', { method: 'POST', body: JSON.stringify({ projectId }) }),
  run: (runId: string) => request<Run>(`/runs/${runId}`),
  cancelRun: (runId: string) => request<void>(`/runs/${runId}/cancel`, { method: 'POST' }),
  runEventsUrl: (runId: string) => `${base}/runs/${runId}/events`,
  providers: () => request<Provider[]>('/providers'),
  createProvider: (input: { name: string; kind: string; baseUrl: string; token: string; capabilities: string[]; weight: number }) => request<Provider>('/providers', { method: 'POST', body: JSON.stringify(input) }),
  updateProvider: (id: string, input: Partial<Omit<Provider, 'id' | 'secretHint'>> & { token?: string }) => request<Provider>(`/providers/${id}`, { method: 'PUT', body: JSON.stringify(input) }),
  toggleProvider: (id: string, enabled: boolean) => request<Provider>(`/providers/${id}`, { method: 'PATCH', body: JSON.stringify({ enabled }) }),
  deleteProvider: (id: string) => request<void>(`/providers/${id}`, { method: 'DELETE' }),
}

