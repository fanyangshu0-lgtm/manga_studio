<script setup lang="ts">
import type { NodeDefinition } from '../types'

defineProps<{ catalog: NodeDefinition[] }>()
const emit = defineEmits<{ add: [definition: NodeDefinition] }>()
</script>

<template>
  <aside class="palette panel">
    <div class="panel-heading">
      <div><span class="eyebrow">NODE LIBRARY</span><h2>节点库</h2></div>
      <span class="count">{{ catalog.length }}</span>
    </div>
    <div v-for="category in [...new Set(catalog.map((item) => item.category))]" :key="category" class="palette-group">
      <h3>{{ category }}</h3>
      <button v-for="item in catalog.filter((node) => node.category === category)" :key="item.type" class="palette-item" type="button" @click="emit('add', item)">
        <i :style="{ background: item.color }">{{ item.label.slice(0, 1) }}</i>
        <span><strong>{{ item.label }}</strong><small>{{ item.description }}</small></span>
        <b>＋</b>
      </button>
    </div>
  </aside>
</template>

