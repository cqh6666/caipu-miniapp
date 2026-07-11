import {
  createSearchBlurController,
  filterRecipes,
  pickRandomRecipe,
} from "../../pages/index/use-recipe-library";
import { createEmptyPlaceDraft, filterPlaces } from "../../pages/index/use-place-library";
import { createMealOrderDraftSyncController, upsertMealOrderItem } from "../../pages/index/use-meal-order";
import { buildInviteShareTitle, memberRoleLabel } from "../../pages/index/use-kitchen-space";
import { buildPlaceDraftFromCandidate } from "../../pages/index/use-smart-add";
import {
	buildRecipeEditPayload,
  buildStepCompletionKeyList,
  createCompletedStepStoragePayload,
  normalizeCompletedStepKeyMap,
  serializeComparableEditDraft,
} from "../../pages/recipe-detail/use-recipe-edit";
import {
  buildFlowchartWaitHint,
  createRecipeAsyncJobsController,
  resolveRemainingWaitSeconds,
  toPositiveInteger,
} from "../../pages/recipe-detail/use-recipe-async-jobs";
import {
  buildFlowchartImageCacheEntry,
  resolveVisibleImageIndex,
} from "../../pages/recipe-detail/use-recipe-images";
import { buildRecipeShareConfig } from "../../pages/recipe-detail/use-recipe-share";

function assertEqual<T>(actual: T, expected: T, label: string) {
  if (actual !== expected) throw new Error(`${label}: expected ${String(expected)}, got ${String(actual)}`);
}

const recipes = [
  { id: "1", title: "番茄炒蛋", mealType: "main", status: "wishlist" },
  { id: "2", title: "早餐粥", mealType: "breakfast", status: "done" },
];
assertEqual(filterRecipes(recipes, { mealType: "main", status: "wishlist" }).length, 1, "菜谱筛选");
assertEqual(filterRecipes(recipes, { mealOrderMode: true }).length, 2, "点菜模式不过滤分类");
assertEqual(pickRandomRecipe(recipes, "1", () => 0)?.id, "2", "随机推荐排除当前项");
let blurCount = 0;
const blurController = createSearchBlurController(() => { blurCount += 1; }, 1);
blurController.schedule();
blurController.cancel();
await new Promise((resolve) => setTimeout(resolve, 8));
assertEqual(blurCount, 0, "搜索失焦调度器取消");
blurController.schedule();
await new Promise((resolve) => setTimeout(resolve, 8));
assertEqual(blurCount, 1, "搜索失焦调度器执行");

const places = [
  { id: "a", name: "小馆", address: "天河", status: "want", tags: [] },
  { id: "b", name: "老店", address: "越秀", status: "visited", tags: [] },
];
assertEqual(filterPlaces(places, "want", "小馆").length, 1, "地点状态与关键词筛选");
assertEqual(createEmptyPlaceDraft().status, "want", "地点草稿默认状态");
assertEqual(buildPlaceDraftFromCandidate({ name: "候选店" }, { source: "share" }).source, "share", "地点候选映射");

assertEqual(buildInviteShareTitle({ kitchenName: "周末厨房" }).includes("空间"), true, "空间邀请文案");
assertEqual(memberRoleLabel("owner"), "创建者", "成员角色文案");
assertEqual(upsertMealOrderItem([], { recipeId: "1", titleSnapshot: "菜" }).length, 1, "菜单条目新增");
let syncCount = 0;
const scheduler = createMealOrderDraftSyncController(async () => { syncCount += 1 }, 1);
scheduler.schedule();
await new Promise((resolve) => setTimeout(resolve, 8));
assertEqual(syncCount, 1, "菜单草稿调度器执行");
scheduler.schedule(10);
scheduler.cancel();
await new Promise((resolve) => setTimeout(resolve, 15));
assertEqual(syncCount, 1, "菜单草稿调度器取消");

const draftA = { title: "菜", mainIngredients: ["蛋"], secondaryIngredients: [], steps: [{ title: "炒", detail: "2分钟" }] };
const draftB = { ...draftA, mainIngredients: [{ value: "蛋" }], steps: [{ title: "炒", detail: "2分钟" }] };
assertEqual(serializeComparableEditDraft(draftA), serializeComparableEditDraft(draftB), "编辑草稿可比较序列化");
const editPayload = buildRecipeEditPayload({ ...draftB, title: " 菜 ", parsedContentMode: "manual" });
assertEqual(editPayload.title, "菜", "编辑保存 payload 归一化");
assertEqual(editPayload.parsedContentEdited, true, "手工编辑标记保留");
const stepKeys = buildStepCompletionKeyList(draftA.steps);
const completedPayload = createCompletedStepStoragePayload({ [stepKeys[0]]: true });
assertEqual(Object.keys(normalizeCompletedStepKeyMap(completedPayload, stepKeys)).length, 1, "步骤完成态恢复");
assertEqual(resolveRemainingWaitSeconds(30, 1000, 11000), 20, "预计等待时间递减");
assertEqual(toPositiveInteger(1.2), 2, "队列数量向上取整");
assertEqual(buildFlowchartWaitHint("pending", 2, 30).includes("前面还有 2 个任务"), true, "流程图排队提示");
let jobPollCount = 0;
let estimateTickCount = 0;
const jobsController = createRecipeAsyncJobsController({
  poll: () => { jobPollCount += 1; },
  onEstimateTick: () => { estimateTickCount += 1; },
  pollInterval: 1,
  estimateInterval: 1,
});
jobsController.sync({ hasActiveJob: true, hasEstimate: true });
await new Promise((resolve) => setTimeout(resolve, 8));
jobsController.stop();
assertEqual(jobPollCount > 0, true, "异步任务轮询执行");
assertEqual(estimateTickCount > 0, true, "等待时间计时执行");
assertEqual(jobsController.status().polling, false, "异步任务轮询停止");

const cacheKey = (url: string, version: string) => `${url}@${version}`;
const flowchartEntry = buildFlowchartImageCacheEntry({ flowchartImageUrl: "img", flowchartUpdatedAt: "v1" }, cacheKey);
assertEqual(flowchartEntry.cacheKey, "img@v1", "流程图缓存键");
assertEqual(resolveVisibleImageIndex([{ cacheKey: "b@v" }], ["a", "b"], "v", cacheKey, 0), 1, "可见图片索引映射");
const share = buildRecipeShareConfig({
  channel: "message",
  recipe: { id: "1", title: "番茄炒蛋" },
  shareToken: "token",
  hasFlowchart: true,
  flowchartImageUrl: "flowchart",
});
assertEqual(share.title, "番茄炒蛋 · 完整做法", "分享标题");
assertEqual(String(share.path).includes("shareToken=token"), true, "公开只读分享 token");

console.log("Miniapp frontend refactor checks passed");
