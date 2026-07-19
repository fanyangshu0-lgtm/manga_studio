<script setup lang="ts">
import { onMounted, ref } from 'vue'
import { useStudioStore } from './stores/studio'
import WorkspaceView from './views/WorkspaceView.vue'
import LoginView from './views/LoginView.vue'
import ProviderDrawer from './components/ProviderDrawer.vue'

const store = useStudioStore()
const showProviders = ref(false)
const showCreate = ref(false)
const newName = ref('')
const newDescription = ref('')

onMounted(() => { void store.initialize() })

async function createProject(): Promise<void> {
  await store.createProject(newName.value, newDescription.value)
  showCreate.value = false; newName.value = ''; newDescription.value = ''
}
</script>

<template>
	<LoginView v-if="store.authReady && !store.authenticated" />
	<div v-else-if="store.authReady" class="app-shell">
    <header class="app-header">
      <a class="brand" href="#" aria-label="绘界首页"><i>绘</i><span><strong>绘界</strong><small>MANGA DRAMA STUDIO</small></span></a>
      <div class="project-switcher">
        <span>当前项目</span>
        <select :value="store.activeProjectId" :disabled="store.projects.length === 0" @change="store.openProject(($event.target as HTMLSelectElement).value)">
          <option v-if="store.projects.length === 0" value="">暂无项目</option>
          <option v-for="item in store.projects" :key="item.id" :value="item.id">{{ item.name }}</option>
        </select>
        <button class="icon-button" type="button" aria-label="新建项目" @click="showCreate = true">＋</button>
      </div>
      <nav>
        <button type="button" class="nav-link active">工作台</button>
        <button type="button" class="nav-link" @click="showProviders = true">模型渠道 <span>{{ store.providers.filter((item) => item.enabled).length }}</span></button>
        <a class="nav-link" href="/healthz" target="_blank">服务状态</a>
      </nav>
		<div class="header-badge"><span /> {{ store.username }}</div>
		<button class="nav-link" type="button" @click="store.logout">退出</button>
    </header>

    <div v-if="store.error" class="global-error"><span>{{ store.error }}</span><button type="button" @click="store.error = ''">×</button></div>
    <div v-if="store.busy" class="loading-screen"><i /><span>正在装载创作空间…</span></div>
    <WorkspaceView v-else-if="store.workflow" />
    <section v-else class="welcome">
      <div class="welcome-art"><i>01</i><i>02</i><i>03</i><span>绘</span></div>
      <span class="eyebrow">VISUAL STORY PIPELINE</span>
      <h1>把故事，连接成一部漫剧</h1>
      <p>从剧本、角色、分镜到配音与成片，用节点工作流掌控每一步生成。</p>
      <button class="primary-button large" type="button" @click="showCreate = true">创建第一个项目</button>
    </section>

    <div v-if="showCreate" class="modal-mask" @click.self="showCreate = false">
      <form class="create-modal" @submit.prevent="createProject">
        <button class="icon-button close" type="button" @click="showCreate = false">×</button>
        <span class="eyebrow">NEW PRODUCTION</span><h2>开始一部新漫剧</h2><p>我们会为你创建一套可运行的示例工作流。</p>
        <label>项目名称<input v-model="newName" required autofocus placeholder="例如：赛博长安夜话" /></label>
        <label>一句话设定<textarea v-model="newDescription" placeholder="主角、世界观与核心冲突" /></label>
        <div class="form-actions"><button class="ghost-button" type="button" @click="showCreate = false">取消</button><button class="primary-button" type="submit">创建并进入工作台</button></div>
      </form>
    </div>
    <ProviderDrawer v-if="showProviders" :providers="store.providers" @close="showProviders = false" @add="store.addProvider" @toggle="store.toggleProvider" @remove="store.deleteProvider" @test="store.testProvider" />
	</div>
	<div v-else class="loading-screen"><i /><span>正在检查登录状态…</span></div>
</template>

