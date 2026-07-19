<script setup lang="ts">
import { ref, watch } from 'vue'
import type { NodeDefinition, StudioNode } from '../types'

const props = defineProps<{ node: StudioNode | null; definition?: NodeDefinition }>()
const emit = defineEmits<{ update: [node: StudioNode]; remove: [nodeId: string] }>()
const name = ref('')
const configText = ref('{}')
const configError = ref('')

watch(() => props.node, (node) => {
  name.value = node?.name ?? ''
  configText.value = JSON.stringify(node?.config ?? {}, null, 2)
  configError.value = ''
}, { immediate: true, deep: true })

function apply(): void {
  if (!props.node) return
  try {
    const config = JSON.parse(configText.value) as Record<string, unknown>
    configError.value = ''
    emit('update', { ...props.node, name: name.value.trim() || props.definition?.label || props.node.name, config })
  } catch { configError.value = '配置必须是有效 JSON' }
}
</script>

<template>
  <aside class="inspector panel">
    <div class="panel-heading"><div><span class="eyebrow">INSPECTOR</span><h2>节点设置</h2></div></div>
    <div v-if="node" class="inspector-body">
      <div class="node-type-pill" :style="{ '--pill': definition?.color }">{{ definition?.label }}</div>
      <label>节点名称<input v-model="name" type="text" @change="apply" /></label>
      <label>节点 ID<input :value="node.id" type="text" disabled /></label>
      <label>参数配置 <small>JSON</small><textarea v-model="configText" spellcheck="false" @blur="apply" /></label>
      <p v-if="configError" class="field-error">{{ configError }}</p>
      <div class="port-summary">
        <h3>端口</h3>
        <div v-for="port in definition?.inputs" :key="`i-${port.id}`"><i class="in" /> 输入 · {{ port.label }} <small>{{ port.dataType }}</small></div>
        <div v-for="port in definition?.outputs" :key="`o-${port.id}`"><i class="out" /> 输出 · {{ port.label }} <small>{{ port.dataType }}</small></div>
      </div>
      <button class="danger-button" type="button" @click="emit('remove', node.id)">删除节点</button>
    </div>
    <div v-else class="empty-inspector"><b>⌁</b><p>选择画布中的节点<br />以编辑名称和参数</p></div>
  </aside>
</template>

