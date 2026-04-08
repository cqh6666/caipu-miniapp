<template>
  <div class="layout-shell">
    <div class="layout-card page-card">
      <aside class="layout-aside">
        <div class="layout-brand">
          <h1>Caipu Admin</h1>
          <p>AI 可观测性与动态配置中心</p>
        </div>
        <el-menu :default-active="route.path" router>
          <el-menu-item index="/dashboard">概览</el-menu-item>
          <el-menu-item index="/ai-jobs">AI 任务</el-menu-item>
          <el-menu-item index="/ai-calls">API 调用</el-menu-item>
          <el-menu-item index="/settings">配置中心</el-menu-item>
        </el-menu>
      </aside>
      <main class="layout-main">
        <div class="layout-topbar">
          <div class="topbar-meta">
            <strong>{{ pageTitle }}</strong>
            <div>统一查看任务成功率、调用明细与在线配置。</div>
          </div>
          <div style="display: flex; align-items: center; gap: 12px">
            <el-tag type="info">{{ username }}</el-tag>
            <el-button text @click="handleLogout">退出登录</el-button>
          </div>
        </div>
        <slot />
      </main>
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import * as adminApi from '@/api/admin'
import { currentUsername, markLoggedOut } from '@/session'

const route = useRoute()
const router = useRouter()

const username = computed(() => currentUsername.value || 'admin')
const pageTitle = computed(() => {
  switch (route.path) {
    case '/ai-jobs':
      return 'AI 任务'
    case '/ai-calls':
      return 'API 调用'
    case '/settings':
      return '配置中心'
    default:
      return '概览'
  }
})

async function handleLogout() {
  await adminApi.logout()
  markLoggedOut()
  await router.replace('/login')
}
</script>
