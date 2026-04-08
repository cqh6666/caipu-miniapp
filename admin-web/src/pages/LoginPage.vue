<template>
  <div class="login-page">
    <div class="login-card page-card">
      <h1 class="login-title">后台登录</h1>
      <p class="login-desc">
        使用独立后台账号进入 AI 可观测性与动态配置中心。
      </p>
      <el-form :model="form" label-position="top" @submit.prevent="handleSubmit">
        <el-form-item label="用户名">
          <el-input v-model="form.username" autocomplete="username" placeholder="请输入后台用户名" />
        </el-form-item>
        <el-form-item label="密码">
          <el-input
            v-model="form.password"
            type="password"
            show-password
            autocomplete="current-password"
            placeholder="请输入后台密码"
            @keyup.enter="handleSubmit"
          />
        </el-form-item>
        <el-button :loading="loading" type="primary" size="large" style="width: 100%" @click="handleSubmit">
          登录
        </el-button>
      </el-form>
    </div>
  </div>
</template>

<script setup lang="ts">
import { reactive, ref } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { ElMessage } from 'element-plus'
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
