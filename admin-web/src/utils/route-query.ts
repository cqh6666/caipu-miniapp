import type { LocationQuery, LocationQueryRaw, LocationQueryValue } from 'vue-router'

export type DateRangeValue = [Date, Date] | []

function firstQueryValue(value?: LocationQueryValue | LocationQueryValue[]) {
  if (Array.isArray(value)) {
    return value[0] || ''
  }
  return value || ''
}

export function readQueryString(query: LocationQuery, key: string) {
  return String(firstQueryValue(query[key])).trim()
}

export function readQueryNumber(query: LocationQuery, key: string, fallback: number) {
  const raw = readQueryString(query, key)
  if (!raw) {
    return fallback
  }
  const parsed = Number.parseInt(raw, 10)
  return Number.isNaN(parsed) || parsed <= 0 ? fallback : parsed
}

export function readDateRange(query: LocationQuery, startKey = 'timeFrom', endKey = 'timeTo'): DateRangeValue {
  const start = readQueryString(query, startKey)
  const end = readQueryString(query, endKey)
  if (!start || !end) {
    return []
  }
  const startDate = new Date(start)
  const endDate = new Date(end)
  if (Number.isNaN(startDate.getTime()) || Number.isNaN(endDate.getTime())) {
    return []
  }
  return [startDate, endDate]
}

export function writeDateRange(range: DateRangeValue, startKey = 'timeFrom', endKey = 'timeTo') {
  if (!range.length) {
    return {}
  }
  return {
    [startKey]: range[0].toISOString(),
    [endKey]: range[1].toISOString()
  }
}

export function buildRouteQuery(entries: Record<string, string | number | undefined | null>) {
  const query: LocationQueryRaw = {}
  for (const [key, value] of Object.entries(entries)) {
    if (value === undefined || value === null || value === '') {
      continue
    }
    query[key] = String(value)
  }
  return query
}
