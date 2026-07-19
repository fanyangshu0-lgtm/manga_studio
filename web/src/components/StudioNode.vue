<script setup lang="ts">
import { Handle, Position } from '@vue-flow/core'
import type { NodeDefinition, NodeRun, StudioNode } from '../types'

defineProps<{
  id: string
  data: { node: StudioNode; definition: NodeDefinition; run?: NodeRun }
}>()
</script>

<template>
  <article class="studio-node" :class="data.run?.status" :style="{ '--node-color': data.definition.color }">
    <div class="node-accent" />
    <header>
      <span class="node-icon">{{ data.definition.label.slice(0, 1) }}</span>
      <div>
        <strong>{{ data.node.name }}</strong>
        <small>{{ data.definition.label }}</small>
      </div>
      <span v-if="data.run" class="node-status" :title="data.run.message">
        {{ data.run.status === 'running' ? `${data.run.progress}%` : data.run.status === 'succeeded' ? '✓' : data.run.status === 'failed' ? '!' : '·' }}
      </span>
    </header>
    <p>{{ data.definition.description }}</p>
    <div v-if="data.run?.status === 'running'" class="node-progress"><i :style="{ width: `${data.run.progress}%` }" /></div>

    <div class="ports inputs">
      <div v-for="(port, index) in data.definition.inputs" :key="port.id" class="port-row">
        <Handle :id="port.id" type="target" :position="Position.Left" :style="{ top: `${92 + index * 26}px` }" />
        <span>{{ port.label }}<em v-if="port.required">*</em></span>
        <small>{{ port.dataType }}</small>
      </div>
    </div>
    <div class="ports outputs">
      <div v-for="(port, index) in data.definition.outputs" :key="port.id" class="port-row output">
        <small>{{ port.dataType }}</small><span>{{ port.label }}</span>
        <Handle :id="port.id" type="source" :position="Position.Right" :style="{ top: `${92 + index * 26}px` }" />
      </div>
    </div>
  </article>
</template>

