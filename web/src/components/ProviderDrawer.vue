<script setup lang="ts">
import { reactive, ref } from 'vue'
import type { Provider } from '../types'

defineProps<{ providers: Provider[] }>()
const emit = defineEmits<{
  close: []
  add: [input: { name: string; kind: string; baseUrl: string; token: string; capabilities: string[]; models: Provider['models']; weight: number }]
  toggle: [id: string, enabled: boolean]
  remove: [id: string]
  test: [id: string]
}>()
const showForm = ref(false)
const form = reactive({
  name: 'New API', kind: 'new-api', baseUrl: '', token: '', capabilities: ['script', 'video'], weight: 1,
  models: { script: 'deepseek-v4-pro', videoFast: 'doubao-seedance-2-0-fast-260128', videoQuality: 'doubao-seedance-2-0-260128', videoDefault: 'doubao-seedance-2-0-fast-260128' },
})

function submit(): void {
  emit('add', { ...form, capabilities: [...form.capabilities], models: { ...form.models } })
  form.name = ''; form.baseUrl = ''; form.token = ''; showForm.value = false
}
</script>

<template>
  <div class="drawer-mask" @click.self="emit('close')">
    <aside class="provider-drawer">
      <header><div><span class="eyebrow">PROVIDER VAULT</span><h2>模型渠道与 Token</h2><p>Token 加密保存，接口只展示末四位。</p></div><button class="icon-button" type="button" aria-label="关闭" @click="emit('close')">×</button></header>
      <div class="provider-list">
        <article v-for="item in providers" :key="item.id" class="provider-card">
          <i :class="{ disabled: !item.enabled }">{{ item.kind.includes('comfy') ? 'C' : 'AI' }}</i>
          <div><strong>{{ item.name }}</strong><small>{{ item.kind }} · {{ item.secretHint }} · 权重 {{ item.weight }}</small><span>{{ item.capabilities.join(' / ') }}</span></div>
          <small v-if="item.models?.script">编剧：{{ item.models.script }} · 视频：{{ item.models.videoDefault }}</small>
          <button class="switch" :class="{ on: item.enabled }" type="button" :aria-label="item.enabled ? '停用渠道' : '启用渠道'" @click="emit('toggle', item.id, !item.enabled)"><i /></button>
          <button class="ghost-button compact" type="button" @click="emit('test', item.id)">测试编剧</button>
          <button class="text-danger" type="button" @click="emit('remove', item.id)">删除</button>
        </article>
        <div v-if="providers.length === 0" class="provider-empty">尚未配置真实模型渠道。Mock 执行器仍可完整演示流程。</div>
      </div>
      <form v-if="showForm" class="provider-form" @submit.prevent="submit">
        <label>渠道名称<input v-model="form.name" required placeholder="例如：主力 LLM" /></label>
        <label>类型<select v-model="form.kind"><option value="new-api">New API</option><option value="openai-compatible">OpenAI Compatible</option></select></label>
        <label>Base URL<input v-model="form.baseUrl" placeholder="https://api.example.com/v1" /></label>
        <label>Token<input v-model="form.token" required type="password" autocomplete="new-password" /></label>
        <label>DeepSeek 编剧模型<input v-model="form.models.script" /></label>
        <label>Seedance 默认模型<select v-model="form.models.videoDefault"><option :value="form.models.videoFast">Seedance 2.0 Fast</option><option :value="form.models.videoQuality">Seedance 2.0</option></select></label>
        <label>权重<input v-model.number="form.weight" type="number" min="1" max="100" /></label>
        <div class="form-actions"><button class="ghost-button" type="button" @click="showForm = false">取消</button><button class="primary-button" type="submit">安全保存</button></div>
      </form>
      <button v-else class="add-provider" type="button" @click="showForm = true">＋ 添加模型渠道</button>
      <footer>本地 MVP 尚未包含登录，不要直接暴露到公网。</footer>
    </aside>
  </div>
</template>

