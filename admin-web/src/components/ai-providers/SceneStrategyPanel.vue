<template>
  <div class="page-card routing-panel routing-panel--strategy">
    <div class="routing-panel__header">
      <div>
        <h3 class="routing-panel__title">
          场景策略 <HelpTip :content="helpTips.sceneStrategy" />
        </h3>
        <div class="routing-panel__subtitle">路由开关、尝试次数、熔断与请求参数。</div>
      </div>
      <div class="routing-panel__tags">
        <StatusTag
          :tone="draftScene.enabled ? 'primary' : 'neutral'"
          :text="draftScene.enabled ? '新路由已启用' : '新路由未启用'"
        />
        <StatusTag :tone="currentChannel.tone" :text="currentChannel.label" />
      </div>
    </div>

    <div class="routing-form-grid">
      <label class="routing-field">
        <span>启用新路由</span>
        <el-switch v-model="draftScene.enabled" inline-prompt active-text="开" inactive-text="关" />
      </label>
      <label class="routing-field">
        <span>调度策略 <HelpTip :content="helpTips.sceneStrategy" /></span>
        <el-select v-model="draftScene.strategy">
          <el-option v-for="item in aiRoutingStrategyOptions" :key="item.value" :label="item.label" :value="item.value" />
        </el-select>
      </label>
      <label class="routing-field routing-field--with-hint">
        <span>最大尝试次数 <HelpTip :content="helpTips.maxAttempts" /></span>
        <el-input-number v-model="draftScene.maxAttempts" :min="minimumMaxAttempts" :max="maxAttemptCeiling" />
        <small class="routing-field__hint" :class="{ 'routing-field__hint--warn': numericWarn.maxAttempts }">
          {{ numericWarn.maxAttempts || '建议 2-5 次；过大会让失败降级变慢' }}
        </small>
      </label>
      <label class="routing-field routing-field--with-hint">
        <span>熔断阈值 <HelpTip :content="helpTips.breaker" /></span>
        <el-input-number v-model="draftScene.breaker.failureThreshold" :min="1" :max="10" />
        <small class="routing-field__hint" :class="{ 'routing-field__hint--warn': numericWarn.failureThreshold }">
          {{ numericWarn.failureThreshold || '连续失败达到该次数后触发熔断，建议 3-10' }}
        </small>
      </label>
      <label class="routing-field routing-field--with-hint">
        <span>冷却时间（秒） <HelpTip :content="helpTips.breaker" /></span>
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
        <span v-for="i in timelineSegments.attempts" :key="`att-${i}`" class="routing-timeline-hint__seg routing-timeline-hint__seg--attempt">试 {{ i }}</span>
        <span class="routing-timeline-hint__seg routing-timeline-hint__seg--breaker">连续失败 {{ draftScene.breaker.failureThreshold }} 次</span>
        <span class="routing-timeline-hint__seg routing-timeline-hint__seg--cooldown">冷却 {{ draftScene.breaker.cooldownSeconds }}s</span>
      </div>
      <div class="routing-timeline-hint__caption">
        预计首轮最长 ≈ <strong>{{ expectedFirstRoundSeconds }}s</strong>
        <span class="routing-timeline-hint__hint">（最大尝试次数 × 启用节点最大超时）</span>
      </div>
    </div>

    <div class="routing-checkbox-block">
      <div class="routing-checkbox-block__title">允许切换到下一个节点的错误类型</div>
      <el-checkbox-group v-model="draftScene.retryOn" class="routing-checkbox-grid">
        <el-checkbox v-for="item in retryOptions" :key="item.value" :value="item.value">{{ item.label }}</el-checkbox>
      </el-checkbox-group>
    </div>

    <div class="routing-request-block">
      <div class="routing-checkbox-block__title">场景级请求参数 <HelpTip :content="helpTips.requestOptions" /></div>
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
      <div v-else class="routing-request-block__note">当前场景默认沿用业务层固定 prompt 参数，无需额外请求选项。</div>
    </div>
  </div>
</template>

<script setup lang="ts">
import HelpTip from '@/components/HelpTip.vue'
import StatusTag from '@/components/StatusTag.vue'
import type { AIRoutingSceneConfig } from '@/types'
import { aiRoutingStrategyOptions, formatDateTime } from '@/utils/admin-display'

defineProps<{
  draftScene: AIRoutingSceneConfig
  currentChannel: { tone: 'neutral' | 'primary' | 'success' | 'warning' | 'danger'; label: string }
  helpTips: { sceneStrategy: string; maxAttempts: string; breaker: string; requestOptions: string }
  minimumMaxAttempts: number
  maxAttemptCeiling: number
  numericWarn: { maxAttempts: string; failureThreshold: string; cooldownSeconds: string }
  timelineSegments: { attempts: number }
  expectedFirstRoundSeconds: number
  retryOptions: Array<{ label: string; value: string }>
}>()
</script>
