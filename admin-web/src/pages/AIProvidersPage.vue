<template>
  <AppShell>
    <template #toolbar>
      <div class="toolbar-cluster toolbar-cluster--status">
        <button
          v-if="latestTestSummary"
          type="button"
          class="test-result-chip"
          @click="scrollToTestCard"
        >
          <StatusTag
            :tone="latestTestSummary.tone"
            :text="latestTestSummary.text"
          />
        </button>
      </div>
      <div class="toolbar-cluster toolbar-cluster--meta">
        <el-tooltip content="重新拉取远端配置" placement="bottom">
          <el-button
            circle
            :loading="pageRefreshing"
            aria-label="刷新"
            @click="refreshPage"
          >
            <el-icon><Refresh /></el-icon>
          </el-button>
        </el-tooltip>
        <el-tooltip :content="discardTooltip" placement="bottom">
          <span class="toolbar-discard-wrap">
            <el-button
              type="danger"
              link
              :disabled="!isDirty"
              @click="handleDiscardDraft"
            >
              放弃草稿
            </el-button>
          </span>
        </el-tooltip>
      </div>
    </template>

    <div class="routing-scene-grid" role="tablist" aria-label="AI 路由场景">
      <button
        v-for="item in sceneCards"
        :key="item.scene"
        :ref="(el) => setSceneCardRef(item.scene, el)"
        type="button"
        class="page-card routing-scene-card"
        :class="{
          'routing-scene-card--active': item.scene === currentSceneKey,
        }"
        role="tab"
        :aria-selected="item.scene === currentSceneKey"
        :tabindex="item.scene === currentSceneKey ? 0 : -1"
        @click="handleSceneChange(item.scene)"
        @keydown.left.prevent="handleSceneArrowKey(-1)"
        @keydown.right.prevent="handleSceneArrowKey(1)"
      >
        <div class="routing-scene-card__header">
          <div>
            <div class="routing-scene-card__eyebrow">
              {{ displayAIRoutingScene(item.scene) }}
            </div>
            <h3>{{ item.title }}</h3>
          </div>
          <StatusTag :tone="item.tone" :text="item.statusText" />
        </div>
        <div class="routing-scene-card__meta">
          <span>策略：{{ displayAIRoutingStrategy(item.strategy) }}</span>
          <span
            >节点：{{ item.activeProviderCount }}/{{ item.providerCount }}</span
          >
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

    <div v-if="sceneLoading && !draftScene" class="page-card routing-panel">
      <PageState mode="loading" title="正在加载场景配置" />
    </div>
    <div v-else-if="sceneError && !draftScene" class="page-card routing-panel">
      <PageState
        mode="error"
        title="场景配置加载失败"
        :description="sceneError"
        @retry="loadCurrentScene"
      />
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
              <button
                type="button"
                class="routing-breadcrumb__channel"
                :class="`routing-breadcrumb__channel--${currentChannel.tone}`"
              >
                线上链路：<code>{{ currentChannel.label }}</code>
                <el-icon class="routing-breadcrumb__channel-icon"
                  ><InfoFilled
                /></el-icon>
              </button>
            </template>
            <div class="channel-popover">
              <div class="channel-popover__title">
                当前状态：{{ currentChannel.reason }}
              </div>
              <table class="channel-popover__table">
                <thead>
                  <tr>
                    <th>草稿/正式</th>
                    <th>新路由</th>
                    <th>实际生效链路</th>
                  </tr>
                </thead>
                <tbody>
                  <tr
                    v-for="(row, idx) in channelMatrix"
                    :key="idx"
                    :class="{ 'is-hit': row.hit }"
                  >
                    <td>{{ row.draft }}</td>
                    <td>{{ row.toggle }}</td>
                    <td>
                      <code>{{ row.effect }}</code>
                    </td>
                  </tr>
                </tbody>
              </table>
            </div>
          </el-popover>
        </span>
        <StatusTag :tone="bottomBarState.tone" :text="statusBarText" />
      </div>

      <div
        class="routing-status-strip"
        :class="`routing-status-strip--${bottomBarState.tone}`"
        aria-live="polite"
      >
        <div class="routing-status-strip__main">
          <strong>{{ currentSceneTitle }}</strong>
          <span
            ><code>{{ currentChannel.label }}</code></span
          >
          <span>{{ currentChannel.reason }}</span>
        </div>
        <div class="routing-status-strip__actions">
          <el-popover placement="bottom-end" :width="340" trigger="click">
            <template #reference>
              <el-button link>告警配置</el-button>
            </template>
            <div class="channel-popover">
              <div class="channel-popover__title">连续异常邮件告警</div>
              <p class="channel-popover__text">
                阈值、SMTP 和收件人统一在配置中心维护，不作为首屏常驻提示。
              </p>
              <el-button type="primary" link @click="goAlertConfig"
                >前往配置 <el-icon><ArrowRight /></el-icon
              ></el-button>
            </div>
          </el-popover>
        </div>
      </div>

      <div class="routing-editor-grid">
        <div class="page-card routing-panel routing-panel--strategy">
          <div class="routing-panel__header">
            <div>
              <h3 class="routing-panel__title">
                场景策略 <HelpTip :content="helpTips.sceneStrategy" />
              </h3>
              <div class="routing-panel__subtitle">
                路由开关、尝试次数、熔断与请求参数。
              </div>
            </div>
            <div class="routing-panel__tags">
              <StatusTag
                :tone="draftScene.enabled ? 'primary' : 'neutral'"
                :text="draftScene.enabled ? '新路由已启用' : '新路由未启用'"
              />
              <StatusTag
                :tone="draftScene.compatibilityMode ? 'warning' : 'success'"
                :text="draftScene.compatibilityMode ? '兼容模式' : '正式模式'"
              />
            </div>
          </div>

          <div class="routing-form-grid">
            <label class="routing-field">
              <span>启用新路由</span>
              <el-switch
                v-model="draftScene.enabled"
                inline-prompt
                active-text="开"
                inactive-text="关"
              />
            </label>
            <label class="routing-field">
              <span
                >调度策略 <HelpTip :content="helpTips.sceneStrategy"
              /></span>
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
              <span
                >最大尝试次数 <HelpTip :content="helpTips.maxAttempts"
              /></span>
              <el-input-number
                v-model="draftScene.maxAttempts"
                :min="1"
                :max="maxAttemptCeiling"
              />
              <small
                class="routing-field__hint"
                :class="{
                  'routing-field__hint--warn': numericWarn.maxAttempts,
                }"
              >
                {{
                  numericWarn.maxAttempts || "建议 2–5 次；过大会让失败降级变慢"
                }}
              </small>
            </label>
            <label class="routing-field routing-field--with-hint">
              <span>熔断阈值 <HelpTip :content="helpTips.breaker" /></span>
              <el-input-number
                v-model="draftScene.breaker.failureThreshold"
                :min="1"
                :max="10"
              />
              <small
                class="routing-field__hint"
                :class="{
                  'routing-field__hint--warn': numericWarn.failureThreshold,
                }"
              >
                {{
                  numericWarn.failureThreshold ||
                  "连续失败达到该次数后触发熔断，建议 3–10"
                }}
              </small>
            </label>
            <label class="routing-field routing-field--with-hint">
              <span
                >冷却时间（秒） <HelpTip :content="helpTips.breaker"
              /></span>
              <el-input-number
                v-model="draftScene.breaker.cooldownSeconds"
                :min="5"
                :max="600"
              />
              <small
                class="routing-field__hint"
                :class="{
                  'routing-field__hint--warn': numericWarn.cooldownSeconds,
                }"
              >
                {{
                  numericWarn.cooldownSeconds ||
                  "熔断冷却时长，低于 30s 容易抖动"
                }}
              </small>
            </label>
            <div class="routing-field routing-field--meta">
              <span>最近修改</span>
              <strong>{{ formatDateTime(draftScene.updatedAt) }}</strong>
              <small>修改人：{{ draftScene.updatedBySubject || "暂无" }}</small>
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
              <span
                class="routing-timeline-hint__seg routing-timeline-hint__seg--breaker"
              >
                连续失败 {{ draftScene.breaker.failureThreshold }} 次
              </span>
              <span
                class="routing-timeline-hint__seg routing-timeline-hint__seg--cooldown"
              >
                冷却 {{ draftScene.breaker.cooldownSeconds }}s
              </span>
            </div>
            <div class="routing-timeline-hint__caption">
              预计首轮最长 ≈ <strong>{{ expectedFirstRoundSeconds }}s</strong>
              <span class="routing-timeline-hint__hint"
                >（最大尝试次数 × 启用节点最大超时）</span
              >
            </div>
          </div>

          <div class="routing-checkbox-block">
            <div class="routing-checkbox-block__title">
              允许切换到下一个节点的错误类型
            </div>
            <el-checkbox-group
              v-model="draftScene.retryOn"
              class="routing-checkbox-grid"
            >
              <el-checkbox
                v-for="item in retryOptions"
                :key="item.value"
                :label="item.value"
              >
                {{ item.label }}
              </el-checkbox>
            </el-checkbox-group>
          </div>

          <div class="routing-request-block">
            <div>
              <div class="routing-checkbox-block__title">
                场景级请求参数 <HelpTip :content="helpTips.requestOptions" />
              </div>
            </div>

            <div
              v-if="draftScene.scene === 'title'"
              class="routing-form-grid routing-form-grid--request"
            >
              <label class="routing-field">
                <span>Stream</span>
                <el-switch
                  v-model="draftScene.requestOptions.stream"
                  inline-prompt
                  active-text="开"
                  inactive-text="关"
                />
              </label>
              <label class="routing-field">
                <span>Temperature</span>
                <el-input-number
                  v-model="draftScene.requestOptions.temperature"
                  :min="0"
                  :max="2"
                  :step="0.1"
                />
              </label>
              <label class="routing-field">
                <span>Max Tokens</span>
                <el-input-number
                  v-model="draftScene.requestOptions.maxTokens"
                  :min="1"
                  :max="512"
                />
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
              <h3 class="routing-panel__title">
                Provider 节点
                <HelpTip content="默认行展示关键状态，展开后编辑完整配置。" />
              </h3>
              <div class="routing-panel__subtitle">
                启停、排序、换密钥和单节点测试。
              </div>
            </div>
            <div class="routing-panel__tags">
              <StatusTag
                tone="neutral"
                :text="`${enabledProviderCount}/${draftScene.providers.length} 个启用节点`"
              />
              <el-button type="primary" plain @click="handleAddProvider"
                >新增节点</el-button
              >
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
                  draggingProviderIndex !== index,
              }"
              @dragover.prevent="handleProviderDragOver(index)"
              @dragenter.prevent="handleProviderDragOver(index)"
              @drop.prevent="handleProviderDrop(index)"
            >
              <div class="provider-editor-card__header">
                <div class="provider-editor-card__main">
                  <div class="provider-editor-card__title">
                    <button
                      type="button"
                      class="provider-icon-button provider-collapse-toggle"
                      :aria-expanded="!isProviderCollapsed(provider)"
                      :aria-label="
                        isProviderCollapsed(provider) ? '展开编辑' : '折叠编辑'
                      "
                      @click="toggleProviderCollapsed(provider)"
                    >
                      <el-icon
                        ><ArrowDown
                          v-if="!isProviderCollapsed(provider)" /><ArrowRight
                          v-else
                      /></el-icon>
                    </button>
                    <button
                      type="button"
                      class="provider-icon-button provider-icon-button--drag provider-drag-handle"
                      :class="{
                        'provider-icon-button--disabled':
                          draftScene.providers.length < 2,
                      }"
                      :disabled="draftScene.providers.length < 2"
                      :draggable="draftScene.providers.length > 1"
                      aria-label="拖拽排序"
                      @dragstart="handleProviderDragStart(index, $event)"
                      @dragend="handleProviderDragEnd"
                    >
                      <el-icon><Rank /></el-icon>
                    </button>
                    <div class="provider-title-stack">
                      <div class="provider-title-stack__name-row">
                        <strong>{{
                          provider.name || `节点 ${index + 1}`
                        }}</strong>
                        <StatusTag
                          v-if="firstProviderError(provider)"
                          tone="warning"
                          text="需修正"
                        />
                      </div>
                    </div>
                  </div>
                  <div class="provider-compact-grid">
                    <span class="provider-compact-grid__item"
                      >顺序 {{ index + 1 }}</span
                    >
                    <span class="provider-compact-grid__item mono-text">{{
                      provider.model || "未填写 Model"
                    }}</span>
                    <span class="provider-compact-grid__item">{{
                      provider.endpointMode || "chat_completions"
                    }}</span>
                    <span class="provider-compact-grid__item"
                      >{{ provider.timeoutSeconds }}s</span
                    >
                    <StatusTag
                      :tone="getProviderSecretStatus(provider).tone"
                      :text="getProviderSecretStatus(provider).text"
                    />
                    <StatusTag
                      :tone="provider.enabled ? 'success' : 'neutral'"
                      :text="provider.enabled ? '参与调度' : '已停用'"
                    />
                    <StatusTag
                      v-if="getProviderTestState(provider)"
                      :tone="
                        getProviderTestState(provider)?.ok
                          ? 'success'
                          : 'warning'
                      "
                      :text="getProviderTestState(provider)?.text || ''"
                    />
                    <span v-else class="provider-test-empty">待测试</span>
                  </div>
                  <div
                    v-if="firstProviderError(provider)"
                    class="provider-inline-error"
                    role="alert"
                  >
                    {{ firstProviderError(provider) }}
                  </div>
                </div>
                <div class="provider-editor-card__controls">
                  <div
                    class="provider-enable-control"
                    :class="{
                      'provider-enable-control--off': !provider.enabled,
                    }"
                  >
                    <span>{{ provider.enabled ? "启用" : "停用" }}</span>
                    <el-switch
                      v-model="provider.enabled"
                      inline-prompt
                      active-text="开"
                      inactive-text="关"
                    />
                  </div>
                  <div class="provider-editor-card__actions">
                    <el-tooltip content="测试当前节点" placement="top">
                      <button
                        type="button"
                        class="provider-icon-button provider-icon-button--primary"
                        :disabled="testingScene"
                        aria-label="测试当前节点"
                        @click="handleTestSingleProvider(index)"
                      >
                        <el-icon
                          v-if="singleTestProviderId === provider.id"
                          class="is-loading"
                          ><Refresh
                        /></el-icon>
                        <el-icon v-else><Promotion /></el-icon>
                      </button>
                    </el-tooltip>
                    <el-dropdown
                      trigger="click"
                      @command="
                        (command) =>
                          handleProviderMenuCommand(String(command), index)
                      "
                    >
                      <button
                        type="button"
                        class="provider-icon-button"
                        aria-label="更多操作"
                      >
                        <el-icon><MoreFilled /></el-icon>
                      </button>
                      <template #dropdown>
                        <el-dropdown-menu>
                          <el-dropdown-item
                            command="move-up"
                            :disabled="index === 0"
                            >上移一位</el-dropdown-item
                          >
                          <el-dropdown-item
                            command="move-down"
                            :disabled="
                              index === draftScene.providers.length - 1
                            "
                            >下移一位</el-dropdown-item
                          >
                          <el-dropdown-item command="duplicate" divided
                            >复制节点</el-dropdown-item
                          >
                          <el-dropdown-item command="delete" divided
                            >删除节点</el-dropdown-item
                          >
                        </el-dropdown-menu>
                      </template>
                    </el-dropdown>
                  </div>
                </div>
              </div>

              <template v-if="!isProviderCollapsed(provider)">
                <section class="provider-editor-section">
                  <div class="provider-editor-section__header">
                    <strong>基础信息</strong>
                    <span>名称、模型与调用身份。</span>
                  </div>
                  <div class="provider-editor-grid provider-editor-grid--basic">
                    <label class="routing-field">
                      <span
                        >Provider Name
                        <HelpTip :content="helpTips.providerName"
                      /></span>
                      <el-input
                        v-model.trim="provider.name"
                        placeholder="主节点 / 备用节点"
                        @blur="touchProviderField(provider, 'name')"
                      />
                      <small
                        v-if="providerFieldError(provider, 'name')"
                        class="routing-field__hint routing-field__hint--warn"
                        role="alert"
                        >{{ providerFieldError(provider, "name") }}</small
                      >
                    </label>
                    <label class="routing-field">
                      <span>Model</span>
                      <el-input
                        v-model.trim="provider.model"
                        placeholder="gpt-4.1-mini"
                        @blur="touchProviderField(provider, 'model')"
                      />
                      <small
                        v-if="providerFieldError(provider, 'model')"
                        class="routing-field__hint routing-field__hint--warn"
                        role="alert"
                        >{{ providerFieldError(provider, "model") }}</small
                      >
                    </label>
                  </div>
                </section>

                <section class="provider-editor-section">
                  <div class="provider-editor-section__header">
                    <strong>请求配置</strong>
                    <span>Base URL、接口模式、超时与适配器。</span>
                  </div>
                  <div
                    class="provider-editor-grid provider-editor-grid--request"
                  >
                    <label class="routing-field provider-editor-grid__wide">
                      <span
                        >Base URL <HelpTip :content="helpTips.baseURL"
                      /></span>
                      <el-input
                        v-model.trim="provider.baseURL"
                        placeholder="https://api.example.com/v1"
                        @blur="touchProviderField(provider, 'baseURL')"
                      />
                      <small
                        v-if="providerFieldError(provider, 'baseURL')"
                        class="routing-field__hint routing-field__hint--warn"
                        role="alert"
                        >{{ providerFieldError(provider, "baseURL") }}</small
                      >
                    </label>
                    <label class="routing-field">
                      <span
                        >Endpoint <HelpTip :content="helpTips.endpoint"
                      /></span>
                      <el-select
                        v-model="provider.endpointMode"
                        @change="handleEndpointModeChange(provider)"
                      >
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
                      <el-input-number
                        v-model="provider.timeoutSeconds"
                        :min="1"
                        :max="600"
                        @blur="touchProviderField(provider, 'timeoutSeconds')"
                      />
                      <small
                        v-if="providerFieldError(provider, 'timeoutSeconds')"
                        class="routing-field__hint routing-field__hint--warn"
                        role="alert"
                        >{{
                          providerFieldError(provider, "timeoutSeconds")
                        }}</small
                      >
                    </label>
                    <label class="routing-field">
                      <span>Adapter</span>
                      <el-input v-model="provider.adapter" disabled />
                    </label>
                    <label
                      v-if="isImageGenerationProvider(provider)"
                      class="routing-field"
                    >
                      <span
                        >Response Format
                        <HelpTip :content="helpTips.responseFormat"
                      /></span>
                      <el-select
                        v-model="provider.responseFormat"
                        :disabled="!isImageGenerationProvider(provider)"
                      >
                        <el-option
                          v-for="item in providerResponseFormatOptions"
                          :key="item.value"
                          :label="item.label"
                          :value="item.value"
                        />
                      </el-select>
                    </label>
                  </div>
                </section>
                <div class="provider-editor-secret">
                  <div class="provider-editor-secret__header">
                    <div class="provider-editor-secret__label-row">
                      <span class="provider-editor-secret__label"
                        >API Key <HelpTip :content="helpTips.apiKey"
                      /></span>
                      <span
                        v-if="provider.hasAPIKey && !provider.clearApiKey"
                        class="provider-secret-chip mono-text"
                      >
                        当前密钥 · {{ provider.apiKeyMasked || "已保存" }}
                      </span>
                      <StatusTag
                        v-else-if="provider.clearApiKey"
                        tone="warning"
                        text="已标记清空"
                      />
                      <span
                        v-else
                        class="provider-secret-chip provider-secret-chip--empty"
                        >当前未配置密钥</span
                      >
                    </div>
                    <div
                      v-if="provider.hasAPIKey"
                      class="provider-editor-secret__actions"
                    >
                      <el-button
                        text
                        :disabled="!!provider.apiKey?.trim()"
                        @click="toggleProviderSecretEditor(provider)"
                      >
                        {{
                          provider.apiKey?.trim()
                            ? "已录入新密钥"
                            : shouldShowProviderSecretEditor(provider)
                              ? "收起更换"
                              : "更换密钥"
                        }}
                      </el-button>
                      <el-button
                        text
                        type="danger"
                        @click="handleClearProviderApiKey(provider)"
                      >
                        {{ provider.clearApiKey ? "撤销清空" : "清空密钥" }}
                      </el-button>
                    </div>
                  </div>

                  <label
                    v-if="shouldShowProviderSecretEditor(provider)"
                    class="routing-field provider-editor-secret__field"
                  >
                    <span>{{
                      provider.hasAPIKey ? "输入新密钥" : "录入密钥"
                    }}</span>
                    <el-input
                      v-model="provider.apiKey"
                      type="password"
                      show-password
                      :placeholder="
                        provider.hasAPIKey
                          ? '输入新密钥，保存后覆盖旧值'
                          : '输入当前节点要使用的密钥'
                      "
                      @update:model-value="
                        handleProviderApiKeyInput(provider, $event)
                      "
                    />
                  </label>
                </div>
                <div class="provider-editor-secret__hint">
                  <template v-if="provider.clearApiKey"
                    >当前已标记为待清空，保存后会彻底移除旧密钥。</template
                  >
                  <template v-else-if="provider.apiKey?.trim()"
                    >已录入新的密钥草稿，保存后会覆盖当前值。</template
                  >
                  <template v-else-if="provider.hasAPIKey"
                    >当前已保存密钥；不输入新值则继续保留旧值。</template
                  >
                  <template v-else
                    >当前没有已保存密钥，可直接录入新值。</template
                  >
                </div>
              </template>
            </div>
          </div>
        </div>
      </div>

      <div
        v-if="testResult"
        ref="testCardRef"
        class="page-card routing-test-card"
      >
        <div class="routing-panel__header">
          <div>
            <h3 class="routing-panel__title">最近测试结果</h3>
            <div class="routing-panel__subtitle">{{ testScope }}</div>
          </div>
          <div class="routing-panel__tags">
            <StatusTag
              :tone="testResult.ok ? 'success' : 'warning'"
              :text="testResult.ok ? '测试成功' : '需要关注'"
            />
            <StatusTag
              tone="neutral"
              :text="`总耗时 ${formatDuration(testResult.attempts.reduce((acc, item) => acc + item.latencyMs, 0))}`"
            />
          </div>
        </div>

        <div class="routing-test-card__summary">
          <span>结果：{{ testResult.message }}</span>
          <span>最终节点：{{ testResult.finalProvider || "-" }}</span>
          <span>最终模型：{{ testResult.finalModel || "-" }}</span>
        </div>

        <div class="table-scroll">
          <el-table
            :data="testResult.attempts"
            size="small"
            style="width: 100%"
          >
            <el-table-column
              label="Provider"
              min-width="150"
              show-overflow-tooltip
            >
              <template #default="{ row }">{{
                providerDisplayName(row.providerId)
              }}</template>
            </el-table-column>
            <el-table-column
              prop="model"
              label="Model"
              min-width="140"
              show-overflow-tooltip
            />
            <el-table-column label="状态" width="120">
              <template #default="{ row }">
                <StatusTag
                  :tone="toneForStatus(row.status)"
                  :text="displayCallStatus(row.status)"
                />
              </template>
            </el-table-column>
            <el-table-column prop="httpStatus" label="HTTP" width="90" />
            <el-table-column prop="latencyMs" label="耗时" width="100">
              <template #default="{ row }">{{
                formatDuration(row.latencyMs)
              }}</template>
            </el-table-column>
            <el-table-column label="错误类型" min-width="120">
              <template #default="{ row }">{{ row.errorType || "-" }}</template>
            </el-table-column>
            <el-table-column label="备注" min-width="220" show-overflow-tooltip>
              <template #default="{ row }">
                {{
                  row.skippedByBreaker
                    ? `breaker until ${formatDateTime(row.breakerOpenUntil)}`
                    : row.errorMessage || "-"
                }}
              </template>
            </el-table-column>
          </el-table>
        </div>
      </div>

      <div class="page-card audit-section">
        <div class="page-header">
          <div>
            <h2 class="page-title" style="font-size: 22px">
              最近审计 <HelpTip :content="helpTips.audit" />
            </h2>
            <div class="page-subtitle">{{ currentAuditGroup }} · 最近 5 条</div>
          </div>
          <el-button :loading="auditsLoading" @click="auditDrawerVisible = true"
            >完整审计</el-button
          >
        </div>

        <PageState
          v-if="recentAuditsLoading && !recentAudits.length"
          mode="loading"
          title="正在加载审计记录"
          compact
        />
        <PageState
          v-else-if="recentAuditsError && !recentAudits.length"
          mode="error"
          title="路由审计加载失败"
          :description="recentAuditsError"
          compact
          @retry="loadRecentAudits"
        />
        <PageState
          v-else-if="!recentAudits.length"
          mode="empty"
          title="暂无路由审计"
          description="当前筛选条件下还没有保存或测试记录。"
          compact
        />
        <template v-else>
          <div class="audit-timeline-list">
            <article
              v-for="item in recentAudits"
              :key="item.id"
              class="audit-timeline-item"
            >
              <div class="audit-timeline-item__rail" aria-hidden="true">
                <span
                  class="audit-timeline-item__dot"
                  :class="`audit-timeline-item__dot--${toneForAuditAction(item)}`"
                ></span>
              </div>
              <div class="audit-timeline-item__body">
                <div class="audit-timeline-item__header">
                  <div>
                    <div class="audit-timeline-item__title-row">
                      <StatusTag
                        :tone="auditBusinessAction(item).tone"
                        :text="auditBusinessAction(item).label"
                      />
                      <strong>{{ auditTargetTitle(item) }}</strong>
                    </div>
                    <div class="audit-timeline-item__meta">
                      <span>{{ auditEventLabel(item) }}</span>
                      <span>{{ item.operatorSubject || "未知操作人" }}</span>
                      <span>{{ formatDateTime(item.createdAt) }}</span>
                    </div>
                  </div>
                  <el-popover
                    placement="left-start"
                    :width="680"
                    trigger="click"
                  >
                    <template #reference>
                      <button
                        type="button"
                        class="audit-detail-button"
                        aria-label="查看审计变化"
                      >
                        查看变化
                      </button>
                    </template>
                    <div class="audit-diff-popover">
                      <div class="audit-diff-popover__header">
                        <div>
                          <strong>{{ auditDiffTitle(item) }}</strong>
                          <span
                            >{{ auditEventLabel(item) }} ·
                            {{ item.operatorSubject || "未知操作人" }} ·
                            {{ formatDateTime(item.createdAt) }}</span
                          >
                        </div>
                        <StatusTag
                          :tone="auditBusinessAction(item).tone"
                          :text="auditDiffStatusText(item)"
                        />
                      </div>
                      <div
                        v-if="auditChangeSummary(item).length"
                        class="audit-diff-summary-strip"
                      >
                        <span
                          v-for="stat in auditChangeStats(item)"
                          :key="stat.kind"
                          :class="`audit-diff-stat audit-diff-stat--${stat.kind}`"
                        >
                          {{ stat.label }} {{ stat.count }} 项
                        </span>
                      </div>
                      <div
                        v-if="auditChangeSummary(item).length"
                        class="audit-diff-popover__changes"
                      >
                        <section
                          v-for="group in groupedAuditChanges(item)"
                          :key="group.name"
                          class="audit-diff-group"
                        >
                          <div class="audit-diff-group__title">
                            {{ group.name }}
                          </div>
                          <div
                            v-for="change in group.changes.slice(0, 10)"
                            :key="change.key"
                            class="audit-diff-row"
                            :class="`audit-diff-row--${change.kind}`"
                          >
                            <span class="audit-diff-row__field">{{
                              change.field
                            }}</span>
                            <span
                              class="audit-diff-row__value audit-diff-row__value--old"
                              :title="change.from"
                              >{{ change.from }}</span
                            >
                            <el-icon aria-hidden="true"><ArrowRight /></el-icon>
                            <span
                              class="audit-diff-row__value audit-diff-row__value--new"
                              :title="change.to"
                              >{{ change.to }}</span
                            >
                            <span class="audit-diff-row__kind">{{
                              auditChangeKindText(change.kind)
                            }}</span>
                          </div>
                        </section>
                        <div
                          v-if="auditChangeSummary(item).length > 10"
                          class="audit-diff-popover__more"
                        >
                          另有
                          {{
                            auditChangeSummary(item).length - 10
                          }}
                          项变化，请在完整审计中查看。
                        </div>
                      </div>
                      <div v-else class="audit-diff-popover__fallback">
                        {{ auditFallbackSummary(item) }}
                      </div>
                      <el-collapse class="audit-raw-collapse">
                        <el-collapse-item title="查看原始值" name="raw">
                          <div class="audit-diff-grid audit-diff-grid--popover">
                            <div>
                              <strong>旧值</strong>
                              <pre>{{
                                formatAuditValue(item.oldValueMasked)
                              }}</pre>
                            </div>
                            <div>
                              <strong>新值</strong>
                              <pre>{{
                                formatAuditValue(item.newValueMasked)
                              }}</pre>
                            </div>
                          </div>
                        </el-collapse-item>
                      </el-collapse>
                    </div>
                  </el-popover>
                </div>
                <div class="audit-timeline-item__summary">
                  <template v-if="auditChangeSummary(item).length">
                    <div class="audit-timeline-item__insight">
                      {{ auditBusinessAction(item).description }}
                    </div>
                    <div
                      v-for="change in auditChangeSummary(item).slice(0, 3)"
                      :key="change.key"
                      class="audit-change-row"
                      :class="`audit-change-row--${change.kind}`"
                    >
                      <span class="audit-change-row__field">{{
                        change.field
                      }}</span>
                      <span class="audit-change-row__value">{{
                        change.from
                      }}</span>
                      <el-icon aria-hidden="true"><ArrowRight /></el-icon>
                      <span
                        class="audit-change-row__value audit-change-row__value--new"
                        >{{ change.to }}</span
                      >
                    </div>
                    <div
                      v-if="auditChangeSummary(item).length > 3"
                      class="audit-timeline-item__more"
                    >
                      另有 {{ auditChangeSummary(item).length - 3 }} 项变化
                    </div>
                  </template>
                  <span v-else class="audit-timeline-item__fallback">{{
                    auditFallbackSummary(item)
                  }}</span>
                </div>
              </div>
            </article>
          </div>
        </template>
      </div>

      <el-drawer
        v-model="auditDrawerVisible"
        title="完整路由审计"
        size="72%"
        append-to-body
      >
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
            <el-button
              type="primary"
              :loading="auditsLoading"
              @click="loadAudits"
              >筛选</el-button
            >
          </template>
        </FilterToolbar>
        <div class="table-scroll">
          <el-table :data="audits.items" size="small" style="width: 100%">
            <el-table-column type="expand">
              <template #default="{ row }">
                <div class="audit-drawer-detail">
                  <div class="audit-diff-popover__header">
                    <div>
                      <strong>{{ auditDiffTitle(row) }}</strong>
                      <span
                        >{{ auditEventLabel(row) }} ·
                        {{ row.operatorSubject || "未知操作人" }} ·
                        {{ formatDateTime(row.createdAt) }}</span
                      >
                    </div>
                    <StatusTag
                      :tone="auditBusinessAction(row).tone"
                      :text="auditDiffStatusText(row)"
                    />
                  </div>
                  <div
                    v-if="auditChangeSummary(row).length"
                    class="audit-diff-summary-strip"
                  >
                    <span
                      v-for="stat in auditChangeStats(row)"
                      :key="stat.kind"
                      :class="`audit-diff-stat audit-diff-stat--${stat.kind}`"
                    >
                      {{ stat.label }} {{ stat.count }} 项
                    </span>
                  </div>
                  <div
                    v-if="auditChangeSummary(row).length"
                    class="audit-diff-popover__changes"
                  >
                    <section
                      v-for="group in groupedAuditChanges(row)"
                      :key="group.name"
                      class="audit-diff-group"
                    >
                      <div class="audit-diff-group__title">
                        {{ group.name }}
                      </div>
                      <div
                        v-for="change in group.changes"
                        :key="change.key"
                        class="audit-diff-row"
                        :class="`audit-diff-row--${change.kind}`"
                      >
                        <span class="audit-diff-row__field">{{
                          change.field
                        }}</span>
                        <span
                          class="audit-diff-row__value audit-diff-row__value--old"
                          :title="change.from"
                          >{{ change.from }}</span
                        >
                        <el-icon aria-hidden="true"><ArrowRight /></el-icon>
                        <span
                          class="audit-diff-row__value audit-diff-row__value--new"
                          :title="change.to"
                          >{{ change.to }}</span
                        >
                        <span class="audit-diff-row__kind">{{
                          auditChangeKindText(change.kind)
                        }}</span>
                      </div>
                    </section>
                  </div>
                  <div v-else class="audit-diff-popover__fallback">
                    {{ auditFallbackSummary(row) }}
                  </div>
                  <el-collapse class="audit-raw-collapse">
                    <el-collapse-item title="查看原始值" name="raw">
                      <div class="audit-diff-grid">
                        <div>
                          <strong>旧值</strong>
                          <pre>{{ formatAuditValue(row.oldValueMasked) }}</pre>
                        </div>
                        <div>
                          <strong>新值</strong>
                          <pre>{{ formatAuditValue(row.newValueMasked) }}</pre>
                        </div>
                      </div>
                    </el-collapse-item>
                  </el-collapse>
                </div>
              </template>
            </el-table-column>
            <el-table-column label="对象" min-width="180" show-overflow-tooltip>
              <template #default="{ row }">{{
                auditTargetTitle(row)
              }}</template>
            </el-table-column>
            <el-table-column label="动作" width="120">
              <template #default="{ row }">
                <StatusTag
                  :tone="auditBusinessAction(row).tone"
                  :text="auditBusinessAction(row).label"
                />
              </template>
            </el-table-column>
            <el-table-column
              prop="operatorSubject"
              label="操作人"
              width="140"
            />
            <el-table-column
              prop="requestId"
              label="Request ID"
              min-width="160"
              show-overflow-tooltip
            />
            <el-table-column label="时间" width="180">
              <template #default="{ row }">{{
                formatDateTime(row.createdAt)
              }}</template>
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
      </el-drawer>

      <div
        class="routing-bottom-bar"
        :class="`routing-bottom-bar--${bottomBarState.tone}`"
        aria-live="polite"
      >
        <div class="routing-bottom-bar__status">
          <el-icon aria-hidden="true"
            ><component :is="bottomBarState.icon"
          /></el-icon>
          <span>{{ bottomBarState.text }}</span>
        </div>
        <el-popover v-if="isDirty" placement="top" :width="360" trigger="click">
          <template #reference>
            <el-button link>草稿摘要</el-button>
          </template>
          <div class="draft-summary-popover">
            <div
              v-for="(item, idx) in sceneDiff.slice(0, 6)"
              :key="idx"
              class="draft-summary-popover__item"
            >
              <strong>{{ item.scope }} · {{ item.path }}</strong>
              <span
                >{{ formatDiffValue(item.from) }} →
                {{ formatDiffValue(item.to) }}</span
              >
            </div>
            <div
              v-if="sceneDiff.length > 6"
              class="draft-summary-popover__more"
            >
              另有 {{ sceneDiff.length - 6 }} 项改动
            </div>
          </div>
        </el-popover>
        <div class="routing-bottom-bar__actions">
          <el-tooltip content="刷新远端配置" placement="top">
            <el-button
              circle
              :loading="pageRefreshing"
              aria-label="刷新远端配置"
              @click="refreshPage"
              ><el-icon><Refresh /></el-icon
            ></el-button>
          </el-tooltip>
          <el-tooltip content="测试当前草稿" placement="top">
            <el-button
              circle
              :loading="testingScene"
              :disabled="!draftScene"
              aria-label="测试当前草稿"
              @click="handleTestScene"
              ><el-icon><Promotion /></el-icon
            ></el-button>
          </el-tooltip>
          <el-tooltip content="保存场景" placement="top">
            <el-button
              circle
              type="primary"
              :loading="savingScene"
              :disabled="!isDirty || savingScene"
              aria-label="保存场景"
              @click="handleSaveScene"
              ><el-icon><Check /></el-icon
            ></el-button>
          </el-tooltip>
        </div>
      </div>
    </template>
  </AppShell>
</template>

<script setup lang="ts">
import {
  computed,
  h,
  nextTick,
  onBeforeUnmount,
  onMounted,
  ref,
  watch,
} from "vue";
import { onBeforeRouteLeave, useRoute, useRouter } from "vue-router";
import { ElMessage, ElMessageBox } from "element-plus";
import {
  ArrowDown,
  ArrowRight,
  ArrowUp,
  Check,
  Clock,
  InfoFilled,
  MoreFilled,
  Promotion,
  Rank,
  Refresh,
  Warning,
} from "@element-plus/icons-vue";
import AppShell from "@/components/AppShell.vue";
import FilterToolbar from "@/components/FilterToolbar.vue";
import HelpTip from "@/components/HelpTip.vue";
import PageState from "@/components/PageState.vue";
import StatusTag from "@/components/StatusTag.vue";
import * as adminApi from "@/api/admin";
import type {
  AIRoutingProviderEndpointMode,
  AIRoutingProviderConfig,
  AIRoutingProviderResponseFormat,
  AIRoutingSceneConfig,
  AIRoutingSceneKey,
  AIRoutingSceneSummary,
  AIRoutingTestResult,
  PaginationResult,
  SettingAuditRecord,
} from "@/types";
import { buildRouteQuery, readQueryString } from "@/utils/route-query";
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
  toneForStatus,
} from "@/utils/admin-display";

const router = useRouter();
const route = useRoute();

const sceneKeys: AIRoutingSceneKey[] = ["summary", "title", "flowchart"];
const currentSceneKey = ref<AIRoutingSceneKey>("summary");
const sceneSummaries = ref<AIRoutingSceneSummary[]>([]);
const remoteScene = ref<AIRoutingSceneConfig | null>(null);
const draftScene = ref<AIRoutingSceneConfig | null>(null);
const testResult = ref<AIRoutingTestResult | null>(null);
const testScope = ref("");

const sceneLoading = ref(false);
const sceneError = ref("");
const pageRefreshing = ref(false);
const savingScene = ref(false);
const testingScene = ref(false);
const singleTestProviderId = ref("");
const testCardRef = ref<HTMLElement | null>(null);
const routeSceneOverride = ref<AIRoutingSceneKey | null>(null);
const draggingProviderIndex = ref<number | null>(null);
const dragOverProviderIndex = ref<number | null>(null);
const shouldFocusSceneAfterChange = ref(false);
const sceneCardRefs: Partial<
  Record<AIRoutingSceneKey, HTMLButtonElement | null>
> = {};

const audits = ref<PaginationResult<SettingAuditRecord>>({
  items: [],
  total: 0,
  page: 1,
  pageSize: 20,
});
const recentAuditItems = ref<SettingAuditRecord[]>([]);
const auditsLoading = ref(false);
const recentAuditsLoading = ref(false);
const auditsError = ref("");
const recentAuditsError = ref("");
const auditAction = ref("");
const auditPage = ref(1);
const auditDrawerVisible = ref(false);
const providerSecretEditorState = ref<Record<string, boolean>>({});
const collapsedProviderKeys = ref<Set<string>>(new Set());
const providerTouchedState = ref<Record<string, boolean>>({});
const providerLastTestState = ref<Record<string, ProviderTestState>>({});
const testingStartedAt = ref<number | null>(null);
const testingElapsedSeconds = ref(0);
let testingTimer: ReturnType<typeof window.setInterval> | null = null;

const providerLocalKeys = new WeakMap<AIRoutingProviderConfig, string>();
let providerLocalKeyCounter = 0;

type ProviderTestState = {
  ok: boolean;
  text: string;
  latencyMs?: number;
  errorType?: string;
  testedAt: string;
};

type ProviderValidationError = {
  name?: string;
  id?: string;
  baseURL?: string;
  model?: string;
  timeoutSeconds?: string;
};

const helpTips = {
  sceneStrategy:
    "调度策略决定多节点失败后如何切换；详细规则可点击线上链路查看。",
  maxAttempts: "建议 2-5 次。尝试次数越大，异常降级链路等待越久。",
  breaker: "连续失败触发熔断后，节点会在冷却时间内跳过。",
  requestOptions:
    "标题清洗场景使用这些请求参数；其他场景由业务层固定 prompt 控制。",
  providerName:
    "用于页面展示、测试结果和审计记录；同一场景内不可重复。内部 ID 由系统自动生成并隐藏。",
  baseURL: "填写兼容 OpenAI 协议的服务地址，例如 https://api.example.com/v1。",
  endpoint:
    "图片生成节点使用 images/generations；普通文本节点使用 chat/completions。",
  responseFormat:
    "images/generations 固定 output_format=png；这里仅配置响应返回格式。",
  apiKey: "已保存密钥留空会继续保留旧值；点击清空并保存后才会移除。",
  audit: "默认只展示最近 5 条，完整审计可在抽屉中筛选和分页。",
};

const retryOptions = [
  { label: "超时 timeout", value: "timeout" },
  { label: "网络 network", value: "network" },
  { label: "限流 rate_limit", value: "rate_limit" },
  { label: "鉴权 auth", value: "auth" },
  { label: "上游 upstream", value: "upstream" },
  { label: "响应异常 invalid_response", value: "invalid_response" },
];

const providerEndpointModeOptions: Array<{
  label: string;
  value: AIRoutingProviderEndpointMode;
}> = [
  { label: "chat/completions", value: "chat_completions" },
  { label: "images/generations", value: "images_generations" },
];

const providerResponseFormatOptions: Array<{
  label: string;
  value: AIRoutingProviderResponseFormat;
}> = [
  { label: "auto", value: "auto" },
  { label: "image_url", value: "image_url" },
  { label: "b64_json", value: "b64_json" },
];

const currentAuditGroup = computed(() => `ai.routing.${currentSceneKey.value}`);
const enabledProviderCount = computed(
  () => draftScene.value?.providers.filter((item) => item.enabled).length || 0,
);
const maxAttemptCeiling = computed(() =>
  Math.max(enabledProviderCount.value || 1, 1),
);
const compatibilityHint = computed(() => {
  if (!draftScene.value?.compatibilityMode) {
    return "";
  }
  return "当前运行时仍优先走旧单 Provider 配置；保存并启用本场景后，summary / title / flowchart 才会正式切到新的多节点路由。";
});

const sceneCards = computed(() => {
  return sceneKeys.map((scene) => {
    const summary = sceneSummaries.value.find((item) => item.scene === scene);
    return {
      scene,
      title:
        scene === "summary"
          ? "正文总结"
          : scene === "title"
            ? "标题清洗"
            : "步骤图生成",
      strategy:
        summary?.strategy ||
        (scene === "title" ? "round_robin_failover" : "priority_failover"),
      providerCount: summary?.providerCount || 0,
      activeProviderCount: summary?.activeProviderCount || 0,
      updatedAt: summary?.updatedAt || "",
      source: summary?.source || "empty",
      compatibilityMode: summary?.compatibilityMode ?? true,
      tone:
        scene === currentSceneKey.value
          ? "primary"
          : summary?.compatibilityMode
            ? "warning"
            : summary?.enabled
              ? "success"
              : "neutral",
      statusText: summary?.compatibilityMode
        ? "兼容模式"
        : summary?.enabled
          ? "正式模式"
          : "未启用",
    };
  });
});

const currentSceneTitle = computed(() => {
  const card = sceneCards.value.find(
    (item) => item.scene === currentSceneKey.value,
  );
  return card?.title || displayAIRoutingScene(currentSceneKey.value);
});

function isImageGenerationProvider(provider: AIRoutingProviderConfig) {
  return (provider.endpointMode || "chat_completions") === "images_generations";
}

function handleEndpointModeChange(provider: AIRoutingProviderConfig) {
  if (!isImageGenerationProvider(provider)) {
    provider.responseFormat = "auto";
  } else if (!provider.responseFormat) {
    provider.responseFormat = "auto";
  }
}

function goAlertConfig() {
  router.push({
    path: "/settings",
    query: { group: "ai.provider_alert" },
    hash: "#ai-provider-alert",
  });
}

function effectiveChannel(params: {
  scene: AIRoutingSceneKey;
  enabled: boolean;
  compatibilityMode: boolean;
  isDraftDirty?: boolean;
}) {
  const { scene, enabled, compatibilityMode, isDraftDirty } = params;
  if (isDraftDirty) {
    return {
      label: `${scene}-draft`,
      tone: "info" as const,
      reason: "草稿未保存 · 线上保持不变，仅测试入口生效",
    };
  }
  if (compatibilityMode) {
    return {
      label: `${scene}-compat`,
      tone: "warning" as const,
      reason: "兼容模式 · 走旧单 Provider 链路",
    };
  }
  if (!enabled) {
    return {
      label: `${scene}-compat`,
      tone: "neutral" as const,
      reason: "新路由未启用 · 回退兼容链路",
    };
  }
  return {
    label: `${scene}-v2`,
    tone: "success" as const,
    reason: "线上走新多节点路由",
  };
}

function sceneEffectiveChannel(scene: AIRoutingSceneKey) {
  const summary = sceneSummaries.value.find((item) => item.scene === scene);
  const isCurrent = scene === currentSceneKey.value && !!draftScene.value;
  const enabled =
    isCurrent && draftScene.value
      ? draftScene.value.enabled
      : (summary?.enabled ?? false);
  const compatibilityMode =
    isCurrent && draftScene.value
      ? draftScene.value.compatibilityMode
      : (summary?.compatibilityMode ?? true);
  const isDraftDirty = isCurrent && isDirty.value;
  return effectiveChannel({ scene, enabled, compatibilityMode, isDraftDirty });
}

const currentChannel = computed(() =>
  sceneEffectiveChannel(currentSceneKey.value),
);

const channelMatrix = computed(() => [
  {
    draft: "正式",
    toggle: "开",
    effect: `${currentSceneKey.value}-v2`,
    hit:
      !isDirty.value &&
      draftScene.value?.enabled &&
      !draftScene.value?.compatibilityMode,
  },
  {
    draft: "正式",
    toggle: "关",
    effect: `${currentSceneKey.value}-compat`,
    hit:
      !isDirty.value &&
      (!draftScene.value?.enabled || draftScene.value?.compatibilityMode),
  },
  { draft: "草稿", toggle: "—", effect: "仅测试入口生效", hit: isDirty.value },
]);

const numericWarn = computed(() => {
  const scene = draftScene.value;
  const warn: {
    maxAttempts: string;
    failureThreshold: string;
    cooldownSeconds: string;
  } = {
    maxAttempts: "",
    failureThreshold: "",
    cooldownSeconds: "",
  };
  if (!scene) return warn;
  const ma = Number(scene.maxAttempts) || 0;
  if (ma > 5) warn.maxAttempts = `当前 ${ma} 次偏大，失败降级会变慢`;
  else if (ma < 2) warn.maxAttempts = "至少 2 次才能触发节点切换";
  const ft = Number(scene.breaker?.failureThreshold) || 0;
  if (ft < 3) warn.failureThreshold = "阈值过低容易误熔断";
  else if (ft > 10) warn.failureThreshold = "阈值过高将延迟熔断保护";
  const cs = Number(scene.breaker?.cooldownSeconds) || 0;
  if (cs < 30) warn.cooldownSeconds = "低于 30s 容易抖动";
  else if (cs > 300) warn.cooldownSeconds = "超过 5 分钟可能影响恢复";
  return warn;
});

const timelineSegments = computed(() => {
  const attempts = Math.max(Number(draftScene.value?.maxAttempts) || 1, 1);
  return { attempts: Math.min(attempts, 6) };
});

const expectedFirstRoundSeconds = computed(() => {
  const scene = draftScene.value;
  if (!scene) return 0;
  const attempts = Math.max(Number(scene.maxAttempts) || 1, 1);
  const enabled = scene.providers.filter((p) => p.enabled);
  const maxTimeout =
    enabled.reduce(
      (acc, p) => Math.max(acc, Number(p.timeoutSeconds) || 0),
      0,
    ) || 30;
  return attempts * maxTimeout;
});

const latestTestSummary = computed(() => {
  if (testingScene.value) {
    return {
      tone: "primary" as const,
      text: `测试中 · ${testingElapsedSeconds.value}s`,
    };
  }
  if (!testResult.value) {
    return null;
  }
  const scopePrefix = testScope.value.includes("单节点") ? "单节点" : "草稿";
  const target = testResult.value.finalProvider
    ? providerDisplayName(testResult.value.finalProvider)
    : "查看详情";
  const latency = testResult.value.attempts.reduce(
    (acc, item) => acc + (Number(item.latencyMs) || 0),
    0,
  );
  return {
    tone: testResult.value.ok ? "success" : "warning",
    text: `${scopePrefix}${testResult.value.ok ? "测试通过" : "测试异常"} · ${target}${latency ? ` · ${formatDuration(latency)}` : ""}`,
  };
});

const providerValidationErrors = computed<
  Record<string, ProviderValidationError>
>(() => {
  const scene = draftScene.value;
  const result: Record<string, ProviderValidationError> = {};
  if (!scene) return result;
  const idCounts = new Map<string, number>();
  const nameCounts = new Map<string, number>();
  scene.providers.forEach((provider) => {
    const id = provider.id.trim();
    const name = provider.name.trim();
    if (id) idCounts.set(id, (idCounts.get(id) || 0) + 1);
    if (name) nameCounts.set(name, (nameCounts.get(name) || 0) + 1);
  });
  scene.providers.forEach((provider) => {
    const key = getProviderLocalKey(provider);
    const errors: ProviderValidationError = {};
    const id = provider.id.trim();
    const name = provider.name.trim();
    if (!id) errors.id = "内部 Provider ID 缺失，请重新新增节点";
    else if ((idCounts.get(id) || 0) > 1)
      errors.id = "内部 Provider ID 重复，请重新新增节点";
    if (!name) errors.name = "Provider Name 不能为空";
    else if ((nameCounts.get(name) || 0) > 1)
      errors.name = "Provider Name 在当前场景内重复";
    if (!isValidHttpUrl(provider.baseURL))
      errors.baseURL = "Base URL 必须是合法的 http(s) 地址";
    if (!provider.model.trim()) errors.model = "Model 不能为空";
    const timeout = Number(provider.timeoutSeconds);
    if (!Number.isFinite(timeout) || timeout < 1 || timeout > 600)
      errors.timeoutSeconds = "超时范围必须为 1-600 秒";
    if (Object.keys(errors).length) result[key] = errors;
  });
  return result;
});

const blockingValidationCount = computed(() => {
  return Object.values(providerValidationErrors.value).reduce(
    (acc, item) => acc + Object.keys(item).length,
    0,
  );
});

const recentAudits = computed(() => recentAuditItems.value);

const providerNameById = computed(() => {
  const map = new Map<string, string>();
  const collect = (scene?: AIRoutingSceneConfig | null) => {
    scene?.providers?.forEach((provider) => {
      const id = provider.id.trim();
      const name = provider.name.trim();
      if (id && name) map.set(id, name);
    });
  };
  collect(remoteScene.value);
  collect(draftScene.value);
  return map;
});

const bottomBarState = computed(() => {
  if (testingScene.value) {
    return {
      tone: "primary" as const,
      text: `测试中 · ${testingElapsedSeconds.value}s`,
      icon: Clock,
    };
  }
  if (testResult.value && !testResult.value.ok) {
    const errorType =
      testResult.value.attempts.find((item) => item.errorType)?.errorType ||
      "查看详情";
    return {
      tone: "warning" as const,
      text: `测试异常 · ${errorType}`,
      icon: Warning,
    };
  }
  if (isDirty.value) {
    return {
      tone: "warning" as const,
      text: `有未保存的更改 · ${diffCount.value} 项`,
      icon: Warning,
    };
  }
  if (testResult.value?.ok) {
    return {
      tone: "success" as const,
      text: latestTestSummary.value?.text || "测试通过",
      icon: Check,
    };
  }
  return { tone: "success" as const, text: "已同步", icon: Check };
});

const statusBarText = computed(() => {
  if (testingScene.value) return `测试中 · ${testingElapsedSeconds.value}s`;
  if (isDirty.value) return `草稿 ${diffCount.value} 项 · 保存后影响线上`;
  if (latestTestSummary.value) return latestTestSummary.value.text;
  return "已同步";
});

const isDirty = computed(() => {
  if (!draftScene.value || !remoteScene.value) {
    return false;
  }
  return (
    comparableScene(draftScene.value) !== comparableScene(remoteScene.value)
  );
});

function diffProviderKey(provider: Record<string, unknown>, index: number) {
  const id = String(provider.id || "").trim();
  return id || `__index_${index}`;
}

function providerDiffSnapshot(provider: Record<string, unknown>) {
  return {
    id: String(provider.id || "").trim(),
    name: String(provider.name || "").trim(),
    adapter: String(provider.adapter || "").trim(),
    enabled: Boolean(provider.enabled),
    baseURL: String(provider.baseURL || "").trim(),
    model: String(provider.model || "").trim(),
    timeoutSeconds: Number(provider.timeoutSeconds) || 0,
    endpointMode: String(provider.endpointMode || "chat_completions").trim(),
    responseFormat: String(provider.responseFormat || "auto").trim(),
    clearApiKey: Boolean(provider.clearApiKey),
    apiKey:
      typeof provider.apiKey === "string" && provider.apiKey.trim()
        ? "[已录入]"
        : "",
  };
}

type SceneDiffItem = {
  scope: string;
  path: string;
  from: unknown;
  to: unknown;
};

const sceneDiff = computed<SceneDiffItem[]>(() => {
  if (!draftScene.value || !remoteScene.value) return [];
  const a = buildScenePayload(draftScene.value) as unknown as Record<
    string,
    unknown
  >;
  const b = buildScenePayload(remoteScene.value) as unknown as Record<
    string,
    unknown
  >;
  const results: Array<{
    scope: string;
    path: string;
    from: unknown;
    to: unknown;
  }> = [];
  const sceneFieldKeys = [
    "enabled",
    "strategy",
    "maxAttempts",
    "retryOn",
    "breaker",
    "requestOptions",
  ] as const;
  for (const key of sceneFieldKeys) {
    if (JSON.stringify(a[key]) !== JSON.stringify(b[key])) {
      results.push({ scope: "scene", path: key, from: b[key], to: a[key] });
    }
  }
  const aps = (a.providers as Array<Record<string, unknown>>) || [];
  const bps = (b.providers as Array<Record<string, unknown>>) || [];
  const aEntries = aps.map(
    (provider, index) => [diffProviderKey(provider, index), provider] as const,
  );
  const bEntries = bps.map(
    (provider, index) => [diffProviderKey(provider, index), provider] as const,
  );
  const aMap = new Map(aEntries);
  const bMap = new Map(bEntries);
  const aKeys = aEntries.map(([key]) => key);
  const bKeys = bEntries.map(([key]) => key);
  const sameProviderSet =
    aKeys.length === bKeys.length && aKeys.every((key) => bMap.has(key));
  if (sameProviderSet && JSON.stringify(aKeys) !== JSON.stringify(bKeys)) {
    results.push({
      scope: "providers",
      path: "order",
      from: bKeys,
      to: aKeys,
    });
  }
  const providerKeys = Array.from(new Set([...bKeys, ...aKeys]));
  for (const key of providerKeys) {
    const ap = aMap.get(key);
    const bp = bMap.get(key);
    if (!ap || !bp) {
      results.push({
        scope: `provider:${key}`,
        path: ap ? "added" : "removed",
        from: bp ? providerDiffSnapshot(bp) : null,
        to: ap ? providerDiffSnapshot(ap) : null,
      });
      continue;
    }
    const nextProvider = providerDiffSnapshot(ap);
    const prevProvider = providerDiffSnapshot(bp);
    const fields = Object.keys({ ...prevProvider, ...nextProvider });
    for (const field of fields) {
      if (
        JSON.stringify(nextProvider[field as keyof typeof nextProvider]) !==
        JSON.stringify(prevProvider[field as keyof typeof prevProvider])
      ) {
        results.push({
          scope: `provider:${key}`,
          path: field,
          from: prevProvider[field as keyof typeof prevProvider],
          to: nextProvider[field as keyof typeof nextProvider],
        });
      }
    }
  }
  return results;
});

const diffCount = computed(() => sceneDiff.value.length);

const discardTooltip = computed(() => {
  if (!isDirty.value) return "当前没有未保存改动";
  const n = diffCount.value;
  return n > 0 ? `将丢弃 ${n} 项未保存改动` : "将丢弃当前未保存改动";
});

function onBeforeUnload(e: BeforeUnloadEvent) {
  if (isDirty.value) {
    e.preventDefault();
    e.returnValue = "";
  }
}

onBeforeRouteLeave(async () => {
  return await guardUnsavedChanges("离开页面");
});

onMounted(async () => {
  window.addEventListener("beforeunload", onBeforeUnload);
  const queryScene = readQueryString(route.query, "scene");
  if (sceneKeys.includes(queryScene as AIRoutingSceneKey)) {
    currentSceneKey.value = queryScene as AIRoutingSceneKey;
  }
  await refreshPage();
});

onBeforeUnmount(() => {
  window.removeEventListener("beforeunload", onBeforeUnload);
  stopTestingTimer();
});

watch(
  () => route.query.scene,
  async (value) => {
    const nextScene = String(value || "").trim();
    if (!sceneKeys.includes(nextScene as AIRoutingSceneKey)) {
      return;
    }
    if (nextScene === currentSceneKey.value) {
      return;
    }
    if (routeSceneOverride.value === nextScene) {
      routeSceneOverride.value = null;
      await applySceneChange(nextScene as AIRoutingSceneKey);
      return;
    }
    const allowed = await guardUnsavedChanges(
      `切换到「${displayAIRoutingScene(nextScene)}」`,
    );
    if (!allowed) {
      await router.replace({
        query: buildRouteQuery({ scene: currentSceneKey.value }),
      });
      return;
    }
    await applySceneChange(nextScene as AIRoutingSceneKey);
  },
);

async function refreshPage() {
  const allowed = await guardUnsavedChanges("刷新页面");
  if (!allowed) {
    return;
  }
  pageRefreshing.value = true;
  resetSceneEditor();
  try {
    await Promise.all([loadSceneSummaries(), loadCurrentScene()]);
  } finally {
    pageRefreshing.value = false;
  }
}

async function handleDiscardDraft() {
  if (!remoteScene.value || !isDirty.value) return;
  try {
    await ElMessageBox.confirm("确定放弃所有未保存的改动？", "放弃草稿", {
      type: "warning",
    });
    discardDraftChanges("已恢复到上次保存的状态");
  } catch {
    return;
  }
}

async function loadSceneSummaries() {
  const response = await adminApi.listAIRoutingScenes();
  sceneSummaries.value = response.items;
}

async function loadCurrentScene() {
  sceneLoading.value = true;
  sceneError.value = "";
  try {
    const response = await adminApi.getAIRoutingScene(currentSceneKey.value);
    remoteScene.value = hydrateScene(response.scene);
    draftScene.value = hydrateScene(response.scene);
    resetProviderUIState();
    testResult.value = null;
    testScope.value = "";
    auditPage.value = 1;
    void Promise.all([loadRecentAudits(), loadAudits()]);
  } catch (error) {
    resetSceneEditor();
    sceneError.value = extractMessage(error);
  } finally {
    sceneLoading.value = false;
  }
}

async function loadAudits() {
  auditsLoading.value = true;
  auditsError.value = "";
  try {
    const query = new URLSearchParams();
    query.set("group", currentAuditGroup.value);
    query.set("page", String(auditPage.value));
    query.set("pageSize", "20");
    if (auditAction.value) {
      query.set("action", auditAction.value);
    }
    const response = await adminApi.listSettingAudits(query);
    audits.value = response.result;
  } catch (error) {
    auditsError.value = extractMessage(error);
  } finally {
    auditsLoading.value = false;
  }
}

async function loadRecentAudits() {
  recentAuditsLoading.value = true;
  recentAuditsError.value = "";
  try {
    const query = new URLSearchParams();
    query.set("group", currentAuditGroup.value);
    query.set("page", "1");
    query.set("pageSize", "5");
    const response = await adminApi.listSettingAudits(query);
    recentAuditItems.value = response.result.items;
  } catch (error) {
    recentAuditsError.value = extractMessage(error);
  } finally {
    recentAuditsLoading.value = false;
  }
}

function hydrateScene(scene: AIRoutingSceneConfig): AIRoutingSceneConfig {
  const clone = JSON.parse(JSON.stringify(scene)) as AIRoutingSceneConfig;
  clone.requestOptions ||= {
    stream: false,
    temperature: 0,
    maxTokens: clone.scene === "title" ? 64 : 0,
  };
  clone.breaker ||= { failureThreshold: 3, cooldownSeconds: 60 };
  clone.retryOn ||= retryOptions.map((item) => item.value);
  clone.providers = (clone.providers || []).map((provider, index) => ({
    adapter: "openai-compatible",
    enabled: true,
    priority: (index + 1) * 10,
    timeoutSeconds: 30,
    baseURL: "",
    name: "",
    id: "",
    hasAPIKey: false,
    apiKeyMasked: "",
    ...provider,
    extra: provider.extra || {},
    endpointMode: (provider.endpointMode ||
      "chat_completions") as AIRoutingProviderEndpointMode,
    responseFormat:
      provider.endpointMode === "images_generations"
        ? provider.responseFormat || "auto"
        : "auto",
    apiKey: "",
    clearApiKey: false,
  }));
  return clone;
}

function comparableScene(scene: AIRoutingSceneConfig) {
  const payload = buildScenePayload(scene);
  return JSON.stringify(payload);
}

function buildScenePayload(scene: AIRoutingSceneConfig): AIRoutingSceneConfig {
  const payload = JSON.parse(JSON.stringify(scene)) as AIRoutingSceneConfig;
  payload.providers = payload.providers.map((provider, index) => ({
    ...provider,
    id: provider.id.trim(),
    name: provider.name.trim(),
    adapter: provider.adapter || "openai-compatible",
    baseURL: provider.baseURL.trim(),
    model: provider.model.trim(),
    priority: (index + 1) * 10,
    timeoutSeconds: Number(provider.timeoutSeconds) || 30,
    endpointMode: provider.endpointMode || "chat_completions",
    responseFormat:
      provider.endpointMode === "images_generations"
        ? provider.responseFormat || "auto"
        : "auto",
    apiKey: (provider.apiKey || "").trim(),
    apiKeyMasked: provider.apiKeyMasked || "",
    hasAPIKey: !!provider.hasAPIKey,
    clearApiKey: !!provider.clearApiKey,
  }));
  payload.maxAttempts = Math.min(
    Math.max(Number(payload.maxAttempts) || 1, 1),
    Math.max(payload.providers.filter((item) => item.enabled).length, 1),
  );
  payload.breaker.failureThreshold = Math.max(
    Number(payload.breaker.failureThreshold) || 1,
    1,
  );
  payload.breaker.cooldownSeconds = Math.max(
    Number(payload.breaker.cooldownSeconds) || 5,
    5,
  );
  if (payload.scene !== "title") {
    payload.requestOptions.stream = false;
    payload.requestOptions.temperature = 0;
    payload.requestOptions.maxTokens = 0;
  }
  return payload;
}

function createProvider(scene: AIRoutingSceneKey): AIRoutingProviderConfig {
  const seed = `${scene}-${Date.now().toString(36)}-${Math.random().toString(36).slice(2, 6)}`;
  return {
    id: seed,
    scene,
    name: "",
    adapter: "openai-compatible",
    enabled: true,
    priority: 10,
    weight: 100,
    baseURL: "",
    apiKey: "",
    apiKeyMasked: "",
    hasAPIKey: false,
    clearApiKey: false,
    model: "",
    timeoutSeconds: scene === "flowchart" ? 120 : scene === "title" ? 5 : 30,
    endpointMode:
      scene === "flowchart" ? "images_generations" : "chat_completions",
    responseFormat: scene === "flowchart" ? "b64_json" : "auto",
    extra: {},
  };
}

async function handleSceneChange(scene: AIRoutingSceneKey) {
  if (scene === currentSceneKey.value) {
    return;
  }
  const allowed = await guardUnsavedChanges(
    `切换到「${displayAIRoutingScene(scene)}」`,
  );
  if (!allowed) {
    return;
  }
  routeSceneOverride.value = scene;
  await router.replace({ query: buildRouteQuery({ scene }) });
}

async function applySceneChange(scene: AIRoutingSceneKey) {
  currentSceneKey.value = scene;
  resetSceneEditor();
  await loadCurrentScene();
  if (shouldFocusSceneAfterChange.value) {
    shouldFocusSceneAfterChange.value = false;
    focusSceneCard(scene);
  }
}

function resetSceneEditor() {
  remoteScene.value = null;
  draftScene.value = null;
  testResult.value = null;
  testScope.value = "";
  resetProviderUIState();
}

function handleAddProvider() {
  if (!draftScene.value) {
    return;
  }
  const provider = createProvider(draftScene.value.scene);
  provider.name = buildUniqueProviderName("新节点");
  draftScene.value.providers.push(provider);
  setProviderSecretEditor(provider, true);
}

async function handleRemoveProvider(index: number) {
  if (!draftScene.value) {
    return;
  }
  try {
    await ElMessageBox.confirm(
      "删除后该节点会从当前草稿里移除，保存后才会真正生效。",
      "确认删除节点",
      {
        type: "warning",
      },
    );
    draftScene.value.providers.splice(index, 1);
  } catch {
    return;
  }
}

function handleProviderMenuCommand(command: string, index: number) {
  if (command === "move-up") {
    moveProvider(index, -1);
    return;
  }
  if (command === "move-down") {
    moveProvider(index, 1);
    return;
  }
  if (command === "duplicate") {
    handleDuplicateProvider(index);
    return;
  }
  if (command === "delete") {
    handleRemoveProvider(index);
  }
}

function handleDuplicateProvider(index: number) {
  if (!draftScene.value) {
    return;
  }
  const source = draftScene.value.providers[index];
  const copied = JSON.parse(JSON.stringify(source)) as AIRoutingProviderConfig;
  copied.id = buildDuplicateProviderId(source.id || `provider-${index + 1}`);
  copied.name = buildUniqueProviderName(
    source.name ? `${source.name} 副本` : "新节点",
  );
  copied.apiKey = "";
  copied.apiKeyMasked = "";
  copied.hasAPIKey = false;
  copied.clearApiKey = false;
  draftScene.value.providers.splice(index + 1, 0, copied);
  setProviderSecretEditor(copied, true);
}

function moveProvider(index: number, offset: number) {
  if (!draftScene.value) {
    return;
  }
  const target = index + offset;
  if (target < 0 || target >= draftScene.value.providers.length) {
    return;
  }
  const items = draftScene.value.providers;
  const [current] = items.splice(index, 1);
  items.splice(target, 0, current);
}

function handleProviderDragStart(index: number, event: DragEvent) {
  draggingProviderIndex.value = index;
  dragOverProviderIndex.value = index;
  if (event.dataTransfer) {
    event.dataTransfer.effectAllowed = "move";
    event.dataTransfer.setData("text/plain", String(index));
  }
}

function handleProviderDragOver(index: number) {
  if (
    draggingProviderIndex.value === null ||
    draggingProviderIndex.value === index
  ) {
    return;
  }
  dragOverProviderIndex.value = index;
}

function handleProviderDrop(index: number) {
  if (!draftScene.value || draggingProviderIndex.value === null) {
    handleProviderDragEnd();
    return;
  }
  const sourceIndex = draggingProviderIndex.value;
  if (sourceIndex !== index) {
    const items = draftScene.value.providers;
    const [current] = items.splice(sourceIndex, 1);
    items.splice(index, 0, current);
  }
  handleProviderDragEnd();
}

function handleProviderDragEnd() {
  draggingProviderIndex.value = null;
  dragOverProviderIndex.value = null;
}

function handleProviderApiKeyInput(
  provider: AIRoutingProviderConfig,
  value: string,
) {
  provider.apiKey = value;
  if (value.trim()) {
    provider.clearApiKey = false;
    setProviderSecretEditor(provider, true);
  }
}

function touchProviderField(
  provider: AIRoutingProviderConfig,
  field: keyof ProviderValidationError,
) {
  providerTouchedState.value = {
    ...providerTouchedState.value,
    [`${getProviderLocalKey(provider)}:${field}`]: true,
  };
}

function providerFieldError(
  provider: AIRoutingProviderConfig,
  field: keyof ProviderValidationError,
) {
  const key = getProviderLocalKey(provider);
  if (
    !providerTouchedState.value[`${key}:${field}`] &&
    !providerTouchedState.value[`${key}:__all`]
  ) {
    return "";
  }
  return providerValidationErrors.value[key]?.[field] || "";
}

function touchAllProviderFields() {
  const next = { ...providerTouchedState.value };
  draftScene.value?.providers.forEach((provider) => {
    next[`${getProviderLocalKey(provider)}:__all`] = true;
  });
  providerTouchedState.value = next;
}

function isValidHttpUrl(value: string) {
  const raw = value.trim();
  if (!raw) return false;
  try {
    const parsed = new URL(raw);
    return parsed.protocol === "http:" || parsed.protocol === "https:";
  } catch {
    return false;
  }
}

function getProviderSecretStatus(provider: AIRoutingProviderConfig) {
  if (provider.clearApiKey) return { tone: "warning" as const, text: "待清空" };
  if (provider.apiKey?.trim())
    return { tone: "primary" as const, text: "新密钥草稿" };
  if (provider.hasAPIKey)
    return { tone: "success" as const, text: "已保存密钥" };
  return { tone: "neutral" as const, text: "未配置密钥" };
}

function getProviderTestState(provider: AIRoutingProviderConfig) {
  return providerLastTestState.value[provider.id.trim()];
}

function firstProviderError(provider: AIRoutingProviderConfig) {
  const errors = providerValidationErrors.value[getProviderLocalKey(provider)];
  return errors ? Object.values(errors)[0] || "" : "";
}

function startTestingTimer() {
  testingStartedAt.value = Date.now();
  testingElapsedSeconds.value = 0;
  stopTestingTimer();
  testingTimer = window.setInterval(() => {
    if (testingStartedAt.value) {
      testingElapsedSeconds.value = Math.floor(
        (Date.now() - testingStartedAt.value) / 1000,
      );
    }
  }, 1000);
}

function stopTestingTimer() {
  if (testingTimer) {
    window.clearInterval(testingTimer);
    testingTimer = null;
  }
}

function validateBeforeAction(actionLabel: string) {
  if (!blockingValidationCount.value) {
    return true;
  }
  touchAllProviderFields();
  ElMessage.warning(
    `${actionLabel}前请先修正 ${blockingValidationCount.value} 个 Provider 字段问题`,
  );
  return false;
}

function formatDiffValue(value: unknown) {
  if (value === null || value === undefined || value === "") return "空";
  if (typeof value === "object") return JSON.stringify(value, null, 2);
  return String(value);
}

function formatAuditValue(value: string) {
  if (!value) return "空";
  try {
    return JSON.stringify(JSON.parse(value), null, 2);
  } catch {
    return value;
  }
}

type AuditTone = "neutral" | "primary" | "success" | "warning" | "danger";
type AuditChangeKind = "added" | "removed" | "changed";
type AuditChangeGroupName =
  | "基础信息"
  | "请求配置"
  | "调度策略"
  | "敏感信息"
  | "其他变化";

type AuditBusinessAction = {
  label: string;
  tone: AuditTone;
  description: string;
};

type AuditChangeItem = {
  key: string;
  field: string;
  from: string;
  to: string;
  kind: AuditChangeKind;
  group: AuditChangeGroupName;
  priority: number;
};

function toneForAuditAction(record: SettingAuditRecord) {
  return auditBusinessAction(record).tone;
}

function auditBusinessAction(record: SettingAuditRecord): AuditBusinessAction {
  if (record.action === "test") {
    const text =
      `${record.newValueMasked} ${record.oldValueMasked}`.toLowerCase();
    const timeout = text.includes("timeout") || text.includes("deadline");
    const failed =
      timeout ||
      text.includes("fail") ||
      text.includes("error") ||
      text.includes("refused");
    if (timeout)
      return {
        label: "测试超时",
        tone: "warning",
        description: "测试请求超时，请检查节点连通性、模型响应速度或超时配置。",
      };
    if (failed)
      return {
        label: "测试异常",
        tone: "warning",
        description: record.newValueMasked || "测试未通过，请查看错误摘要。",
      };
    return {
      label: "测试通过",
      tone: "success",
      description: record.newValueMasked || "当前草稿路由测试通过。",
    };
  }

  const changes = auditChangeSummary(record);
  const keys = changes.map((item) => item.key);
  const before = parseAuditKeyValue(record.oldValueMasked);
  const after = parseAuditKeyValue(record.newValueMasked);
  const allAdded =
    changes.length > 0 && changes.every((item) => item.kind === "added");
  const allRemoved =
    changes.length > 0 && changes.every((item) => item.kind === "removed");
  const providerRecord = isProviderAuditRecord(record);

  if (providerRecord && allAdded)
    return {
      label: "新增节点",
      tone: "primary",
      description: providerAddedAuditDescription(after),
    };
  if (providerRecord && allRemoved)
    return {
      label: "删除节点",
      tone: "danger",
      description: "该 Provider 节点已从当前场景移除。",
    };
  if (providerRecord && keys.length === 1) {
    const key = keys[0];
    if (key === "apiKey") {
      if (isAuditEmptyValue(before.apiKey) && !isAuditEmptyValue(after.apiKey))
        return {
          label: "新增密钥",
          tone: "warning",
          description: "该节点已新增密钥，密钥内容已脱敏。",
        };
      if (!isAuditEmptyValue(before.apiKey) && isAuditEmptyValue(after.apiKey))
        return {
          label: "清空密钥",
          tone: "danger",
          description: "该节点旧密钥已清空。",
        };
      return {
        label: "更新密钥",
        tone: "warning",
        description: "该节点密钥已更新，密钥内容已脱敏。",
      };
    }
    if (key === "enabled") {
      return after.enabled === "true"
        ? {
            label: "启用节点",
            tone: "success",
            description: "该 Provider 节点已启用，并参与当前场景调度。",
          }
        : {
            label: "停用节点",
            tone: "neutral",
            description: "该 Provider 节点已停用，不再参与当前场景调度。",
          };
    }
    if (key === "priority")
      return {
        label: "调整顺序",
        tone: "neutral",
        description: "节点调度顺序已更新。",
      };
    if (key === "name")
      return {
        label: "重命名节点",
        tone: "primary",
        description: "节点名称已更新，页面展示、测试结果和审计标识会使用新名称。",
      };
    if (key === "model")
      return {
        label: "修改 Model",
        tone: "primary",
        description: "该节点请求模型已更新。",
      };
    if (key === "baseURL")
      return {
        label: "修改地址",
        tone: "warning",
        description: "该节点 Base URL 已更新。",
      };
    if (key === "timeout")
      return {
        label: "修改超时",
        tone: "neutral",
        description: "该节点请求超时时间已更新。",
      };
  }

  if (providerRecord)
    return {
      label: "修改配置",
      tone: "primary",
      description: `该节点配置已更新，以下 ${changes.length} 项字段发生变化。`,
    };

  if (keys.length === 1) {
    const key = keys[0];
    if (key === "enabled") {
      return after.enabled === "true"
        ? {
            label: "启用场景",
            tone: "success",
            description: "当前场景已启用新路由配置。",
          }
        : {
            label: "停用场景",
            tone: "neutral",
            description: "当前场景已停用新路由配置。",
          };
    }
    if (key === "strategy")
      return {
        label: "修改调度策略",
        tone: "primary",
        description: "当前场景已使用新的 Provider 调度策略。",
      };
    if (key === "maxAttempts")
      return {
        label: "修改重试次数",
        tone: "primary",
        description: "当前场景最大尝试次数已更新。",
      };
    if (key === "providers")
      return {
        label: "节点数量变化",
        tone:
          Number(after.providers) > Number(before.providers)
            ? "primary"
            : "danger",
        description: "当前场景的 Provider 节点数量已变化。",
      };
    if (key === "breaker")
      return {
        label: "修改熔断策略",
        tone: "warning",
        description: "当前场景熔断阈值或冷却时间已更新。",
      };
  }

  if (keys.some((key) => key === "providers"))
    return {
      label: "调整场景策略",
      tone: "primary",
      description: "场景策略和 Provider 数量已同步更新。",
    };
  if (changes.length)
    return {
      label: "调整场景策略",
      tone: "primary",
      description: `场景策略已更新，以下 ${changes.length} 项配置发生变化。`,
    };
  return {
    label: displayAuditAction(record.action),
    tone: "neutral",
    description: auditFallbackSummary(record),
  };
}

function auditEventLabel(record: SettingAuditRecord) {
  if (record.action === "test") return "测试执行";
  if (record.action === "update" || record.action === "save") return "保存发布";
  return displayAuditAction(record.action);
}

function providerAddedAuditDescription(after: Record<string, string>) {
  if (after.enabled === "false") return "已新增停用节点，暂不参与当前场景调度。";
  return "已新增节点，并参与当前场景调度。";
}

function isProviderAuditRecord(record: SettingAuditRecord) {
  return record.settingKey.includes(".provider.");
}

function providerDisplayName(providerId: string) {
  const id = String(providerId || "").trim();
  if (!id) return "未知 Provider";
  return providerNameById.value.get(id) || id;
}

function providerNameFromAuditRecord(record: SettingAuditRecord) {
  const after = parseAuditKeyValue(record.newValueMasked);
  const before = parseAuditKeyValue(record.oldValueMasked);
  return after.name || before.name || "";
}

function auditTargetTitle(target: SettingAuditRecord | string) {
  const settingKey = typeof target === "string" ? target : target.settingKey;
  const prefix = `ai.routing.${currentSceneKey.value}.`;
  const key = settingKey.startsWith(prefix)
    ? settingKey.slice(prefix.length)
    : settingKey;
  if (key === "scene") return "场景策略";
  if (key.startsWith("provider.")) {
    const providerId = key.slice("provider.".length);
    const auditName =
      typeof target === "string" ? "" : providerNameFromAuditRecord(target);
    return (
      auditName || providerNameById.value.get(providerId) || "Provider 节点"
    );
  }
  return key || settingKey;
}

function auditFallbackSummary(record: SettingAuditRecord) {
  if (record.action === "test") {
    return record.newValueMasked || record.oldValueMasked || "测试记录已写入";
  }
  if (record.action === "save" || record.action === "update") {
    return "配置已更新，点击查看变化获取完整对比。";
  }
  return record.newValueMasked || record.oldValueMasked || "暂无摘要";
}

function auditDiffTitle(record: SettingAuditRecord) {
  const target = auditTargetTitle(record);
  if (record.action === "test")
    return `${target} · ${auditBusinessAction(record).label}`;
  return `${target} · ${auditBusinessAction(record).label}`;
}

function auditDiffStatusText(record: SettingAuditRecord) {
  const changes = auditChangeSummary(record);
  if (!changes.length) return auditBusinessAction(record).label;
  const stats = auditChangeStats(record);
  return stats.map((item) => `${item.label} ${item.count}`).join(" · ");
}

function auditChangeSummary(record: SettingAuditRecord): AuditChangeItem[] {
  if (record.action === "test") return [];
  const before = parseAuditKeyValue(record.oldValueMasked);
  const after = parseAuditKeyValue(record.newValueMasked);
  const keys = Array.from(
    new Set([...Object.keys(before), ...Object.keys(after)]),
  );
  return keys
    .filter((key) => !auditValuesEqual(before[key], after[key]))
    .map((key) => ({
      key,
      field: displayAuditField(key),
      from: compactAuditValue(before[key], key),
      to: compactAuditValue(after[key], key),
      kind: auditChangeKind(before[key], after[key]),
      group: auditChangeGroup(key),
      priority: auditChangePriority(key),
    }))
    .sort(
      (left, right) =>
        left.priority - right.priority || left.field.localeCompare(right.field),
    );
}

function groupedAuditChanges(record: SettingAuditRecord) {
  const groups: AuditChangeGroupName[] = [
    "基础信息",
    "请求配置",
    "调度策略",
    "敏感信息",
    "其他变化",
  ];
  const changes = auditChangeSummary(record);
  return groups
    .map((name) => ({
      name,
      changes: changes.filter((item) => item.group === name),
    }))
    .filter((group) => group.changes.length);
}

function auditChangeStats(record: SettingAuditRecord) {
  const changes = auditChangeSummary(record);
  const configs = [
    { kind: "added" as const, label: "新增" },
    { kind: "changed" as const, label: "修改" },
    { kind: "removed" as const, label: "删除" },
  ];
  return configs
    .map((config) => ({
      ...config,
      count: changes.filter((item) => item.kind === config.kind).length,
    }))
    .filter((item) => item.count > 0);
}

function auditChangeKind(before?: string, after?: string): AuditChangeKind {
  const oldEmpty = isAuditEmptyValue(before);
  const newEmpty = isAuditEmptyValue(after);
  if (!oldEmpty && newEmpty) return "removed";
  if (oldEmpty && !newEmpty) return "added";
  return "changed";
}

function auditValuesEqual(before?: string, after?: string) {
  if (isAuditEmptyValue(before) && isAuditEmptyValue(after)) return true;
  return before === after;
}

function auditChangeKindText(kind: string) {
  if (kind === "removed") return "删除";
  if (kind === "added") return "新增";
  return "修改";
}

function isAuditEmptyValue(value?: string) {
  return (
    value === undefined ||
    value === null ||
    value === "" ||
    value === "空" ||
    value === "未设置" ||
    value === "-"
  );
}

function parseAuditKeyValue(value: string) {
  const result: Record<string, string> = {};
  const raw = value.trim();
  if (!raw) return result;
  try {
    const parsed = JSON.parse(raw) as Record<string, unknown>;
    Object.entries(parsed || {}).forEach(([key, val]) => {
      result[key] = String(val ?? "");
    });
    return result;
  } catch {
    // Continue with key=value parser.
  }
  const matches = raw.matchAll(
    /([A-Za-z][\w.-]*)=([^=]+?)(?=\s+[A-Za-z][\w.-]*=|$)/g,
  );
  for (const match of matches) {
    result[match[1]] = match[2].trim();
  }
  return result;
}

function displayAuditField(field: string) {
  const map: Record<string, string> = {
    enabled: "启用状态",
    strategy: "调度策略",
    maxAttempts: "最大尝试次数",
    retryOn: "重试条件",
    breaker: "熔断策略",
    providers: "节点数量",
    requestOptions: "请求参数",
    name: "名称",
    model: "Model",
    baseURL: "Base URL",
    timeout: "超时",
    timeoutSeconds: "超时",
    endpoint: "Endpoint",
    endpointMode: "Endpoint",
    responseFormat: "返回格式",
    priority: "顺序",
    adapter: "适配器",
    apiKey: "密钥状态",
    hasAPIKey: "密钥状态",
    clearApiKey: "清空密钥",
  };
  return map[field] || field;
}

function auditChangeGroup(field: string): AuditChangeGroupName {
  if (["name", "enabled", "model"].includes(field)) return "基础信息";
  if (
    [
      "baseURL",
      "timeout",
      "timeoutSeconds",
      "endpoint",
      "endpointMode",
      "responseFormat",
      "adapter",
      "requestOptions",
    ].includes(field)
  )
    return "请求配置";
  if (
    [
      "priority",
      "strategy",
      "maxAttempts",
      "retryOn",
      "breaker",
      "providers",
    ].includes(field)
  )
    return "调度策略";
  if (/apiKey|token|secret|password|clearApiKey|hasAPIKey/i.test(field))
    return "敏感信息";
  return "其他变化";
}

function auditChangePriority(field: string) {
  const map: Record<string, number> = {
    apiKey: 1,
    enabled: 2,
    name: 3,
    model: 4,
    baseURL: 5,
    timeout: 6,
    timeoutSeconds: 6,
    endpoint: 7,
    endpointMode: 7,
    responseFormat: 8,
    priority: 9,
    strategy: 10,
    maxAttempts: 11,
    breaker: 12,
    providers: 13,
    requestOptions: 14,
  };
  return map[field] || 99;
}

function compactAuditValue(value: unknown, field = "") {
  if (value === undefined || value === null || value === "" || value === "空")
    return "未设置";
  if (/apiKey|token|secret|password/i.test(field)) return "已设置（脱敏）";
  if (typeof value === "boolean") return value ? "启用" : "停用";
  if (typeof value === "object") return JSON.stringify(value);
  const text = String(value);
  if (field === "enabled")
    return text === "true" ? "启用" : text === "false" ? "停用" : text;
  if (/^(sk-|Bearer\s+)/i.test(text)) return "已设置（脱敏）";
  if (text === "true") return "是";
  if (text === "false") return "否";
  return text.length > 72 ? `${text.slice(0, 69)}...` : text;
}

async function handleClearProviderApiKey(provider: AIRoutingProviderConfig) {
  if (provider.clearApiKey) {
    provider.clearApiKey = false;
    ElMessage.info("已撤销清空密钥");
    return;
  }
  try {
    await ElMessageBox.confirm(
      "清空后需要保存当前场景才会真正移除旧密钥，是否继续？",
      "确认清空密钥",
      {
        type: "warning",
      },
    );
  } catch {
    return;
  }
  provider.apiKey = "";
  provider.clearApiKey = true;
  setProviderSecretEditor(provider, false);
  ElMessage.warning("已标记为清空密钥，保存后生效");
}

async function handleSaveScene() {
  await saveCurrentScene();
}

async function saveCurrentScene(successMessage = "场景配置已保存") {
  if (!draftScene.value) {
    return false;
  }
  if (!validateBeforeAction("保存")) {
    return false;
  }
  if (isDirty.value) {
    try {
      await ElMessageBox.confirm(
        buildSaveConfirmVNode(),
        `发布「${currentSceneTitle.value}」配置变更`,
        {
          type: "warning",
          customClass: "save-confirm-message-box",
          confirmButtonText: "保存并发布",
          cancelButtonText: "再检查一下",
          dangerouslyUseHTMLString: false,
        },
      );
    } catch {
      return false;
    }
  }
  savingScene.value = true;
  try {
    const response = await adminApi.updateAIRoutingScene(
      currentSceneKey.value,
      buildScenePayload(draftScene.value),
    );
    remoteScene.value = hydrateScene(response.scene);
    draftScene.value = hydrateScene(response.scene);
    resetProviderUIState();
    await loadSceneSummaries();
    await Promise.all([loadRecentAudits(), loadAudits()]);
    ElMessage.success(successMessage);
    return true;
  } catch (error) {
    ElMessage.error(extractMessage(error));
    return false;
  } finally {
    savingScene.value = false;
  }
}

type SaveConfirmDiffKind =
  | "added"
  | "removed"
  | "changed"
  | "secret"
  | "warning";

type SaveConfirmDiffRow = {
  kind: SaveConfirmDiffKind;
  tag: string;
  title: string;
  description: string;
  before?: string;
  after?: string;
  beforeLabel?: string;
  afterLabel?: string;
};

function providerSnapshotTitle(value: unknown, fallback: string) {
  if (!value || typeof value !== "object") return providerDisplayName(fallback);
  const item = value as Record<string, unknown>;
  const name = String(item.name || "").trim();
  const id = String(item.id || fallback).trim();
  return name || providerDisplayName(id);
}

function diffFieldLabel(path: string) {
  const labels: Record<string, string> = {
    enabled: "启用状态",
    strategy: "调度策略",
    maxAttempts: "最大尝试次数",
    retryOn: "重试条件",
    breaker: "熔断策略",
    requestOptions: "请求参数",
    order: "Provider 顺序",
    name: "展示名称",
    adapter: "适配器",
    baseURL: "Base URL",
    model: "Model",
    timeoutSeconds: "超时时间",
    endpointMode: "接口模式",
    responseFormat: "响应格式",
    apiKey: "密钥",
    clearApiKey: "密钥清空标记",
  };
  return labels[path] || path;
}

function saveConfirmDiffKind(item: SceneDiffItem): SaveConfirmDiffKind {
  if (item.path === "added") return "added";
  if (item.path === "removed") return "removed";
  if (item.path === "apiKey" || item.path === "clearApiKey") return "secret";
  if (item.scope === "providers" && item.path === "order") return "warning";
  return "changed";
}

function saveConfirmDiffTag(kind: SaveConfirmDiffKind) {
  const tags: Record<SaveConfirmDiffKind, string> = {
    added: "新增",
    removed: "删除",
    changed: "修改",
    secret: "密钥",
    warning: "顺序",
  };
  return tags[kind];
}

function secretStateLabel(value: unknown) {
  if (value === null || value === undefined || value === "" || value === false)
    return "未录入";
  const text = String(value);
  if (text === "空" || text === "false") return "未录入";
  if (text === "true") return "已录入";
  return text.replace(/^\[|\]$/g, "");
}

function formatSaveConfirmDiffItem(item: SceneDiffItem): SaveConfirmDiffRow {
  const kind = saveConfirmDiffKind(item);
  const providerId = item.scope.startsWith("provider:")
    ? item.scope.replace("provider:", "")
    : "";
  if (kind === "added") {
    const title = providerSnapshotTitle(item.to, providerId);
    return {
      kind,
      tag: saveConfirmDiffTag(kind),
      title,
      description: "Provider 将新增。",
    };
  }
  if (kind === "removed") {
    const title = providerSnapshotTitle(item.from, providerId);
    return {
      kind,
      tag: saveConfirmDiffTag(kind),
      title,
      description: "Provider 将移除。",
    };
  }
  if (item.scope === "providers" && item.path === "order") {
    return {
      kind,
      tag: saveConfirmDiffTag(kind),
      title: "Provider 顺序",
      description: "调度顺序将调整。",
      before: formatDiffValue(item.from),
      after: formatDiffValue(item.to),
    };
  }
  const field = diffFieldLabel(item.path);
  if (providerId) {
    const title = providerSnapshotTitle(item.to || item.from, providerId);
    if (kind === "secret") {
      return {
        kind,
        tag: saveConfirmDiffTag(kind),
        title,
        description: "密钥将更新，内容已脱敏。",
        before: secretStateLabel(item.from),
        after: secretStateLabel(item.to),
        beforeLabel: "当前",
        afterLabel: "发布后",
      };
    }
    return {
      kind,
      tag: saveConfirmDiffTag(kind),
      title,
      description: `${field} 将更新。`,
      before: formatDiffValue(item.from),
      after: formatDiffValue(item.to),
      beforeLabel: "当前",
      afterLabel: "发布后",
    };
  }
  return {
    kind,
    tag: saveConfirmDiffTag(kind),
    title: `场景${field}`,
    description: "配置将更新。",
    before: formatDiffValue(item.from),
    after: formatDiffValue(item.to),
    beforeLabel: "当前",
    afterLabel: "发布后",
  };
}

function saveConfirmTestChipText() {
  if (testingScene.value) return `测试中 · ${testingElapsedSeconds.value}s`;
  if (!testResult.value) return "未测试";
  return testResult.value.ok ? "测试通过" : "测试异常";
}

function saveConfirmTestChipKind(): SaveConfirmDiffKind {
  if (testingScene.value) return "changed";
  if (!testResult.value) return "warning";
  return testResult.value.ok ? "added" : "removed";
}

function saveConfirmNoticeText() {
  if (testingScene.value) return "测试仍在进行，建议等待结果后发布。";
  if (!testResult.value) return "当前草稿未测试，建议先测试。";
  return testResult.value.ok
    ? "最近测试通过，可以发布。"
    : "最近测试异常，建议先排查后发布。";
}

function buildSaveConfirmVNode() {
  const rows = sceneDiff.value.slice(0, 5).map(formatSaveConfirmDiffItem);
  const moreCount = Math.max(sceneDiff.value.length - rows.length, 0);
  const testKind = saveConfirmTestChipKind();
  return h("div", { class: "save-confirm", role: "document" }, [
    h("div", { class: "save-confirm__chips", "aria-label": "保存影响摘要" }, [
      h(
        "span",
        { class: "save-confirm-chip save-confirm-chip--changed" },
        `${diffCount.value} 项变更`,
      ),
      h(
        "span",
        { class: "save-confirm-chip save-confirm-chip--warning" },
        "线上生效",
      ),
      h(
        "span",
        { class: `save-confirm-chip save-confirm-chip--${testKind}` },
        saveConfirmTestChipText(),
      ),
    ]),
    h(
      "p",
      { class: "save-confirm__lead" },
      `保存后立即更新线上「${currentSceneTitle.value}」。`,
    ),
    h(
      "div",
      {
        class: `save-confirm__notice save-confirm__notice--${testKind}`,
        role: testKind === "added" ? "status" : "alert",
      },
      saveConfirmNoticeText(),
    ),
    h("div", { class: "save-confirm__section-title" }, "变更摘要"),
    h(
      "div",
      { class: "save-confirm__list" },
      rows.map((row) =>
        h("div", { class: `save-confirm-row save-confirm-row--${row.kind}` }, [
          h(
            "span",
            {
              class: `save-confirm-row__tag save-confirm-row__tag--${row.kind}`,
            },
            row.tag,
          ),
          h("div", { class: "save-confirm-row__content" }, [
            h("strong", row.title),
            h("span", row.description),
            row.before !== undefined || row.after !== undefined
              ? h("div", { class: "save-confirm-row__values" }, [
                  h(
                    "span",
                    { class: "save-confirm-row__value-pill" },
                    `${row.beforeLabel || "当前"}：${row.before || "空"}`,
                  ),
                  h("span", { class: "save-confirm-row__arrow" }, "→"),
                  h(
                    "span",
                    {
                      class:
                        "save-confirm-row__value-pill save-confirm-row__value-pill--new",
                    },
                    `${row.afterLabel || "发布后"}：${row.after || "空"}`,
                  ),
                ])
              : null,
          ]),
        ]),
      ),
    ),
    moreCount > 0
      ? h(
          "div",
          { class: "save-confirm__more" },
          `另有 ${moreCount} 项改动，可在底部「草稿摘要」查看。`,
        )
      : null,
  ]);
}

async function handleTestScene() {
  if (testingScene.value) {
    return;
  }
  if (!validateBeforeAction("测试")) {
    return;
  }
  await runSceneTest("当前草稿测试", draftScene.value);
}

async function handleTestSingleProvider(index: number) {
  if (!draftScene.value || testingScene.value) {
    return;
  }
  const provider = draftScene.value.providers[index];
  if (!validateBeforeAction("测试")) {
    return;
  }
  const payload = buildScenePayload(draftScene.value);
  payload.enabled = true;
  payload.maxAttempts = 1;
  payload.providers = payload.providers.map((item, itemIndex) => ({
    ...item,
    enabled: itemIndex === index,
  }));
  singleTestProviderId.value = provider.id;
  await runSceneTest(`单节点测试：${provider.name || provider.id}`, payload);
  singleTestProviderId.value = "";
}

async function runSceneTest(scope: string, scene: AIRoutingSceneConfig | null) {
  if (!scene || testingScene.value) {
    return;
  }
  testingScene.value = true;
  startTestingTimer();
  try {
    const response = await adminApi.testAIRoutingScene(
      currentSceneKey.value,
      buildScenePayload(scene),
    );
    testResult.value = response.result;
    testScope.value = scope;
    recordProviderTestState(response.result);
    await Promise.all([loadRecentAudits(), loadAudits()]);
    if (response.result.ok) {
      ElMessage.success("路由测试通过");
    } else {
      ElMessage.warning(response.result.message || "路由测试失败");
    }
  } catch (error) {
    ElMessage.error(extractMessage(error));
  } finally {
    testingScene.value = false;
    stopTestingTimer();
  }
}

function recordProviderTestState(result: AIRoutingTestResult) {
  const testedAt = new Date().toISOString();
  const next = { ...providerLastTestState.value };
  result.attempts.forEach((attempt) => {
    const ok = attempt.status === "success";
    next[attempt.providerId] = {
      ok,
      text: ok
        ? `通过 · ${formatDuration(attempt.latencyMs)}`
        : `${attempt.errorType || displayCallStatus(attempt.status)} · ${formatDuration(attempt.latencyMs)}`,
      latencyMs: attempt.latencyMs,
      errorType: attempt.errorType,
      testedAt,
    };
  });
  providerLastTestState.value = next;
}

function getProviderLocalKey(provider: AIRoutingProviderConfig) {
  const existing = providerLocalKeys.get(provider);
  if (existing) {
    return existing;
  }
  const key = `provider-local-${(providerLocalKeyCounter += 1)}`;
  providerLocalKeys.set(provider, key);
  return key;
}

function setProviderSecretEditor(
  provider: AIRoutingProviderConfig,
  open: boolean,
) {
  providerSecretEditorState.value = {
    ...providerSecretEditorState.value,
    [getProviderLocalKey(provider)]: open,
  };
}

function preserveViewportAfterUpdate(update: () => void) {
  const scrollX = window.scrollX;
  const scrollY = window.scrollY;
  update();
  nextTick(() => {
    window.scrollTo(scrollX, scrollY);
  });
}

function toggleProviderSecretEditor(provider: AIRoutingProviderConfig) {
  preserveViewportAfterUpdate(() => {
    if (provider.clearApiKey) {
      provider.clearApiKey = false;
    }
    setProviderSecretEditor(
      provider,
      !shouldShowProviderSecretEditor(provider),
    );
  });
}

function shouldShowProviderSecretEditor(provider: AIRoutingProviderConfig) {
  return (
    !provider.hasAPIKey ||
    !!provider.apiKey?.trim() ||
    !!providerSecretEditorState.value[getProviderLocalKey(provider)]
  );
}

function isProviderCollapsed(provider: AIRoutingProviderConfig) {
  return collapsedProviderKeys.value.has(getProviderLocalKey(provider));
}

function toggleProviderCollapsed(provider: AIRoutingProviderConfig) {
  preserveViewportAfterUpdate(() => {
    const key = getProviderLocalKey(provider);
    const next = new Set(collapsedProviderKeys.value);
    if (next.has(key)) next.delete(key);
    else next.add(key);
    collapsedProviderKeys.value = next;
  });
}

function resetProviderUIState() {
  providerSecretEditorState.value = {};
  handleProviderDragEnd();
  if (draftScene.value && draftScene.value.providers.length > 3) {
    const keys = new Set<string>();
    draftScene.value.providers.forEach((p, idx) => {
      if (idx > 0) keys.add(getProviderLocalKey(p));
    });
    collapsedProviderKeys.value = keys;
  } else {
    collapsedProviderKeys.value = new Set();
  }
}

function discardDraftChanges(message?: string) {
  if (!remoteScene.value) {
    return;
  }
  draftScene.value = hydrateScene(remoteScene.value);
  testResult.value = null;
  testScope.value = "";
  resetProviderUIState();
  if (message) {
    ElMessage.info(message);
  }
}

async function guardUnsavedChanges(actionLabel: string) {
  if (!isDirty.value) {
    return true;
  }
  const decision = await resolveUnsavedDraftAction(actionLabel);
  if (decision === "cancel") {
    return false;
  }
  if (decision === "save") {
    return await saveCurrentScene(`${actionLabel}前已保存当前场景`);
  }
  discardDraftChanges("已放弃未保存草稿");
  return true;
}

async function resolveUnsavedDraftAction(
  actionLabel: string,
): Promise<"save" | "discard" | "cancel"> {
  try {
    await ElMessageBox.confirm(
      `当前场景有未保存的改动。${actionLabel}前，请先处理这些草稿。`,
      "处理未保存草稿",
      {
        type: "warning",
        confirmButtonText: "保存并继续",
        cancelButtonText: "放弃更改",
        distinguishCancelAndClose: true,
        closeOnClickModal: false,
        closeOnPressEscape: false,
      },
    );
    return "save";
  } catch (error) {
    return error === "cancel" ? "discard" : "cancel";
  }
}

function buildUniqueProviderName(seed: string) {
  const base = seed.trim() || "新节点";
  const used = new Set(
    draftScene.value?.providers
      .map((item) => item.name.trim())
      .filter(Boolean) || [],
  );
  if (!used.has(base)) return base;
  let index = 2;
  let candidate = `${base} ${index}`;
  while (used.has(candidate)) {
    index += 1;
    candidate = `${base} ${index}`;
  }
  return candidate;
}

function buildDuplicateProviderId(seed: string) {
  if (!draftScene.value) {
    return `${seed}-copy`;
  }
  const used = new Set(draftScene.value.providers.map((item) => item.id));
  let candidate = `${seed}-copy`;
  let index = 2;
  while (used.has(candidate)) {
    candidate = `${seed}-copy-${index}`;
    index += 1;
  }
  return candidate;
}

function scrollToTestCard() {
  nextTick(() => {
    testCardRef.value?.scrollIntoView({ behavior: "smooth", block: "start" });
  });
}

function handleSceneArrowKey(offset: number) {
  const currentIndex = sceneKeys.indexOf(currentSceneKey.value);
  if (currentIndex < 0) {
    return;
  }
  const nextIndex =
    (currentIndex + offset + sceneKeys.length) % sceneKeys.length;
  shouldFocusSceneAfterChange.value = true;
  handleSceneChange(sceneKeys[nextIndex]);
}

function setSceneCardRef(scene: AIRoutingSceneKey, element: Element | null) {
  sceneCardRefs[scene] = element instanceof HTMLButtonElement ? element : null;
}

function focusSceneCard(scene: AIRoutingSceneKey) {
  nextTick(() => {
    sceneCardRefs[scene]?.focus();
  });
}

function resetAuditFilters() {
  auditAction.value = "";
  auditPage.value = 1;
  loadAudits();
}

function handleAuditPageChange(page: number) {
  auditPage.value = page;
  loadAudits();
}

function extractMessage(error: unknown) {
  return error instanceof Error ? error.message : "请求失败";
}
</script>

<style scoped>
.toolbar-cluster {
  display: inline-flex;
  align-items: center;
  gap: 8px;
}

.toolbar-discard-wrap {
  display: inline-flex;
}

:global(.layout-main:has(.routing-bottom-bar)) {
  padding-bottom: 104px;
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
  background: linear-gradient(
    180deg,
    rgba(255, 255, 255, 0.98),
    rgba(247, 250, 252, 0.94)
  );
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
  background: linear-gradient(
    180deg,
    rgba(239, 246, 255, 0.96),
    rgba(255, 255, 255, 0.96)
  );
  box-shadow: 0 18px 40px rgba(37, 99, 235, 0.08);
}

.routing-scene-card--active::before {
  content: "";
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
  background: color-mix(
    in srgb,
    var(--color-bg-elevated, #ffffff) 85%,
    transparent
  );
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

.routing-status-strip {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 14px;
  padding: 12px 16px;
  margin: -4px 0 16px;
  border-radius: 12px;
  border: 1px solid rgba(148, 163, 184, 0.2);
  background: rgba(248, 250, 252, 0.88);
  color: var(--color-text-subtle, #64748b);
  font-size: 13px;
}

.routing-status-strip__main,
.routing-status-strip__actions {
  display: inline-flex;
  align-items: center;
  flex-wrap: wrap;
  gap: 8px;
}

.routing-status-strip__main strong {
  color: var(--color-text, #1f2937);
}

.routing-status-strip code {
  padding: 1px 7px;
  border-radius: 999px;
  background: rgba(15, 23, 42, 0.06);
  font-size: 12px;
}

.channel-popover__text {
  margin: 0 0 10px;
  color: var(--color-text-subtle, #64748b);
  font-size: 13px;
  line-height: 1.7;
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
  border-color: color-mix(
    in srgb,
    var(--color-success, #10b981) 40%,
    transparent
  );
  color: var(--color-success, #10b981);
}

.routing-breadcrumb__channel--warning {
  border-color: color-mix(
    in srgb,
    var(--color-warning, #d97706) 40%,
    transparent
  );
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
  background: color-mix(
    in srgb,
    var(--color-primary, #409eff) 10%,
    transparent
  );
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
  border-color: color-mix(
    in srgb,
    var(--color-primary, #409eff) 30%,
    transparent
  );
}

.routing-timeline-hint__seg--breaker {
  color: var(--color-warning, #d97706);
  border-color: color-mix(
    in srgb,
    var(--color-warning, #d97706) 35%,
    transparent
  );
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
  overflow-anchor: none;
}

.provider-editor-card {
  overflow-anchor: none;
  padding: 16px 18px 18px;
  border-radius: 18px;
  border: 1px solid rgba(148, 163, 184, 0.2);
  background: linear-gradient(
    180deg,
    rgba(255, 255, 255, 0.99),
    rgba(248, 250, 252, 0.94)
  );
  transition:
    border-color 0.18s ease,
    box-shadow 0.18s ease,
    transform 0.18s ease;
}

.provider-editor-card:hover {
  border-color: rgba(37, 99, 235, 0.26);
  box-shadow: 0 14px 34px rgba(15, 23, 42, 0.07);
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
  gap: 18px;
  margin-bottom: 14px;
}

.provider-editor-card__main {
  min-width: 0;
  flex: 1;
}

.provider-editor-card__title {
  display: flex;
  align-items: center;
  gap: 8px;
  min-width: 0;
}

.provider-title-stack {
  display: grid;
  gap: 3px;
  min-width: 0;
}

.provider-title-stack__name-row {
  display: flex;
  align-items: center;
  flex-wrap: wrap;
  gap: 8px;
}

.provider-title-stack__name-row strong {
  color: var(--color-text, #111827);
  font-size: 16px;
  line-height: 1.35;
}

.provider-title-stack__id {
  max-width: 360px;
  overflow: hidden;
  color: var(--color-text-subtle, #64748b);
  font-size: 13px;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.provider-editor-card__meta {
  display: flex;
  flex-wrap: wrap;
  gap: 8px 12px;
  margin-top: 8px;
  color: var(--color-text-subtle);
  font-size: 12px;
}

.provider-compact-grid {
  display: flex;
  flex-wrap: wrap;
  align-items: center;
  gap: 8px;
  margin-top: 10px;
  color: var(--color-text-subtle);
  font-size: 12px;
}

.provider-compact-grid__item {
  display: inline-flex;
  align-items: center;
  min-height: 24px;
  max-width: 280px;
  padding: 0 9px;
  overflow: hidden;
  border-radius: 999px;
  background: rgba(148, 163, 184, 0.1);
  color: #475569;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.provider-test-empty {
  display: inline-flex;
  align-items: center;
  min-height: 26px;
  padding: 0 10px;
  border-radius: 999px;
  background: rgba(148, 163, 184, 0.1);
  color: var(--color-text-subtle);
  font-size: 12px;
  font-weight: 600;
}

.provider-inline-error {
  margin-top: 8px;
  color: var(--color-warning, #d97706);
  font-size: 12px;
  font-weight: 600;
}

.provider-editor-card__controls {
  display: flex;
  flex-wrap: nowrap;
  gap: 10px;
  align-items: center;
  justify-content: flex-end;
}

.provider-enable-control {
  display: inline-flex;
  align-items: center;
  gap: 8px;
  min-height: 34px;
  padding: 0 8px 0 10px;
  border-radius: 999px;
  background: rgba(37, 99, 235, 0.08);
  color: var(--color-primary, #2563eb);
  font-size: 12px;
  font-weight: 700;
  white-space: nowrap;
}

.provider-enable-control--off {
  background: rgba(148, 163, 184, 0.12);
  color: var(--color-text-subtle, #64748b);
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

.provider-icon-button:focus-visible {
  outline: none;
  border-color: rgba(37, 99, 235, 0.42);
  box-shadow: 0 0 0 3px rgba(37, 99, 235, 0.16);
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

.provider-icon-button--primary {
  border-color: rgba(37, 99, 235, 0.22);
  color: var(--color-primary, #2563eb);
  background: rgba(37, 99, 235, 0.08);
}

.provider-drag-handle {
  color: #94a3b8;
}

.provider-editor-section {
  margin-top: 14px;
  padding-top: 14px;
  border-top: 1px solid rgba(148, 163, 184, 0.14);
}

.provider-editor-section__header {
  display: flex;
  align-items: baseline;
  gap: 10px;
  margin-bottom: 12px;
}

.provider-editor-section__header strong {
  color: var(--color-text, #1f2937);
  font-size: 14px;
}

.provider-editor-section__header span {
  color: var(--color-text-subtle, #64748b);
  font-size: 12px;
}

.provider-editor-grid {
  display: grid;
  grid-template-columns: repeat(2, minmax(0, 1fr));
  gap: 14px 16px;
}

.provider-editor-grid__wide {
  grid-column: 1 / -1;
}

.provider-editor-secret {
  margin-top: 14px;
  padding: 12px 14px;
  border-radius: 14px;
  border: 1px solid rgba(148, 163, 184, 0.16);
  background: rgba(248, 250, 252, 0.62);
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

.audit-diff-grid {
  display: grid;
  grid-template-columns: repeat(2, minmax(0, 1fr));
  gap: 14px;
  padding: 12px 18px;
  background: rgba(248, 250, 252, 0.82);
}

.audit-diff-grid--popover {
  padding: 0;
  background: transparent;
}

.audit-diff-popover {
  display: grid;
  gap: 14px;
}

.audit-diff-popover__header {
  display: flex;
  align-items: flex-start;
  justify-content: space-between;
  gap: 12px;
  padding-bottom: 12px;
  border-bottom: 1px solid rgba(148, 163, 184, 0.18);
}

.audit-diff-popover__header strong,
.audit-diff-popover__header span {
  display: block;
}

.audit-diff-popover__header strong {
  color: var(--color-text, #1f2937);
  font-size: 15px;
}

.audit-diff-popover__header span {
  margin-top: 5px;
  color: var(--color-text-subtle, #64748b);
  font-size: 12px;
}

.audit-diff-popover__changes {
  display: grid;
  gap: 8px;
}

.audit-diff-row {
  display: grid;
  grid-template-columns: minmax(92px, 136px) minmax(0, 1fr) 16px minmax(
      0,
      1fr
    ) auto;
  align-items: center;
  gap: 8px;
  min-height: 34px;
  padding: 6px 8px;
  border: 1px solid rgba(148, 163, 184, 0.16);
  border-radius: 10px;
  background: rgba(248, 250, 252, 0.82);
  font-size: 12px;
}

.audit-diff-row--added {
  border-color: rgba(22, 163, 74, 0.22);
  background: rgba(22, 163, 74, 0.07);
}

.audit-diff-row--removed {
  border-color: rgba(220, 38, 38, 0.2);
  background: rgba(220, 38, 38, 0.06);
}

.audit-diff-row__field {
  color: var(--color-text, #1f2937);
  font-weight: 700;
}

.audit-diff-row__value {
  min-width: 0;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
  font-family: ui-monospace, SFMono-Regular, Menlo, Monaco, Consolas, monospace;
  color: var(--color-text-subtle, #64748b);
}

.audit-diff-row__value--new {
  color: var(--color-primary, #2563eb);
}

.audit-diff-row__kind {
  display: inline-flex;
  align-items: center;
  min-height: 22px;
  padding: 0 8px;
  border-radius: 999px;
  background: rgba(15, 23, 42, 0.06);
  color: var(--color-text-subtle, #64748b);
  font-size: 11px;
  font-weight: 700;
  white-space: nowrap;
}

.audit-diff-row--added .audit-diff-row__kind {
  background: rgba(22, 163, 74, 0.12);
  color: #15803d;
}

.audit-diff-row--removed .audit-diff-row__kind {
  background: rgba(220, 38, 38, 0.1);
  color: #b91c1c;
}

.audit-diff-popover__fallback,
.audit-diff-popover__more {
  color: var(--color-text-subtle, #64748b);
  font-size: 12px;
  line-height: 1.7;
}

.audit-raw-collapse {
  border-top: 1px solid rgba(148, 163, 184, 0.16);
  border-bottom: none;
}

.audit-raw-collapse :deep(.el-collapse-item__header) {
  height: 38px;
  border-bottom: none;
  color: var(--color-text-subtle, #64748b);
  font-size: 12px;
  font-weight: 700;
}

.audit-raw-collapse :deep(.el-collapse-item__wrap) {
  border-bottom: none;
}

.audit-drawer-detail {
  display: grid;
  gap: 14px;
  padding: 14px 18px 18px;
  background: linear-gradient(180deg, rgba(248, 250, 252, 0.92), #fff);
}

.audit-timeline-item__insight {
  padding: 8px 10px;
  border: 1px solid rgba(99, 102, 241, 0.16);
  border-radius: 10px;
  background: rgba(99, 102, 241, 0.07);
  color: #3730a3;
  font-size: 12px;
  line-height: 1.6;
}

.audit-diff-summary-strip {
  display: flex;
  flex-wrap: wrap;
  gap: 8px;
}

.audit-diff-stat {
  display: inline-flex;
  align-items: center;
  min-height: 24px;
  padding: 2px 10px;
  border: 1px solid rgba(148, 163, 184, 0.2);
  border-radius: 999px;
  background: rgba(248, 250, 252, 0.86);
  color: var(--color-text-subtle, #64748b);
  font-size: 12px;
  font-weight: 700;
}

.audit-diff-stat--added {
  border-color: rgba(22, 163, 74, 0.24);
  background: rgba(22, 163, 74, 0.08);
  color: #15803d;
}

.audit-diff-stat--changed {
  border-color: rgba(37, 99, 235, 0.24);
  background: rgba(37, 99, 235, 0.08);
  color: #1d4ed8;
}

.audit-diff-stat--removed {
  border-color: rgba(220, 38, 38, 0.22);
  background: rgba(220, 38, 38, 0.07);
  color: #b91c1c;
}

.audit-diff-group {
  display: grid;
  gap: 8px;
}

.audit-diff-group + .audit-diff-group {
  margin-top: 4px;
}

.audit-diff-group__title {
  display: flex;
  align-items: center;
  gap: 8px;
  color: var(--color-text, #1f2937);
  font-size: 12px;
  font-weight: 800;
}

.audit-diff-group__title::before {
  width: 6px;
  height: 6px;
  border-radius: 999px;
  background: var(--color-primary, #2563eb);
  content: '';
}

.audit-diff-row--changed {
  border-color: rgba(37, 99, 235, 0.18);
  background: rgba(37, 99, 235, 0.045);
}

.audit-diff-row--changed .audit-diff-row__kind {
  background: rgba(37, 99, 235, 0.1);
  color: #1d4ed8;
}

.audit-change-row--added .audit-change-row__value--new,
.audit-diff-row--added .audit-diff-row__value--new {
  color: #15803d;
}

.audit-change-row--removed .audit-change-row__value,
.audit-diff-row--removed .audit-diff-row__value--old {
  color: #b91c1c;
  text-decoration: line-through;
  text-decoration-thickness: 1px;
}

.audit-timeline-list {
  display: flex;
  flex-direction: column;
  gap: 12px;
}

.audit-timeline-item {
  display: grid;
  grid-template-columns: 24px minmax(0, 1fr);
  gap: 12px;
}

.audit-timeline-item__rail {
  position: relative;
  display: flex;
  justify-content: center;
  padding-top: 8px;
}

.audit-timeline-item__rail::after {
  position: absolute;
  top: 28px;
  bottom: -20px;
  width: 1px;
  background: rgba(148, 163, 184, 0.24);
  content: "";
}

.audit-timeline-item:last-child .audit-timeline-item__rail::after {
  display: none;
}

.audit-timeline-item__dot {
  position: relative;
  z-index: 1;
  width: 10px;
  height: 10px;
  border-radius: 50%;
  background: #64748b;
  box-shadow: 0 0 0 4px #f8fafc;
}

.audit-timeline-item__dot--success {
  background: #16a34a;
}

.audit-timeline-item__dot--warning {
  background: #d97706;
}

.audit-timeline-item__dot--primary {
  background: #2563eb;
}

.audit-timeline-item__dot--danger {
  background: #dc2626;
}

.audit-timeline-item__body {
  padding: 12px 14px;
  border: 1px solid rgba(148, 163, 184, 0.18);
  border-radius: 12px;
  background: rgba(248, 250, 252, 0.72);
  transition:
    border-color 0.18s ease,
    background 0.18s ease;
}

.audit-timeline-item__body:hover {
  border-color: rgba(37, 99, 235, 0.22);
  background: rgba(255, 255, 255, 0.94);
}

.audit-timeline-item__header {
  display: flex;
  align-items: flex-start;
  justify-content: space-between;
  gap: 14px;
}

.audit-timeline-item__title-row,
.audit-timeline-item__meta {
  display: flex;
  align-items: center;
  flex-wrap: wrap;
  gap: 8px;
}

.audit-timeline-item__title-row strong {
  color: var(--color-text, #1f2937);
  font-size: 14px;
}

.audit-timeline-item__meta {
  margin-top: 8px;
  color: var(--color-text-subtle, #64748b);
  font-size: 12px;
}

.audit-timeline-item__meta code {
  max-width: 420px;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.audit-detail-button {
  display: inline-flex;
  align-items: center;
  min-height: 30px;
  padding: 0 10px;
  border: 1px solid rgba(148, 163, 184, 0.28);
  border-radius: 8px;
  background: #fff;
  color: var(--color-text-subtle, #64748b);
  font-size: 12px;
  font-weight: 700;
  cursor: pointer;
  transition:
    color 0.18s ease,
    border-color 0.18s ease,
    box-shadow 0.18s ease;
}

.audit-detail-button:hover,
.audit-detail-button:focus-visible {
  border-color: rgba(37, 99, 235, 0.32);
  color: var(--color-primary, #2563eb);
  outline: none;
  box-shadow: 0 0 0 3px rgba(37, 99, 235, 0.12);
}

.audit-timeline-item__summary {
  display: grid;
  gap: 8px;
  margin-top: 12px;
}

.audit-change-row {
  display: grid;
  grid-template-columns: minmax(90px, 150px) minmax(0, 1fr) 16px minmax(0, 1fr);
  align-items: center;
  gap: 8px;
  min-height: 28px;
  color: var(--color-text-subtle, #64748b);
  font-size: 12px;
}

.audit-change-row__field {
  color: var(--color-text, #1f2937);
  font-weight: 700;
}

.audit-change-row__value {
  min-width: 0;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
  font-family: ui-monospace, SFMono-Regular, Menlo, Monaco, Consolas, monospace;
}

.audit-change-row__value--new {
  color: var(--color-primary, #2563eb);
}

.audit-timeline-item__fallback,
.audit-timeline-item__more {
  color: var(--color-text-subtle, #64748b);
  font-size: 12px;
}

.audit-diff-grid strong {
  display: block;
  margin-bottom: 8px;
  color: var(--color-text, #1f2937);
  font-size: 13px;
}

.audit-diff-grid pre {
  min-height: 80px;
  max-height: 280px;
  margin: 0;
  padding: 12px;
  overflow: auto;
  border-radius: 10px;
  background: #0f172a;
  color: #e2e8f0;
  font-size: 12px;
  line-height: 1.6;
  white-space: pre-wrap;
  word-break: break-word;
}

:global(.save-confirm-message-box) {
  width: min(640px, calc(100vw - 32px));
}

:global(.save-confirm-message-box .el-message-box__message) {
  width: 100%;
}

:global(.save-confirm) {
  display: flex;
  flex-direction: column;
  gap: 14px;
  color: var(--color-text, #1f2937);
}

:global(.save-confirm__chips) {
  display: flex;
  flex-wrap: wrap;
  gap: 8px;
}

:global(.save-confirm-chip) {
  display: inline-flex;
  align-items: center;
  min-height: 24px;
  padding: 2px 10px;
  border-radius: 999px;
  border: 1px solid transparent;
  font-size: 12px;
  font-weight: 600;
}

:global(.save-confirm-chip--added) {
  color: #047857;
  background: rgba(16, 185, 129, 0.12);
  border-color: rgba(16, 185, 129, 0.24);
}

:global(.save-confirm-chip--removed) {
  color: #b91c1c;
  background: rgba(239, 68, 68, 0.1);
  border-color: rgba(239, 68, 68, 0.22);
}

:global(.save-confirm-chip--changed) {
  color: #1d4ed8;
  background: rgba(59, 130, 246, 0.12);
  border-color: rgba(59, 130, 246, 0.24);
}

:global(.save-confirm-chip--warning) {
  color: #b45309;
  background: rgba(245, 158, 11, 0.14);
  border-color: rgba(245, 158, 11, 0.28);
}

:global(.save-confirm__lead) {
  margin: 0;
  color: var(--color-text-secondary, #4b5563);
  line-height: 1.7;
}

:global(.save-confirm__notice) {
  padding: 10px 12px;
  border-radius: 12px;
  color: #92400e;
  background: rgba(245, 158, 11, 0.12);
  border: 1px solid rgba(245, 158, 11, 0.28);
  line-height: 1.6;
}

:global(.save-confirm__notice--added) {
  color: #047857;
  background: rgba(16, 185, 129, 0.1);
  border-color: rgba(16, 185, 129, 0.24);
}

:global(.save-confirm__notice--removed) {
  color: #b91c1c;
  background: rgba(239, 68, 68, 0.1);
  border-color: rgba(239, 68, 68, 0.22);
}

:global(.save-confirm__notice--changed) {
  color: #1d4ed8;
  background: rgba(59, 130, 246, 0.1);
  border-color: rgba(59, 130, 246, 0.24);
}

:global(.save-confirm__section-title) {
  font-size: 13px;
  font-weight: 700;
  color: var(--color-text, #1f2937);
}

:global(.save-confirm__list) {
  display: flex;
  flex-direction: column;
  gap: 10px;
}

:global(.save-confirm-row) {
  display: grid;
  grid-template-columns: auto 1fr;
  gap: 10px;
  padding: 11px 12px;
  border-radius: 12px;
  border: 1px solid rgba(148, 163, 184, 0.2);
  background: rgba(248, 250, 252, 0.72);
  box-shadow: inset 3px 0 0 rgba(59, 130, 246, 0.48);
}

:global(.save-confirm-row--added) {
  box-shadow: inset 3px 0 0 rgba(16, 185, 129, 0.72);
}

:global(.save-confirm-row--removed) {
  box-shadow: inset 3px 0 0 rgba(239, 68, 68, 0.72);
}

:global(.save-confirm-row--secret) {
  box-shadow: inset 3px 0 0 rgba(124, 58, 237, 0.68);
}

:global(.save-confirm-row--warning) {
  box-shadow: inset 3px 0 0 rgba(245, 158, 11, 0.78);
}

:global(.save-confirm-row__tag) {
  align-self: start;
  padding: 2px 8px;
  border-radius: 999px;
  font-size: 12px;
  font-weight: 700;
  white-space: nowrap;
}

:global(.save-confirm-row__tag--added) {
  color: #047857;
  background: rgba(16, 185, 129, 0.12);
}

:global(.save-confirm-row__tag--removed) {
  color: #b91c1c;
  background: rgba(239, 68, 68, 0.1);
}

:global(.save-confirm-row__tag--changed) {
  color: #1d4ed8;
  background: rgba(59, 130, 246, 0.12);
}

:global(.save-confirm-row__tag--secret) {
  color: #6d28d9;
  background: rgba(124, 58, 237, 0.12);
}

:global(.save-confirm-row__tag--warning) {
  color: #b45309;
  background: rgba(245, 158, 11, 0.14);
}

:global(.save-confirm-row__content) {
  display: flex;
  flex-direction: column;
  gap: 4px;
  min-width: 0;
}

:global(.save-confirm-row__content strong) {
  color: var(--color-text, #1f2937);
  font-size: 13px;
}

:global(.save-confirm-row__content span) {
  color: var(--color-text-secondary, #4b5563);
  font-size: 12px;
  line-height: 1.5;
}

:global(.save-confirm-row__values) {
  display: flex;
  align-items: center;
  flex-wrap: wrap;
  gap: 6px;
  margin-top: 2px;
  font-size: 12px;
}

:global(.save-confirm-row__value-pill) {
  display: inline-flex;
  align-items: center;
  min-height: 26px;
  max-width: 240px;
  padding: 2px 9px;
  overflow: hidden;
  border-radius: 999px;
  color: #475569;
  background: rgba(15, 23, 42, 0.06);
  text-overflow: ellipsis;
  white-space: nowrap;
}

:global(.save-confirm-row__value-pill--new) {
  color: #1d4ed8;
  background: rgba(37, 99, 235, 0.1);
}

:global(.save-confirm-row__arrow) {
  color: var(--color-text-subtle, #94a3b8);
}

:global(.save-confirm__more) {
  color: var(--color-text-subtle, #64748b);
  font-size: 12px;
}

.routing-bottom-bar {
  position: fixed;
  left: 50%;
  bottom: 22px;
  z-index: 20;
  display: flex;
  align-items: center;
  gap: 14px;
  min-height: 54px;
  padding: 8px 10px 8px 16px;
  border-radius: 18px;
  border: 1px solid rgba(148, 163, 184, 0.26);
  background: rgba(255, 255, 255, 0.96);
  box-shadow: 0 18px 42px rgba(15, 23, 42, 0.16);
  transform: translateX(-50%);
  backdrop-filter: blur(14px);
}

.routing-bottom-bar__status,
.routing-bottom-bar__actions {
  display: inline-flex;
  align-items: center;
  gap: 8px;
}

.routing-bottom-bar__status {
  color: var(--color-text, #1f2937);
  font-size: 13px;
  font-weight: 700;
  white-space: nowrap;
}

.routing-bottom-bar--warning .routing-bottom-bar__status {
  color: var(--color-warning, #d97706);
}

.routing-bottom-bar--primary .routing-bottom-bar__status {
  color: var(--color-primary, #2563eb);
}

.routing-bottom-bar--success .routing-bottom-bar__status {
  color: var(--color-success, #16a34a);
}

.draft-summary-popover {
  display: flex;
  flex-direction: column;
  gap: 10px;
  max-height: 320px;
  overflow: auto;
}

.draft-summary-popover__item {
  display: grid;
  gap: 4px;
  padding-bottom: 8px;
  border-bottom: 1px solid rgba(148, 163, 184, 0.16);
  font-size: 12px;
}

.draft-summary-popover__item strong {
  color: var(--color-text, #1f2937);
}

.draft-summary-popover__item span,
.draft-summary-popover__more {
  color: var(--color-text-subtle, #64748b);
  word-break: break-word;
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
  .routing-status-strip,
  .provider-editor-secret,
  .provider-editor-secret__header {
    flex-direction: column;
    align-items: stretch;
  }

  .routing-bottom-bar {
    left: 12px;
    right: 12px;
    bottom: max(12px, env(safe-area-inset-bottom));
    transform: none;
    justify-content: space-between;
  }

  .audit-diff-grid {
    grid-template-columns: 1fr;
  }

  .audit-timeline-item__header {
    flex-direction: column;
  }

  .audit-change-row {
    grid-template-columns: 1fr;
  }

  .audit-diff-popover__header {
    flex-direction: column;
  }

  .audit-diff-row {
    grid-template-columns: 1fr;
  }

  .provider-editor-card__controls,
  .provider-editor-card__actions,
  .routing-panel__tags {
    justify-content: flex-start;
  }
}
</style>
