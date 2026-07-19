<script setup lang="ts">
import { computed } from 'vue'
import { api } from '../api/client'
import type { Asset, Run } from '../types'

const props = defineProps<{ run: Run | null; assets: Asset[] }>()
const emit = defineEmits<{ run: []; cancel: []; resume: [] }>()
const finalAsset = computed(() => props.assets.at(-1))
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
      <a v-if="run.status === 'succeeded' && finalAsset" class="primary-button compact" :href="api.assetUrl(finalAsset.id)" target="_blank">查看成片</a>
      <button v-if="run.status === 'running' || run.status === 'queued'" class="ghost-button compact" type="button" @click="emit('cancel')">停止</button>
      <button v-else-if="run.status === 'failed' || run.status === 'cancelled' || run.status === 'interrupted'" class="primary-button compact" type="button" @click="emit('resume')">恢复任务</button>
      <button v-else class="ghost-button compact" type="button" @click="emit('run')">再次运行</button>
    </template>
    <template v-else>
      <div class="run-copy"><span class="run-orb ready" /><div><strong>工作流已就绪</strong><small>保存后运行全部节点</small></div></div>
      <button class="primary-button" type="button" @click="emit('run')"><span>▶</span> 运行工作流</button>
    </template>
  </section>
</template>
