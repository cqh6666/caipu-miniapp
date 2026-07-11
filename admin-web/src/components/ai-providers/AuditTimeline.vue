<template>
          <div class="page-header">
            <div>
              <h2 class="page-title" style="font-size: 22px">
                最近审计 <HelpTip :content="helpTips.audit" />
              </h2>
              <div class="page-subtitle">
                {{ currentAuditGroup }} · 最近 {{ recentAudits.length }} 条
              </div>
            </div>
            <div class="audit-section__actions">
              <el-segmented
                v-model="recentAuditKindFilter"
                :options="recentAuditKindOptions"
                size="small"
              />
              <el-button
                :loading="auditsLoading"
                @click="emit('open-full')"
                >完整审计</el-button
              >
            </div>
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
                      :width="largePopoverWidth"
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
                              <el-icon aria-hidden="true"
                                ><ArrowRight
                              /></el-icon>
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
                            {{ auditChangeSummary(item).length - 10 }}
                            项变化，请在完整审计中查看。
                          </div>
                        </div>
                        <div v-else class="audit-diff-popover__fallback">
                          {{ auditFallbackSummary(item) }}
                        </div>
                        <el-collapse class="audit-raw-collapse">
                          <el-collapse-item title="查看原始值" name="raw">
                            <div
                              class="audit-diff-grid audit-diff-grid--popover"
                            >
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
</template>

<script setup lang="ts">
import { computed, toRefs } from "vue";
import { ArrowRight } from "@element-plus/icons-vue";
import HelpTip from "@/components/HelpTip.vue";
import PageState from "@/components/PageState.vue";
import StatusTag from "@/components/StatusTag.vue";
import type { SettingAuditRecord } from "@/types";
import { formatDateTime } from "@/utils/admin-display";
import {
  auditChangeKindText,
  auditChangeStats,
  auditChangeSummary,
  formatAuditValue,
  groupedAuditChanges,
  type AuditBusinessAction,
  type AuditTone,
} from "@/utils/ai-provider-audit";

const props = defineProps<{
  recentAudits: SettingAuditRecord[];
  currentAuditGroup: string;
  kindFilter: "all" | "changes" | "tests";
  kindOptions: Array<{ label: string; value: string }>;
  auditsLoading: boolean;
  recentAuditsLoading: boolean;
  recentAuditsError: string;
  largePopoverWidth: string | number;
  helpTip: string;
  toneForAuditAction: (record: SettingAuditRecord) => AuditTone;
  auditBusinessAction: (record: SettingAuditRecord) => AuditBusinessAction;
  auditTargetTitle: (record: SettingAuditRecord) => string;
  auditEventLabel: (record: SettingAuditRecord) => string;
  auditDiffTitle: (record: SettingAuditRecord) => string;
  auditDiffStatusText: (record: SettingAuditRecord) => string;
  auditFallbackSummary: (record: SettingAuditRecord) => string;
}>();
const emit = defineEmits<{
  "update:kindFilter": [value: "all" | "changes" | "tests"];
  "open-full": [];
  retry: [];
}>();
const {
  recentAudits,
  currentAuditGroup,
  kindOptions: recentAuditKindOptions,
  auditsLoading,
  recentAuditsLoading,
  recentAuditsError,
  largePopoverWidth,
} = toRefs(props);
const {
  toneForAuditAction,
  auditBusinessAction,
  auditTargetTitle,
  auditEventLabel,
  auditDiffTitle,
  auditDiffStatusText,
  auditFallbackSummary,
} = props;
const helpTips = { audit: props.helpTip };
const recentAuditKindFilter = computed({
  get: () => props.kindFilter,
  set: (value) => emit("update:kindFilter", value),
});
function loadRecentAudits() { emit("retry"); }
</script>
