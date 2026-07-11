import { onBeforeUnmount, onMounted, ref } from "vue";
import { init, type ECharts, type EChartsCoreOption } from "echarts/core";

export type DashboardChartKey = "trend" | "scene" | "provider" | "model";

export function useDashboardCharts() {
  const chartRef = ref<HTMLDivElement | null>(null);
  const sceneChartRef = ref<HTMLDivElement | null>(null);
  const providerChartRef = ref<HTMLDivElement | null>(null);
  const modelChartRef = ref<HTMLDivElement | null>(null);
  const instances = new Map<DashboardChartKey, ECharts>();

  function containerFor(key: DashboardChartKey) {
    if (key === "trend") return chartRef.value;
    if (key === "scene") return sceneChartRef.value;
    if (key === "provider") return providerChartRef.value;
    return modelChartRef.value;
  }

  function ensure(key: DashboardChartKey, container = containerFor(key)) {
    if (!container) return null;
    const current = instances.get(key);
    if (current?.getDom() === container) return current;
    current?.dispose();
    const next = init(container);
    instances.set(key, next);
    return next;
  }

  function render(
    key: DashboardChartKey,
    option: EChartsCoreOption,
    container = containerFor(key),
  ) {
    const instance = ensure(key, container);
    if (!instance) return;
    instance.setOption(option, true);
    instance.resize();
    requestAnimationFrame(() => instance.resize());
  }

  function dispose(key: DashboardChartKey) {
    instances.get(key)?.dispose();
    instances.delete(key);
  }

  function resizeAll() {
    instances.forEach((instance) => instance.resize());
  }

  function disposeAll() {
    instances.forEach((instance) => instance.dispose());
    instances.clear();
  }

  onMounted(() => window.addEventListener("resize", resizeAll));
  onBeforeUnmount(() => {
    window.removeEventListener("resize", resizeAll);
    disposeAll();
  });

  return {
    chartRef,
    dispose,
    modelChartRef,
    providerChartRef,
    render,
    sceneChartRef,
  };
}
