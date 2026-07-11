<template>
  <div
    v-if="showShortcut"
    class="routing-scene-section-hint"
    aria-label="场景切换快捷键提示"
  >
    <span>切换场景</span>
    <span v-for="token in shortcutPrev" :key="`prev-${token}`" class="shortcut-key">{{ token }}</span>
    <span class="routing-scene-section-hint__sep">/</span>
    <span v-for="token in shortcutNext" :key="`next-${token}`" class="shortcut-key">{{ token }}</span>
  </div>
  <div class="routing-scene-grid" role="tablist" aria-label="AI 路由场景">
    <button
      v-for="item in cards"
      :key="item.scene"
      :ref="(element) => setCardRef(item.scene, element)"
      type="button"
      class="page-card routing-scene-card"
      :class="{ 'routing-scene-card--active': item.scene === currentScene }"
      role="tab"
      :aria-selected="item.scene === currentScene"
      :tabindex="item.scene === currentScene ? 0 : -1"
      @click="emit('select', item.scene)"
      @keydown.left.prevent="focusOffset(-1)"
      @keydown.right.prevent="focusOffset(1)"
    >
      <div class="routing-scene-card__header">
        <div>
          <div class="routing-scene-card__eyebrow">{{ displayAIRoutingScene(item.scene) }}</div>
          <h3>{{ item.title }}</h3>
        </div>
        <StatusTag :tone="item.aggregate.tone" :text="item.aggregate.text" />
      </div>
      <div class="routing-scene-card__meta">
        <span>策略：{{ displayAIRoutingStrategy(item.strategy) }}</span>
        <span>节点：{{ item.activeProviderCount }}/{{ item.providerCount }}</span>
        <span>来源：{{ displaySettingSource(item.source) }}</span>
      </div>
      <div
        v-if="item.issue"
        class="routing-scene-card__issue"
        :class="`routing-scene-card__issue--${item.issue.tone}`"
      >
        <el-icon class="routing-scene-card__issue-icon"><Warning /></el-icon>
        <div class="routing-scene-card__issue-body">
          <span class="routing-scene-card__issue-title">{{ item.issue.title }}</span>
          <span v-if="item.issue.detail" class="routing-scene-card__issue-detail">{{ item.issue.detail }}</span>
        </div>
      </div>
      <div class="routing-scene-card__footer">
        <span>最近修改：{{ formatDateTime(item.updatedAt) }}</span>
        <span class="routing-scene-card__channel">线上：{{ item.savedChannelLabel }}</span>
      </div>
    </button>
  </div>
</template>

<script setup lang="ts">
import { Warning } from "@element-plus/icons-vue";
import StatusTag from "@/components/StatusTag.vue";
import type { AIRoutingSceneKey, AIRoutingStrategy } from "@/types";
import {
  displayAIRoutingScene,
  displayAIRoutingStrategy,
  displaySettingSource,
  formatDateTime,
} from "@/utils/admin-display";

type SceneCard = {
  scene: AIRoutingSceneKey;
  title: string;
  strategy: AIRoutingStrategy;
  providerCount: number;
  activeProviderCount: number;
  updatedAt: string;
  source: string;
  savedChannelLabel: string;
  aggregate: { tone: "neutral" | "primary" | "success" | "warning" | "danger"; text: string };
  issue: { tone: "warning" | "danger"; title: string; detail: string } | null;
};

const props = defineProps<{
  cards: SceneCard[];
  currentScene: AIRoutingSceneKey;
  showShortcut: boolean;
  shortcutPrev: string[];
  shortcutNext: string[];
}>();
const emit = defineEmits<{ select: [scene: AIRoutingSceneKey] }>();
const cardRefs: Partial<Record<AIRoutingSceneKey, HTMLButtonElement | null>> = {};

function setCardRef(scene: AIRoutingSceneKey, element: Element | null) {
  cardRefs[scene] = element instanceof HTMLButtonElement ? element : null;
}

function focusScene(scene: AIRoutingSceneKey) {
  cardRefs[scene]?.focus();
}

function focusOffset(offset: number) {
  const currentIndex = props.cards.findIndex((item) => item.scene === props.currentScene);
  const nextIndex = (currentIndex + offset + props.cards.length) % props.cards.length;
  const next = props.cards[nextIndex];
  if (!next) return;
  emit("select", next.scene);
  requestAnimationFrame(() => focusScene(next.scene));
}

defineExpose({ focusOffset, focusScene });
</script>
