<script setup lang="ts">
import type { Run } from '../types'

defineProps<{ run: Run | null }>()
const emit = defineEmits<{ run: []; cancel: [] }>()
</script>

<template>
  <section class="run-dock">
    <template v-if="run">
      <div class="run-copy">
        <span class="run-orb" :class="run.status" />
        <div><strong>{{ run.message }}</strong><small>RUN · {{ run.id.slice(-8) }}</small></div>
      </div>
      <div class="overall-progress"><i :style="{ width: `${run.progress}%` }" /></div>
      <b>{{ run.progress }}%</b>
      <button v-if="run.status === 'running' || run.status === 'queued'" class="ghost-button compact" type="button" @click="emit('cancel')">停止</button>
      <button v-else class="primary-button compact" type="button" @click="emit('run')">再次运行</button>
    </template>
    <template v-else>
      <div class="run-copy"><span class="run-orb ready" /><div><strong>工作流已就绪</strong><small>保存后运行全部节点</small></div></div>
      <button class="primary-button" type="button" @click="emit('run')"><span>▶</span> 运行工作流</button>
    </template>
  </section>
</template>

