import { computed, ref } from 'vue'
import { defineStore } from 'pinia'
import { api } from '../api/client'
import type { NodeDefinition, Project, Provider, Run, RunEvent, StudioNode, Workflow } from '../types'

export const useStudioStore = defineStore('studio', () => {
  const projects = ref<Project[]>([])
  const activeProjectId = ref('')
  const workflow = ref<Workflow | null>(null)
  const catalog = ref<NodeDefinition[]>([])
  const providers = ref<Provider[]>([])
  const run = ref<Run | null>(null)
  const selectedNodeId = ref('')
  const busy = ref(false)
  const saving = ref(false)
  const error = ref('')
  let eventSource: EventSource | null = null

  const activeProject = computed(() => projects.value.find((item) => item.id === activeProjectId.value) ?? null)
  const selectedNode = computed(() => workflow.value?.nodes.find((item) => item.id === selectedNodeId.value) ?? null)
  const definitionByType = computed(() => new Map(catalog.value.map((item) => [item.type, item])))

  async function bootstrap(): Promise<void> {
    busy.value = true
    error.value = ''
    try {
      const [projectItems, definitions, providerItems] = await Promise.all([api.projects(), api.nodeCatalog(), api.providers()])
      projects.value = projectItems
      catalog.value = definitions
      providers.value = providerItems
      if (projectItems[0]) await openProject(projectItems[0].id)
    } catch (cause) { error.value = messageOf(cause) } finally { busy.value = false }
  }

  async function createProject(name: string, description: string): Promise<void> {
    busy.value = true
    try {
      const created = await api.createProject({ name, description })
      projects.value.unshift(created.project)
      activeProjectId.value = created.project.id
      workflow.value = created.workflow
      selectedNodeId.value = created.workflow.nodes[0]?.id ?? ''
    } catch (cause) { error.value = messageOf(cause); throw cause } finally { busy.value = false }
  }

  async function openProject(projectId: string): Promise<void> {
    activeProjectId.value = projectId
    workflow.value = await api.workflow(projectId)
    selectedNodeId.value = workflow.value.nodes[0]?.id ?? ''
    run.value = null
    closeEvents()
  }

  function replaceGraph(nodes: StudioNode[], edges: Workflow['edges'], viewport?: Workflow['viewport']): void {
    if (!workflow.value) return
    workflow.value.nodes = nodes
    workflow.value.edges = edges
    if (viewport) workflow.value.viewport = viewport
  }

  function updateNode(node: StudioNode): void {
    if (!workflow.value) return
    const index = workflow.value.nodes.findIndex((item) => item.id === node.id)
    if (index >= 0) workflow.value.nodes[index] = node
  }

  async function save(): Promise<void> {
    if (!workflow.value || !activeProjectId.value) return
    saving.value = true
    error.value = ''
    try { workflow.value = await api.saveWorkflow(activeProjectId.value, workflow.value) }
    catch (cause) { error.value = messageOf(cause); throw cause }
    finally { saving.value = false }
  }

  async function startRun(): Promise<void> {
    if (!activeProjectId.value) return
    await save()
    run.value = await api.createRun(activeProjectId.value)
    subscribe(run.value.id)
  }

  async function cancelRun(): Promise<void> { if (run.value) await api.cancelRun(run.value.id) }

  function subscribe(runId: string): void {
    closeEvents()
    eventSource = new EventSource(api.runEventsUrl(runId))
    const consume = async (event: MessageEvent<string>): Promise<void> => {
      const update = JSON.parse(event.data) as RunEvent
      if (run.value) { run.value.status = update.status; run.value.progress = update.progress; run.value.message = update.message }
      run.value = await api.run(runId)
      if (update.type === 'run' && ['succeeded', 'failed', 'cancelled'].includes(update.status)) closeEvents()
    }
    for (const type of ['snapshot', 'run', 'node']) eventSource.addEventListener(type, (event) => { void consume(event as MessageEvent<string>) })
    eventSource.onerror = () => { if (run.value && !['succeeded', 'failed', 'cancelled'].includes(run.value.status)) error.value = '实时进度连接已断开，可刷新任务状态' }
  }

  function closeEvents(): void { eventSource?.close(); eventSource = null }

  async function refreshProviders(): Promise<void> { providers.value = await api.providers() }
  async function addProvider(input: Parameters<typeof api.createProvider>[0]): Promise<void> { await api.createProvider(input); await refreshProviders() }
  async function toggleProvider(id: string, enabled: boolean): Promise<void> { await api.toggleProvider(id, enabled); await refreshProviders() }
  async function deleteProvider(id: string): Promise<void> { await api.deleteProvider(id); await refreshProviders() }

  return { projects, activeProjectId, activeProject, workflow, catalog, providers, run, selectedNodeId, selectedNode, definitionByType, busy, saving, error, bootstrap, createProject, openProject, replaceGraph, updateNode, save, startRun, cancelRun, refreshProviders, addProvider, toggleProvider, deleteProvider }
})

function messageOf(cause: unknown): string { return cause instanceof Error ? cause.message : '发生未知错误' }
