<template>
  <div class="layout-shell">
    <div class="layout-card page-card">
      <aside v-if="!isCompactLayout" class="layout-aside">
        <div class="layout-brand">
          <h1>Caipu Admin</h1>
          <p>AI 观测与运行时配置控制台</p>
        </div>
        <div class="layout-aside__meta">稳重数据台 · 排障优先 · 单一浅色主题</div>
        <el-menu class="layout-menu" :default-active="route.path" router @select="handleMenuSelect">
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
          <div class="layout-topbar__heading">
            <el-button
              v-if="isCompactLayout"
              text
              class="layout-topbar__menu-trigger"
              @click="navDrawerVisible = true"
            >
              导航
            </el-button>
            <div class="topbar-meta">
              <strong>{{ pageTitle }}</strong>
              <div>{{ pageDescription }}</div>
            </div>
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

    <el-drawer
      v-model="navDrawerVisible"
      class="layout-drawer"
      direction="ltr"
      :size="drawerSize"
      :with-header="false"
    >
      <div class="layout-drawer__content">
        <div class="layout-brand">
          <h1>Caipu Admin</h1>
          <p>AI 观测与运行时配置控制台</p>
        </div>
        <div class="layout-aside__meta">稳重数据台 · 排障优先 · 单一浅色主题</div>
        <el-menu class="layout-menu" :default-active="route.path" router @select="handleMenuSelect">
          <el-menu-item v-for="item in navItems" :key="item.path" :index="item.path">
            <div class="layout-menu__label">
              <strong>{{ item.label }}</strong>
              <span>{{ item.description }}</span>
            </div>
          </el-menu-item>
        </el-menu>
      </div>
    </el-drawer>
  </div>
</template>

<script setup lang="ts">
import { computed, ref, watch } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import * as adminApi from '@/api/admin'
import { useResponsive } from '@/composables/useResponsive'
import { currentUsername, markLoggedOut } from '@/session'

const route = useRoute()
const router = useRouter()
const { isCompactLayout, isMobile } = useResponsive()
const navDrawerVisible = ref(false)

const navItems = [
  {
    path: '/dashboard',
    label: '概览',
    description: '全局健康度与失败信号'
  },
  {
    path: '/ai-providers',
    label: 'AI Provider',
    description: '多节点路由、熔断与草稿测试'
  },
  {
    path: '/server-health',
    label: '服务健康',
    description: '主机资源与核心链路状态'
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
const drawerSize = computed(() => (isMobile.value ? '100%' : '320px'))

watch(
  () => route.path,
  () => {
    navDrawerVisible.value = false
  }
)

function handleMenuSelect() {
  if (isCompactLayout.value) {
    navDrawerVisible.value = false
  }
}

async function handleLogout() {
  await adminApi.logout()
  markLoggedOut()
  navDrawerVisible.value = false
  await router.replace('/login')
}
</script>
