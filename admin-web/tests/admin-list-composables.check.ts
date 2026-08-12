import { readFileSync } from "node:fs";
import { resolve } from "node:path";
import { reactive } from "vue";
import { useJobDetailDrawer } from "../src/composables/useJobDetailDrawer";
import { useRoutedAdminList } from "../src/composables/useRoutedAdminList";
import { assertEqual, assertIncludes } from "../../tests/check-assertions";

const route = reactive<any>({
  query: { page: "2", scene: "summary", timeFrom: "2026-08-01T00:00:00.000Z", timeTo: "2026-08-02T00:00:00.000Z" },
});
const replacements: any[] = [];
const requests: string[] = [];
const list = useRoutedAdminList<any, { scene: string; status: string }>({
  route,
  router: { replace: async (target: any) => { replacements.push(target); } } as any,
  initialFilters: { scene: "", status: "" },
  fields: [{ key: "scene" }, { key: "status" }],
  fetchList: async (query) => {
    requests.push(query.toString());
    return { items: [{ id: 1 }], total: 1, page: 2, pageSize: 20 };
  },
  loadErrorMessage: "加载失败",
});
list.syncStateFromRoute();
assertEqual(list.page.value, 2, "列表页码从路由恢复");
assertEqual(list.filters.scene, "summary", "列表筛选从路由恢复");
await list.loadList();
assertEqual(requests[0]?.includes("scene=summary"), true, "列表请求编码筛选字段");
assertEqual(requests[0]?.includes("timeFrom=2026-08-01T00%3A00%3A00.000Z"), true, "列表请求编码时间范围");
list.filters.status = "failed";
await list.applyFilters();
assertEqual(replacements[0]?.query?.page, undefined, "应用筛选回到第一页");
assertEqual(replacements[0]?.query?.status, "failed", "应用筛选同步路由");

let detailFailure = "";
const drawer = useJobDetailDrawer({
  loadDetail: async (jobId) => {
    if (jobId === 2) throw new Error("detail failed");
    return { job: { id: jobId } as any, calls: [] };
  },
  onError: (error) => { detailFailure = (error as Error).message; },
});
await drawer.openJobDetail(1);
assertEqual(drawer.jobDrawerVisible.value, true, "任务详情打开抽屉");
assertEqual(drawer.jobDetail.value?.job.id, 1, "任务详情保存加载结果");
drawer.openCallDetail({ id: 10 } as any);
assertEqual(drawer.callDrawerVisible.value, true, "任务详情可打开调用抽屉");
await drawer.openJobDetail(2);
assertEqual(drawer.callDrawerVisible.value, false, "切回任务详情时关闭调用抽屉");
assertEqual(drawer.jobDetail.value, null, "任务详情失败时清空旧数据");
assertEqual(detailFailure, "detail failed", "任务详情保留错误回调");

const jobsPageSource = readFileSync(resolve(process.cwd(), "src/pages/JobsPage.vue"), "utf8");
assertIncludes(jobsPageSource, "buildListRouteQuery(page.value, { jobId })", "Jobs 详情保留 jobId 深链写入");
assertIncludes(jobsPageSource, "activeJobId.value = readQueryNumber(route.query, 'jobId', 0)", "Jobs 详情保留 jobId 深链读取");
assertIncludes(jobsPageSource, "const nextQuery = buildListRouteQuery(page.value)", "Jobs 关闭详情清理 jobId");

const callsPageSource = readFileSync(resolve(process.cwd(), "src/pages/CallsPage.vue"), "utf8");
assertEqual(callsPageSource.includes("path: '/ai-jobs'"), false, "Calls 页面继续原地打开任务抽屉");

const dashboardPageSource = readFileSync(resolve(process.cwd(), "src/pages/DashboardPage.vue"), "utf8");
assertIncludes(dashboardPageSource, "path: '/ai-jobs'", "Dashboard 继续跳转 Jobs 页面");

console.log("Admin routed list and job drawer composable checks passed");
