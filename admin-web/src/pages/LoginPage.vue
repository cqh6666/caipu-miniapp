<template>
  <div class="login-page">
    <aside class="login-side">
      <div class="login-brand">
        <div class="login-brand__logo" aria-hidden="true">
          <el-icon><Monitor /></el-icon>
        </div>
        <div class="login-brand__text">
          <h1>Caipu Admin</h1>
          <p>AI 观测与运行时配置控制台</p>
        </div>
      </div>

      <div class="login-hero">
        <h2>一个页面，看清 AI 任务、API 调用与配置中心。</h2>
        <p>
          统一指标仪表盘、多 Provider 路由、实时告警与审计记录，
          让运维同学第一时间发现问题并完成处置。
        </p>
      </div>

      <ul class="login-highlights">
        <li class="login-highlights__item">
          <span class="login-highlights__dot"><el-icon><DataLine /></el-icon></span>
          <span>任务成功率、超时率、P95 一屏可见，异常直达详情</span>
        </li>
        <li class="login-highlights__item">
          <span class="login-highlights__dot"><el-icon><Connection /></el-icon></span>
          <span>多 Provider 路由草稿、熔断阈值与邮件告警全链路托管</span>
        </li>
        <li class="login-highlights__item">
          <span class="login-highlights__dot"><el-icon><Lock /></el-icon></span>
          <span>独立后台账号 + 审计留痕，运行时配置变更全部可溯</span>
        </li>
      </ul>
    </aside>

    <main class="login-main">
      <div class="login-card">
        <h1 class="login-title">后台登录</h1>
        <p class="login-desc">
          使用独立后台账号进入 AI 可观测性与动态配置中心。
        </p>
        <el-form :model="form" label-position="top" @submit.prevent="handleSubmit">
          <el-form-item label="用户名">
            <el-input
              v-model="form.username"
              autocomplete="username"
              placeholder="请输入后台用户名"
              :prefix-icon="User"
            />
          </el-form-item>
          <el-form-item label="密码">
            <el-input
              v-model="form.password"
              type="password"
              show-password
              autocomplete="current-password"
              placeholder="请输入后台密码"
              :prefix-icon="Lock"
              @keyup.enter="handleSubmit"
            />
          </el-form-item>
          <el-button
            :loading="loading"
            type="primary"
            size="large"
            style="width: 100%"
            @click="handleSubmit"
          >
            登录
          </el-button>
        </el-form>
      </div>
    </main>
  </div>
</template>

<script setup lang="ts">
import { reactive, ref } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { ElMessage } from 'element-plus'
import { Connection, DataLine, Lock, Monitor, User } from '@element-plus/icons-vue'
import * as adminApi from '@/api/admin'
import { markLoggedIn } from '@/session'

const router = useRouter()
const route = useRoute()
const loading = ref(false)
const form = reactive({
  username: '',
  password: ''
})

async function handleSubmit() {
  if (!form.username.trim() || !form.password.trim()) {
    ElMessage.warning('请输入用户名和密码')
    return
  }

  loading.value = true
  try {
    const data = await adminApi.login(form.username.trim(), form.password)
    markLoggedIn(data.username)
    const redirect = typeof route.query.redirect === 'string' ? route.query.redirect : '/dashboard'
    await router.replace(redirect)
  } catch (error) {
    ElMessage.error(error instanceof Error ? error.message : '登录失败')
  } finally {
    loading.value = false
  }
}
</script>
