<template>
  <AppShell>
    <el-alert
      v-if="dirtyGroupCount"
      class="settings-summary"
      type="warning"
      :closable="false"
      :title="`当前有 ${dirtyGroupCount} 个分组存在未保存变更`"
      description="建议先检查 diff 再保存，敏感配置清空后只有保存才会真正生效。"
    />

    <div v-if="groupsLoading && !groups.length" class="page-card setting-card">
      <PageState mode="loading" title="正在加载运行时配置" />
    </div>
    <div v-else-if="groupsError && !groups.length" class="page-card setting-card">
      <PageState
        mode="error"
        title="运行时配置加载失败"
        :description="groupsError"
        @retry="loadGroups"
      />
    </div>

    <template v-else>
      <div v-for="group in groups" :key="group.name" class="page-card setting-card">
        <div class="setting-card__header">
          <div>
            <h3 class="setting-card__title">{{ group.title }}</h3>
            <div class="setting-card__desc">{{ group.description }}</div>
          </div>
          <div class="setting-card__header-actions">
            <StatusTag
              v-if="isGroupDirty(group.name)"
              tone="warning"
              :text="`${getGroupDiff(group.name).length} 项待保存`"
            />
            <el-button :loading="testingGroups[group.name]" @click="handleTest(group.name)">测试连接</el-button>
            <el-button type="primary" :loading="savingGroups[group.name]" @click="handleSave(group.name)">保存</el-button>
          </div>
        </div>

        <el-alert
          v-if="isGroupDirty(group.name)"
          class="setting-alert"
          type="warning"
          :closable="false"
          :title="`该分组尚有 ${getGroupDiff(group.name).length} 项未保存变更`"
        />

        <el-alert
          v-if="isAIRuntimeCompatGroup(group.name)"
          class="setting-alert"
          type="info"
          :closable="false"
          title="该分组已降级为兼容模式入口"
        >
          <template #default>
            多 Provider 正式入口已迁移到
            <router-link to="/ai-providers">AI Provider</router-link>
            页面；这里保留为旧单节点兜底配置，仅在新路由未启用时生效。
          </template>
        </el-alert>

        <div v-if="testResults[group.name]" class="test-result-card">
          <div class="detail-panel__header">
            <div>
              <div class="detail-panel__title">最近测试结果</div>
              <div class="detail-panel__subtitle">{{ testResults[group.name]?.message }}</div>
            </div>
            <StatusTag
              :tone="testResults[group.name]?.ok ? 'success' : 'warning'"
              :text="testResults[group.name]?.ok ? '连通正常' : '需要关注'"
            />
          </div>
          <div class="test-result-card__meta">
            <span>耗时：{{ testResults[group.name]?.latencyMs ?? 0 }} ms</span>
            <span>测试时间：{{ formatDateTime(testResults[group.name]?.testedAt) }}</span>
          </div>
        </div>

        <div v-for="field in group.fields" :key="field.key" class="setting-row">
          <div>
            <div class="setting-field__title">
              <strong>{{ field.label }}</strong>
              <div class="setting-field__tags">
                <el-tag size="small" effect="plain">{{ displayValueType(field.valueType) }}</el-tag>
                <el-tag v-if="field.isSecret" size="small" type="warning" effect="plain">敏感</el-tag>
                <el-tag v-if="field.isRestartRequired" size="small" type="danger" effect="plain">需重启</el-tag>
                <el-tag
                  v-if="isFieldQueuedForClear(group.name, field.key)"
                  size="small"
                  type="danger"
                  effect="plain"
                >
                  待清空
                </el-tag>
              </div>
            </div>
            <div class="setting-meta">{{ field.description }}</div>
          </div>

          <div class="setting-field__control">
            <template v-if="field.valueType === 'bool'">
              <el-switch
                v-model="drafts[group.name][field.key]"
                inline-prompt
                active-text="开"
                inactive-text="关"
              />
            </template>
            <template v-else-if="field.isSecret">
              <div class="setting-field__control-row">
                <el-input
                  v-model="drafts[group.name][field.key]"
                  type="password"
                  show-password
                  :placeholder="field.maskedValue || '当前未配置，输入新的值'"
                  @update:model-value="handleFieldInput(group.name, field, $event)"
                />
                <el-button text @click="handleClearSecret(group.name, field.key)">清空</el-button>
              </div>
            </template>
            <template v-else>
              <el-input
                v-model="drafts[group.name][field.key]"
                :type="field.valueType === 'int' || field.valueType === 'float' ? 'number' : 'text'"
                :placeholder="field.maskedValue || '请输入值'"
                @update:model-value="handleFieldInput(group.name, field, $event)"
              />
            </template>

            <div v-if="field.isSecret && getSecretHint(group.name, field.key)" class="setting-meta">
              {{ getSecretHint(group.name, field.key) }}
            </div>
          </div>

          <div class="setting-meta">
            <div>来源：{{ displaySettingSource(field.source) }}</div>
            <div>当前值：{{ field.maskedValue || '未配置' }}</div>
            <div>最近修改：{{ formatDateTime(field.updatedAt) }}</div>
            <div>修改人：{{ field.updatedBySubject || '暂无' }}</div>
          </div>
        </div>
      </div>
    </template>

    <div class="page-card audit-section">
      <div class="page-header">
        <div>
          <h2 class="page-title" style="font-size: 22px">最近审计</h2>
          <div class="page-subtitle">记录保存与测试动作，便于回看配置变更与排障过程。</div>
        </div>
      </div>

      <FilterToolbar>
        <el-select v-model="auditFilters.group" clearable placeholder="配置分组">
          <el-option v-for="group in groups" :key="group.name" :label="group.title" :value="group.name" />
        </el-select>
        <el-select v-model="auditFilters.action" clearable placeholder="动作">
          <el-option
            v-for="item in auditActionOptions"
            :key="item.value"
            :label="item.label"
            :value="item.value"
          />
        </el-select>
        <template #actions>
          <el-button @click="resetAuditFilters">重置</el-button>
          <el-button type="primary" :loading="auditsLoading" @click="applyAuditFilters">筛选</el-button>
        </template>
      </FilterToolbar>

      <el-alert
        v-if="auditsError && audits.items.length"
        class="setting-alert"
        type="warning"
        :closable="false"
        :title="auditsError"
      />

      <PageState v-if="auditsLoading && !audits.items.length" mode="loading" title="正在加载审计记录" compact />
      <PageState
        v-else-if="auditsError && !audits.items.length"
        mode="error"
        title="审计记录加载失败"
        :description="auditsError"
        compact
        @retry="loadAudits"
      />
      <PageState
        v-else-if="!audits.items.length"
        mode="empty"
        title="暂无审计记录"
        description="当前筛选条件下没有命中的保存或测试记录。"
        compact
      />
      <template v-else>
        <div class="table-scroll">
          <el-table :data="audits.items" size="small" style="width: 100%">
            <el-table-column label="分组" min-width="160">
              <template #default="{ row }">{{ groupTitleMap[row.groupName] || row.groupName }}</template>
            </el-table-column>
            <el-table-column prop="settingKey" label="配置键" min-width="180" show-overflow-tooltip />
            <el-table-column label="动作" width="100">
              <template #default="{ row }">{{ displayAuditAction(row.action) }}</template>
            </el-table-column>
            <el-table-column prop="oldValueMasked" label="旧值" min-width="120" show-overflow-tooltip />
            <el-table-column prop="newValueMasked" label="新值" min-width="120" show-overflow-tooltip />
            <el-table-column prop="operatorSubject" label="操作人" width="120" />
            <el-table-column label="时间" width="180">
              <template #default="{ row }">{{ formatDateTime(row.createdAt) }}</template>
            </el-table-column>
          </el-table>
        </div>

        <div style="display: flex; justify-content: flex-end; margin-top: 16px">
          <el-pagination
            v-model:current-page="auditPage"
            layout="total, prev, pager, next"
            background
            :total="audits.total"
            @current-change="handleAuditPageChange"
          />
        </div>
      </template>
    </div>
  </AppShell>
</template>

<script setup lang="ts">
import { computed, h, onMounted, reactive, ref } from 'vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import AppShell from '@/components/AppShell.vue'
import FilterToolbar from '@/components/FilterToolbar.vue'
import PageState from '@/components/PageState.vue'
import StatusTag from '@/components/StatusTag.vue'
import * as adminApi from '@/api/admin'
import type { GroupTestResult, PaginationResult, RuntimeSettingFieldView, RuntimeSettingGroupView, SettingAuditRecord } from '@/types'
import {
  auditActionOptions,
  displayAuditAction,
  displaySettingSource,
  displayValueType,
  formatDateTime
} from '@/utils/admin-display'

interface StoredTestResult extends GroupTestResult {
  testedAt: string
}

interface DiffItem {
  key: string
  label: string
  summary: string
}

const groups = ref<RuntimeSettingGroupView[]>([])
const groupsLoading = ref(false)
const groupsError = ref('')
const drafts = reactive<Record<string, Record<string, string | boolean>>>({})
const clearKeyMap = reactive<Record<string, string[]>>({})
const savingGroups = reactive<Record<string, boolean>>({})
const testingGroups = reactive<Record<string, boolean>>({})
const testResults = reactive<Record<string, StoredTestResult | null>>({})

const auditFilters = reactive({
  group: '',
  action: ''
})
const auditPage = ref(1)
const auditsLoading = ref(false)
const auditsError = ref('')
const audits = ref<PaginationResult<SettingAuditRecord>>({
  items: [],
  total: 0,
  page: 1,
  pageSize: 20
})

const groupTitleMap = computed(() =>
  Object.fromEntries(groups.value.map((group) => [group.name, group.title]))
)

const dirtyGroupCount = computed(() => groups.value.filter((group) => isGroupDirty(group.name)).length)

function resetReactiveRecord(target: Record<string, unknown>) {
  for (const key of Object.keys(target)) {
    delete target[key]
  }
}

function initializeDrafts() {
  resetReactiveRecord(drafts)
  resetReactiveRecord(clearKeyMap)
  for (const group of groups.value) {
    drafts[group.name] = {}
    clearKeyMap[group.name] = []
    for (const field of group.fields) {
      if (field.valueType === 'bool') {
        drafts[group.name][field.key] = field.value === 'true'
      } else if (field.isSecret) {
        drafts[group.name][field.key] = ''
      } else {
        drafts[group.name][field.key] = field.value || ''
      }
    }
  }
}

function getGroupByName(groupName: string) {
  return groups.value.find((item) => item.name === groupName)
}

function isFieldQueuedForClear(groupName: string, key: string) {
  return (clearKeyMap[groupName] || []).includes(key)
}

function queueFieldClear(groupName: string, key: string) {
  if (!isFieldQueuedForClear(groupName, key)) {
    clearKeyMap[groupName] = [...(clearKeyMap[groupName] || []), key]
  }
}

function dequeueFieldClear(groupName: string, key: string) {
  clearKeyMap[groupName] = (clearKeyMap[groupName] || []).filter((item) => item !== key)
}

function formatDraftValue(field: RuntimeSettingFieldView, value: string | boolean) {
  if (field.valueType === 'bool') {
    return value ? '开' : '关'
  }
  const raw = typeof value === 'string' ? value.trim() : String(value)
  return raw || '空'
}

function getGroupDiff(groupName: string): DiffItem[] {
  const group = getGroupByName(groupName)
  if (!group) {
    return []
  }

  const items: DiffItem[] = []
  for (const field of group.fields) {
    const draft = drafts[groupName]?.[field.key]
    const queuedForClear = isFieldQueuedForClear(groupName, field.key)

    if (field.isSecret) {
      if (queuedForClear) {
        items.push({
          key: field.key,
          label: field.label,
          summary: `清空 ${field.label}`
        })
        continue
      }

      if (typeof draft === 'string' && draft.trim()) {
        items.push({
          key: field.key,
          label: field.label,
          summary: `更新 ${field.label}`
        })
      }
      continue
    }

    if (field.valueType === 'bool') {
      const currentValue = field.value === 'true'
      const nextValue = Boolean(draft)
      if (currentValue !== nextValue) {
        items.push({
          key: field.key,
          label: field.label,
          summary: `更新 ${field.label}：${currentValue ? '开' : '关'} -> ${nextValue ? '开' : '关'}`
        })
      }
      continue
    }

    const currentValue = (field.value || '').trim()
    const nextValue = typeof draft === 'string' ? draft.trim() : String(draft ?? '')
    if (currentValue !== nextValue) {
      items.push({
        key: field.key,
        label: field.label,
        summary: nextValue
          ? `更新 ${field.label}：${currentValue || '空'} -> ${nextValue}`
          : `清空 ${field.label}，恢复默认来源`
      })
    }
  }

  return items
}

function isGroupDirty(groupName: string) {
  return getGroupDiff(groupName).length > 0
}

function getSecretHint(groupName: string, key: string) {
  if (isFieldQueuedForClear(groupName, key)) {
    return '该字段已加入清空队列，保存后会删除覆盖值。'
  }
  const draft = drafts[groupName]?.[key]
  if (typeof draft === 'string' && draft.trim()) {
    return '已输入新值，保存后会覆盖当前密钥。'
  }
  return ''
}

function isAIRuntimeCompatGroup(groupName: string) {
  return groupName === 'ai.summary' || groupName === 'ai.title' || groupName === 'ai.flowchart'
}

function handleFieldInput(groupName: string, field: RuntimeSettingFieldView, value: string | boolean) {
  drafts[groupName][field.key] = value
  if (field.isSecret && typeof value === 'string' && value.trim()) {
    dequeueFieldClear(groupName, field.key)
  }
}

function buildPayload(groupName: string) {
  const group = getGroupByName(groupName)
  if (!group) {
    return { values: {}, clearKeys: [] as string[], diffItems: [] as DiffItem[] }
  }

  const values: Record<string, unknown> = {}
  const clearKeys: string[] = []
  const diffItems = getGroupDiff(groupName)
  const diffKeySet = new Set(diffItems.map((item) => item.key))

  for (const field of group.fields) {
    if (!diffKeySet.has(field.key)) {
      continue
    }
    const draft = drafts[groupName]?.[field.key]
    if (field.isSecret) {
      if (isFieldQueuedForClear(groupName, field.key)) {
        clearKeys.push(field.key)
        continue
      }
      if (typeof draft === 'string' && draft.trim()) {
        values[field.key] = draft.trim()
      }
      continue
    }
    if (field.valueType === 'bool') {
      values[field.key] = Boolean(draft)
      continue
    }
    values[field.key] = typeof draft === 'string' ? draft.trim() : draft
  }

  return { values, clearKeys, diffItems }
}

async function loadGroups() {
  groupsLoading.value = true
  groupsError.value = ''
  try {
    const data = await adminApi.getRuntimeSettings()
    groups.value = data.groups
    initializeDrafts()
  } catch (error) {
    groupsError.value = error instanceof Error ? error.message : '加载配置失败'
  } finally {
    groupsLoading.value = false
  }
}

async function loadAudits() {
  auditsLoading.value = true
  auditsError.value = ''
  try {
    const query = new URLSearchParams({
      page: String(auditPage.value),
      pageSize: '20'
    })
    if (auditFilters.group) query.set('group', auditFilters.group)
    if (auditFilters.action) query.set('action', auditFilters.action)
    const data = await adminApi.listSettingAudits(query)
    audits.value = data.result
  } catch (error) {
    auditsError.value = error instanceof Error ? error.message : '加载审计失败'
  } finally {
    auditsLoading.value = false
  }
}

async function handleClearSecret(groupName: string, key: string) {
  try {
    await ElMessageBox.confirm('保存后会真正清空当前敏感配置，确定加入清空队列吗？', '确认清空敏感值', {
      type: 'warning',
      confirmButtonText: '加入清空队列',
      cancelButtonText: '取消'
    })
    queueFieldClear(groupName, key)
    drafts[groupName][key] = ''
    ElMessage.info('已加入清空队列，保存后生效')
  } catch {
    // 用户取消时不提示
  }
}

async function handleSave(groupName: string) {
  const payload = buildPayload(groupName)
  if (!payload.diffItems.length) {
    ElMessage.info('当前分组没有待保存变更')
    return
  }

  try {
    await ElMessageBox({
      title: '确认保存配置',
      message: h(
        'div',
        { style: 'display:grid;gap:8px;line-height:1.6;' },
        payload.diffItems.map((item, index) => h('div', `${index + 1}. ${item.summary}`))
      ),
      showCancelButton: true,
      confirmButtonText: '确认保存',
      cancelButtonText: '取消'
    })
  } catch {
    return
  }

  savingGroups[groupName] = true
  try {
    await adminApi.updateRuntimeGroup(groupName, {
      values: payload.values,
      clearKeys: payload.clearKeys
    })
    ElMessage.success('保存成功')
    testResults[groupName] = null
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
    const payload = buildPayload(groupName)
    const data = await adminApi.testRuntimeGroup(groupName, {
      values: payload.values,
      clearKeys: payload.clearKeys
    })
    testResults[groupName] = {
      ...data.result,
      testedAt: new Date().toISOString()
    }
    if (data.result.ok) {
      ElMessage.success(`${data.result.message}（${data.result.latencyMs} ms）`)
    } else {
      ElMessage.warning(`${data.result.message}（${data.result.latencyMs} ms）`)
    }
    auditPage.value = 1
    await loadAudits()
  } catch (error) {
    ElMessage.error(error instanceof Error ? error.message : '测试失败')
  } finally {
    testingGroups[groupName] = false
  }
}

async function applyAuditFilters() {
  auditPage.value = 1
  await loadAudits()
}

async function resetAuditFilters() {
  auditFilters.group = ''
  auditFilters.action = ''
  auditPage.value = 1
  await loadAudits()
}

async function handleAuditPageChange(nextPage: number) {
  auditPage.value = nextPage
  await loadAudits()
}

onMounted(async () => {
  await Promise.all([loadGroups(), loadAudits()])
})
</script>
