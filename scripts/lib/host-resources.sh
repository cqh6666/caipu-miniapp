#!/usr/bin/env bash

host_resources_cpu_count() {
  if [[ -n "${HOST_RESOURCES_CPU_COUNT:-}" ]]; then
    printf '%s\n' "$HOST_RESOURCES_CPU_COUNT"
    return
  fi
  if command -v nproc >/dev/null 2>&1; then
    nproc
    return
  fi
  if command -v getconf >/dev/null 2>&1; then
    getconf _NPROCESSORS_ONLN 2>/dev/null && return
  fi
  if command -v sysctl >/dev/null 2>&1; then
    sysctl -n hw.ncpu 2>/dev/null && return
  fi
  echo 1
}

host_resources_mem_total_mb() {
  if [[ -n "${HOST_RESOURCES_MEM_TOTAL_MB:-}" ]]; then
    printf '%s\n' "$HOST_RESOURCES_MEM_TOTAL_MB"
    return
  fi
  if [[ -r /proc/meminfo ]]; then
    awk '/MemTotal:/ { printf "%d\n", $2 / 1024 }' /proc/meminfo
    return
  fi
  if command -v sysctl >/dev/null 2>&1; then
    local bytes
    bytes="$(sysctl -n hw.memsize 2>/dev/null || true)"
    if [[ "$bytes" =~ ^[0-9]+$ ]]; then
      echo $(( bytes / 1024 / 1024 ))
      return
    fi
  fi
  echo 0
}

host_resources_swap_total_mb() {
  if [[ -n "${HOST_RESOURCES_SWAP_TOTAL_MB:-}" ]]; then
    printf '%s\n' "$HOST_RESOURCES_SWAP_TOTAL_MB"
    return
  fi
  if [[ -r /proc/meminfo ]]; then
    awk '/SwapTotal:/ { printf "%d\n", $2 / 1024 }' /proc/meminfo
    return
  fi
  if command -v sysctl >/dev/null 2>&1; then
    local amount parsed swap_usage unit
    swap_usage="$(sysctl -n vm.swapusage 2>/dev/null || true)"
    parsed="$(sed -E -n 's/.*total = ([0-9.]+)([MG]).*/\1 \2/p' <<< "$swap_usage")"
    amount="${parsed%% *}"
    unit="${parsed##* }"
    if [[ "$amount" =~ ^[0-9]+([.][0-9]+)?$ ]]; then
      awk -v amount="$amount" -v unit="$unit" 'BEGIN {
        if (unit == "G") amount *= 1024
        printf "%d\n", amount
      }'
      return
    fi
  fi
  echo 0
}

host_resources_is_low_resource() {
  local min_cpu="$1"
  local min_mem_mb="$2"
  local min_swap_mb="$3"
  local cpu_count mem_total_mb swap_total_mb

  cpu_count="$(host_resources_cpu_count)"
  mem_total_mb="$(host_resources_mem_total_mb)"
  swap_total_mb="$(host_resources_swap_total_mb)"

  (( cpu_count < min_cpu || mem_total_mb < min_mem_mb || swap_total_mb < min_swap_mb ))
}

host_resources_run_low_priority() {
  local nice_value="$1"
  shift
  if command -v ionice >/dev/null 2>&1; then
    ionice -c3 nice -n "$nice_value" "$@"
    return
  fi
  nice -n "$nice_value" "$@"
}

host_resources_print_summary() {
  local cpu_count mem_total_mb swap_total_mb

  cpu_count="$(host_resources_cpu_count)"
  mem_total_mb="$(host_resources_mem_total_mb)"
  swap_total_mb="$(host_resources_swap_total_mb)"

  cat <<EOF
Host resources:
- cpu: ${cpu_count} vCPU
- memory: ${mem_total_mb} MiB
- swap: ${swap_total_mb} MiB
EOF
}
