<template>
  <AppShell>
    <div class="page-header">
      <div>
        <h2 class="page-title">配置中心</h2>
        <div class="page-subtitle">在线修改 AI / sidecar 配置，并查看最近审计记录。</div>
      </div>
    </div>

    <div v-for="group in groups" :key="group.name" class="page-card setting-card">
      <div class="setting-card__header">
        <div>
          <h3 class="setting-card__title">{{ group.title }}</h3>
          <div class="setting-card__desc">{{ group.description }}</div>
        </div>
        <div style="display: flex; gap: 10px">
          <el-button :loading="testingGroups[group.name]" @click="handleTest(group.name)">测试连接</el-button>
          <el-button type="primary" :loading="savingGroups[group.name]" @click="handleSave(group.name)">保存</el-button>
        </div>
      </div>

      <div v-for="field in group.fields" :key="field.key" class="setting-row">
        <div>
          <strong>{{ field.label }}</strong>
          <div class="setting-meta">{{ field.description }}</div>
        </div>

        <div>
          <template v-if="field.valueType === 'bool'">
            <el-switch
              v-model="drafts[group.name][field.key]"
              inline-prompt
              active-text="开"
              inactive-text="关"
            />
          </template>
          <template v-else-if="field.valueType === 'int' || field.valueType === 'float'">
            <el-input v-model="drafts[group.name][field.key]" :placeholder="field.maskedValue || '请输入值'" />
          </template>
          <template v-else-if="field.isSecret">
            <div style="display: flex; gap: 10px; align-items: center">
              <el-input
                v-model="drafts[group.name][field.key]"
                type="password"
                show-password
                :placeholder="field.maskedValue || '当前未配置，输入新的值'"
              />
              <el-button text @click="handleClearSecret(group.name, field.key)">清空</el-button>
            </div>
          </template>
          <template v-else>
            <el-input v-model="drafts[group.name][field.key]" :placeholder="field.maskedValue || '请输入值'" />
          </template>
        </div>

        <div class="setting-meta">
          <div>来源：{{ field.source || '-' }}</div>
          <div>当前值：{{ field.isSecret ? field.maskedValue || '未配置' : field.maskedValue || '未配置' }}</div>
          <div>最近修改：{{ field.updatedAt || '暂无' }}</div>
          <div>修改人：{{ field.updatedBySubject || '暂无' }}</div>
        </div>
      </div>
    </div>

    <div class="page-card table-card">
      <div class="page-header">
        <div>
          <h2 class="page-title" style="font-size: 22px">最近审计</h2>
          <div class="page-subtitle">记录保存和测试动作。</div>
        </div>
      </div>
      <el-table :data="audits.items" size="small">
        <el-table-column prop="groupName" label="分组" min-width="160" />
        <el-table-column prop="settingKey" label="配置键" min-width="180" show-overflow-tooltip />
        <el-table-column prop="action" label="动作" width="90" />
        <el-table-column prop="oldValueMasked" label="旧值" min-width="120" show-overflow-tooltip />
        <el-table-column prop="newValueMasked" label="新值" min-width="120" show-overflow-tooltip />
        <el-table-column prop="operatorSubject" label="操作人" width="120" />
        <el-table-column prop="createdAt" label="时间" width="180" />
      </el-table>
    </div>
  </AppShell>
</template>

<script setup lang="ts">
import { reactive, ref } from 'vue'
import { ElMessage } from 'element-plus'
import AppShell from '@/components/AppShell.vue'
import * as adminApi from '@/api/admin'
import type { PaginationResult, RuntimeSettingGroupView, SettingAuditRecord } from '@/types'

const groups = ref<RuntimeSettingGroupView[]>([])
const drafts = reactive<Record<string, Record<string, string | boolean>>>({})
const clearKeys = reactive<Record<string, Set<string>>>({})
const savingGroups = reactive<Record<string, boolean>>({})
const testingGroups = reactive<Record<string, boolean>>({})

const audits = ref<PaginationResult<SettingAuditRecord>>({
  items: [],
  total: 0,
  page: 1,
  pageSize: 20
})

function initializeDrafts() {
  for (const group of groups.value) {
    drafts[group.name] = {}
    clearKeys[group.name] = new Set<string>()
    for (const field of group.fields) {
      if (field.valueType === 'bool') {
        drafts[group.name][field.key] = field.value === 'true'
      } else if (!field.isSecret) {
        drafts[group.name][field.key] = field.value || ''
      } else {
        drafts[group.name][field.key] = ''
      }
    }
  }
}

async function loadGroups() {
  try {
    const data = await adminApi.getRuntimeSettings()
    groups.value = data.groups
    initializeDrafts()
  } catch (error) {
    ElMessage.error(error instanceof Error ? error.message : '加载配置失败')
  }
}

async function loadAudits() {
  try {
    const query = new URLSearchParams({ page: '1', pageSize: '20' })
    const data = await adminApi.listSettingAudits(query)
    audits.value = data.result
  } catch (error) {
    ElMessage.error(error instanceof Error ? error.message : '加载审计失败')
  }
}

function buildPayload(groupName: string) {
  const group = groups.value.find((item) => item.name === groupName)
  if (!group) {
    return { values: {}, clearKeys: [] as string[] }
  }

  const values: Record<string, unknown> = {}
  for (const field of group.fields) {
    const draft = drafts[groupName]?.[field.key]
    if (field.valueType === 'bool') {
      values[field.key] = Boolean(draft)
      continue
    }
    if (field.isSecret) {
      if (typeof draft === 'string' && draft.trim()) {
        values[field.key] = draft.trim()
      }
      continue
    }
    values[field.key] = typeof draft === 'string' ? draft.trim() : draft
  }

  return {
    values,
    clearKeys: Array.from(clearKeys[groupName] || [])
  }
}

function handleClearSecret(groupName: string, key: string) {
  clearKeys[groupName]?.add(key)
  drafts[groupName][key] = ''
  ElMessage.info('已加入清空队列，保存后生效')
}

async function handleSave(groupName: string) {
  savingGroups[groupName] = true
  try {
    await adminApi.updateRuntimeGroup(groupName, buildPayload(groupName))
    ElMessage.success('保存成功')
    await Promise.all([loadGroups(), loadAudits()])
  } catch (error) {
    ElMessage.error(error instanceof Error ? error.message : '保存失败')
  } finally {
    savingGroups[groupName] = false
  }
}

async function handleTest(groupName: string) {
  testingGroups[groupName] = true
  try {
    const data = await adminApi.testRuntimeGroup(groupName, buildPayload(groupName))
    if (data.result.ok) {
      ElMessage.success(`${data.result.message}（${data.result.latencyMs} ms）`)
    } else {
      ElMessage.warning(`${data.result.message}（${data.result.latencyMs} ms）`)
    }
    await loadAudits()
  } catch (error) {
    ElMessage.error(error instanceof Error ? error.message : '测试失败')
  } finally {
    testingGroups[groupName] = false
  }
}

Promise.all([loadGroups(), loadAudits()])
</script>
