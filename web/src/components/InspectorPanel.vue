<script setup lang="ts">
import { computed } from 'vue'
import type { NodeDefinition, StudioNode } from '../types'

const props = defineProps<{ node: StudioNode | null; definition?: NodeDefinition }>()
const emit = defineEmits<{ update: [node: StudioNode]; remove: [nodeId: string] }>()

const prompt = computed({
  get: () => String(props.node?.config.prompt ?? ''),
  set: (value: string) => updateConfig('prompt', value),
})
const ratio = computed({
  get: () => String(props.node?.config.ratio ?? '9:16'),
  set: (value: string) => updateConfig('ratio', value),
})
const quality = computed({
  get: () => String(props.node?.config.quality ?? 'fast'),
  set: (value: string) => updateConfig('quality', value),
})

function updateName(value: string): void {
  if (props.node) emit('update', { ...props.node, name: value.trim() || props.definition?.label || props.node.name })
}

function updateConfig(key: string, value: unknown): void {
  if (props.node) emit('update', { ...props.node, config: { ...props.node.config, [key]: value } })
}
</script>

<template>
  <aside class="inspector panel">
    <div class="panel-heading"><div><span class="eyebrow">INSPECTOR</span><h2>节点设置</h2></div></div>
    <div v-if="node" class="inspector-body">
      <div class="node-type-pill" :style="{ '--pill': definition?.color }">{{ definition?.label }}</div>
      <label>节点名称<input :value="node.name" type="text" @change="updateName(($event.target as HTMLInputElement).value)"></label>
      <label v-if="node.type === 'story-input'">故事创意<textarea v-model="prompt" rows="8" placeholder="主角、世界观、核心冲突和期望结局…" /></label>
      <template v-else-if="node.type === 'storyboard'">
        <label>画面比例<select v-model="ratio"><option value="9:16">9:16 竖屏</option><option value="16:9">16:9 横屏</option><option value="1:1">1:1 方形</option></select></label>
        <p class="field-help">DeepSeek 会自动规划 1–12 个连续分镜，每个镜头 1–15 秒。</p>
      </template>
      <template v-else-if="node.type === 'image'">
        <label>Seedance 模式<select v-model="quality"><option value="fast">2.0 Fast（默认，速度优先）</option><option value="quality">2.0（质量优先）</option></select></label>
        <p class="field-help">每个分镜都会生成带原生音频的 720p 视频，默认并发数为 2。</p>
      </template>
      <p v-else-if="node.type === 'script'">使用 DeepSeek V4 Pro 生成结构化中文剧本。</p>
      <p v-else-if="node.type === 'character'">根据剧本生成跨镜头一致的角色外观档案。</p>
      <p v-else-if="node.type === 'compose'">使用 FFmpeg 拼接全部镜头、补齐音轨并烧录中文字幕。</p>
      <div class="port-summary">
        <h3>端口</h3>
        <div v-for="port in definition?.inputs" :key="`i-${port.id}`"><i class="in" /> 输入 · {{ port.label }} <small>{{ port.dataType }}</small></div>
        <div v-for="port in definition?.outputs" :key="`o-${port.id}`"><i class="out" /> 输出 · {{ port.label }} <small>{{ port.dataType }}</small></div>
      </div>
      <button class="danger-button" type="button" @click="emit('remove', node.id)">删除节点</button>
    </div>
    <div v-else class="empty-inspector"><b>◎</b><p>选择画布中的节点<br>查看和修改生成参数</p></div>
  </aside>
</template>
