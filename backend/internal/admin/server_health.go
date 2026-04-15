package admin

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"os"
	osexec "os/exec"
	goruntime "runtime"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/cqh6666/caipu-miniapp/backend/internal/appsettings"
	"github.com/cqh6666/caipu-miniapp/backend/internal/config"
)

const serverHealthProbeTimeout = 2 * time.Second

type ServerHealthStatus string

const (
	ServerHealthStatusHealthy  ServerHealthStatus = "healthy"
	ServerHealthStatusWarning  ServerHealthStatus = "warning"
	ServerHealthStatusCritical ServerHealthStatus = "critical"
	ServerHealthStatusUnknown  ServerHealthStatus = "unknown"
)

type ServerHealthSummary struct {
	Status        ServerHealthStatus `json:"status"`
	HealthyCount  int                `json:"healthyCount"`
	WarningCount  int                `json:"warningCount"`
	CriticalCount int                `json:"criticalCount"`
	UnknownCount  int                `json:"unknownCount"`
}

type ServerHealthHost struct {
	Hostname           string   `json:"hostname"`
	Platform           string   `json:"platform"`
	UptimeSeconds      int64    `json:"uptimeSeconds"`
	CPUUsagePercent    *float64 `json:"cpuUsagePercent"`
	MemoryUsagePercent *float64 `json:"memoryUsagePercent"`
	DiskUsagePercent   *float64 `json:"diskUsagePercent"`
	Load1              *float64 `json:"load1"`
	Load5              *float64 `json:"load5"`
	Load15             *float64 `json:"load15"`
}

type ServerHealthCheck struct {
	Key       string             `json:"key"`
	Label     string             `json:"label"`
	Category  string             `json:"category"`
	Status    ServerHealthStatus `json:"status"`
	Target    string             `json:"target"`
	Detail    string             `json:"detail"`
	LatencyMS *int64             `json:"latencyMs"`
	CheckedAt string             `json:"checkedAt"`
}

type ServerHealthOverview struct {
	GeneratedAt string              `json:"generatedAt"`
	Summary     ServerHealthSummary `json:"summary"`
	Host        ServerHealthHost    `json:"host"`
	Checks      []ServerHealthCheck `json:"checks"`
}

type serverHealthHostSnapshot struct {
	Host    ServerHealthHost
	Signals []ServerHealthStatus
}

type execCommandFunc func(ctx context.Context, name string, args ...string) ([]byte, error)
type collectHostSnapshotFunc func(ctx context.Context, diskPath string) serverHealthHostSnapshot

type ServerHealthService struct {
	cfg                 config.Config
	runtime             *appsettings.RuntimeProvider
	httpClient          *http.Client
	execCommand         execCommandFunc
	collectHostSnapshot collectHostSnapshotFunc
	now                 func() time.Time
}

func NewServerHealthService(cfg config.Config, runtimeProvider *appsettings.RuntimeProvider) *ServerHealthService {
	return &ServerHealthService{
		cfg:     cfg,
		runtime: runtimeProvider,
		httpClient: &http.Client{
			Timeout: serverHealthProbeTimeout,
		},
		execCommand:         runExecCommand,
		collectHostSnapshot: collectServerHealthHostSnapshot,
		now:                 time.Now,
	}
}

func (s *ServerHealthService) Overview(ctx context.Context) (ServerHealthOverview, error) {
	now := s.now().UTC()
	hostSnapshot := s.collectHostSnapshot(ctx, s.resolveDiskPath())
	sidecarConfig := s.resolveSidecarConfig(ctx)

	checks := []ServerHealthCheck{
		s.checkSystemdService(ctx, "nginx", "Nginx", true),
		s.checkSystemdService(ctx, "caipu-backend", "Caipu Backend", true),
		s.checkSystemdService(ctx, "caipu-linkparse-sidecar", "Linkparse Sidecar", sidecarConfig.Enabled),
		s.checkHTTP(ctx, "backend-healthz", "Backend /healthz", "http://127.0.0.1:8080/healthz", true, ""),
		s.checkHTTP(ctx, "backend-api-healthz", "Backend /api/healthz", "http://127.0.0.1:8080/api/healthz", true, ""),
		s.checkHTTP(ctx, "sidecar-health", "Linkparse Sidecar /v1/health", sidecarHealthTarget(sidecarConfig.BaseURL), sidecarConfig.Enabled, sidecarConfig.APIKey),
	}

	statuses := append([]ServerHealthStatus{}, hostSnapshot.Signals...)
	for _, check := range checks {
		statuses = append(statuses, check.Status)
	}

	return ServerHealthOverview{
		GeneratedAt: now.Format(time.RFC3339),
		Summary:     summarizeServerHealthStatuses(statuses),
		Host:        hostSnapshot.Host,
		Checks:      checks,
	}, nil
}

func (s *ServerHealthService) resolveSidecarConfig(ctx context.Context) appsettings.LinkparseSidecarConfig {
	if s.runtime != nil {
		cfg := s.runtime.LinkparseSidecar(ctx)
		if strings.TrimSpace(cfg.BaseURL) != "" || cfg.Enabled {
			return cfg
		}
	}

	return appsettings.LinkparseSidecarConfig{
		Enabled: s.cfg.LinkparseSidecarEnabled,
		BaseURL: strings.TrimSpace(s.cfg.LinkparseSidecarBaseURL),
		APIKey:  strings.TrimSpace(s.cfg.LinkparseSidecarAPIKey),
		Timeout: time.Duration(s.cfg.LinkparseSidecarTimeoutSec) * time.Second,
	}
}

func (s *ServerHealthService) resolveDiskPath() string {
	path := strings.TrimSpace(s.cfg.UploadDir)
	if path == "" {
		return "/"
	}
	return path
}

func (s *ServerHealthService) checkSystemdService(ctx context.Context, key, label string, enabled bool) ServerHealthCheck {
	checkedAt := s.now().UTC().Format(time.RFC3339)
	if !enabled {
		return ServerHealthCheck{
			Key:       key,
			Label:     label,
			Category:  "systemd",
			Status:    ServerHealthStatusUnknown,
			Target:    key,
			Detail:    "当前环境未启用该服务",
			CheckedAt: checkedAt,
		}
	}

	commandCtx, cancel := context.WithTimeout(ctx, serverHealthProbeTimeout)
	defer cancel()

	output, err := s.execCommand(commandCtx, "systemctl", "is-active", key)
	detail := strings.TrimSpace(string(output))
	if detail == "" && err == nil {
		detail = "active"
	}

	check := ServerHealthCheck{
		Key:       key,
		Label:     label,
		Category:  "systemd",
		Target:    key,
		CheckedAt: checkedAt,
	}

	if err != nil {
		if isSystemdUnavailable(err, detail) {
			check.Status = ServerHealthStatusUnknown
			check.Detail = fallbackDetail(detail, "当前环境不支持 systemd 检查")
			return check
		}

		check.Status = ServerHealthStatusCritical
		check.Detail = fallbackDetail(detail, strings.TrimSpace(err.Error()))
		return check
	}

	if detail == "active" {
		check.Status = ServerHealthStatusHealthy
		check.Detail = "服务运行正常"
		return check
	}

	check.Status = ServerHealthStatusCritical
	check.Detail = fallbackDetail(detail, "systemd 返回非 active 状态")
	return check
}

func (s *ServerHealthService) checkHTTP(ctx context.Context, key, label, target string, enabled bool, bearerToken string) ServerHealthCheck {
	checkedAt := s.now().UTC().Format(time.RFC3339)
	if !enabled {
		return ServerHealthCheck{
			Key:       key,
			Label:     label,
			Category:  "http",
			Status:    ServerHealthStatusUnknown,
			Target:    target,
			Detail:    "当前环境未启用该检查",
			CheckedAt: checkedAt,
		}
	}
	if strings.TrimSpace(target) == "" {
		return ServerHealthCheck{
			Key:       key,
			Label:     label,
			Category:  "http",
			Status:    ServerHealthStatusUnknown,
			Target:    target,
			Detail:    "当前环境缺少可用探测地址",
			CheckedAt: checkedAt,
		}
	}

	requestCtx, cancel := context.WithTimeout(ctx, serverHealthProbeTimeout)
	defer cancel()

	req, err := http.NewRequestWithContext(requestCtx, http.MethodGet, target, nil)
	if err != nil {
		return ServerHealthCheck{
			Key:       key,
			Label:     label,
			Category:  "http",
			Status:    ServerHealthStatusCritical,
			Target:    target,
			Detail:    strings.TrimSpace(err.Error()),
			CheckedAt: checkedAt,
		}
	}
	if strings.TrimSpace(bearerToken) != "" {
		req.Header.Set("Authorization", "Bearer "+strings.TrimSpace(bearerToken))
	}

	startedAt := time.Now()
	resp, err := s.httpClient.Do(req)
	latencyMS := time.Since(startedAt).Milliseconds()
	latency := int64Ptr(latencyMS)

	if err != nil {
		return ServerHealthCheck{
			Key:       key,
			Label:     label,
			Category:  "http",
			Status:    ServerHealthStatusCritical,
			Target:    target,
			Detail:    strings.TrimSpace(err.Error()),
			LatencyMS: latency,
			CheckedAt: checkedAt,
		}
	}
	defer resp.Body.Close()

	statusText := fmt.Sprintf("HTTP %d %s", resp.StatusCode, http.StatusText(resp.StatusCode))
	if resp.StatusCode >= 200 && resp.StatusCode < 300 {
		return ServerHealthCheck{
			Key:       key,
			Label:     label,
			Category:  "http",
			Status:    ServerHealthStatusHealthy,
			Target:    target,
			Detail:    statusText,
			LatencyMS: latency,
			CheckedAt: checkedAt,
		}
	}

	return ServerHealthCheck{
		Key:       key,
		Label:     label,
		Category:  "http",
		Status:    ServerHealthStatusCritical,
		Target:    target,
		Detail:    statusText,
		LatencyMS: latency,
		CheckedAt: checkedAt,
	}
}

func summarizeServerHealthStatuses(statuses []ServerHealthStatus) ServerHealthSummary {
	summary := ServerHealthSummary{
		Status: ServerHealthStatusUnknown,
	}

	for _, status := range statuses {
		switch status {
		case ServerHealthStatusHealthy:
			summary.HealthyCount++
		case ServerHealthStatusWarning:
			summary.WarningCount++
		case ServerHealthStatusCritical:
			summary.CriticalCount++
		default:
			summary.UnknownCount++
		}
	}

	switch {
	case summary.CriticalCount > 0:
		summary.Status = ServerHealthStatusCritical
	case summary.WarningCount > 0:
		summary.Status = ServerHealthStatusWarning
	case summary.HealthyCount > 0:
		summary.Status = ServerHealthStatusHealthy
	default:
		summary.Status = ServerHealthStatusUnknown
	}

	return summary
}

func collectServerHealthHostSnapshot(ctx context.Context, diskPath string) serverHealthHostSnapshot {
	snapshot := serverHealthHostSnapshot{
		Host: ServerHealthHost{
			Platform: goruntime.GOOS,
		},
		Signals: make([]ServerHealthStatus, 0, 3),
	}

	if hostname, err := os.Hostname(); err == nil {
		snapshot.Host.Hostname = hostname
	}

	if goruntime.GOOS == "linux" {
		if uptime, err := readLinuxUptimeSeconds(); err == nil {
			snapshot.Host.UptimeSeconds = uptime
		}
		if usage, err := readLinuxCPUUsagePercent(ctx); err == nil {
			snapshot.Host.CPUUsagePercent = float64Ptr(usage)
			snapshot.Signals = append(snapshot.Signals, statusForUsage(usage, 75, 90))
		} else {
			snapshot.Signals = append(snapshot.Signals, ServerHealthStatusUnknown)
		}
		if usage, err := readLinuxMemoryUsagePercent(); err == nil {
			snapshot.Host.MemoryUsagePercent = float64Ptr(usage)
			snapshot.Signals = append(snapshot.Signals, statusForUsage(usage, 75, 90))
		} else {
			snapshot.Signals = append(snapshot.Signals, ServerHealthStatusUnknown)
		}
		if load1, load5, load15, err := readLinuxLoadAverage(); err == nil {
			snapshot.Host.Load1 = float64Ptr(load1)
			snapshot.Host.Load5 = float64Ptr(load5)
			snapshot.Host.Load15 = float64Ptr(load15)
		}
	} else {
		snapshot.Signals = append(snapshot.Signals, ServerHealthStatusUnknown, ServerHealthStatusUnknown)
	}

	if usage, err := readDiskUsagePercent(diskPath); err == nil {
		snapshot.Host.DiskUsagePercent = float64Ptr(usage)
		snapshot.Signals = append(snapshot.Signals, statusForUsage(usage, 80, 90))
	} else {
		snapshot.Signals = append(snapshot.Signals, ServerHealthStatusUnknown)
	}

	return snapshot
}

func runExecCommand(ctx context.Context, name string, args ...string) ([]byte, error) {
	return osexec.CommandContext(ctx, name, args...).CombinedOutput()
}

func readLinuxUptimeSeconds() (int64, error) {
	content, err := os.ReadFile("/proc/uptime")
	if err != nil {
		return 0, err
	}

	parts := strings.Fields(string(content))
	if len(parts) == 0 {
		return 0, errors.New("invalid /proc/uptime content")
	}

	value, err := strconv.ParseFloat(parts[0], 64)
	if err != nil {
		return 0, err
	}

	return int64(value), nil
}

func readLinuxCPUUsagePercent(ctx context.Context) (float64, error) {
	first, err := readLinuxCPUStat()
	if err != nil {
		return 0, err
	}

	timer := time.NewTimer(120 * time.Millisecond)
	defer timer.Stop()

	select {
	case <-ctx.Done():
		return 0, ctx.Err()
	case <-timer.C:
	}

	second, err := readLinuxCPUStat()
	if err != nil {
		return 0, err
	}

	totalDelta := second.total - first.total
	idleDelta := second.idle - first.idle
	if totalDelta == 0 {
		return 0, errors.New("invalid cpu sample delta")
	}

	usage := float64(totalDelta-idleDelta) / float64(totalDelta) * 100
	return clampPercent(usage), nil
}

func readLinuxCPUStat() (struct {
	idle  uint64
	total uint64
}, error) {
	content, err := os.ReadFile("/proc/stat")
	if err != nil {
		return struct {
			idle  uint64
			total uint64
		}{}, err
	}

	lines := strings.Split(string(content), "\n")
	if len(lines) == 0 {
		return struct {
			idle  uint64
			total uint64
		}{}, errors.New("invalid /proc/stat content")
	}

	fields := strings.Fields(lines[0])
	if len(fields) < 5 || fields[0] != "cpu" {
		return struct {
			idle  uint64
			total uint64
		}{}, errors.New("invalid cpu stat row")
	}

	var total uint64
	values := make([]uint64, 0, len(fields)-1)
	for _, field := range fields[1:] {
		value, err := strconv.ParseUint(field, 10, 64)
		if err != nil {
			return struct {
				idle  uint64
				total uint64
			}{}, err
		}
		values = append(values, value)
		total += value
	}

	idle := values[3]
	if len(values) > 4 {
		idle += values[4]
	}

	return struct {
		idle  uint64
		total uint64
	}{
		idle:  idle,
		total: total,
	}, nil
}

func readLinuxMemoryUsagePercent() (float64, error) {
	content, err := os.ReadFile("/proc/meminfo")
	if err != nil {
		return 0, err
	}

	lines := strings.Split(string(content), "\n")
	values := make(map[string]uint64, 3)
	for _, line := range lines {
		fields := strings.Fields(line)
		if len(fields) < 2 {
			continue
		}

		value, err := strconv.ParseUint(fields[1], 10, 64)
		if err != nil {
			continue
		}
		values[strings.TrimSuffix(fields[0], ":")] = value
	}

	total := values["MemTotal"]
	available := values["MemAvailable"]
	if total == 0 {
		return 0, errors.New("meminfo missing MemTotal")
	}
	if available == 0 {
		if free, ok := values["MemFree"]; ok {
			available = free
		}
	}

	usage := (float64(total-available) / float64(total)) * 100
	return clampPercent(usage), nil
}

func readLinuxLoadAverage() (float64, float64, float64, error) {
	content, err := os.ReadFile("/proc/loadavg")
	if err != nil {
		return 0, 0, 0, err
	}

	fields := strings.Fields(string(content))
	if len(fields) < 3 {
		return 0, 0, 0, errors.New("invalid /proc/loadavg content")
	}

	load1, err := strconv.ParseFloat(fields[0], 64)
	if err != nil {
		return 0, 0, 0, err
	}
	load5, err := strconv.ParseFloat(fields[1], 64)
	if err != nil {
		return 0, 0, 0, err
	}
	load15, err := strconv.ParseFloat(fields[2], 64)
	if err != nil {
		return 0, 0, 0, err
	}

	return load1, load5, load15, nil
}

func readDiskUsagePercent(path string) (float64, error) {
	target := strings.TrimSpace(path)
	if target == "" {
		target = "/"
	}

	var stat syscall.Statfs_t
	if err := syscall.Statfs(target, &stat); err != nil {
		return 0, err
	}

	total := float64(stat.Blocks) * float64(stat.Bsize)
	available := float64(stat.Bavail) * float64(stat.Bsize)
	if total <= 0 {
		return 0, errors.New("invalid filesystem size")
	}

	usage := ((total - available) / total) * 100
	return clampPercent(usage), nil
}

func statusForUsage(value, warningThreshold, criticalThreshold float64) ServerHealthStatus {
	switch {
	case value >= criticalThreshold:
		return ServerHealthStatusCritical
	case value >= warningThreshold:
		return ServerHealthStatusWarning
	default:
		return ServerHealthStatusHealthy
	}
}

func sidecarHealthTarget(baseURL string) string {
	baseURL = strings.TrimRight(strings.TrimSpace(baseURL), "/")
	if baseURL == "" {
		return ""
	}
	return baseURL + "/v1/health"
}

func isSystemdUnavailable(err error, detail string) bool {
	if errors.Is(err, osexec.ErrNotFound) {
		return true
	}

	message := strings.ToLower(strings.TrimSpace(detail))
	if message == "" {
		message = strings.ToLower(strings.TrimSpace(err.Error()))
	}

	return strings.Contains(message, "system has not been booted with systemd") ||
		strings.Contains(message, "failed to connect to bus") ||
		strings.Contains(message, "launchctl") ||
		strings.Contains(message, "executable file not found") ||
		strings.Contains(message, "command not found")
}

func fallbackDetail(value, fallback string) string {
	value = strings.TrimSpace(value)
	if value != "" {
		return value
	}
	return strings.TrimSpace(fallback)
}

func clampPercent(value float64) float64 {
	if value < 0 {
		return 0
	}
	if value > 100 {
		return 100
	}
	return value
}

func float64Ptr(value float64) *float64 {
	return &value
}

func int64Ptr(value int64) *int64 {
	return &value
}
