<template>
<el-drawer
  v-model="auditDrawerVisible"
  title="完整路由审计"
  size="72%"
  append-to-body
>
  <FilterToolbar
    :active-filters="activeAuditFilters"
    :on-clear-all="
      activeAuditFilters.length ? resetAuditFilters : undefined
    "
  >
    <el-select v-model="auditAction" clearable placeholder="动作">
      <el-option
        v-for="item in auditActionOptions"
        :key="item.value"
        :label="item.label"
        :value="item.value"
      />
    </el-select>
    <el-input
      v-model.trim="auditOperator"
      clearable
      placeholder="操作人"
      @keyup.enter="applyAuditFilters"
    />
    <el-input
      v-model.trim="auditSettingKey"
      clearable
      placeholder="配置键搜索"
      @keyup.enter="applyAuditFilters"
    />
    <el-date-picker
      v-model="auditTimeRange"
      type="datetimerange"
      unlink-panels
      clearable
      range-separator="至"
      start-placeholder="开始时间"
      end-placeholder="结束时间"
    />
    <el-select
      v-model="auditPageSize"
      placeholder="每页条数"
      @change="handleAuditPageSizeChange"
    >
      <el-option
        v-for="item in auditPageSizeOptions"
        :key="item"
        :label="`每页 ${item} 条`"
        :value="item"
      />
    </el-select>
    <template #actions>
      <el-button @click="resetAuditFilters">重置</el-button>
      <el-button
        type="primary"
        :loading="auditsLoading"
        @click="applyAuditFilters"
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
                    <pre>{{
                      formatAuditValue(row.oldValueMasked)
                    }}</pre>
                  </div>
                  <div>
                    <strong>新值</strong>
                    <pre>{{
                      formatAuditValue(row.newValueMasked)
                    }}</pre>
                  </div>
                </div>
              </el-collapse-item>
            </el-collapse>
          </div>
        </template>
      </el-table-column>
      <el-table-column
        label="对象"
        min-width="180"
        show-overflow-tooltip
      >
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
      :page-size="audits.pageSize || auditPageSize"
      @current-change="handleAuditPageChange"
    />
  </div>
</el-drawer>
</template>

<script setup lang="ts">
import { computed, toRefs } from "vue";
import { ArrowRight } from "@element-plus/icons-vue";
import FilterToolbar from "@/components/FilterToolbar.vue";
import StatusTag from "@/components/StatusTag.vue";
import type { PaginationResult, SettingAuditRecord } from "@/types";
import type { DateRangeValue } from "@/utils/route-query";
import { formatDateTime } from "@/utils/admin-display";
import {
  auditChangeKindText,
  auditChangeStats,
  auditChangeSummary,
  formatAuditValue,
  groupedAuditChanges,
  type AuditBusinessAction,
} from "@/utils/ai-provider-audit";

type FilterChip = { key: string; label: string; onRemove?: () => void };
const props = defineProps<{
  visible: boolean;
  audits: PaginationResult<SettingAuditRecord>;
  action: string;
  operator: string;
  settingKey: string;
  timeRange: DateRangeValue;
  pageSize: number;
  page: number;
  pageSizeOptions: number[];
  actionOptions: Array<{ label: string; value: string }>;
  activeFilters: FilterChip[];
  auditsLoading: boolean;
  auditDiffTitle: (record: SettingAuditRecord) => string;
  auditEventLabel: (record: SettingAuditRecord) => string;
  auditBusinessAction: (record: SettingAuditRecord) => AuditBusinessAction;
  auditDiffStatusText: (record: SettingAuditRecord) => string;
  auditFallbackSummary: (record: SettingAuditRecord) => string;
  auditTargetTitle: (record: SettingAuditRecord) => string;
}>();
const emit = defineEmits<{
  "update:visible": [value: boolean];
  "update:action": [value: string];
  "update:operator": [value: string];
  "update:settingKey": [value: string];
  "update:timeRange": [value: DateRangeValue];
  "update:pageSize": [value: number];
  "update:page": [value: number];
  reset: [];
  apply: [];
  "page-size-change": [value: number | string];
  "page-change": [value: number];
}>();
const {
  audits,
  pageSizeOptions: auditPageSizeOptions,
  actionOptions: auditActionOptions,
  activeFilters: activeAuditFilters,
  auditsLoading,
} = toRefs(props);
const {
  auditDiffTitle,
  auditEventLabel,
  auditBusinessAction,
  auditDiffStatusText,
  auditFallbackSummary,
  auditTargetTitle,
} = props;
const auditDrawerVisible = computed({ get: () => props.visible, set: (value) => emit("update:visible", value) });
const auditAction = computed({ get: () => props.action, set: (value) => emit("update:action", value) });
const auditOperator = computed({ get: () => props.operator, set: (value) => emit("update:operator", value) });
const auditSettingKey = computed({ get: () => props.settingKey, set: (value) => emit("update:settingKey", value) });
const auditTimeRange = computed({ get: () => props.timeRange, set: (value) => emit("update:timeRange", value) });
const auditPageSize = computed({ get: () => props.pageSize, set: (value) => emit("update:pageSize", value) });
const auditPage = computed({ get: () => props.page, set: (value) => emit("update:page", value) });
function resetAuditFilters() { emit("reset"); }
function applyAuditFilters() { emit("apply"); }
function handleAuditPageSizeChange(value: number | string) { emit("page-size-change", value); }
function handleAuditPageChange(value: number) { emit("page-change", value); }
</script>
