<template>
  <div class="layout-shell">
    <div class="layout-card page-card">
      <aside class="layout-aside">
        <div class="layout-brand">
          <h1>Caipu Admin</h1>
          <p>AI 观测与运行时配置控制台</p>
        </div>
        <div class="layout-aside__meta">稳重数据台 · 排障优先 · 单一浅色主题</div>
        <el-menu class="layout-menu" :default-active="route.path" router>
          <el-menu-item v-for="item in navItems" :key="item.path" :index="item.path">
            <div class="layout-menu__label">
              <strong>{{ item.label }}</strong>
              <span>{{ item.description }}</span>
            </div>
          </el-menu-item>
        </el-menu>
      </aside>
      <main class="layout-main">
        <div class="layout-topbar">
          <div class="topbar-meta">
            <strong>{{ pageTitle }}</strong>
            <div>{{ pageDescription }}</div>
          </div>
          <div class="layout-topbar__actions">
            <div class="layout-topbar__identity">
              <span>当前账号</span>
              <strong>{{ username }}</strong>
            </div>
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

const navItems = [
  {
    path: '/dashboard',
    label: '概览',
    description: '全局健康度与失败信号'
  },
  {
    path: '/ai-jobs',
    label: 'AI 任务',
    description: '任务结果与链路回溯'
  },
  {
    path: '/ai-calls',
    label: 'API 调用',
    description: 'Provider 调用明细'
  },
  {
    path: '/settings',
    label: '配置中心',
    description: '运行时配置与审计'
  }
]

const username = computed(() => currentUsername.value || 'admin')
const pageTitle = computed(() => String(route.meta.title || 'Caipu Admin'))
const pageDescription = computed(() => String(route.meta.description || '统一查看任务成功率、调用明细与在线配置。'))

async function handleLogout() {
  await adminApi.logout()
  markLoggedOut()
  await router.replace('/login')
}
</script>
