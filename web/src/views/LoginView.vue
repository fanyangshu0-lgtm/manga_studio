<script setup lang="ts">
import { ref } from 'vue'
import { useStudioStore } from '../stores/studio'

const store = useStudioStore()
const username = ref('')
const password = ref('')
const submitting = ref(false)

async function submit(): Promise<void> {
  submitting.value = true
  try {
    await store.login(username.value, password.value)
  } catch (cause) {
    store.error = cause instanceof Error ? cause.message : '登录失败'
  } finally {
    submitting.value = false
  }
}
</script>

<template>
  <main class="login-screen">
    <form class="login-card" @submit.prevent="submit">
      <span class="eyebrow">MANGA DRAMA STUDIO</span>
      <h1>登录创作工作台</h1>
      <p>使用服务器环境变量中配置的管理员账号登录。</p>
      <label>管理员账号<input v-model="username" autocomplete="username" required autofocus></label>
      <label>管理员密码<input v-model="password" type="password" autocomplete="current-password" required></label>
      <p v-if="store.error" class="login-error">{{ store.error }}</p>
      <button class="primary-button large" type="submit" :disabled="submitting">
        {{ submitting ? '正在登录…' : '进入工作台' }}
      </button>
    </form>
  </main>
</template>

<style scoped>
.login-screen { min-height: 100vh; display: grid; place-items: center; padding: 24px; background: radial-gradient(circle at 20% 10%, #39231d, #111 52%); }
.login-card { width: min(440px, 100%); display: grid; gap: 18px; padding: 42px; color: #f7f0e7; background: rgba(23, 20, 18, .94); border: 1px solid rgba(255,255,255,.12); box-shadow: 0 24px 80px rgba(0,0,0,.45); }
.login-card h1, .login-card p { margin: 0; }
.login-card label { display: grid; gap: 8px; color: #d7c9bc; }
.login-card input { padding: 13px 14px; color: #fff; background: #0e0d0c; border: 1px solid #51443b; border-radius: 4px; }
.login-error { color: #ff9b86; }
</style>
