import { computed, type Ref } from "vue";
import type { AIRoutingSceneConfig } from "@/types";
import {
  buildSceneDiff,
  comparableScene,
} from "@/utils/ai-provider-draft";

type AIRoutingDraftOptions = {
  draftScene: Ref<AIRoutingSceneConfig | null>;
  remoteScene: Ref<AIRoutingSceneConfig | null>;
};

export function useAIRoutingDraft({
  draftScene,
  remoteScene,
}: AIRoutingDraftOptions) {
  const isDirty = computed(() => {
    if (!draftScene.value || !remoteScene.value) return false;
    return comparableScene(draftScene.value) !== comparableScene(remoteScene.value);
  });

  const sceneDiff = computed(() =>
    buildSceneDiff(draftScene.value, remoteScene.value),
  );
  const diffCount = computed(() => sceneDiff.value.length);
  const pendingClearKeyCount = computed(
    () =>
      draftScene.value?.providers.filter((provider) => provider.clearApiKey)
        .length || 0,
  );
  const pendingRemovedProviderCount = computed(
    () =>
      sceneDiff.value.filter(
        (item) =>
          item.scope.startsWith("provider:") && item.path === "removed",
      ).length,
  );

  return {
    diffCount,
    isDirty,
    pendingClearKeyCount,
    pendingRemovedProviderCount,
    sceneDiff,
  };
}
