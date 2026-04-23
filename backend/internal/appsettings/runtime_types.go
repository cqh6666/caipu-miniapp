package appsettings

import (
	"context"
	"time"

	"github.com/cqh6666/caipu-miniapp/backend/internal/aialert"
)

type RuntimeConfigProvider interface {
	SummaryAI(ctx context.Context) SummaryAIConfig
	FlowchartAI(ctx context.Context) FlowchartAIConfig
	TitleAI(ctx context.Context) TitleAIConfig
	AIProviderAlert(ctx context.Context) aialert.Config
	LinkparseSidecar(ctx context.Context) LinkparseSidecarConfig
}

type SummaryAIConfig struct {
	BaseURL string
	APIKey  string
	Model   string
	Timeout time.Duration
}

type FlowchartAIConfig struct {
	BaseURL        string
	APIKey         string
	Model          string
	EndpointMode   string
	ResponseFormat string
	Timeout        time.Duration
}

type TitleAIConfig struct {
	Enabled     bool
	BaseURL     string
	APIKey      string
	Model       string
	Stream      bool
	Temperature float64
	MaxTokens   int
	Timeout     time.Duration
}

type LinkparseSidecarConfig struct {
	Enabled bool
	BaseURL string
	APIKey  string
	Timeout time.Duration
}

type RuntimeSettingFieldView struct {
	Key               string `json:"key"`
	Label             string `json:"label"`
	Description       string `json:"description"`
	ValueType         string `json:"valueType"`
	IsSecret          bool   `json:"isSecret"`
	IsRestartRequired bool   `json:"isRestartRequired"`
	HasValue          bool   `json:"hasValue"`
	Value             string `json:"value"`
	MaskedValue       string `json:"maskedValue"`
	Source            string `json:"source"`
	UpdatedAt         string `json:"updatedAt"`
	UpdatedBySubject  string `json:"updatedBySubject"`
}

type RuntimeSettingGroupView struct {
	Name        string                    `json:"name"`
	Title       string                    `json:"title"`
	Description string                    `json:"description"`
	Fields      []RuntimeSettingFieldView `json:"fields"`
}

type SettingAuditRecord struct {
	ID              int64  `json:"id"`
	GroupName       string `json:"groupName"`
	SettingKey      string `json:"settingKey"`
	Action          string `json:"action"`
	OldValueMasked  string `json:"oldValueMasked"`
	NewValueMasked  string `json:"newValueMasked"`
	OperatorSubject string `json:"operatorSubject"`
	RequestID       string `json:"requestId"`
	CreatedAt       string `json:"createdAt"`
}

type SettingAuditFilter struct {
	GroupName string
	Action    string
	Page      int
	PageSize  int
}

type SettingAuditList struct {
	Items    []SettingAuditRecord `json:"items"`
	Total    int                  `json:"total"`
	Page     int                  `json:"page"`
	PageSize int                  `json:"pageSize"`
}

type GroupTestResult struct {
	OK        bool   `json:"ok"`
	LatencyMS int64  `json:"latencyMs"`
	Message   string `json:"message"`
}

type runtimeSettingRecord struct {
	Key               string
	GroupName         string
	ValueText         string
	ValueCiphertext   string
	ValueType         string
	IsSecret          bool
	IsRestartRequired bool
	Description       string
	UpdatedBySubject  string
	UpdatedAt         string
}

type settingAuditRecord struct {
	GroupName       string
	SettingKey      string
	Action          string
	OldValueMasked  string
	NewValueMasked  string
	OperatorSubject string
	RequestID       string
	CreatedAt       string
}
