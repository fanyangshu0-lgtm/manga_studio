export interface Position { x: number; y: number }
export interface Viewport extends Position { zoom: number }

export interface StudioNode {
  id: string
  type: string
  name: string
  position: Position
  config: Record<string, unknown>
}

export interface StudioEdge {
  id: string
  source: string
  sourceHandle: string
  target: string
  targetHandle: string
}

export interface Workflow {
  id: string
  projectId: string
  revision: number
  nodes: StudioNode[]
  edges: StudioEdge[]
  viewport: Viewport
  updatedAt: string
}

export interface Project {
  id: string
  name: string
  description: string
  createdAt: string
  updatedAt: string
}

export interface PortDefinition {
  id: string
  label: string
  dataType: string
  required?: boolean
}

export interface NodeDefinition {
  type: string
  label: string
  description: string
  category: string
  color: string
  inputs: PortDefinition[]
  outputs: PortDefinition[]
}

export type RunStatus = 'queued' | 'running' | 'succeeded' | 'failed' | 'cancelled' | 'interrupted'

export interface NodeRun {
  nodeId: string
  nodeName: string
  status: RunStatus
  progress: number
  message: string
  outputs?: Record<string, unknown>
}

export interface Run {
  id: string
  projectId: string
  status: RunStatus
  progress: number
  message: string
  nodes: Record<string, NodeRun>
  createdAt: string
}

export interface Asset {
  id: string
  projectId: string
  runId: string
  nodeId: string
  kind: string
  mime: string
  size: number
  sha256: string
  createdAt: string
}

export interface RunEvent {
  type: string
  runId: string
  nodeId?: string
  status: RunStatus
  progress: number
  message: string
  timestamp: string
}

export interface Provider {
  id: string
  name: string
  kind: string
  baseUrl: string
  capabilities: string[]
  models: { script: string; videoFast: string; videoQuality: string; videoDefault: string }
  weight: number
  enabled: boolean
  secretHint: string
}

export interface ApiFailure {
  error: { code: string; message: string; details?: Array<{ message: string; nodeId?: string }> }
}

