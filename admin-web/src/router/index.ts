import { createRouter, createWebHistory } from 'vue-router'
import DashboardPage from '@/pages/DashboardPage.vue'
import JobsPage from '@/pages/JobsPage.vue'
import CallsPage from '@/pages/CallsPage.vue'
import SettingsPage from '@/pages/SettingsPage.vue'
import LoginPage from '@/pages/LoginPage.vue'
import { bootstrapSession, currentUsername } from '@/session'

const router = createRouter({
  history: createWebHistory('/admin/'),
  routes: [
    {
      path: '/login',
      name: 'login',
      component: LoginPage,
      meta: { public: true }
    },
    {
      path: '/',
      redirect: '/dashboard'
    },
    {
      path: '/dashboard',
      name: 'dashboard',
      component: DashboardPage
    },
    {
      path: '/ai-jobs',
      name: 'ai-jobs',
      component: JobsPage
    },
    {
      path: '/ai-calls',
      name: 'ai-calls',
      component: CallsPage
    },
    {
      path: '/settings',
      name: 'settings',
      component: SettingsPage
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
