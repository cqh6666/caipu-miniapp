<template>
  <div class="layout-shell">
    <div class="layout-card page-card">
      <aside v-if="!isCompactLayout" class="layout-aside">
        <div class="layout-brand">
          <div class="layout-brand__logo" aria-hidden="true">
            <el-icon><Monitor /></el-icon>
          </div>
          <div class="layout-brand__text">
            <h1>Caipu Admin</h1>
            <p>AI 观测与运行时配置</p>
          </div>
        </div>

        <el-menu class="layout-menu" :default-active="route.path" router @select="handleMenuSelect">
          <el-menu-item v-for="item in navItems" :key="item.path" :index="item.path">
            <el-icon class="layout-menu__icon">
              <component :is="item.icon" />
            </el-icon>
            <div class="layout-menu__label">
              <strong>{{ item.label }}</strong>
              <span>{{ item.description }}</span>
            </div>
          </el-menu-item>
        </el-menu>

        <RouterLink
          to="/server-health"
          class="layout-aside__footer"
          :class="`layout-aside__footer--${backendHealthTone}`"
          :title="backendHealthTitle"
        >
          <span
            class="layout-aside__env-dot"
            :class="`layout-aside__env-dot--${backendHealthTone}`"
            aria-hidden="true"
          ></span>
          <span>{{ backendHealthText }}</span>
        </RouterLink>
      </aside>

      <main class="layout-main">
        <header class="layout-topbar">
          <div class="layout-topbar__heading">
            <nav class="breadcrumb" aria-label="面包屑">
              <el-button
                v-if="isCompactLayout"
                text
                class="layout-topbar__menu-trigger"
                @click="navDrawerVisible = true"
              >
                <el-icon><Menu /></el-icon>
                导航
              </el-button>
              <template v-else>
                <span class="breadcrumb__item breadcrumb__item--root">Caipu Admin</span>
                <el-icon class="breadcrumb__sep"><ArrowRight /></el-icon>
              </template>
              <span class="breadcrumb__item breadcrumb__item--current">{{ pageTitle }}</span>
            </nav>
            <h2 class="layout-topbar__title">{{ pageTitle }}</h2>
            <p class="layout-topbar__subtitle">{{ pageDescription }}</p>
          </div>

          <div class="layout-topbar__actions">
            <slot name="toolbar" />

            <el-dropdown trigger="click" placement="bottom-end" @command="handleAccountCommand">
              <button type="button" class="account-trigger" aria-label="账号菜单">
                <span class="account-trigger__avatar" aria-hidden="true">{{ avatarLabel }}</span>
                <span class="account-trigger__meta">
                  <span class="account-trigger__meta-label">当前账号</span>
                  <strong class="account-trigger__meta-name">{{ username }}</strong>
                </span>
                <el-icon class="account-trigger__caret"><ArrowDown /></el-icon>
              </button>
              <template #dropdown>
                <el-dropdown-menu>
                  <el-dropdown-item disabled>
                    <el-icon><User /></el-icon>
                    {{ username }}
                  </el-dropdown-item>
                  <el-dropdown-item divided command="logout">
                    <el-icon><SwitchButton /></el-icon>
                    退出登录
                  </el-dropdown-item>
                </el-dropdown-menu>
              </template>
            </el-dropdown>
          </div>
        </header>

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
          <div class="layout-brand__logo" aria-hidden="true">
            <el-icon><Monitor /></el-icon>
          </div>
          <div class="layout-brand__text">
            <h1>Caipu Admin</h1>
            <p>AI 观测与运行时配置</p>
          </div>
        </div>

        <el-menu class="layout-menu" :default-active="route.path" router @select="handleMenuSelect">
          <el-menu-item v-for="item in navItems" :key="item.path" :index="item.path">
            <el-icon class="layout-menu__icon">
              <component :is="item.icon" />
            </el-icon>
            <div class="layout-menu__label">
              <strong>{{ item.label }}</strong>
              <span>{{ item.description }}</span>
            </div>
          </el-menu-item>
        </el-menu>

        <RouterLink
          to="/server-health"
          class="layout-aside__footer"
          :class="`layout-aside__footer--${backendHealthTone}`"
          :title="backendHealthTitle"
          @click="navDrawerVisible = false"
        >
          <span
            class="layout-aside__env-dot"
            :class="`layout-aside__env-dot--${backendHealthTone}`"
            aria-hidden="true"
          ></span>
          <span>{{ backendHealthText }}</span>
        </RouterLink>
      </div>
    </el-drawer>
  </div>
</template>

<script setup lang="ts">
import { computed, ref, watch } from 'vue'
import { RouterLink, useRoute, useRouter } from 'vue-router'
import { ElMessageBox } from 'element-plus'
import {
  ArrowDown,
  ArrowRight,
  Connection,
  DataLine,
  Menu,
  Monitor,
  Notebook,
  Odometer,
  Setting,
  SwitchButton,
  Tickets,
  User
} from '@element-plus/icons-vue'
import * as adminApi from '@/api/admin'
import { useBackendHealth } from '@/composables/useBackendHealth'
import { useResponsive } from '@/composables/useResponsive'
import { currentUsername, markLoggedOut } from '@/session'

const route = useRoute()
const router = useRouter()
const { isCompactLayout, isMobile } = useResponsive()
const { state: backendHealthState, lastCheckedAt: backendHealthCheckedAt } = useBackendHealth()
const navDrawerVisible = ref(false)

const navItems = [
  {
    path: '/dashboard',
    label: '概览',
    description: '全局健康度与失败信号',
    icon: Odometer
  },
  {
    path: '/ai-providers',
    label: 'AI Provider',
    description: '多节点路由、熔断与草稿测试',
    icon: Connection
  },
  {
    path: '/server-health',
    label: '服务健康',
    description: '主机资源与核心链路状态',
    icon: Monitor
  },
  {
    path: '/ai-jobs',
    label: 'AI 任务',
    description: '任务结果与链路回溯',
    icon: Notebook
  },
  {
    path: '/ai-calls',
    label: 'API 调用',
    description: 'Provider 调用明细',
    icon: DataLine
  },
  {
    path: '/settings',
    label: '配置中心',
    description: '运行时配置与审计',
    icon: Setting
  }
]

// Preserve Tickets import for future sub-menu usage; avoid unused warning.
void Tickets

const username = computed(() => currentUsername.value || 'admin')
const avatarLabel = computed(() => (username.value[0] || 'A').toUpperCase())
const pageTitle = computed(() => String(route.meta.title || 'Caipu Admin'))
const pageDescription = computed(() =>
  String(route.meta.description || '统一查看任务成功率、调用明细与在线配置。')
)
const drawerSize = computed(() => (isMobile.value ? '100%' : '320px'))

const backendHealthTone = computed(() => {
  switch (backendHealthState.value) {
    case 'online':
      return 'online'
    case 'degraded':
      return 'degraded'
    case 'critical':
    case 'offline':
      return 'critical'
    default:
      return 'unknown'
  }
})

const backendHealthText = computed(() => {
  switch (backendHealthState.value) {
    case 'online':
      return '后端在线'
    case 'degraded':
      return '部分服务告警'
    case 'critical':
      return '后端异常'
    case 'offline':
      return '无法连接后端'
    default:
      return '正在检测后端状态'
  }
})

const backendHealthTitle = computed(() => {
  const checked = backendHealthCheckedAt.value
  const base = '点击查看服务健康详情'
  if (!checked) {
    return base
  }
  const parsed = new Date(checked)
  if (Number.isNaN(parsed.getTime())) {
    return base
  }
  return `${base}（更新于 ${parsed.toLocaleTimeString()}）`
})

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

async function handleAccountCommand(command: string | number | object) {
  if (command === 'logout') {
    await handleLogout()
  }
}

async function handleLogout() {
  try {
    await ElMessageBox.confirm('退出后需要重新登录后台账号。', '退出登录', {
      type: 'warning',
      confirmButtonText: '确认退出',
      cancelButtonText: '取消'
    })
  } catch {
    return
  }
  await adminApi.logout()
  markLoggedOut()
  navDrawerVisible.value = false
  await router.replace('/login')
}
</script>
