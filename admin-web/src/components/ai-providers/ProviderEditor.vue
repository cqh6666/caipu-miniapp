<template>
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
              description="从常见模板开始，后续再补 Base URL、密钥和模型细节。"
              compact
            />
            <div v-if="!draftScene.providers.length" class="provider-preset-grid">
              <button
                v-for="preset in providerPresetOptions"
                :key="preset.key"
                type="button"
                class="provider-preset-card"
                @click="handleAddProviderPreset(preset.key)"
              >
                <strong>{{ preset.title }}</strong>
                <span>{{ preset.description }}</span>
              </button>
            </div>

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
                          isProviderCollapsed(provider)
                            ? '展开编辑'
                            : '折叠编辑'
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
                      <span
                        v-if="providerThinkingLabel(provider)"
                        class="provider-compact-grid__item"
                      >
                        {{ providerThinkingLabel(provider) }}
                      </span>
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
                    <div
                      class="provider-editor-grid provider-editor-grid--basic"
                    >
                      <label class="routing-field">
                        <span
                          >节点名称
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
                      <template v-else>
                        <label class="routing-field">
                          <span
                            >Thinking
                            <HelpTip :content="helpTips.thinkingType"
                          /></span>
                          <el-select
                            v-model="provider.extra.thinking_type"
                            @change="handleThinkingTypeChange(provider)"
                          >
                            <el-option
                              v-for="item in thinkingTypeOptions"
                              :key="item.value"
                              :label="item.label"
                              :value="item.value"
                            />
                          </el-select>
                        </label>
                        <label
                          v-if="provider.extra.thinking_type !== 'disabled'"
                          class="routing-field"
                        >
                          <span
                            >Reasoning Effort
                            <HelpTip :content="helpTips.reasoningEffort"
                          /></span>
                          <el-select v-model="provider.extra.reasoning_effort">
                            <el-option
                              v-for="item in reasoningEffortOptions"
                              :key="item.value"
                              :label="item.label"
                              :value="item.value"
                            />
                          </el-select>
                        </label>
                      </template>
                      <template v-if="isImageGenerationProvider(provider)">
                        <label class="routing-field">
                          <span
                            >图片尺寸
                            <HelpTip :content="helpTips.imageSize"
                          /></span>
                          <el-select
                            v-model="provider.extra.size"
                            allow-create
                            filterable
                            placeholder="1536x1024"
                          >
                            <el-option
                              v-for="item in imageSizeOptions"
                              :key="item.value"
                              :label="item.label"
                              :value="item.value"
                            />
                          </el-select>
                        </label>
                        <label class="routing-field">
                          <span>质量</span>
                          <el-select v-model="provider.extra.quality">
                            <el-option
                              v-for="item in imageQualityOptions"
                              :key="item.value"
                              :label="item.label"
                              :value="item.value"
                            />
                          </el-select>
                        </label>
                        <label class="routing-field">
                          <span
                            >背景
                            <HelpTip :content="helpTips.imageBackground"
                          /></span>
                          <el-select
                            v-model="provider.extra.background"
                            popper-class="image-background-select-popper"
                          >
                            <el-option
                              v-for="item in imageBackgroundOptions"
                              :key="item.value"
                              :label="item.label"
                              :value="item.value"
                            >
                              <div class="image-option">
                                <span class="image-option__label">{{
                                  item.label
                                }}</span>
                                <small class="image-option__tip">{{
                                  item.tip
                                }}</small>
                              </div>
                            </el-option>
                          </el-select>
                        </label>
                        <label class="routing-field">
                          <span>输出格式</span>
                          <el-select v-model="provider.extra.output_format">
                            <el-option
                              v-for="item in imageOutputFormatOptions"
                              :key="item.value"
                              :label="item.label"
                              :value="item.value"
                            />
                          </el-select>
                        </label>
                        <label
                          v-if="provider.extra.output_format !== 'png'"
                          class="routing-field"
                        >
                          <span
                            >压缩率
                            <HelpTip :content="helpTips.imageCompression"
                          /></span>
                          <el-input-number
                            v-model="provider.extra.output_compression"
                            :min="0"
                            :max="100"
                          />
                        </label>
                        <label class="routing-field">
                          <span>数量</span>
                          <el-input-number
                            v-model="provider.extra.n"
                            :min="1"
                            :max="10"
                          />
                        </label>
                      </template>
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
</template>

<script setup lang="ts">
import { computed, toRefs } from "vue";
import { ArrowDown, ArrowRight, MoreFilled, Promotion, Rank, Refresh } from "@element-plus/icons-vue";
import HelpTip from "@/components/HelpTip.vue";
import PageState from "@/components/PageState.vue";
import StatusTag from "@/components/StatusTag.vue";
import type { AIRoutingProviderConfig, AIRoutingSceneConfig } from "@/types";
import { getProviderSecretStatus, type ProviderValidationError } from "@/utils/ai-provider-validation";

type SelectOption = { label: string; value: string };
type ProviderPresetOption = { key: string; title: string; description: string };
type ProviderTestState = { ok: boolean; text: string; latencyMs?: number; errorType?: string; testedAt: string };
type ProviderSelectOptions = {
  endpointModes: SelectOption[];
  responseFormats: SelectOption[];
  thinkingTypes: SelectOption[];
  reasoningEfforts: SelectOption[];
  imageSizes: SelectOption[];
  imageQualities: SelectOption[];
  imageBackgrounds: SelectOption[];
  imageOutputFormats: SelectOption[];
};
type ProviderEditorContract = {
  draggingProviderIndex: number | null;
  dragOverProviderIndex: number | null;
  getProviderLocalKey: (provider: AIRoutingProviderConfig) => string;
  isProviderCollapsed: (provider: AIRoutingProviderConfig) => boolean;
  firstProviderError: (provider: AIRoutingProviderConfig) => string;
  getProviderTestState: (provider: AIRoutingProviderConfig) => ProviderTestState | undefined;
  providerFieldError: (provider: AIRoutingProviderConfig, field: keyof ProviderValidationError) => string;
  isImageGenerationProvider: (provider: AIRoutingProviderConfig) => boolean;
  providerThinkingLabel: (provider: AIRoutingProviderConfig) => string;
  shouldShowProviderSecretEditor: (provider: AIRoutingProviderConfig) => boolean;
  add: () => void;
  addPreset: (key: string) => void;
  dragOver: (index: number) => void;
  drop: (index: number) => void;
  dragStart: (index: number, event: DragEvent) => void;
  dragEnd: () => void;
  toggleCollapse: (provider: AIRoutingProviderConfig) => void;
  testSingle: (index: number) => void;
  menu: (command: string, index: number) => void;
  touchField: (provider: AIRoutingProviderConfig, field: keyof ProviderValidationError) => void;
  endpointChange: (provider: AIRoutingProviderConfig) => void;
  thinkingChange: (provider: AIRoutingProviderConfig) => void;
  toggleSecret: (provider: AIRoutingProviderConfig) => void;
  clearSecret: (provider: AIRoutingProviderConfig) => void;
  apiKeyInput: (provider: AIRoutingProviderConfig, value: string) => void;
};
type ProviderHelpTips = {
  providerName: string;
  baseURL: string;
  endpoint: string;
  responseFormat: string;
  thinkingType: string;
  reasoningEffort: string;
  imageSize: string;
  imageBackground: string;
  imageCompression: string;
  apiKey: string;
};

const props = defineProps<{
  draftScene: AIRoutingSceneConfig;
  enabledProviderCount: number;
  helpTips: ProviderHelpTips;
  providerPresetOptions: ProviderPresetOption[];
  singleTestProviderId: string;
  testingScene: boolean;
  savingScene: boolean;
  selectOptions: ProviderSelectOptions;
  editor: ProviderEditorContract;
}>();

const {
  draftScene,
  enabledProviderCount,
  helpTips,
  providerPresetOptions,
  singleTestProviderId,
  testingScene,
  savingScene,
} = toRefs(props);
const draggingProviderIndex = computed(() => props.editor.draggingProviderIndex);
const dragOverProviderIndex = computed(() => props.editor.dragOverProviderIndex);
const providerEndpointModeOptions = computed(() => props.selectOptions.endpointModes);
const providerResponseFormatOptions = computed(() => props.selectOptions.responseFormats);
const thinkingTypeOptions = computed(() => props.selectOptions.thinkingTypes);
const reasoningEffortOptions = computed(() => props.selectOptions.reasoningEfforts);
const imageSizeOptions = computed(() => props.selectOptions.imageSizes);
const imageQualityOptions = computed(() => props.selectOptions.imageQualities);
const imageBackgroundOptions = computed(() => props.selectOptions.imageBackgrounds);
const imageOutputFormatOptions = computed(() => props.selectOptions.imageOutputFormats);

function getProviderLocalKey(provider: AIRoutingProviderConfig) { return props.editor.getProviderLocalKey(provider); }
function isProviderCollapsed(provider: AIRoutingProviderConfig) { return props.editor.isProviderCollapsed(provider); }
function firstProviderError(provider: AIRoutingProviderConfig) { return props.editor.firstProviderError(provider); }
function getProviderTestState(provider: AIRoutingProviderConfig) { return props.editor.getProviderTestState(provider); }
function providerFieldError(provider: AIRoutingProviderConfig, field: keyof ProviderValidationError) { return props.editor.providerFieldError(provider, field); }
function isImageGenerationProvider(provider: AIRoutingProviderConfig) { return props.editor.isImageGenerationProvider(provider); }
function providerThinkingLabel(provider: AIRoutingProviderConfig) { return props.editor.providerThinkingLabel(provider); }
function shouldShowProviderSecretEditor(provider: AIRoutingProviderConfig) { return props.editor.shouldShowProviderSecretEditor(provider); }
function handleAddProvider() { props.editor.add(); }
function handleAddProviderPreset(key: string) { props.editor.addPreset(key); }
function handleProviderDragOver(index: number) { props.editor.dragOver(index); }
function handleProviderDrop(index: number) { props.editor.drop(index); }
function handleProviderDragStart(index: number, event: DragEvent) { props.editor.dragStart(index, event); }
function handleProviderDragEnd() { props.editor.dragEnd(); }
function toggleProviderCollapsed(provider: AIRoutingProviderConfig) { props.editor.toggleCollapse(provider); }
function handleTestSingleProvider(index: number) { props.editor.testSingle(index); }
function handleProviderMenuCommand(command: string, index: number) { props.editor.menu(String(command), index); }
function touchProviderField(provider: AIRoutingProviderConfig, field: keyof ProviderValidationError) { props.editor.touchField(provider, field); }
function handleEndpointModeChange(provider: AIRoutingProviderConfig) { props.editor.endpointChange(provider); }
function handleThinkingTypeChange(provider: AIRoutingProviderConfig) { props.editor.thinkingChange(provider); }
function toggleProviderSecretEditor(provider: AIRoutingProviderConfig) { props.editor.toggleSecret(provider); }
function handleClearProviderApiKey(provider: AIRoutingProviderConfig) { props.editor.clearSecret(provider); }
function handleProviderApiKeyInput(provider: AIRoutingProviderConfig, value: string) { props.editor.apiKeyInput(provider, value); }
</script>
