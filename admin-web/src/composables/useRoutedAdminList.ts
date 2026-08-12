import { reactive, ref } from 'vue'
import type { LocationQueryRaw, RouteLocationNormalizedLoaded, Router } from 'vue-router'
import type { PaginationResult } from '@/types'
import {
  buildRouteQuery,
  readDateRange,
  readQueryNumber,
  readQueryString,
  writeDateRange,
  type DateRangeValue
} from '@/utils/route-query'

type StringFilterRecord = Record<string, string>

export type RoutedListField<Filters extends StringFilterRecord> = {
  key: Extract<keyof Filters, string>
  routeKey?: string
  requestKey?: string
  decode?: (value: string) => string
  encode?: (value: string) => string
}

type RoutedAdminListOptions<Item, Filters extends StringFilterRecord> = {
  route: RouteLocationNormalizedLoaded
  router: Router
  initialFilters: Filters
  fields: RoutedListField<Filters>[]
  fetchList: (query: URLSearchParams) => Promise<PaginationResult<Item>>
  loadErrorMessage: string
  onLoaded?: () => void
  pageSize?: number
}

function sameQuery(left: object, right: object) {
  return JSON.stringify(left) === JSON.stringify(right)
}

export function useRoutedAdminList<Item, Filters extends StringFilterRecord>({
  route,
  router,
  initialFilters,
  fields,
  fetchList,
  loadErrorMessage,
  onLoaded,
  pageSize = 20
}: RoutedAdminListOptions<Item, Filters>) {
  const page = ref(1)
  const loading = ref(false)
  const errorMessage = ref('')
  const timeRange = ref<DateRangeValue>([])
  const filters = reactive({ ...initialFilters }) as Filters
  const result = ref<PaginationResult<Item>>({
    items: [],
    total: 0,
    page: 1,
    pageSize
  })

  function syncStateFromRoute() {
    page.value = readQueryNumber(route.query, 'page', 1)
    for (const field of fields) {
      const raw = readQueryString(route.query, field.routeKey || field.key)
      filters[field.key] = (field.decode ? field.decode(raw) : raw) as Filters[typeof field.key]
    }
    timeRange.value = readDateRange(route.query)
  }

  function buildListRouteQuery(
    nextPage = page.value,
    extraEntries: Record<string, string | number | undefined | null> = {}
  ): LocationQueryRaw {
    const entries: Record<string, string | number | undefined | null> = {
      page: nextPage > 1 ? nextPage : undefined
    }
    for (const field of fields) {
      const value = filters[field.key]
      entries[field.routeKey || field.key] = value
        ? (field.encode ? field.encode(value) : value)
        : undefined
    }
    return buildRouteQuery({
      ...entries,
      ...extraEntries,
      ...writeDateRange(timeRange.value)
    })
  }

  function buildRequestQuery() {
    const query = new URLSearchParams()
    query.set('page', String(page.value))
    query.set('pageSize', String(pageSize))
    for (const field of fields) {
      const value = filters[field.key]
      if (value) query.set(field.requestKey || field.key, field.encode ? field.encode(value) : value)
    }
    if (timeRange.value.length === 2) {
      query.set('timeFrom', timeRange.value[0].toISOString())
      query.set('timeTo', timeRange.value[1].toISOString())
    }
    return query
  }

  async function loadList() {
    loading.value = true
    errorMessage.value = ''
    try {
      result.value = await fetchList(buildRequestQuery())
      onLoaded?.()
    } catch (error) {
      errorMessage.value = error instanceof Error ? error.message : loadErrorMessage
    } finally {
      loading.value = false
    }
  }

  async function applyFilters() {
    const nextQuery = buildListRouteQuery(1)
    if (sameQuery(route.query, nextQuery)) {
      page.value = 1
      await loadList()
      return
    }
    await router.replace({ query: nextQuery })
  }

  async function resetFilters() {
    for (const field of fields) filters[field.key] = '' as Filters[typeof field.key]
    timeRange.value = []
    if (!Object.keys(route.query).length) {
      page.value = 1
      await loadList()
      return
    }
    await router.replace({ query: {} })
  }

  async function handlePageChange(nextPage: number) {
    await router.replace({ query: buildListRouteQuery(nextPage) })
  }

  function removeFilter(key: Extract<keyof Filters, string> | 'timeRange') {
    if (key === 'timeRange') {
      timeRange.value = []
    } else {
      filters[key] = '' as Filters[typeof key]
    }
    void applyFilters()
  }

  return {
    applyFilters,
    buildListRouteQuery,
    buildRequestQuery,
    errorMessage,
    filters,
    handlePageChange,
    loadList,
    loading,
    page,
    removeFilter,
    resetFilters,
    result,
    syncStateFromRoute,
    timeRange
  }
}
