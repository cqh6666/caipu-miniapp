import { createRouter, createWebHistory } from 'vue-router'
import { bootstrapSession, currentUsername } from '@/session'

const router = createRouter({
  history: createWebHistory('/admin/'),
  routes: [
    {
      path: '/login',
      name: 'login',
      component: () => import('@/pages/LoginPage.vue'),
      meta: {
        public: true,
        title: '后台登录',
        description: '进入 Caipu Admin 统一运维与配置控制台。'
      }
    },
    {
      path: '/',
      redirect: '/dashboard'
    },
    {
      path: '/dashboard',
      name: 'dashboard',
      component: () => import('@/pages/DashboardPage.vue'),
      meta: {
        title: 'AI 概览',
        description: '统一查看任务成功率、调用健康度与最近失败信号。'
      }
    },
    {
      path: '/ai-jobs',
      name: 'ai-jobs',
      component: () => import('@/pages/JobsPage.vue'),
      meta: {
        title: 'AI 任务',
        description: '按任务维度追踪场景、目标对象、最终状态与关联调用。'
      }
    },
    {
      path: '/ai-calls',
      name: 'ai-calls',
      component: () => import('@/pages/CallsPage.vue'),
      meta: {
        title: 'API 调用',
        description: '按 provider 与 request 维度排查超时、失败与异常模式。'
      }
    },
    {
      path: '/settings',
      name: 'settings',
      component: () => import('@/pages/SettingsPage.vue'),
      meta: {
        title: '配置中心',
        description: '在线管理运行时参数、测试连通性并追踪审计记录。'
      }
    }
  ]
})

router.beforeEach(async (to) => {
  const username = await bootstrapSession()
  if (to.meta.public) {
    if (username) {
      return '/dashboard'
    }
    return true
  }

  if (!currentUsername.value) {
    return `/login?redirect=${encodeURIComponent(to.fullPath)}`
  }
  return true
})

export default router
