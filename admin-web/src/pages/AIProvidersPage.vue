<template>
  <AppShell>
    <template #toolbar>
      <div class="toolbar-cluster toolbar-cluster--status">
        <button v-if="latestTestSummary" type="button" class="test-result-chip" @click="scrollToTestCard">
          <StatusTag :tone="latestTestSummary.tone" :text="latestTestSummary.text" />
        </button>
      </div>
      <div class="toolbar-cluster toolbar-cluster--meta">
        <el-tooltip content="重新拉取远端配置" placement="bottom">
          <el-button circle :loading="pageRefreshing" aria-label="刷新" @click="refreshPage">
            <el-icon><Refresh /></el-icon>
          </el-button>
        </el-tooltip>
        <el-tooltip :content="discardTooltip" placement="bottom">
          <span class="toolbar-discard-wrap">
            <el-button type="danger" link :disabled="!isDirty" @click="handleDiscardDraft">
              放弃草稿
            </el-button>
          </span>
        </el-tooltip>
      </div>
      <span class="toolbar-divider" aria-hidden="true" />
      <div class="toolbar-cluster toolbar-cluster--action">
        <el-button :loading="testingScene" :disabled="!draftScene" @click="handleTestScene">
          测试当前草稿
        </el-button>
        <el-badge :is-dot="isDirty" class="toolbar-save-dot">
          <el-button
            type="primary"
            :loading="savingScene"
            :disabled="!isDirty || savingScene"
            @click="handleSaveScene"
          >
            保存场景
          </el-button>
        </el-badge>
      </div>
    </template>

    <div class="routing-scene-grid" role="tablist" aria-label="AI 路由场景">
      <button
        v-for="item in sceneCards"
        :key="item.scene"
        :ref="(el) => setSceneCardRef(item.scene, el)"
        type="button"
        class="page-card routing-scene-card"
        :class="{ 'routing-scene-card--active': item.scene === currentSceneKey }"
        role="tab"
        :aria-selected="item.scene === currentSceneKey"
        :tabindex="item.scene === currentSceneKey ? 0 : -1"
        @click="handleSceneChange(item.scene)"
        @keydown.left.prevent="handleSceneArrowKey(-1)"
        @keydown.right.prevent="handleSceneArrowKey(1)"
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
          <span class="routing-scene-card__channel">
            线上链路：<code>{{ sceneEffectiveChannel(item.scene).label }}</code>
          </span>
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

    <el-alert
      class="routing-alert routing-alert--with-action"
      type="info"
      show-icon
      :closable="false"
      title="连续异常邮件告警已接入配置中心"
    >
      <div class="routing-alert__content">
        <span>如果你需要在某个 Provider 连续异常达到阈值后发 QQ 邮箱告警，请前往配置中心的"AI Provider 告警"分组配置阈值、SMTP 和收件人。</span>
        <div class="routing-alert__action">
          <el-button type="primary" link @click="goAlertConfig">
            前往配置
            <el-icon><ArrowRight /></el-icon>
          </el-button>
        </div>
      </div>
    </el-alert>

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

      <div class="routing-breadcrumb" aria-live="polite">
        <span class="routing-breadcrumb__crumbs">
          场景策略
          <el-icon class="routing-breadcrumb__sep"><ArrowRight /></el-icon>
          正在编辑：<strong>{{ currentSceneTitle }}</strong>
          <el-popover placement="bottom-start" :width="340" trigger="click">
            <template #reference>
              <button type="button" class="routing-breadcrumb__channel" :class="`routing-breadcrumb__channel--${currentChannel.tone}`">
                线上链路：<code>{{ currentChannel.label }}</code>
                <el-icon class="routing-breadcrumb__channel-icon"><InfoFilled /></el-icon>
              </button>
            </template>
            <div class="channel-popover">
              <div class="channel-popover__title">当前状态：{{ currentChannel.reason }}</div>
              <table class="channel-popover__table">
                <thead>
                  <tr><th>草稿/正式</th><th>新路由</th><th>实际生效链路</th></tr>
                </thead>
                <tbody>
                  <tr v-for="(row, idx) in channelMatrix" :key="idx" :class="{ 'is-hit': row.hit }">
                    <td>{{ row.draft }}</td>
                    <td>{{ row.toggle }}</td>
                    <td><code>{{ row.effect }}</code></td>
                  </tr>
                </tbody>
              </table>
            </div>
          </el-popover>
        </span>
        <StatusTag v-if="isDirty" tone="warning" :text="`📝 ${diffCount} 项未保存`" />
        <span v-else class="routing-breadcrumb__clean">已同步</span>
      </div>

      <div class="routing-editor-grid">
        <div class="page-card routing-panel routing-panel--strategy">
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
            <label class="routing-field routing-field--with-hint">
              <span>最大尝试次数</span>
              <el-input-number v-model="draftScene.maxAttempts" :min="1" :max="maxAttemptCeiling" />
              <small class="routing-field__hint" :class="{ 'routing-field__hint--warn': numericWarn.maxAttempts }">
                {{ numericWarn.maxAttempts || '建议 2–5 次；过大会让失败降级变慢' }}
              </small>
            </label>
            <label class="routing-field routing-field--with-hint">
              <span>熔断阈值</span>
              <el-input-number v-model="draftScene.breaker.failureThreshold" :min="1" :max="10" />
              <small class="routing-field__hint" :class="{ 'routing-field__hint--warn': numericWarn.failureThreshold }">
                {{ numericWarn.failureThreshold || '连续失败达到该次数后触发熔断，建议 3–10' }}
              </small>
            </label>
            <label class="routing-field routing-field--with-hint">
              <span>冷却时间（秒）</span>
              <el-input-number v-model="draftScene.breaker.cooldownSeconds" :min="5" :max="600" />
              <small class="routing-field__hint" :class="{ 'routing-field__hint--warn': numericWarn.cooldownSeconds }">
                {{ numericWarn.cooldownSeconds || '熔断冷却时长，低于 30s 容易抖动' }}
              </small>
            </label>
            <div class="routing-field routing-field--meta">
              <span>最近修改</span>
              <strong>{{ formatDateTime(draftScene.updatedAt) }}</strong>
              <small>修改人：{{ draftScene.updatedBySubject || '暂无' }}</small>
            </div>
          </div>

          <div class="routing-timeline-hint" aria-label="重试与熔断时序示意">
            <div class="routing-timeline-hint__track">
              <span
                v-for="i in timelineSegments.attempts"
                :key="`att-${i}`"
                class="routing-timeline-hint__seg routing-timeline-hint__seg--attempt"
              >
                试 {{ i }}
              </span>
              <span class="routing-timeline-hint__seg routing-timeline-hint__seg--breaker">
                连续失败 {{ draftScene.breaker.failureThreshold }} 次
              </span>
              <span class="routing-timeline-hint__seg routing-timeline-hint__seg--cooldown">
                冷却 {{ draftScene.breaker.cooldownSeconds }}s
              </span>
            </div>
            <div class="routing-timeline-hint__caption">
              预计首轮最长 ≈ <strong>{{ expectedFirstRoundSeconds }}s</strong>
              <span class="routing-timeline-hint__hint">（最大尝试次数 × 启用节点最大超时）</span>
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
            <div
              v-for="(provider, index) in draftScene.providers"
              :key="getProviderLocalKey(provider)"
              class="provider-editor-card"
              :class="{
                'provider-editor-card--drag-over':
                  dragOverProviderIndex === index &&
                  draggingProviderIndex !== null &&
                  draggingProviderIndex !== index
              }"
              @dragover.prevent="handleProviderDragOver(index)"
              @dragenter.prevent="handleProviderDragOver(index)"
              @drop.prevent="handleProviderDrop(index)"
            >
              <div class="provider-editor-card__header">
                <div>
                  <div class="provider-editor-card__title">
                    <button
                      type="button"
                      class="provider-icon-button provider-collapse-toggle"
                      :aria-expanded="!isProviderCollapsed(provider)"
                      :aria-label="isProviderCollapsed(provider) ? '展开编辑' : '折叠编辑'"
                      @click="toggleProviderCollapsed(provider)"
                    >
                      <el-icon><ArrowDown v-if="!isProviderCollapsed(provider)" /><ArrowRight v-else /></el-icon>
                    </button>
                    <strong>{{ provider.name || `节点 ${index + 1}` }}</strong>
                    <span class="mono-text">{{ provider.id }}</span>
                  </div>
                  <div class="provider-editor-card__meta">
                    <span>顺序 {{ index + 1 }}</span>
                    <span>{{ provider.adapter }}</span>
                    <span v-if="provider.model">Model: {{ provider.model }}</span>
                    <span>Endpoint: {{ provider.endpointMode || 'chat_completions' }}</span>
                    <span v-if="isImageGenerationProvider(provider)">Format: {{ provider.responseFormat || 'auto' }}</span>
                    <span>{{ provider.enabled ? '参与调度' : '已停用' }}</span>
                  </div>
                </div>
                <div class="provider-editor-card__controls">
                  <el-switch v-model="provider.enabled" inline-prompt active-text="开" inactive-text="关" />
                  <div class="provider-editor-card__actions">
                    <el-tooltip content="拖拽排序" placement="top">
                      <button
                        type="button"
                        class="provider-icon-button provider-icon-button--drag"
                        :class="{ 'provider-icon-button--disabled': draftScene.providers.length < 2 }"
                        :disabled="draftScene.providers.length < 2"
                        :draggable="draftScene.providers.length > 1"
                        aria-label="拖拽排序"
                        @dragstart="handleProviderDragStart(index, $event)"
                        @dragend="handleProviderDragEnd"
                      >
                        <el-icon><Rank /></el-icon>
                      </button>
                    </el-tooltip>
                    <el-tooltip content="上移一位" placement="top">
                      <button
                        type="button"
                        class="provider-icon-button"
                        :disabled="index === 0"
                        aria-label="上移一位"
                        @click="moveProvider(index, -1)"
                      >
                        <el-icon><ArrowUp /></el-icon>
                      </button>
                    </el-tooltip>
                    <el-tooltip content="下移一位" placement="top">
                      <button
                        type="button"
                        class="provider-icon-button"
                        :disabled="index === draftScene.providers.length - 1"
                        aria-label="下移一位"
                        @click="moveProvider(index, 1)"
                      >
                        <el-icon><ArrowDown /></el-icon>
                      </button>
                    </el-tooltip>
                    <el-tooltip content="测试当前节点" placement="top">
                      <button
                        type="button"
                        class="provider-icon-button"
                        :disabled="singleTestProviderId === provider.id"
                        aria-label="测试当前节点"
                        @click="handleTestSingleProvider(index)"
                      >
                        <el-icon v-if="singleTestProviderId === provider.id" class="is-loading"><Refresh /></el-icon>
                        <el-icon v-else><Promotion /></el-icon>
                      </button>
                    </el-tooltip>
                    <el-dropdown trigger="click" @command="(command) => handleProviderMenuCommand(String(command), index)">
                      <button type="button" class="provider-icon-button" aria-label="更多操作">
                        <el-icon><MoreFilled /></el-icon>
                      </button>
                      <template #dropdown>
                        <el-dropdown-menu>
                          <el-dropdown-item command="duplicate">复制节点</el-dropdown-item>
                          <el-dropdown-item command="delete" divided>删除节点</el-dropdown-item>
                        </el-dropdown-menu>
                      </template>
                    </el-dropdown>
                  </div>
                </div>
              </div>

              <template v-if="!isProviderCollapsed(provider)">
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
                    <span>Endpoint</span>
                    <el-select v-model="provider.endpointMode" @change="handleEndpointModeChange(provider)">
                      <el-option
                        v-for="item in providerEndpointModeOptions"
                        :key="item.value"
                        :label="item.label"
                        :value="item.value"
                      />
                    </el-select>
                  </label>
                  <label class="routing-field">
                    <span>超时（秒）</span>
                    <el-input-number v-model="provider.timeoutSeconds" :min="1" :max="600" />
                  </label>
                  <label class="routing-field">
                    <span>Adapter</span>
                    <el-input v-model="provider.adapter" disabled />
                  </label>
                  <label class="routing-field">
                    <span>Response Format</span>
                    <el-select v-model="provider.responseFormat" :disabled="!isImageGenerationProvider(provider)">
                      <el-option
                        v-for="item in providerResponseFormatOptions"
                        :key="item.value"
                        :label="item.label"
                        :value="item.value"
                      />
                    </el-select>
                  </label>
                </div>
                <div v-if="isImageGenerationProvider(provider)" class="provider-editor-secret__hint">
                  `images/generations` 当前固定按 `quality=high`、`output_format=png`
                  发请求；这里仅配置返回格式。
                </div>

                <div class="provider-editor-secret">
                <div class="provider-editor-secret__header">
                  <div class="provider-editor-secret__label-row">
                    <span class="provider-editor-secret__label">API Key</span>
                    <span
                      v-if="provider.hasAPIKey && !provider.clearApiKey"
                      class="provider-secret-chip mono-text"
                    >
                      当前密钥 · {{ provider.apiKeyMasked || '已保存' }}
                    </span>
                    <StatusTag v-else-if="provider.clearApiKey" tone="warning" text="已标记清空" />
                    <span v-else class="provider-secret-chip provider-secret-chip--empty">当前未配置密钥</span>
                  </div>
                  <div v-if="provider.hasAPIKey" class="provider-editor-secret__actions">
                    <el-button text :disabled="!!provider.apiKey?.trim()" @click="toggleProviderSecretEditor(provider)">
                      {{ provider.apiKey?.trim() ? '已录入新密钥' : shouldShowProviderSecretEditor(provider) ? '收起更换' : '更换密钥' }}
                    </el-button>
                    <el-button text type="danger" @click="handleClearProviderApiKey(provider)">
                      {{ provider.clearApiKey ? '撤销清空' : '清空密钥' }}
                    </el-button>
                  </div>
                </div>

                <label v-if="shouldShowProviderSecretEditor(provider)" class="routing-field provider-editor-secret__field">
                  <span>{{ provider.hasAPIKey ? '输入新密钥' : '录入密钥' }}</span>
                  <el-input
                    v-model="provider.apiKey"
                    type="password"
                    show-password
                    :placeholder="provider.hasAPIKey ? '输入新密钥，保存后覆盖旧值' : '输入当前节点要使用的密钥'"
                    @update:model-value="handleProviderApiKeyInput(provider, $event)"
                  />
                </label>
              </div>
              <div class="provider-editor-secret__hint">
                <template v-if="provider.clearApiKey">当前已标记为待清空，保存后会彻底移除旧密钥。</template>
                <template v-else-if="provider.apiKey?.trim()">已录入新的密钥草稿，保存后会覆盖当前值。</template>
                <template v-else-if="provider.hasAPIKey">当前已保存密钥；不输入新值则继续保留旧值。</template>
                <template v-else>当前没有已保存密钥，可直接录入新值。</template>
              </div>
              </template>
            </div>
          </div>
        </div>
      </div>

      <div v-if="testResult" ref="testCardRef" class="page-card routing-test-card">
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

          <div class="pagination-row">
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
import { computed, nextTick, onBeforeUnmount, onMounted, ref, watch } from 'vue'
import { onBeforeRouteLeave, useRoute, useRouter } from 'vue-router'
import { ElMessage, ElMessageBox } from 'element-plus'
import { ArrowDown, ArrowRight, ArrowUp, InfoFilled, MoreFilled, Promotion, Rank, Refresh } from '@element-plus/icons-vue'
import AppShell from '@/components/AppShell.vue'
import FilterToolbar from '@/components/FilterToolbar.vue'
import PageState from '@/components/PageState.vue'
import StatusTag from '@/components/StatusTag.vue'
import * as adminApi from '@/api/admin'
import type {
  AIRoutingProviderEndpointMode,
  AIRoutingProviderConfig,
  AIRoutingProviderResponseFormat,
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
const testCardRef = ref<HTMLElement | null>(null)
const routeSceneOverride = ref<AIRoutingSceneKey | null>(null)
const draggingProviderIndex = ref<number | null>(null)
const dragOverProviderIndex = ref<number | null>(null)
const sceneCardRefs: Partial<Record<AIRoutingSceneKey, HTMLButtonElement | null>> = {}

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
const providerSecretEditorState = ref<Record<string, boolean>>({})
const collapsedProviderKeys = ref<Set<string>>(new Set())

const providerLocalKeys = new WeakMap<AIRoutingProviderConfig, string>()
let providerLocalKeyCounter = 0

const retryOptions = [
  { label: '超时 timeout', value: 'timeout' },
  { label: '网络 network', value: 'network' },
  { label: '限流 rate_limit', value: 'rate_limit' },
  { label: '鉴权 auth', value: 'auth' },
  { label: '上游 upstream', value: 'upstream' },
  { label: '响应异常 invalid_response', value: 'invalid_response' }
]

const providerEndpointModeOptions: Array<{ label: string; value: AIRoutingProviderEndpointMode }> = [
  { label: 'chat/completions', value: 'chat_completions' },
  { label: 'images/generations', value: 'images_generations' }
]

const providerResponseFormatOptions: Array<{ label: string; value: AIRoutingProviderResponseFormat }> = [
  { label: 'auto', value: 'auto' },
  { label: 'image_url', value: 'image_url' },
  { label: 'b64_json', value: 'b64_json' }
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

const currentSceneTitle = computed(() => {
  const card = sceneCards.value.find((item) => item.scene === currentSceneKey.value)
  return card?.title || displayAIRoutingScene(currentSceneKey.value)
})

function isImageGenerationProvider(provider: AIRoutingProviderConfig) {
  return (provider.endpointMode || 'chat_completions') === 'images_generations'
}

function handleEndpointModeChange(provider: AIRoutingProviderConfig) {
  if (!isImageGenerationProvider(provider)) {
    provider.responseFormat = 'auto'
  } else if (!provider.responseFormat) {
    provider.responseFormat = 'auto'
  }
}

function goAlertConfig() {
  router.push({ path: '/settings', query: { group: 'ai.provider_alert' }, hash: '#ai-provider-alert' })
}

function effectiveChannel(params: { scene: AIRoutingSceneKey; enabled: boolean; compatibilityMode: boolean; isDraftDirty?: boolean }) {
  const { scene, enabled, compatibilityMode, isDraftDirty } = params
  if (isDraftDirty) {
    return {
      label: `${scene}-draft`,
      tone: 'info' as const,
      reason: '草稿未保存 · 线上保持不变，仅测试入口生效'
    }
  }
  if (compatibilityMode) {
    return { label: `${scene}-compat`, tone: 'warning' as const, reason: '兼容模式 · 走旧单 Provider 链路' }
  }
  if (!enabled) {
    return { label: `${scene}-compat`, tone: 'neutral' as const, reason: '新路由未启用 · 回退兼容链路' }
  }
  return { label: `${scene}-v2`, tone: 'success' as const, reason: '线上走新多节点路由' }
}

function sceneEffectiveChannel(scene: AIRoutingSceneKey) {
  const summary = sceneSummaries.value.find((item) => item.scene === scene)
  const isCurrent = scene === currentSceneKey.value && !!draftScene.value
  const enabled = isCurrent && draftScene.value ? draftScene.value.enabled : summary?.enabled ?? false
  const compatibilityMode = isCurrent && draftScene.value
    ? draftScene.value.compatibilityMode
    : summary?.compatibilityMode ?? true
  const isDraftDirty = isCurrent && isDirty.value
  return effectiveChannel({ scene, enabled, compatibilityMode, isDraftDirty })
}

const currentChannel = computed(() => sceneEffectiveChannel(currentSceneKey.value))

const channelMatrix = computed(() => [
  { draft: '正式', toggle: '开', effect: `${currentSceneKey.value}-v2`, hit: !isDirty.value && draftScene.value?.enabled && !draftScene.value?.compatibilityMode },
  { draft: '正式', toggle: '关', effect: `${currentSceneKey.value}-compat`, hit: !isDirty.value && (!draftScene.value?.enabled || draftScene.value?.compatibilityMode) },
  { draft: '草稿', toggle: '—', effect: '仅测试入口生效', hit: isDirty.value }
])

const numericWarn = computed(() => {
  const scene = draftScene.value
  const warn: { maxAttempts: string; failureThreshold: string; cooldownSeconds: string } = {
    maxAttempts: '',
    failureThreshold: '',
    cooldownSeconds: ''
  }
  if (!scene) return warn
  const ma = Number(scene.maxAttempts) || 0
  if (ma > 5) warn.maxAttempts = `当前 ${ma} 次偏大，失败降级会变慢`
  else if (ma < 2) warn.maxAttempts = '至少 2 次才能触发节点切换'
  const ft = Number(scene.breaker?.failureThreshold) || 0
  if (ft < 3) warn.failureThreshold = '阈值过低容易误熔断'
  else if (ft > 10) warn.failureThreshold = '阈值过高将延迟熔断保护'
  const cs = Number(scene.breaker?.cooldownSeconds) || 0
  if (cs < 30) warn.cooldownSeconds = '低于 30s 容易抖动'
  else if (cs > 300) warn.cooldownSeconds = '超过 5 分钟可能影响恢复'
  return warn
})

const timelineSegments = computed(() => {
  const attempts = Math.max(Number(draftScene.value?.maxAttempts) || 1, 1)
  return { attempts: Math.min(attempts, 6) }
})

const expectedFirstRoundSeconds = computed(() => {
  const scene = draftScene.value
  if (!scene) return 0
  const attempts = Math.max(Number(scene.maxAttempts) || 1, 1)
  const enabled = scene.providers.filter((p) => p.enabled)
  const maxTimeout = enabled.reduce((acc, p) => Math.max(acc, Number(p.timeoutSeconds) || 0), 0) || 30
  return attempts * maxTimeout
})

const latestTestSummary = computed(() => {
  if (!testResult.value) {
    return null
  }
  const scopePrefix = testScope.value.includes('单节点') ? '单节点' : '草稿'
  const target = testResult.value.finalProvider || '查看详情'
  return {
    tone: testResult.value.ok ? 'success' : 'warning',
    text: `${scopePrefix}${testResult.value.ok ? '测试通过' : '测试异常'} · ${target}`
  }
})

const isDirty = computed(() => {
  if (!draftScene.value || !remoteScene.value) {
    return false
  }
  return comparableScene(draftScene.value) !== comparableScene(remoteScene.value)
})

function diffProviderKey(provider: Record<string, unknown>, index: number) {
  const id = String(provider.id || '').trim()
  return id || `__index_${index}`
}

function providerDiffSnapshot(provider: Record<string, unknown>) {
  return {
    id: String(provider.id || '').trim(),
    name: String(provider.name || '').trim(),
    adapter: String(provider.adapter || '').trim(),
    enabled: Boolean(provider.enabled),
    baseURL: String(provider.baseURL || '').trim(),
    model: String(provider.model || '').trim(),
    timeoutSeconds: Number(provider.timeoutSeconds) || 0,
    endpointMode: String(provider.endpointMode || 'chat_completions').trim(),
    responseFormat: String(provider.responseFormat || 'auto').trim(),
    clearApiKey: Boolean(provider.clearApiKey),
    apiKey: typeof provider.apiKey === 'string' && provider.apiKey.trim() ? '[已录入]' : ''
  }
}

const sceneDiff = computed<Array<{ scope: string; path: string; from: unknown; to: unknown }>>(() => {
  if (!draftScene.value || !remoteScene.value) return []
  const a = buildScenePayload(draftScene.value) as unknown as Record<string, unknown>
  const b = buildScenePayload(remoteScene.value) as unknown as Record<string, unknown>
  const results: Array<{ scope: string; path: string; from: unknown; to: unknown }> = []
  const sceneFieldKeys = ['enabled', 'strategy', 'maxAttempts', 'retryOn', 'breaker', 'requestOptions'] as const
  for (const key of sceneFieldKeys) {
    if (JSON.stringify(a[key]) !== JSON.stringify(b[key])) {
      results.push({ scope: 'scene', path: key, from: b[key], to: a[key] })
    }
  }
  const aps = (a.providers as Array<Record<string, unknown>>) || []
  const bps = (b.providers as Array<Record<string, unknown>>) || []
  const aEntries = aps.map((provider, index) => [diffProviderKey(provider, index), provider] as const)
  const bEntries = bps.map((provider, index) => [diffProviderKey(provider, index), provider] as const)
  const aMap = new Map(aEntries)
  const bMap = new Map(bEntries)
  const aKeys = aEntries.map(([key]) => key)
  const bKeys = bEntries.map(([key]) => key)
  const sameProviderSet = aKeys.length === bKeys.length && aKeys.every((key) => bMap.has(key))
  if (sameProviderSet && JSON.stringify(aKeys) !== JSON.stringify(bKeys)) {
    results.push({
      scope: 'providers',
      path: 'order',
      from: bKeys,
      to: aKeys
    })
  }
  const providerKeys = Array.from(new Set([...bKeys, ...aKeys]))
  for (const key of providerKeys) {
    const ap = aMap.get(key)
    const bp = bMap.get(key)
    if (!ap || !bp) {
      results.push({
        scope: `provider:${key}`,
        path: ap ? 'added' : 'removed',
        from: bp ? providerDiffSnapshot(bp) : null,
        to: ap ? providerDiffSnapshot(ap) : null
      })
      continue
    }
    const nextProvider = providerDiffSnapshot(ap)
    const prevProvider = providerDiffSnapshot(bp)
    const fields = Object.keys({ ...prevProvider, ...nextProvider })
    for (const field of fields) {
      if (JSON.stringify(nextProvider[field as keyof typeof nextProvider]) !== JSON.stringify(prevProvider[field as keyof typeof prevProvider])) {
        results.push({
          scope: `provider:${key}`,
          path: field,
          from: prevProvider[field as keyof typeof prevProvider],
          to: nextProvider[field as keyof typeof nextProvider]
        })
      }
    }
  }
  return results
})

const diffCount = computed(() => sceneDiff.value.length)

const discardTooltip = computed(() => {
  if (!isDirty.value) return '当前没有未保存改动'
  const n = diffCount.value
  return n > 0 ? `将丢弃 ${n} 项未保存改动` : '将丢弃当前未保存改动'
})

function onBeforeUnload(e: BeforeUnloadEvent) {
  if (isDirty.value) {
    e.preventDefault()
    e.returnValue = ''
  }
}

onBeforeRouteLeave(async () => {
  return await guardUnsavedChanges('离开页面')
})

onMounted(async () => {
  window.addEventListener('beforeunload', onBeforeUnload)
  const queryScene = readQueryString(route.query, 'scene')
  if (sceneKeys.includes(queryScene as AIRoutingSceneKey)) {
    currentSceneKey.value = queryScene as AIRoutingSceneKey
  }
  await refreshPage()
})

onBeforeUnmount(() => {
  window.removeEventListener('beforeunload', onBeforeUnload)
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
    if (routeSceneOverride.value === nextScene) {
      routeSceneOverride.value = null
      await applySceneChange(nextScene as AIRoutingSceneKey)
      return
    }
    const allowed = await guardUnsavedChanges(`切换到「${displayAIRoutingScene(nextScene)}」`)
    if (!allowed) {
      await router.replace({ query: buildRouteQuery({ scene: currentSceneKey.value }) })
      return
    }
    await applySceneChange(nextScene as AIRoutingSceneKey)
  }
)

async function refreshPage() {
  const allowed = await guardUnsavedChanges('刷新页面')
  if (!allowed) {
    return
  }
  pageRefreshing.value = true
  resetSceneEditor()
  try {
    await Promise.all([loadSceneSummaries(), loadCurrentScene()])
  } finally {
    pageRefreshing.value = false
  }
}

async function handleDiscardDraft() {
  if (!remoteScene.value || !isDirty.value) return
  try {
    await ElMessageBox.confirm('确定放弃所有未保存的改动？', '放弃草稿', { type: 'warning' })
    discardDraftChanges('已恢复到上次保存的状态')
  } catch {
    return
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
    resetProviderUIState()
    testResult.value = null
    testScope.value = ''
    auditPage.value = 1
    void loadAudits()
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
    extra: provider.extra || {},
    endpointMode: (provider.endpointMode || 'chat_completions') as AIRoutingProviderEndpointMode,
    responseFormat: provider.endpointMode === 'images_generations' ? (provider.responseFormat || 'auto') : 'auto',
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
    endpointMode: provider.endpointMode || 'chat_completions',
    responseFormat: provider.endpointMode === 'images_generations' ? (provider.responseFormat || 'auto') : 'auto',
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
    timeoutSeconds: scene === 'flowchart' ? 120 : scene === 'title' ? 5 : 30,
    endpointMode: scene === 'flowchart' ? 'images_generations' : 'chat_completions',
    responseFormat: scene === 'flowchart' ? 'b64_json' : 'auto',
    extra: {}
  }
}

async function handleSceneChange(scene: AIRoutingSceneKey) {
  if (scene === currentSceneKey.value) {
    return
  }
  const allowed = await guardUnsavedChanges(`切换到「${displayAIRoutingScene(scene)}」`)
  if (!allowed) {
    return
  }
  routeSceneOverride.value = scene
  await router.replace({ query: buildRouteQuery({ scene }) })
}

async function applySceneChange(scene: AIRoutingSceneKey) {
  currentSceneKey.value = scene
  resetSceneEditor()
  await loadCurrentScene()
  focusSceneCard(scene)
}

function resetSceneEditor() {
  remoteScene.value = null
  draftScene.value = null
  testResult.value = null
  testScope.value = ''
  resetProviderUIState()
}

function handleAddProvider() {
  if (!draftScene.value) {
    return
  }
  const provider = createProvider(draftScene.value.scene)
  draftScene.value.providers.push(provider)
  setProviderSecretEditor(provider, true)
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

function handleProviderMenuCommand(command: string, index: number) {
  if (command === 'duplicate') {
    handleDuplicateProvider(index)
    return
  }
  if (command === 'delete') {
    handleRemoveProvider(index)
  }
}

function handleDuplicateProvider(index: number) {
  if (!draftScene.value) {
    return
  }
  const source = draftScene.value.providers[index]
  const copied = JSON.parse(JSON.stringify(source)) as AIRoutingProviderConfig
  copied.id = buildDuplicateProviderId(source.id || `provider-${index + 1}`)
  copied.name = source.name ? `${source.name} 副本` : ''
  copied.apiKey = ''
  copied.apiKeyMasked = ''
  copied.hasAPIKey = false
  copied.clearApiKey = false
  draftScene.value.providers.splice(index + 1, 0, copied)
  setProviderSecretEditor(copied, true)
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

function handleProviderDragStart(index: number, event: DragEvent) {
  draggingProviderIndex.value = index
  dragOverProviderIndex.value = index
  if (event.dataTransfer) {
    event.dataTransfer.effectAllowed = 'move'
    event.dataTransfer.setData('text/plain', String(index))
  }
}

function handleProviderDragOver(index: number) {
  if (draggingProviderIndex.value === null || draggingProviderIndex.value === index) {
    return
  }
  dragOverProviderIndex.value = index
}

function handleProviderDrop(index: number) {
  if (!draftScene.value || draggingProviderIndex.value === null) {
    handleProviderDragEnd()
    return
  }
  const sourceIndex = draggingProviderIndex.value
  if (sourceIndex !== index) {
    const items = draftScene.value.providers
    const [current] = items.splice(sourceIndex, 1)
    items.splice(index, 0, current)
  }
  handleProviderDragEnd()
}

function handleProviderDragEnd() {
  draggingProviderIndex.value = null
  dragOverProviderIndex.value = null
}

function handleProviderApiKeyInput(provider: AIRoutingProviderConfig, value: string) {
  provider.apiKey = value
  if (value.trim()) {
    provider.clearApiKey = false
    setProviderSecretEditor(provider, true)
  }
}

async function handleClearProviderApiKey(provider: AIRoutingProviderConfig) {
  if (provider.clearApiKey) {
    provider.clearApiKey = false
    ElMessage.info('已撤销清空密钥')
    return
  }
  try {
    await ElMessageBox.confirm('清空后需要保存当前场景才会真正移除旧密钥，是否继续？', '确认清空密钥', {
      type: 'warning'
    })
  } catch {
    return
  }
  provider.apiKey = ''
  provider.clearApiKey = true
  setProviderSecretEditor(provider, false)
  ElMessage.warning('已标记为清空密钥，保存后生效')
}

async function handleSaveScene() {
  await saveCurrentScene()
}

async function saveCurrentScene(successMessage = '场景配置已保存') {
  if (!draftScene.value) {
    return false
  }
  savingScene.value = true
  try {
    const response = await adminApi.updateAIRoutingScene(currentSceneKey.value, buildScenePayload(draftScene.value))
    remoteScene.value = hydrateScene(response.scene)
    draftScene.value = hydrateScene(response.scene)
    resetProviderUIState()
    await loadSceneSummaries()
    await loadAudits()
    ElMessage.success(successMessage)
    return true
  } catch (error) {
    ElMessage.error(extractMessage(error))
    return false
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
    await nextTick()
  } catch (error) {
    ElMessage.error(extractMessage(error))
  } finally {
    testingScene.value = false
  }
}

function getProviderLocalKey(provider: AIRoutingProviderConfig) {
  const existing = providerLocalKeys.get(provider)
  if (existing) {
    return existing
  }
  const key = `provider-local-${providerLocalKeyCounter += 1}`
  providerLocalKeys.set(provider, key)
  return key
}

function setProviderSecretEditor(provider: AIRoutingProviderConfig, open: boolean) {
  providerSecretEditorState.value = {
    ...providerSecretEditorState.value,
    [getProviderLocalKey(provider)]: open
  }
}

function toggleProviderSecretEditor(provider: AIRoutingProviderConfig) {
  if (provider.clearApiKey) {
    provider.clearApiKey = false
  }
  setProviderSecretEditor(provider, !shouldShowProviderSecretEditor(provider))
}

function shouldShowProviderSecretEditor(provider: AIRoutingProviderConfig) {
  return !provider.hasAPIKey || !!provider.apiKey?.trim() || !!providerSecretEditorState.value[getProviderLocalKey(provider)]
}

function isProviderCollapsed(provider: AIRoutingProviderConfig) {
  return collapsedProviderKeys.value.has(getProviderLocalKey(provider))
}

function toggleProviderCollapsed(provider: AIRoutingProviderConfig) {
  const key = getProviderLocalKey(provider)
  const next = new Set(collapsedProviderKeys.value)
  if (next.has(key)) next.delete(key)
  else next.add(key)
  collapsedProviderKeys.value = next
}

function resetProviderUIState() {
  providerSecretEditorState.value = {}
  handleProviderDragEnd()
  if (draftScene.value && draftScene.value.providers.length > 3) {
    const keys = new Set<string>()
    draftScene.value.providers.forEach((p, idx) => {
      if (idx > 0) keys.add(getProviderLocalKey(p))
    })
    collapsedProviderKeys.value = keys
  } else {
    collapsedProviderKeys.value = new Set()
  }
}

function discardDraftChanges(message?: string) {
  if (!remoteScene.value) {
    return
  }
  draftScene.value = hydrateScene(remoteScene.value)
  testResult.value = null
  testScope.value = ''
  resetProviderUIState()
  if (message) {
    ElMessage.info(message)
  }
}

async function guardUnsavedChanges(actionLabel: string) {
  if (!isDirty.value) {
    return true
  }
  const decision = await resolveUnsavedDraftAction(actionLabel)
  if (decision === 'cancel') {
    return false
  }
  if (decision === 'save') {
    return await saveCurrentScene(`${actionLabel}前已保存当前场景`)
  }
  discardDraftChanges('已放弃未保存草稿')
  return true
}

async function resolveUnsavedDraftAction(actionLabel: string): Promise<'save' | 'discard' | 'cancel'> {
  try {
    await ElMessageBox.confirm(`当前场景有未保存的改动。${actionLabel}前，请先处理这些草稿。`, '处理未保存草稿', {
      type: 'warning',
      confirmButtonText: '保存并继续',
      cancelButtonText: '放弃更改',
      distinguishCancelAndClose: true,
      closeOnClickModal: false,
      closeOnPressEscape: false
    })
    return 'save'
  } catch (error) {
    return error === 'cancel' ? 'discard' : 'cancel'
  }
}

function buildDuplicateProviderId(seed: string) {
  if (!draftScene.value) {
    return `${seed}-copy`
  }
  const used = new Set(draftScene.value.providers.map((item) => item.id))
  let candidate = `${seed}-copy`
  let index = 2
  while (used.has(candidate)) {
    candidate = `${seed}-copy-${index}`
    index += 1
  }
  return candidate
}

function scrollToTestCard() {
  nextTick(() => {
    testCardRef.value?.scrollIntoView({ behavior: 'smooth', block: 'start' })
  })
}

function handleSceneArrowKey(offset: number) {
  const currentIndex = sceneKeys.indexOf(currentSceneKey.value)
  if (currentIndex < 0) {
    return
  }
  const nextIndex = (currentIndex + offset + sceneKeys.length) % sceneKeys.length
  handleSceneChange(sceneKeys[nextIndex])
}

function setSceneCardRef(scene: AIRoutingSceneKey, element: Element | null) {
  sceneCardRefs[scene] = element instanceof HTMLButtonElement ? element : null
}

function focusSceneCard(scene: AIRoutingSceneKey) {
  nextTick(() => {
    sceneCardRefs[scene]?.focus()
  })
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
.toolbar-cluster {
  display: inline-flex;
  align-items: center;
  gap: 8px;
}

.toolbar-cluster--action {
  gap: 12px;
}

.toolbar-divider {
  display: inline-block;
  width: 1px;
  height: 20px;
  margin: 0 4px;
  background: rgba(148, 163, 184, 0.35);
}

.toolbar-discard-wrap {
  display: inline-flex;
}

.toolbar-save-dot :deep(.el-badge__content.is-dot) {
  width: 8px;
  height: 8px;
  background: var(--color-primary, #409eff);
  box-shadow: 0 0 0 2px #fff;
  right: 6px;
  top: 4px;
}

.routing-scene-grid {
  display: grid;
  grid-template-columns: repeat(3, minmax(0, 1fr));
  gap: 16px;
  margin-bottom: 20px;
}

.routing-scene-card {
  position: relative;
  width: 100%;
  padding: 18px 18px 20px;
  text-align: left;
  cursor: pointer;
  overflow: hidden;
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
  box-shadow: 0 18px 40px rgba(37, 99, 235, 0.08);
}

.routing-scene-card--active::before {
  content: '';
  position: absolute;
  inset: 0 auto 0 0;
  width: 4px;
  background: linear-gradient(180deg, #2563eb, #60a5fa);
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
  font-weight: 600;
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

.routing-breadcrumb {
  --routing-breadcrumb-h: 44px;
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 12px;
  min-height: var(--routing-breadcrumb-h);
  padding: 8px 16px;
  margin-bottom: 16px;
  border-radius: 12px;
  background: color-mix(in srgb, var(--color-bg-elevated, #ffffff) 85%, transparent);
  border: 1px solid rgba(148, 163, 184, 0.18);
  backdrop-filter: blur(10px);
  font-size: 13px;
  color: var(--color-text-subtle, #64748b);
}

.routing-breadcrumb__crumbs {
  display: inline-flex;
  align-items: center;
  gap: 6px;
}

.routing-breadcrumb__crumbs strong {
  color: var(--color-text, #1f2937);
  font-weight: 600;
}

.routing-breadcrumb__sep {
  font-size: 12px;
  color: var(--color-text-subtle, #94a3b8);
}

.routing-breadcrumb__clean {
  color: var(--color-success, #10b981);
  font-weight: 500;
}

.routing-breadcrumb__channel {
  display: inline-flex;
  align-items: center;
  gap: 4px;
  margin-left: 8px;
  padding: 2px 8px;
  border-radius: 999px;
  border: 1px solid rgba(148, 163, 184, 0.3);
  background: #fff;
  font-size: 12px;
  color: var(--color-text, #1f2937);
  cursor: pointer;
  line-height: 1.5;
}

.routing-breadcrumb__channel code {
  font-size: 12px;
  background: transparent;
  padding: 0;
}

.routing-breadcrumb__channel-icon {
  font-size: 12px;
  color: var(--color-text-subtle, #94a3b8);
}

.routing-breadcrumb__channel--success {
  border-color: color-mix(in srgb, var(--color-success, #10b981) 40%, transparent);
  color: var(--color-success, #10b981);
}

.routing-breadcrumb__channel--warning {
  border-color: color-mix(in srgb, var(--color-warning, #d97706) 40%, transparent);
  color: var(--color-warning, #d97706);
}

.routing-breadcrumb__channel--info {
  border-color: color-mix(in srgb, var(--color-info, #0ea5e9) 40%, transparent);
  color: var(--color-info, #0ea5e9);
}

.channel-popover__title {
  font-size: 13px;
  margin-bottom: 10px;
  color: var(--color-text, #1f2937);
}

.channel-popover__table {
  width: 100%;
  border-collapse: collapse;
  font-size: 12px;
}

.channel-popover__table th,
.channel-popover__table td {
  padding: 6px 8px;
  text-align: left;
  border-bottom: 1px solid rgba(148, 163, 184, 0.18);
}

.channel-popover__table th {
  color: var(--color-text-subtle, #94a3b8);
  font-weight: 500;
}

.channel-popover__table tr.is-hit {
  background: color-mix(in srgb, var(--color-primary, #409eff) 10%, transparent);
}

.channel-popover__table tr.is-hit td {
  color: var(--color-primary, #409eff);
  font-weight: 600;
}

.routing-scene-card__channel code {
  font-size: 12px;
  padding: 1px 6px;
  border-radius: 4px;
  background: rgba(148, 163, 184, 0.14);
}

.routing-field--with-hint {
  display: flex;
  flex-direction: column;
  gap: 6px;
}

.routing-field__hint {
  font-size: 12px;
  line-height: 1.5;
  color: var(--color-text-subtle, #94a3b8);
}

.routing-field__hint--warn {
  color: var(--color-warning, #d97706);
}

.routing-timeline-hint {
  margin: 16px 0 4px;
  padding: 12px 14px;
  border-radius: 12px;
  background: rgba(148, 163, 184, 0.08);
  border: 1px dashed rgba(148, 163, 184, 0.3);
}

.routing-timeline-hint__track {
  display: flex;
  align-items: center;
  flex-wrap: wrap;
  gap: 6px;
  font-size: 12px;
}

.routing-timeline-hint__seg {
  display: inline-flex;
  align-items: center;
  padding: 4px 10px;
  border-radius: 999px;
  background: #fff;
  border: 1px solid rgba(148, 163, 184, 0.25);
  color: var(--color-text-subtle, #64748b);
  white-space: nowrap;
}

.routing-timeline-hint__seg--attempt {
  color: var(--color-primary, #409eff);
  border-color: color-mix(in srgb, var(--color-primary, #409eff) 30%, transparent);
}

.routing-timeline-hint__seg--breaker {
  color: var(--color-warning, #d97706);
  border-color: color-mix(in srgb, var(--color-warning, #d97706) 35%, transparent);
}

.routing-timeline-hint__seg--cooldown {
  color: var(--color-info, #0ea5e9);
  border-color: color-mix(in srgb, var(--color-info, #0ea5e9) 35%, transparent);
}

.routing-timeline-hint__caption {
  margin-top: 8px;
  font-size: 12px;
  color: var(--color-text-subtle, #64748b);
}

.routing-timeline-hint__caption strong {
  color: var(--color-text, #1f2937);
  font-weight: 600;
}

.routing-timeline-hint__hint {
  margin-left: 6px;
  opacity: 0.75;
}

.routing-panel {
  padding: 22px;
}

.provider-collapse-toggle {
  margin-right: 6px;
}

@media (min-width: 1200px) {
  .routing-editor-grid > .routing-panel--strategy {
    position: sticky;
    top: 24px;
    align-self: start;
    max-height: calc(100vh - 120px);
    overflow: auto;
  }
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
  transition:
    border-color 0.18s ease,
    box-shadow 0.18s ease,
    transform 0.18s ease;
}

.provider-editor-card--drag-over {
  border-color: rgba(37, 99, 235, 0.34);
  box-shadow: 0 16px 30px rgba(37, 99, 235, 0.08);
  transform: translateY(-1px);
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

.provider-editor-card__controls {
  display: flex;
  flex-wrap: wrap;
  gap: 10px;
  align-items: center;
  justify-content: flex-end;
}

.provider-editor-card__actions {
  display: flex;
  align-items: center;
  flex-wrap: wrap;
  gap: 8px;
}

.provider-icon-button {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  width: 34px;
  height: 34px;
  padding: 0;
  border: 1px solid rgba(148, 163, 184, 0.22);
  border-radius: 10px;
  background: rgba(255, 255, 255, 0.92);
  color: var(--color-text-subtle);
  cursor: pointer;
  transition:
    border-color 0.18s ease,
    color 0.18s ease,
    background 0.18s ease,
    transform 0.18s ease;
}

.provider-icon-button:hover:not(:disabled) {
  border-color: rgba(37, 99, 235, 0.3);
  color: var(--color-primary);
  background: rgba(239, 246, 255, 0.96);
  transform: translateY(-1px);
}

.provider-icon-button:disabled,
.provider-icon-button--disabled {
  cursor: not-allowed;
  opacity: 0.45;
}

.provider-icon-button--drag {
  cursor: grab;
}

.provider-icon-button--drag:active {
  cursor: grabbing;
}

.provider-editor-grid {
  display: grid;
  grid-template-columns: repeat(2, minmax(0, 1fr));
  gap: 14px 16px;
}

.provider-editor-secret {
  margin-top: 16px;
  padding: 14px 16px;
  border-radius: 16px;
  border: 1px solid rgba(148, 163, 184, 0.14);
  background: rgba(248, 250, 252, 0.86);
}

.provider-editor-secret__header {
  display: flex;
  flex-wrap: wrap;
  gap: 12px;
  align-items: center;
  justify-content: space-between;
}

.provider-editor-secret__label-row {
  display: flex;
  flex-wrap: wrap;
  gap: 10px;
  align-items: center;
}

.provider-editor-secret__label {
  color: var(--color-text-subtle);
  font-size: 13px;
  font-weight: 700;
}

.provider-secret-chip {
  display: inline-flex;
  align-items: center;
  min-height: 28px;
  padding: 0 12px;
  border-radius: 999px;
  background: rgba(15, 23, 42, 0.06);
  color: var(--color-text-strong);
  font-size: 12px;
  font-weight: 600;
}

.provider-secret-chip--empty {
  background: rgba(148, 163, 184, 0.12);
  color: var(--color-text-subtle);
}

.provider-editor-secret__actions {
  display: flex;
  flex-wrap: wrap;
  gap: 6px;
}

.provider-editor-secret__field {
  margin-top: 14px;
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

.test-result-chip {
  display: inline-flex;
  align-items: center;
  padding: 0;
  border: none;
  background: transparent;
  cursor: pointer;
}

.routing-alert {
  margin-bottom: 18px;
}

.routing-alert--with-action {
  position: relative;
  padding-right: 120px;
}

.routing-alert__content {
  min-height: 24px;
}

.routing-alert__action {
  position: absolute;
  right: 16px;
  top: 50%;
  transform: translateY(-50%);
}

@media (max-width: 720px) {
  .routing-alert--with-action {
    padding-right: 16px;
  }

  .routing-alert__action {
    position: static;
    transform: none;
    margin-top: 8px;
  }
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
  .provider-editor-secret,
  .provider-editor-secret__header {
    flex-direction: column;
    align-items: stretch;
  }

  .provider-editor-card__controls,
  .provider-editor-card__actions,
  .routing-panel__tags {
    justify-content: flex-start;
  }
}
</style>
