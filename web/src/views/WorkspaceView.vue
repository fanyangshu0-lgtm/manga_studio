<script setup lang="ts">
import { computed, ref, watch } from 'vue'
import { MarkerType, VueFlow, type Connection } from '@vue-flow/core'
import { useStudioStore } from '../stores/studio'
import type { NodeDefinition, NodeRun, Position, StudioNode } from '../types'
import StudioNodeCard from '../components/StudioNode.vue'
import NodePalette from '../components/NodePalette.vue'
import InspectorPanel from '../components/InspectorPanel.vue'
import RunDock from '../components/RunDock.vue'

const store = useStudioStore()
interface FlowNode { id: string; type: string; position: Position; data: { node: StudioNode; definition: NodeDefinition; run?: NodeRun } }
interface FlowEdge { id: string; source: string; target: string; sourceHandle?: string | null; targetHandle?: string | null; markerEnd?: MarkerType; animated?: boolean }
const graphNodes = ref<FlowNode[]>([])
const graphEdges = ref<FlowEdge[]>([])
const connectionError = ref('')
let syncing = false

watch(() => store.workflow, (graph) => {
  if (!graph || syncing) return
  graphNodes.value = graph.nodes.map((node) => ({ id: node.id, type: 'studio', position: { ...node.position }, data: nodeData(node) }))
  graphEdges.value = graph.edges.map((edge) => ({ ...edge, markerEnd: MarkerType.ArrowClosed, animated: store.run?.status === 'running' }))
}, { immediate: true, deep: true })

watch(() => store.run, () => {
  graphNodes.value = graphNodes.value.map((node) => ({ ...node, data: nodeData((node.data as { node: StudioNode }).node) }))
  graphEdges.value = graphEdges.value.map((edge) => ({ ...edge, animated: store.run?.status === 'running' }))
}, { deep: true })

const selectedDefinition = computed(() => store.selectedNode ? store.definitionByType.get(store.selectedNode.type) : undefined)

function nodeData(node: StudioNode): FlowNode['data'] {
  const definition = store.definitionByType.get(node.type)
  if (!definition) throw new Error(`未知节点类型: ${node.type}`)
  return { node, definition, run: store.run?.nodes[node.id] }
}

function addNode(definition: NodeDefinition): void {
  if (!store.workflow) return
  const id = `${definition.type}-${crypto.randomUUID().slice(0, 8)}`
  const node: StudioNode = { id, type: definition.type, name: definition.label, position: { x: 260 + graphNodes.value.length * 42, y: 180 + (graphNodes.value.length % 5) * 80 }, config: {} }
  graphNodes.value.push({ id, type: 'studio', position: node.position, data: nodeData(node) })
  store.workflow.nodes.push(node)
  store.selectedNodeId = id
}

function connect(connection: Connection): void {
  if (!connection.source || !connection.target || !connection.sourceHandle || !connection.targetHandle) return
  const sourceNode = store.workflow?.nodes.find((item) => item.id === connection.source)
  const targetNode = store.workflow?.nodes.find((item) => item.id === connection.target)
  const sourceDef = sourceNode ? store.definitionByType.get(sourceNode.type) : undefined
  const targetDef = targetNode ? store.definitionByType.get(targetNode.type) : undefined
  const output = sourceDef?.outputs.find((item) => item.id === connection.sourceHandle)
  const input = targetDef?.inputs.find((item) => item.id === connection.targetHandle)
  if (!output || !input || output.dataType !== input.dataType) { connectionError.value = `端口类型不兼容：${output?.dataType ?? '?'} → ${input?.dataType ?? '?'}`; window.setTimeout(() => { connectionError.value = '' }, 2800); return }
  const id = `edge-${crypto.randomUUID().slice(0, 8)}`
  graphEdges.value.push({ id, ...connection, markerEnd: MarkerType.ArrowClosed })
  commitGraph()
}

function selectNode(event: { node: { id: string } }): void { store.selectedNodeId = event.node.id }

function updateNode(node: StudioNode): void {
  store.updateNode(node)
  const graphNode = graphNodes.value.find((item) => item.id === node.id)
  if (graphNode) graphNode.data = nodeData(node)
}

function removeNode(nodeId: string): void {
  if (!store.workflow) return
  graphNodes.value = graphNodes.value.filter((item) => item.id !== nodeId)
  graphEdges.value = graphEdges.value.filter((edge) => edge.source !== nodeId && edge.target !== nodeId)
  store.selectedNodeId = ''
  commitGraph()
}

function commitGraph(): void {
  if (!store.workflow) return
  syncing = true
  const nodes = graphNodes.value.map((item) => {
    const studioNode = (item.data as { node: StudioNode }).node
    return { ...studioNode, position: { x: item.position.x, y: item.position.y } }
  })
  const edges = graphEdges.value.map((edge) => ({ id: edge.id, source: edge.source, sourceHandle: edge.sourceHandle ?? '', target: edge.target, targetHandle: edge.targetHandle ?? '' }))
  store.replaceGraph(nodes, edges)
  syncing = false
}

async function save(): Promise<void> { commitGraph(); await store.save() }
async function run(): Promise<void> { commitGraph(); await store.startRun() }
</script>

<template>
  <main class="workspace">
    <NodePalette :catalog="store.catalog" @add="addNode" />
    <section class="canvas-shell">
      <div class="canvas-toolbar">
        <div><span class="live-dot" />工作流画布 <small>REV {{ store.workflow?.revision ?? 0 }}</small></div>
        <div><button class="ghost-button compact" type="button" :disabled="store.saving" @click="save">{{ store.saving ? '保存中…' : '保存工作流' }}</button></div>
      </div>
      <VueFlow v-model:nodes="graphNodes" v-model:edges="graphEdges" class="studio-flow" :min-zoom="0.25" :max-zoom="1.8" :default-viewport="store.workflow?.viewport" :delete-key-code="'Delete'" :snap-to-grid="true" :snap-grid="[16, 16]" @connect="connect" @node-click="selectNode" @pane-click="store.selectedNodeId = ''" @node-drag-stop="commitGraph" @edges-change="commitGraph">
        <template #node-studio="slotProps"><StudioNodeCard v-bind="slotProps" /></template>
        <template #connection-line><div /></template>
      </VueFlow>
      <div class="canvas-hint"><kbd>滚轮</kbd> 缩放 <kbd>拖拽</kbd> 平移 <kbd>Delete</kbd> 删除</div>
      <div v-if="connectionError" class="toast-error">{{ connectionError }}</div>
      <RunDock :run="store.run" @run="run" @cancel="store.cancelRun" />
    </section>
    <InspectorPanel :node="store.selectedNode" :definition="selectedDefinition" @update="updateNode" @remove="removeNode" />
  </main>
</template>
