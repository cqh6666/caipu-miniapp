<template>
  <AppShell>
    <div class="page-header">
      <div>
        <h2 class="page-title">AI Provider</h2>
        <div class="page-subtitle">按场景维护多 Provider 路由、失败切换和兼容模式收口。</div>
      </div>
	    <div class="page-header__actions">
	        <el-button :loading="pageRefreshing" @click="refreshPage">刷新</el-button>
	        <el-button :loading="testingScene" :disabled="!draftScene" @click="handleTestScene">测试当前草稿</el-button>
	        <el-button type="primary" :loading="savingScene" :disabled="!draftScene" @click="handleSaveScene">保存场景</el-button>
	      </div>
    </div>

    <div class="routing-scene-grid">
      <button
        v-for="item in sceneCards"
        :key="item.scene"
        type="button"
        class="page-card routing-scene-card"
        :class="{ 'routing-scene-card--active': item.scene === currentSceneKey }"
        @click="handleSceneChange(item.scene)"
      >
        <div class="routing-scene-card__header">
          <div>
            <div class="routing-scene-card__eyebrow">{{ displayAIRoutingScene(item.scene) }}</div>
            <h3>{{ item.title }}</h3>
          </div>
          <StatusTag :tone="item.tone" :text="item.statusText" />
        </div>
        <div class="routing-scene-card__meta">
          <span>策略：{{ displayAIRoutingStrategy(item.strategy) }}</span>
          <span>节点：{{ item.activeProviderCount }}/{{ item.providerCount }}</span>
          <span>来源：{{ displaySettingSource(item.source) }}</span>
        </div>
        <div class="routing-scene-card__footer">
          <span>最近修改：{{ formatDateTime(item.updatedAt) }}</span>
          <span>{{ item.compatibilityMode ? '运行时仍走兼容链路' : '运行时已走新路由' }}</span>
        </div>
      </button>
    </div>

    <el-alert
      v-if="isDirty"
      class="settings-summary"
      type="warning"
      :closable="false"
      title="当前场景存在未保存草稿"
      description="建议先测试，再保存；敏感密钥留空表示保留旧值，点击清空后保存才会真正移除。"
    />

    <div v-if="sceneLoading && !draftScene" class="page-card routing-panel">
      <PageState mode="loading" title="正在加载场景配置" />
    </div>
    <div v-else-if="sceneError && !draftScene" class="page-card routing-panel">
      <PageState mode="error" title="场景配置加载失败" :description="sceneError" @retry="loadCurrentScene" />
    </div>
    <template v-else-if="draftScene">
      <el-alert
        v-if="draftScene.compatibilityMode"
        class="routing-alert"
        type="warning"
        :closable="false"
        title="当前仍处于兼容模式"
        :description="compatibilityHint"
      />

      <div class="routing-editor-grid">
        <div class="page-card routing-panel">
          <div class="routing-panel__header">
            <div>
              <h3 class="routing-panel__title">场景策略</h3>
              <div class="routing-panel__subtitle">统一编辑路由开关、尝试次数、熔断和场景级请求参数。</div>
            </div>
            <div class="routing-panel__tags">
              <StatusTag :tone="draftScene.enabled ? 'primary' : 'neutral'" :text="draftScene.enabled ? '新路由已启用' : '新路由未启用'" />
              <StatusTag :tone="draftScene.compatibilityMode ? 'warning' : 'success'" :text="draftScene.compatibilityMode ? '兼容模式' : '正式模式'" />
            </div>
          </div>

          <div class="routing-form-grid">
            <label class="routing-field">
              <span>启用新路由</span>
              <el-switch v-model="draftScene.enabled" inline-prompt active-text="开" inactive-text="关" />
            </label>
            <label class="routing-field">
              <span>调度策略</span>
              <el-select v-model="draftScene.strategy">
                <el-option
                  v-for="item in aiRoutingStrategyOptions"
                  :key="item.value"
                  :label="item.label"
                  :value="item.value"
                />
              </el-select>
            </label>
            <label class="routing-field">
              <span>最大尝试次数</span>
              <el-input-number v-model="draftScene.maxAttempts" :min="1" :max="maxAttemptCeiling" />
            </label>
            <label class="routing-field">
              <span>熔断阈值</span>
              <el-input-number v-model="draftScene.breaker.failureThreshold" :min="1" :max="10" />
            </label>
            <label class="routing-field">
              <span>冷却时间（秒）</span>
              <el-input-number v-model="draftScene.breaker.cooldownSeconds" :min="5" :max="600" />
            </label>
            <div class="routing-field routing-field--meta">
              <span>最近修改</span>
              <strong>{{ formatDateTime(draftScene.updatedAt) }}</strong>
              <small>修改人：{{ draftScene.updatedBySubject || '暂无' }}</small>
            </div>
          </div>

          <div class="routing-checkbox-block">
            <div class="routing-checkbox-block__title">允许切换到下一个节点的错误类型</div>
            <el-checkbox-group v-model="draftScene.retryOn" class="routing-checkbox-grid">
              <el-checkbox v-for="item in retryOptions" :key="item.value" :label="item.value">
                {{ item.label }}
              </el-checkbox>
            </el-checkbox-group>
          </div>

          <div class="routing-request-block">
            <div>
              <div class="routing-checkbox-block__title">场景级请求参数</div>
              <div class="routing-panel__subtitle">
                标题精修优先使用这里的 `stream / temperature / maxTokens`；其他场景保持最小化参数。
              </div>
            </div>

            <div v-if="draftScene.scene === 'title'" class="routing-form-grid routing-form-grid--request">
              <label class="routing-field">
                <span>Stream</span>
                <el-switch v-model="draftScene.requestOptions.stream" inline-prompt active-text="开" inactive-text="关" />
              </label>
              <label class="routing-field">
                <span>Temperature</span>
                <el-input-number v-model="draftScene.requestOptions.temperature" :min="0" :max="2" :step="0.1" />
              </label>
              <label class="routing-field">
                <span>Max Tokens</span>
                <el-input-number v-model="draftScene.requestOptions.maxTokens" :min="1" :max="512" />
              </label>
            </div>
            <div v-else class="routing-request-block__note">
              当前场景默认沿用业务层固定 prompt 参数，无需额外请求选项。
            </div>
          </div>
        </div>

        <div class="page-card routing-panel">
          <div class="routing-panel__header">
            <div>
              <h3 class="routing-panel__title">Provider 节点</h3>
              <div class="routing-panel__subtitle">支持启停、排序、局部换密钥和单节点测试。</div>
            </div>
            <div class="routing-panel__tags">
              <StatusTag tone="neutral" :text="`${enabledProviderCount}/${draftScene.providers.length} 个启用节点`" />
              <el-button type="primary" plain @click="handleAddProvider">新增节点</el-button>
            </div>
          </div>

          <PageState
            v-if="!draftScene.providers.length"
            mode="empty"
            title="当前还没有 Provider 节点"
            description="先新增一个节点，再决定是否启用新路由。"
            compact
          />

          <div v-else class="provider-editor-list">
            <div v-for="(provider, index) in draftScene.providers" :key="provider.id" class="provider-editor-card">
              <div class="provider-editor-card__header">
                <div>
                  <div class="provider-editor-card__title">
                    <strong>{{ provider.name || `节点 ${index + 1}` }}</strong>
                    <span class="mono-text">{{ provider.id }}</span>
                  </div>
                  <div class="provider-editor-card__meta">
                    <span>顺序 {{ index + 1 }}</span>
                    <span>{{ provider.adapter }}</span>
                    <span>{{ provider.enabled ? '参与调度' : '已停用' }}</span>
                  </div>
                </div>
                <div class="provider-editor-card__actions">
                  <el-switch v-model="provider.enabled" inline-prompt active-text="开" inactive-text="关" />
                  <el-button text :disabled="index === 0" @click="moveProvider(index, -1)">上移</el-button>
                  <el-button text :disabled="index === draftScene.providers.length - 1" @click="moveProvider(index, 1)">下移</el-button>
                  <el-button text :loading="singleTestProviderId === provider.id" @click="handleTestSingleProvider(index)">测试</el-button>
                  <el-button text type="danger" @click="handleRemoveProvider(index)">删除</el-button>
                </div>
              </div>

              <div class="provider-editor-grid">
                <label class="routing-field">
                  <span>Provider ID</span>
                  <el-input v-model.trim="provider.id" placeholder="summary-main" />
                </label>
                <label class="routing-field">
                  <span>展示名称</span>
                  <el-input v-model.trim="provider.name" placeholder="主节点 / 备用节点" />
                </label>
                <label class="routing-field">
                  <span>Base URL</span>
                  <el-input v-model.trim="provider.baseURL" placeholder="https://api.example.com/v1" />
                </label>
                <label class="routing-field">
                  <span>Model</span>
                  <el-input v-model.trim="provider.model" placeholder="gpt-4.1-mini" />
                </label>
                <label class="routing-field">
                  <span>超时（秒）</span>
                  <el-input-number v-model="provider.timeoutSeconds" :min="1" :max="600" />
                </label>
                <label class="routing-field">
                  <span>Adapter</span>
                  <el-input v-model="provider.adapter" disabled />
                </label>
              </div>

              <div class="provider-editor-secret">
                <label class="routing-field" style="flex: 1">
                  <span>API Key</span>
                  <el-input
                    v-model="provider.apiKey"
                    type="password"
                    show-password
                    :placeholder="provider.apiKeyMasked || '当前未配置，可输入新密钥'"
                    @update:model-value="handleProviderApiKeyInput(provider, $event)"
                  />
                </label>
                <el-button text @click="handleClearProviderApiKey(provider)">清空旧密钥</el-button>
              </div>
              <div class="provider-editor-secret__hint">
                <template v-if="provider.clearApiKey">当前已标记为待清空，保存后会彻底移除旧密钥。</template>
                <template v-else-if="provider.hasAPIKey">当前已保存密钥；这里留空表示继续保留旧值。</template>
                <template v-else>当前没有已保存密钥，可直接录入新值。</template>
              </div>
            </div>
          </div>
        </div>
      </div>

      <div v-if="testResult" class="page-card routing-test-card">
        <div class="routing-panel__header">
          <div>
            <h3 class="routing-panel__title">最近测试结果</h3>
            <div class="routing-panel__subtitle">{{ testScope }}</div>
          </div>
          <StatusTag :tone="testResult.ok ? 'success' : 'warning'" :text="testResult.ok ? '测试成功' : '需要关注'" />
        </div>

        <div class="routing-test-card__summary">
          <span>结果：{{ testResult.message }}</span>
          <span>最终节点：{{ testResult.finalProvider || '-' }}</span>
          <span>最终模型：{{ testResult.finalModel || '-' }}</span>
        </div>

        <div class="table-scroll">
          <el-table :data="testResult.attempts" size="small" style="width: 100%">
            <el-table-column prop="providerId" label="Provider" min-width="150" show-overflow-tooltip />
            <el-table-column prop="model" label="Model" min-width="140" show-overflow-tooltip />
            <el-table-column label="状态" width="120">
              <template #default="{ row }">
                <StatusTag :tone="toneForStatus(row.status)" :text="displayCallStatus(row.status)" />
              </template>
            </el-table-column>
            <el-table-column prop="httpStatus" label="HTTP" width="90" />
            <el-table-column prop="latencyMs" label="耗时" width="100">
              <template #default="{ row }">{{ formatDuration(row.latencyMs) }}</template>
            </el-table-column>
            <el-table-column label="错误类型" min-width="120">
              <template #default="{ row }">{{ row.errorType || '-' }}</template>
            </el-table-column>
            <el-table-column label="备注" min-width="220" show-overflow-tooltip>
              <template #default="{ row }">
                {{ row.skippedByBreaker ? `breaker until ${formatDateTime(row.breakerOpenUntil)}` : row.errorMessage || '-' }}
              </template>
            </el-table-column>
          </el-table>
        </div>
      </div>

      <div class="page-card audit-section">
        <div class="page-header">
          <div>
            <h2 class="page-title" style="font-size: 22px">路由审计</h2>
            <div class="page-subtitle">当前只看 `{{ currentAuditGroup }}` 这组保存与测试记录。</div>
          </div>
        </div>

        <FilterToolbar>
          <el-select v-model="auditAction" clearable placeholder="动作">
            <el-option
              v-for="item in auditActionOptions"
              :key="item.value"
              :label="item.label"
              :value="item.value"
            />
          </el-select>
          <template #actions>
            <el-button @click="resetAuditFilters">重置</el-button>
            <el-button type="primary" :loading="auditsLoading" @click="loadAudits">筛选</el-button>
          </template>
        </FilterToolbar>

        <PageState v-if="auditsLoading && !audits.items.length" mode="loading" title="正在加载审计记录" compact />
        <PageState
          v-else-if="auditsError && !audits.items.length"
          mode="error"
          title="路由审计加载失败"
          :description="auditsError"
          compact
          @retry="loadAudits"
        />
        <PageState
          v-else-if="!audits.items.length"
          mode="empty"
          title="暂无路由审计"
          description="当前筛选条件下还没有保存或测试记录。"
          compact
        />
        <template v-else>
          <div class="table-scroll">
            <el-table :data="audits.items" size="small" style="width: 100%">
              <el-table-column prop="settingKey" label="配置键" min-width="240" show-overflow-tooltip />
              <el-table-column label="动作" width="100">
                <template #default="{ row }">{{ displayAuditAction(row.action) }}</template>
              </el-table-column>
              <el-table-column prop="oldValueMasked" label="旧值" min-width="180" show-overflow-tooltip />
              <el-table-column prop="newValueMasked" label="新值" min-width="180" show-overflow-tooltip />
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
    </template>
  </AppShell>
</template>

<script setup lang="ts">
import { computed, onMounted, ref, watch } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { ElMessage, ElMessageBox } from 'element-plus'
import AppShell from '@/components/AppShell.vue'
import FilterToolbar from '@/components/FilterToolbar.vue'
import PageState from '@/components/PageState.vue'
import StatusTag from '@/components/StatusTag.vue'
import * as adminApi from '@/api/admin'
import type {
  AIRoutingProviderConfig,
  AIRoutingSceneConfig,
  AIRoutingSceneKey,
  AIRoutingSceneSummary,
  AIRoutingTestResult,
  PaginationResult,
  SettingAuditRecord
} from '@/types'
import { buildRouteQuery, readQueryString } from '@/utils/route-query'
import {
  aiRoutingStrategyOptions,
  auditActionOptions,
  displayAIRoutingScene,
  displayAIRoutingStrategy,
  displayAuditAction,
  displayCallStatus,
  displaySettingSource,
  formatDateTime,
  formatDuration,
  toneForStatus
} from '@/utils/admin-display'

const router = useRouter()
const route = useRoute()

const sceneKeys: AIRoutingSceneKey[] = ['summary', 'title', 'flowchart']
const currentSceneKey = ref<AIRoutingSceneKey>('summary')
const sceneSummaries = ref<AIRoutingSceneSummary[]>([])
const remoteScene = ref<AIRoutingSceneConfig | null>(null)
const draftScene = ref<AIRoutingSceneConfig | null>(null)
const testResult = ref<AIRoutingTestResult | null>(null)
const testScope = ref('')

const sceneLoading = ref(false)
const sceneError = ref('')
const pageRefreshing = ref(false)
const savingScene = ref(false)
const testingScene = ref(false)
const singleTestProviderId = ref('')

const audits = ref<PaginationResult<SettingAuditRecord>>({
  items: [],
  total: 0,
  page: 1,
  pageSize: 20
})
const auditsLoading = ref(false)
const auditsError = ref('')
const auditAction = ref('')
const auditPage = ref(1)

const retryOptions = [
  { label: '超时 timeout', value: 'timeout' },
  { label: '网络 network', value: 'network' },
  { label: '限流 rate_limit', value: 'rate_limit' },
  { label: '鉴权 auth', value: 'auth' },
  { label: '上游 upstream', value: 'upstream' },
  { label: '响应异常 invalid_response', value: 'invalid_response' }
]

const currentAuditGroup = computed(() => `ai.routing.${currentSceneKey.value}`)
const enabledProviderCount = computed(() => draftScene.value?.providers.filter((item) => item.enabled).length || 0)
const maxAttemptCeiling = computed(() => Math.max(enabledProviderCount.value || 1, 1))
const compatibilityHint = computed(() => {
  if (!draftScene.value?.compatibilityMode) {
    return ''
  }
  return '当前运行时仍优先走旧单 Provider 配置；保存并启用本场景后，summary / title / flowchart 才会正式切到新的多节点路由。'
})

const sceneCards = computed(() => {
  return sceneKeys.map((scene) => {
    const summary = sceneSummaries.value.find((item) => item.scene === scene)
    return {
      scene,
      title: scene === 'summary' ? '正文总结' : scene === 'title' ? '标题清洗' : '步骤图生成',
      strategy: summary?.strategy || (scene === 'title' ? 'round_robin_failover' : 'priority_failover'),
      providerCount: summary?.providerCount || 0,
      activeProviderCount: summary?.activeProviderCount || 0,
      updatedAt: summary?.updatedAt || '',
      source: summary?.source || 'empty',
      compatibilityMode: summary?.compatibilityMode ?? true,
      tone: scene === currentSceneKey.value ? 'primary' : summary?.compatibilityMode ? 'warning' : summary?.enabled ? 'success' : 'neutral',
      statusText: summary?.compatibilityMode ? '兼容模式' : summary?.enabled ? '正式模式' : '未启用'
    }
  })
})

const isDirty = computed(() => {
  if (!draftScene.value || !remoteScene.value) {
    return false
  }
  return comparableScene(draftScene.value) !== comparableScene(remoteScene.value)
})

onMounted(async () => {
  const queryScene = readQueryString(route.query, 'scene')
  if (sceneKeys.includes(queryScene as AIRoutingSceneKey)) {
    currentSceneKey.value = queryScene as AIRoutingSceneKey
  }
  await refreshPage()
})

watch(
  () => route.query.scene,
  async (value) => {
    const nextScene = String(value || '').trim()
    if (!sceneKeys.includes(nextScene as AIRoutingSceneKey)) {
      return
    }
    if (nextScene === currentSceneKey.value) {
      return
    }
    currentSceneKey.value = nextScene as AIRoutingSceneKey
    resetSceneEditor()
    await loadCurrentScene()
  }
)

async function refreshPage() {
  pageRefreshing.value = true
  resetSceneEditor()
  try {
    await Promise.all([loadSceneSummaries(), loadCurrentScene()])
  } finally {
    pageRefreshing.value = false
  }
}

async function loadSceneSummaries() {
  const response = await adminApi.listAIRoutingScenes()
  sceneSummaries.value = response.items
}

async function loadCurrentScene() {
  sceneLoading.value = true
  sceneError.value = ''
  try {
    const response = await adminApi.getAIRoutingScene(currentSceneKey.value)
    remoteScene.value = hydrateScene(response.scene)
    draftScene.value = hydrateScene(response.scene)
    testResult.value = null
    testScope.value = ''
    auditPage.value = 1
    await loadAudits()
  } catch (error) {
    resetSceneEditor()
    sceneError.value = extractMessage(error)
  } finally {
    sceneLoading.value = false
  }
}

async function loadAudits() {
  auditsLoading.value = true
  auditsError.value = ''
  try {
    const query = new URLSearchParams()
    query.set('group', currentAuditGroup.value)
    query.set('page', String(auditPage.value))
    query.set('pageSize', '20')
    if (auditAction.value) {
      query.set('action', auditAction.value)
    }
    const response = await adminApi.listSettingAudits(query)
    audits.value = response.result
  } catch (error) {
    auditsError.value = extractMessage(error)
  } finally {
    auditsLoading.value = false
  }
}

function hydrateScene(scene: AIRoutingSceneConfig): AIRoutingSceneConfig {
  const clone = JSON.parse(JSON.stringify(scene)) as AIRoutingSceneConfig
  clone.requestOptions ||= { stream: false, temperature: 0, maxTokens: clone.scene === 'title' ? 64 : 0 }
  clone.breaker ||= { failureThreshold: 3, cooldownSeconds: 60 }
  clone.retryOn ||= retryOptions.map((item) => item.value)
  clone.providers = (clone.providers || []).map((provider, index) => ({
    adapter: 'openai-compatible',
    enabled: true,
    priority: (index + 1) * 10,
    timeoutSeconds: 30,
    baseURL: '',
    name: '',
    id: '',
    hasAPIKey: false,
    apiKeyMasked: '',
    ...provider,
    apiKey: '',
    clearApiKey: false
  }))
  return clone
}

function comparableScene(scene: AIRoutingSceneConfig) {
  const payload = buildScenePayload(scene)
  return JSON.stringify(payload)
}

function buildScenePayload(scene: AIRoutingSceneConfig): AIRoutingSceneConfig {
  const payload = JSON.parse(JSON.stringify(scene)) as AIRoutingSceneConfig
  payload.providers = payload.providers.map((provider, index) => ({
    ...provider,
    id: provider.id.trim(),
    name: provider.name.trim(),
    adapter: provider.adapter || 'openai-compatible',
    baseURL: provider.baseURL.trim(),
    model: provider.model.trim(),
    priority: (index + 1) * 10,
    timeoutSeconds: Number(provider.timeoutSeconds) || 30,
    apiKey: (provider.apiKey || '').trim(),
    apiKeyMasked: provider.apiKeyMasked || '',
    hasAPIKey: !!provider.hasAPIKey,
    clearApiKey: !!provider.clearApiKey
  }))
  payload.maxAttempts = Math.min(Math.max(Number(payload.maxAttempts) || 1, 1), Math.max(payload.providers.filter((item) => item.enabled).length, 1))
  payload.breaker.failureThreshold = Math.max(Number(payload.breaker.failureThreshold) || 1, 1)
  payload.breaker.cooldownSeconds = Math.max(Number(payload.breaker.cooldownSeconds) || 5, 5)
  if (payload.scene !== 'title') {
    payload.requestOptions.stream = false
    payload.requestOptions.temperature = 0
    payload.requestOptions.maxTokens = 0
  }
  return payload
}

function createProvider(scene: AIRoutingSceneKey): AIRoutingProviderConfig {
  const seed = `${scene}-${Date.now().toString(36)}-${Math.random().toString(36).slice(2, 6)}`
  return {
    id: seed,
    scene,
    name: '',
    adapter: 'openai-compatible',
    enabled: true,
    priority: 10,
    weight: 100,
    baseURL: '',
    apiKey: '',
    apiKeyMasked: '',
    hasAPIKey: false,
    clearApiKey: false,
    model: '',
    timeoutSeconds: scene === 'flowchart' ? 120 : scene === 'title' ? 5 : 30
  }
}

async function handleSceneChange(scene: AIRoutingSceneKey) {
  if (scene === currentSceneKey.value) {
    return
  }
  currentSceneKey.value = scene
  resetSceneEditor()
  await router.replace({ query: buildRouteQuery({ scene }) })
  await loadCurrentScene()
}

function resetSceneEditor() {
  remoteScene.value = null
  draftScene.value = null
  testResult.value = null
  testScope.value = ''
}

function handleAddProvider() {
  if (!draftScene.value) {
    return
  }
  draftScene.value.providers.push(createProvider(draftScene.value.scene))
}

async function handleRemoveProvider(index: number) {
  if (!draftScene.value) {
    return
  }
  try {
    await ElMessageBox.confirm('删除后该节点会从当前草稿里移除，保存后才会真正生效。', '确认删除节点', {
      type: 'warning'
    })
    draftScene.value.providers.splice(index, 1)
  } catch {
    return
  }
}

function moveProvider(index: number, offset: number) {
  if (!draftScene.value) {
    return
  }
  const target = index + offset
  if (target < 0 || target >= draftScene.value.providers.length) {
    return
  }
  const items = draftScene.value.providers
  const [current] = items.splice(index, 1)
  items.splice(target, 0, current)
}

function handleProviderApiKeyInput(provider: AIRoutingProviderConfig, value: string) {
  provider.apiKey = value
  provider.clearApiKey = value.trim() ? false : provider.clearApiKey
}

function handleClearProviderApiKey(provider: AIRoutingProviderConfig) {
  provider.apiKey = ''
  provider.clearApiKey = true
}

async function handleSaveScene() {
  if (!draftScene.value) {
    return
  }
  savingScene.value = true
  try {
    const response = await adminApi.updateAIRoutingScene(currentSceneKey.value, buildScenePayload(draftScene.value))
    remoteScene.value = hydrateScene(response.scene)
    draftScene.value = hydrateScene(response.scene)
    await loadSceneSummaries()
    await loadAudits()
    ElMessage.success('场景配置已保存')
  } catch (error) {
    ElMessage.error(extractMessage(error))
  } finally {
    savingScene.value = false
  }
}

async function handleTestScene() {
  await runSceneTest('当前草稿测试', draftScene.value)
}

async function handleTestSingleProvider(index: number) {
  if (!draftScene.value) {
    return
  }
  const provider = draftScene.value.providers[index]
  const payload = buildScenePayload(draftScene.value)
  payload.enabled = true
  payload.maxAttempts = 1
  payload.providers = payload.providers.map((item, itemIndex) => ({
    ...item,
    enabled: itemIndex === index
  }))
  singleTestProviderId.value = provider.id
  await runSceneTest(`单节点测试：${provider.name || provider.id}`, payload)
  singleTestProviderId.value = ''
}

async function runSceneTest(scope: string, scene: AIRoutingSceneConfig | null) {
  if (!scene) {
    return
  }
  testingScene.value = true
  try {
    const response = await adminApi.testAIRoutingScene(currentSceneKey.value, buildScenePayload(scene))
    testResult.value = response.result
    testScope.value = scope
    await loadAudits()
    if (response.result.ok) {
      ElMessage.success('路由测试通过')
    } else {
      ElMessage.warning(response.result.message || '路由测试失败')
    }
  } catch (error) {
    ElMessage.error(extractMessage(error))
  } finally {
    testingScene.value = false
  }
}

function resetAuditFilters() {
  auditAction.value = ''
  auditPage.value = 1
  loadAudits()
}

function handleAuditPageChange(page: number) {
  auditPage.value = page
  loadAudits()
}

function extractMessage(error: unknown) {
  return error instanceof Error ? error.message : '请求失败'
}
</script>

<style scoped>
.routing-scene-grid {
  display: grid;
  grid-template-columns: repeat(3, minmax(0, 1fr));
  gap: 16px;
  margin-bottom: 20px;
}

.routing-scene-card {
  width: 100%;
  padding: 18px 18px 20px;
  text-align: left;
  cursor: pointer;
  border: 1px solid rgba(148, 163, 184, 0.18);
  background:
    linear-gradient(180deg, rgba(255, 255, 255, 0.98), rgba(247, 250, 252, 0.94));
  transition:
    border-color 0.22s ease,
    transform 0.22s ease,
    box-shadow 0.22s ease;
}

.routing-scene-card:hover {
  border-color: rgba(37, 99, 235, 0.28);
  transform: translateY(-1px);
  box-shadow: 0 18px 36px rgba(15, 23, 42, 0.08);
}

.routing-scene-card--active {
  border-color: rgba(37, 99, 235, 0.4);
  background:
    linear-gradient(180deg, rgba(239, 246, 255, 0.96), rgba(255, 255, 255, 0.96));
}

.routing-scene-card__header {
  display: flex;
  align-items: flex-start;
  justify-content: space-between;
  gap: 12px;
}

.routing-scene-card__header h3 {
  margin: 6px 0 0;
  font-size: 20px;
  line-height: 1.2;
}

.routing-scene-card__eyebrow {
  color: var(--color-text-subtle);
  font-size: 12px;
  font-weight: 700;
  letter-spacing: 0.08em;
  text-transform: uppercase;
}

.routing-scene-card__meta,
.routing-scene-card__footer {
  display: flex;
  flex-wrap: wrap;
  gap: 10px 16px;
  margin-top: 14px;
  color: var(--color-text-subtle);
  font-size: 13px;
}

.routing-scene-card__footer {
  padding-top: 14px;
  border-top: 1px solid rgba(148, 163, 184, 0.14);
}

.routing-editor-grid {
  display: grid;
  grid-template-columns: minmax(0, 0.95fr) minmax(0, 1.05fr);
  gap: 20px;
}

.routing-panel {
  padding: 22px;
}

.routing-panel__header {
  display: flex;
  align-items: flex-start;
  justify-content: space-between;
  gap: 16px;
  margin-bottom: 18px;
}

.routing-panel__title {
  margin: 0;
  font-size: 22px;
  line-height: 1.2;
}

.routing-panel__subtitle {
  margin-top: 8px;
  color: var(--color-text-subtle);
  font-size: 13px;
  line-height: 1.7;
}

.routing-panel__tags {
  display: flex;
  flex-wrap: wrap;
  gap: 10px;
  justify-content: flex-end;
}

.routing-form-grid {
  display: grid;
  grid-template-columns: repeat(2, minmax(0, 1fr));
  gap: 14px 16px;
}

.routing-form-grid--request {
  margin-top: 16px;
}

.routing-field {
  display: flex;
  flex-direction: column;
  gap: 10px;
}

.routing-field > span {
  color: var(--color-text-subtle);
  font-size: 13px;
  font-weight: 600;
}

.routing-field--meta {
  justify-content: flex-end;
  padding: 16px 18px;
  border-radius: 18px;
  background: rgba(15, 23, 42, 0.04);
}

.routing-field--meta strong {
  font-size: 16px;
}

.routing-field--meta small {
  color: var(--color-text-subtle);
  font-size: 12px;
}

.routing-checkbox-block,
.routing-request-block {
  margin-top: 22px;
  padding-top: 18px;
  border-top: 1px solid rgba(148, 163, 184, 0.14);
}

.routing-checkbox-block__title {
  font-size: 14px;
  font-weight: 700;
}

.routing-checkbox-grid {
  display: grid;
  grid-template-columns: repeat(2, minmax(0, 1fr));
  gap: 10px 14px;
  margin-top: 14px;
}

.routing-request-block__note {
  margin-top: 14px;
  padding: 14px 16px;
  border-radius: 16px;
  background: rgba(37, 99, 235, 0.08);
  color: #1d4ed8;
  font-size: 13px;
  line-height: 1.7;
}

.provider-editor-list {
  display: flex;
  flex-direction: column;
  gap: 14px;
}

.provider-editor-card {
  padding: 18px;
  border-radius: 18px;
  border: 1px solid rgba(148, 163, 184, 0.18);
  background:
    linear-gradient(180deg, rgba(255, 255, 255, 0.98), rgba(248, 250, 252, 0.95));
}

.provider-editor-card__header {
  display: flex;
  align-items: flex-start;
  justify-content: space-between;
  gap: 16px;
  margin-bottom: 16px;
}

.provider-editor-card__title {
  display: flex;
  flex-direction: column;
  gap: 6px;
}

.provider-editor-card__meta {
  display: flex;
  flex-wrap: wrap;
  gap: 8px 12px;
  margin-top: 8px;
  color: var(--color-text-subtle);
  font-size: 12px;
}

.provider-editor-card__actions {
  display: flex;
  flex-wrap: wrap;
  gap: 6px;
  justify-content: flex-end;
}

.provider-editor-grid {
  display: grid;
  grid-template-columns: repeat(2, minmax(0, 1fr));
  gap: 14px 16px;
}

.provider-editor-secret {
  display: flex;
  gap: 12px;
  align-items: flex-end;
  margin-top: 16px;
}

.provider-editor-secret__hint {
  margin-top: 10px;
  color: var(--color-text-subtle);
  font-size: 12px;
  line-height: 1.7;
}

.routing-test-card {
  margin-top: 20px;
  padding: 22px;
}

.routing-test-card__summary {
  display: flex;
  flex-wrap: wrap;
  gap: 10px 18px;
  margin-bottom: 16px;
  color: var(--color-text-subtle);
  font-size: 13px;
}

.routing-alert {
  margin-bottom: 18px;
}

@media (max-width: 1440px) {
  .routing-scene-grid {
    grid-template-columns: repeat(2, minmax(0, 1fr));
  }
}

@media (max-width: 1200px) {
  .routing-editor-grid {
    grid-template-columns: 1fr;
  }
}

@media (max-width: 768px) {
  .routing-scene-grid,
  .routing-form-grid,
  .provider-editor-grid,
  .routing-checkbox-grid {
    grid-template-columns: 1fr;
  }

  .provider-editor-card__header,
  .routing-panel__header,
  .provider-editor-secret {
    flex-direction: column;
    align-items: stretch;
  }

  .provider-editor-card__actions,
  .routing-panel__tags {
    justify-content: flex-start;
  }
}
</style>
