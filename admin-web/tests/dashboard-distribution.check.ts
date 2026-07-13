import {
  buildDistributionRankItems,
  middleEllipsis,
  normalizeDistributionItems,
} from "../src/utils/dashboard-distribution";

function assertEqual<T>(actual: T, expected: T, label: string) {
  if (actual !== expected) throw new Error(`${label}: expected ${String(expected)}, got ${String(actual)}`);
}

const ranked = buildDistributionRankItems([
  { name: "", total: 99, successRate: 0.5 },
  { name: "provider-b", total: 3, successRate: 0.8 },
  { name: "provider-a", total: 3, successRate: 0.96 },
]);
assertEqual(ranked[0]?.label, "provider-a", "并列数量按名称排序");
assertEqual(ranked[0]?.tone, "success", "成功率 tone");
assertEqual(ranked[1]?.showAlert, true, "低成功率提示");
assertEqual(ranked.at(-1)?.label, "未指定", "未知名称置底");
assertEqual(ranked.at(-1)?.rankLabel, "—", "未知名称不参与排行");
assertEqual(normalizeDistributionItems([{ name: "(empty)" }])[0]?.name, "未指定", "图表未知名称归一化");
assertEqual(middleEllipsis("1234567890abcdefghij", 4, 3), "1234…hij", "长名称中间省略");
assertEqual(buildDistributionRankItems([{ name: "summary", total: 1, successRate: 0.5 }], { showAlert: false })[0]?.showAlert, false, "场景排行关闭告警图标");

console.log("Dashboard distribution utils checks passed");
