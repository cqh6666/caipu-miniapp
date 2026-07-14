import {
  createSearchBlurController,
  filterRecipes,
  pickRandomRecipe,
  recipeLibraryModule,
} from "../pages/index/use-recipe-library";
import { createEmptyPlaceDraft, filterPlaces, placeLibraryModule } from "../pages/index/use-place-library";
import { createMealOrderDraftSyncController, mealOrderModule, upsertMealOrderItem } from "../pages/index/use-meal-order";
import { buildInviteShareTitle, kitchenSpaceModule, memberRoleLabel } from "../pages/index/use-kitchen-space";
import { buildPlaceDraftFromCandidate, smartAddModule } from "../pages/index/use-smart-add";
import {
  defineIndexPageModule,
  installIndexPageModules,
  runIndexPageModuleLifecycle,
  validateIndexPageModuleContext,
} from "../pages/index/page-module";
import {
  createAddPreviewFlowController,
  hasParseableShareHint,
  readClipboardText,
} from "../pages/index/use-add-preview-flow";
import { createCountUpController, easeOutCubic } from "../utils/count-up";
import { createImageDisplayController } from "../utils/image-cache";
import { normalizePublicAppConfig } from "../utils/public-app-config-api";
import { createChunkDecoder } from "../utils/diet-assistant-stream-decoder";
import { createDietAssistantStreamParser } from "../utils/diet-assistant-sse";
import {
  MAX_RECIPE_IMAGES,
  buildRecipePayload,
  normalizeParsedContentView,
  normalizeRecipe,
} from "../utils/recipe-model";
import { loadRecipesForKitchen, saveRecipesForKitchen } from "../utils/recipe-cache";
import {
  buildRecipeEditPayload,
  buildStepCompletionKeyList,
  createCompletedStepStoragePayload,
  normalizeCompletedStepKeyMap,
  serializeComparableEditDraft,
} from "../pages/recipe-detail/use-recipe-edit";
import {
  createActionFeedbackController,
  createRecipeLoadController,
  createStepCompletionController,
} from "../pages/recipe-detail/use-recipe-detail-state";
import {
  buildFlowchartWaitHint,
  createRecipeAsyncJobsController,
  resolveRemainingWaitSeconds,
  toPositiveInteger,
} from "../pages/recipe-detail/use-recipe-async-jobs";
import {
  buildRecipeShareConfig,
} from "../pages/recipe-detail/use-recipe-share";
import {
  buildFlowchartImageCacheEntry,
  resolveVisibleImageIndex,
} from "../pages/recipe-detail/use-recipe-images";

function assertEqual<T>(actual: T, expected: T, label: string) {
  if (actual !== expected) throw new Error(`${label}: expected ${String(expected)}, got ${String(actual)}`);
}

const recipes = [
  { id: "1", title: "番茄炒蛋", mealType: "main", status: "wishlist" },
  { id: "2", title: "早餐粥", mealType: "breakfast", status: "done" },
];
assertEqual(normalizePublicAppConfig().features.dietAssistantEnabled, false, "AI 助手配置缺失时默认关闭");
assertEqual(normalizePublicAppConfig({ features: { dietAssistantEnabled: "invalid" } }).features.dietAssistantEnabled, false, "AI 助手非法配置保持关闭");
assertEqual(normalizePublicAppConfig({ features: { dietAssistantEnabled: true } }).features.dietAssistantEnabled, true, "AI 助手仅显式开启时展示");
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
const indexModules = [placeLibraryModule, recipeLibraryModule, mealOrderModule, kitchenSpaceModule, smartAddModule];
const installedIndexModules = installIndexPageModules(indexModules);
assertEqual(Object.keys(installedIndexModules.methods).length > 20, true, "首页模块方法统一安装");
assertEqual(Object.keys(installedIndexModules.computed).length > 20, true, "首页模块计算属性统一安装");
assertEqual(validateIndexPageModuleContext({}, indexModules).length > 0, true, "首页模块显式声明上下文依赖");
let moduleDeactivateCount = 0;
let moduleDisposeCount = 0;
const lifecycleModule = defineIndexPageModule({
  name: "lifecycle-test",
  lifecycle: {
    deactivate() { moduleDeactivateCount += 1; },
    dispose() { moduleDisposeCount += 1; },
  },
});
runIndexPageModuleLifecycle({}, [lifecycleModule], "dispose");
assertEqual(moduleDeactivateCount, 1, "首页模块销毁前统一停用");
assertEqual(moduleDisposeCount, 1, "首页模块统一销毁");
let kitchenChangePayload = 0;
const kitchenLifecycleModule = defineIndexPageModule({
  name: "kitchen-lifecycle-test",
  lifecycle: { onKitchenChange({ nextKitchenId }) { kitchenChangePayload = nextKitchenId; } },
});
runIndexPageModuleLifecycle({}, [kitchenLifecycleModule], "onKitchenChange", { nextKitchenId: 2 });
assertEqual(kitchenChangePayload, 2, "空间切换上下文按模块分发");

assertEqual(hasParseableShareHint("复制 https://example.com/a"), true, "分享链接提示识别");
assertEqual(hasParseableShareHint("来自小红书的分享"), true, "分享平台提示识别");
assertEqual(hasParseableShareHint("普通文字"), false, "普通文字不触发菜谱解析");
assertEqual(await readClipboardText({ getClipboardData: ({ success }) => success({ data: "  链接  " }) }), "链接", "剪贴板内容归一化");
let intervalCallback: (() => void) | null = null;
let clearedPreviewTimer = 0;
const previewStates: Array<{ isParsing: boolean; stage: string; duration: number }> = [];
const previewController = createAddPreviewFlowController({
  onState: (state) => previewStates.push(state),
  setIntervalFn: (callback) => {
    intervalCallback = callback;
    return 1;
  },
  clearIntervalFn: () => { clearedPreviewTimer += 1; },
});
const previewRunId = previewController.start();
intervalCallback?.();
previewController.setStage(previewRunId, "fetching");
assertEqual(previewController.status().duration, 1, "预览流程计时");
assertEqual(previewController.status().stage, "fetching", "预览流程阶段切换");
assertEqual(previewController.stop(previewRunId), true, "预览流程主动停止");
assertEqual(previewController.stop(previewRunId), false, "旧预览流程不能重复收尾");
assertEqual(clearedPreviewTimer, 1, "预览流程清理计时器");
assertEqual(previewStates.at(-1)?.isParsing, false, "预览流程停止后复位状态");

assertEqual(easeOutCubic(0), 0, "数字缓动起点");
assertEqual(easeOutCubic(1), 1, "数字缓动终点");
const animatedValues: Record<string, number> = { total: 0, average: 2 };
const countUpCallbacks = new Map<number, () => void>();
let countUpTimerSequence = 0;
const countUpController = createCountUpController({
  read: (key) => animatedValues[key] || 0,
  write: (key, value) => { animatedValues[key] = value; },
  duration: 80,
  step: 40,
  setIntervalFn: (callback) => {
    const id = ++countUpTimerSequence;
    countUpCallbacks.set(id, callback);
    return id;
  },
  clearIntervalFn: (id) => { countUpCallbacks.delete(id); },
});
countUpController.animate("total", 9, { round: true });
countUpCallbacks.get(1)?.();
assertEqual(Number.isInteger(animatedValues.total), true, "整数数字滚动保持取整");
countUpCallbacks.get(1)?.();
assertEqual(animatedValues.total, 9, "数字滚动到达精确终值");
countUpController.animate("total", 12);
countUpController.animate("average", 3.5);
assertEqual(countUpController.activeKeys().length, 2, "多键数字滚动并行");
assertEqual(countUpTimerSequence, 2, "同批数字滚动复用单个计时器");
countUpController.clear(["total"]);
assertEqual(countUpController.activeKeys()[0], "average", "数字滚动按键取消");
countUpController.clear();
assertEqual(countUpController.activeKeys().length, 0, "数字滚动批量清理");

const imageDisplayStates: Array<Record<string, Record<string, unknown>>> = [];
let invalidatedImageCount = 0;
const imageDisplayController = createImageDisplayController({
  onState: (state) => imageDisplayStates.push(state),
  getCachedImagePathFn: async () => "/local/cover.jpg",
  warmImageCacheFn: async () => {},
  invalidateCachedImageFn: async () => { invalidatedImageCount += 1; },
});
await imageDisplayController.sync([{ key: "cover", url: "https://img/cover.jpg", version: "v1" }]);
assertEqual(imageDisplayController.displayURL("cover"), "/local/cover.jpg", "图片缓存命中本地路径");
assertEqual(await imageDisplayController.handleError({ key: "cover", displayURL: "/local/cover.jpg" }), "fallback", "本地图片失败后回退远端");
assertEqual(imageDisplayController.displayURL("cover"), "https://img/cover.jpg", "图片缓存回退保留远端地址");
assertEqual(invalidatedImageCount, 1, "损坏本地图片缓存失效");
assertEqual(await imageDisplayController.handleError({ key: "cover", displayURL: "https://img/cover.jpg" }), "hidden", "远端图片失败后隐藏");
assertEqual(imageDisplayController.displayURL("cover"), "", "隐藏态不再返回失败图片");

const manualDecoder = createChunkDecoder({ TextDecoderClass: null });
assertEqual(manualDecoder.decode(new Uint8Array([0xe4])), "", "UTF-8 中文半包暂存");
assertEqual(manualDecoder.decode(new Uint8Array([0xbd, 0xa0])), "你", "UTF-8 中文跨包还原");
assertEqual(manualDecoder.flush(), "", "UTF-8 完整流无残留");
const streamEvents: string[] = [];
let streamMutationRecipeId = "";
let streamErrorMessage = "";
const streamParser = createDietAssistantStreamParser({
  onDelta: (value) => streamEvents.push(`delta:${value}`),
  onStatus: (event) => {
    streamEvents.push(`status:${event.type}`);
    streamMutationRecipeId = event.mutation?.recipeId || "";
  },
  onError: (error) => { streamErrorMessage = error.message; },
  onDone: () => streamEvents.push("done"),
});
streamParser.push('data: {"type":"delta","delta":"你');
streamParser.push('好"}\r\n\r\ndata: {"type":"tool_done","message":"ok","mutation":{"type":"recipe_created","recipeId":"42"}}\n\n');
streamParser.push('data: {"type":"error","message":"失败"}\n\n');
streamParser.push("data: [DONE]\n\n");
streamParser.flush();
assertEqual(streamEvents[0], "delta:你好", "SSE JSON 半包合并");
assertEqual(streamEvents[1], "status:tool_done", "SSE 工具状态分发");
assertEqual(streamMutationRecipeId, "42", "SSE mutation 归一化");
assertEqual(streamErrorMessage, "失败", "SSE 错误事件分发");
assertEqual(streamEvents.at(-1), "done", "SSE 结束事件分发");

const legacyRecipe = normalizeRecipe({
  id: "legacy",
  title: "旧菜谱",
  imageUrls: Array.from({ length: 12 }, (_, index) => `https://img/${index}.jpg`),
  parsedContent: { ingredients: ["鸡蛋 2个", "盐 少许"], steps: ["炒熟"] },
});
assertEqual(legacyRecipe.imageUrls.length, MAX_RECIPE_IMAGES, "菜谱图片上限");
assertEqual(normalizeParsedContentView(legacyRecipe.parsedContent).steps[0]?.detail, "炒熟", "旧步骤字段归一化");
const fallbackRecipe = normalizeRecipe({ id: "fallback", title: "空菜谱" });
assertEqual(fallbackRecipe.parsedContent.steps.length, 3, "缺少解析内容时生成 fallback");
assertEqual(buildRecipePayload(fallbackRecipe).parsedContent.steps.length, 0, "fallback 不写回后端");
const recipeStorage = new Map<string, unknown>();
const storageAdapter = {
  getStorageSync: (key: string) => recipeStorage.get(key),
  setStorageSync: (key: string, value: unknown) => recipeStorage.set(key, value),
};
saveRecipesForKitchen(1, [{ id: "a", kitchenId: 1, title: "空间一" }], storageAdapter);
saveRecipesForKitchen(2, [{ id: "b", kitchenId: 2, title: "空间二" }], storageAdapter);
assertEqual(loadRecipesForKitchen(1, storageAdapter)[0]?.id, "a", "菜谱缓存按空间读取");
assertEqual(loadRecipesForKitchen(2, storageAdapter)[0]?.id, "b", "菜谱缓存空间隔离");

let feedbackTimeout: (() => void) | null = null;
const feedbackStates: Array<Record<string, unknown>> = [];
const feedbackController = createActionFeedbackController({
  onState: (state) => feedbackStates.push(state),
  setTimeoutFn: (callback) => { feedbackTimeout = callback; return 1; },
  clearTimeoutFn: () => { feedbackTimeout = null; },
});
assertEqual(feedbackController.show({ title: "保存成功", duration: 1 }), true, "动作反馈展示");
feedbackTimeout?.();
assertEqual(feedbackStates.at(-1)?.visible, false, "动作反馈自动隐藏");
feedbackController.show({ title: "再次保存" });
feedbackController.clear();
assertEqual(feedbackController.status().active, false, "动作反馈卸载清理");

const stepStorageMap = new Map<string, unknown>();
const stepStorage = {
  getStorageSync: (key: string) => stepStorageMap.get(key),
  setStorageSync: (key: string, value: unknown) => stepStorageMap.set(key, value),
  removeStorageSync: (key: string) => stepStorageMap.delete(key),
};
let completedSnapshot: Record<string, boolean> = {};
const controllerStepKeys = buildStepCompletionKeyList([{ title: "备料", detail: "切好" }]);
const stepController = createStepCompletionController({
  storage: stepStorage,
  onChange: (value) => { completedSnapshot = value; },
});
stepController.load("recipe-1", controllerStepKeys);
assertEqual(stepController.toggle("recipe-1", controllerStepKeys, 0), true, "步骤完成态切换");
assertEqual(completedSnapshot[controllerStepKeys[0]], true, "步骤完成态响应式快照");
const restoredStepController = createStepCompletionController({ storage: stepStorage });
assertEqual(restoredStepController.load("recipe-1", controllerStepKeys)[controllerStepKeys[0]], true, "步骤完成态持久化恢复");
stepController.reset("recipe-1");
assertEqual(Object.keys(stepController.snapshot()).length, 0, "步骤完成态重置");

const loadedRecipeSources: string[] = [];
const loaderStates: Array<Record<string, boolean>> = [];
let privateEnsureCount = 0;
const recipeLoader = createRecipeLoadController({
  getCachedRecipe: () => ({ id: "cached" }),
  getRecipe: async () => ({ id: "remote" }),
  getPublicRecipe: async () => ({ recipe: { id: "public" }, kitchenName: "空间" }),
  ensurePrivateAccess: () => { privateEnsureCount += 1; },
  onRecipe: (_recipe, meta) => loadedRecipeSources.push(meta.source),
  onState: (state) => loaderStates.push(state),
});
await recipeLoader.load({ recipeId: "1" });
assertEqual(loadedRecipeSources.join(","), "cache,remote", "详情加载缓存后远端刷新");
assertEqual(privateEnsureCount, 1, "详情私有加载并行确保分享 token");
assertEqual(loaderStates.at(-1)?.resolved, true, "详情加载完成态");
await recipeLoader.load({ isPublicView: true, publicViewToken: "token" });
assertEqual(loadedRecipeSources.at(-1), "public", "详情公开只读加载不走私有缓存");

let syncCount = 0;
const scheduler = createMealOrderDraftSyncController(async () => { syncCount += 1; }, 1);
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

console.log("Miniapp frontend checks passed");
