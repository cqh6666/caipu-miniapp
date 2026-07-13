<template>
  <div class="save-confirm" role="document">
    <div class="save-confirm__chips" aria-label="保存影响摘要">
      <span class="save-confirm-chip save-confirm-chip--changed">
        {{ model.diffCount }} 项变更
      </span>
      <span class="save-confirm-chip save-confirm-chip--warning">线上生效</span>
      <span :class="`save-confirm-chip save-confirm-chip--${model.test.kind}`">
        {{ model.test.text }}
      </span>
    </div>
    <p class="save-confirm__lead">
      保存后立即更新线上「{{ model.sceneTitle }}」。
    </p>
    <div
      :class="`save-confirm__notice save-confirm__notice--${model.test.kind}`"
      :role="model.test.role"
    >
      {{ model.test.notice }}
    </div>
    <div class="save-confirm__section-title">变更摘要</div>
    <div class="save-confirm__list">
      <div
        v-for="(row, index) in model.rows"
        :key="`${row.kind}-${row.title}-${index}`"
        :class="`save-confirm-row save-confirm-row--${row.kind}`"
      >
        <span :class="`save-confirm-row__tag save-confirm-row__tag--${row.kind}`">
          {{ row.tag }}
        </span>
        <div class="save-confirm-row__content">
          <strong>{{ row.title }}</strong>
          <span>{{ row.description }}</span>
          <div
            v-if="row.before !== undefined || row.after !== undefined"
            class="save-confirm-row__values"
          >
            <span class="save-confirm-row__value-pill">
              {{ row.beforeLabel || "当前" }}：{{ row.before || "空" }}
            </span>
            <span class="save-confirm-row__arrow">→</span>
            <span class="save-confirm-row__value-pill save-confirm-row__value-pill--new">
              {{ row.afterLabel || "发布后" }}：{{ row.after || "空" }}
            </span>
          </div>
        </div>
      </div>
    </div>
    <div v-if="model.moreCount > 0" class="save-confirm__more">
      另有 {{ model.moreCount }} 项改动，可在底部「草稿摘要」查看。
    </div>
  </div>
</template>

<script setup lang="ts">
import type { SaveSceneConfirmModel } from "@/utils/ai-provider-save-confirm";

defineProps<{ model: SaveSceneConfirmModel }>();
</script>
