package portal

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/wooveep/aigateway-console/backend/internal/dao"
	"github.com/wooveep/aigateway-console/backend/internal/model/do"
	"github.com/wooveep/aigateway-console/backend/internal/model/entity"
)

type AISensitiveMenuState struct {
	Enabled           bool `json:"enabled"`
	EnabledRouteCount int  `json:"enabledRouteCount"`
}

const aiSensitivePluginName = "ai-data-masking"

type AISensitiveDetectRule struct {
	ID          int64      `json:"id,omitempty"`
	Pattern     string     `json:"pattern"`
	MatchType   string     `json:"matchType"`
	Description string     `json:"description,omitempty"`
	Priority    int        `json:"priority,omitempty"`
	Enabled     bool       `json:"enabled"`
	CreatedAt   *time.Time `json:"createdAt,omitempty"`
	UpdatedAt   *time.Time `json:"updatedAt,omitempty"`
}

type AISensitiveReplaceRule struct {
	ID           int64      `json:"id,omitempty"`
	Pattern      string     `json:"pattern"`
	ReplaceType  string     `json:"replaceType"`
	ReplaceValue string     `json:"replaceValue,omitempty"`
	Restore      bool       `json:"restore,omitempty"`
	Description  string     `json:"description,omitempty"`
	Priority     int        `json:"priority,omitempty"`
	Enabled      bool       `json:"enabled"`
	CreatedAt    *time.Time `json:"createdAt,omitempty"`
	UpdatedAt    *time.Time `json:"updatedAt,omitempty"`
}

type AISensitiveBlockAudit struct {
	ID                int64                      `json:"id"`
	RequestID         string                     `json:"requestId,omitempty"`
	RouteName         string                     `json:"routeName,omitempty"`
	ConsumerName      string                     `json:"consumerName,omitempty"`
	DisplayName       string                     `json:"displayName,omitempty"`
	BlockedAt         *time.Time                 `json:"blockedAt,omitempty"`
	BlockedBy         string                     `json:"blockedBy,omitempty"`
	RequestPhase      string                     `json:"requestPhase,omitempty"`
	BlockedReasonJSON string                     `json:"blockedReasonJson,omitempty"`
	GuardCode         *int                       `json:"guardCode,omitempty"`
	BlockedDetails    []AISensitiveBlockedDetail `json:"blockedDetails,omitempty"`
	MatchType         string                     `json:"matchType,omitempty"`
	MatchedRule       string                     `json:"matchedRule,omitempty"`
	MatchedExcerpt    string                     `json:"matchedExcerpt,omitempty"`
	ProviderID        *int64                     `json:"providerId,omitempty"`
	CostUSD           string                     `json:"costUsd,omitempty"`
}

type AISensitiveBlockedDetail struct {
	Type       string `json:"type,omitempty"`
	Level      string `json:"level,omitempty"`
	Suggestion string `json:"suggestion,omitempty"`
}

type AISensitiveSystemConfig struct {
	SystemDenyEnabled bool       `json:"systemDenyEnabled"`
	DictionaryText    string     `json:"dictionaryText"`
	UpdatedBy         string     `json:"updatedBy,omitempty"`
	UpdatedAt         *time.Time `json:"updatedAt,omitempty"`
}

type AISensitiveRuntimeAuditSink struct {
	ServiceName string `json:"serviceName,omitempty"`
	Namespace   string `json:"namespace,omitempty"`
	Port        int    `json:"port,omitempty"`
	Path        string `json:"path,omitempty"`
	TimeoutMs   int    `json:"timeoutMs,omitempty"`
}

type AISensitiveRuntimeConfig struct {
	DenyOpenai      bool                        `json:"denyOpenai"`
	DenyJsonpath    []string                    `json:"denyJsonpath,omitempty"`
	DenyRaw         bool                        `json:"denyRaw"`
	DenyCode        int                         `json:"denyCode"`
	DenyMessage     string                      `json:"denyMessage"`
	DenyRawMessage  string                      `json:"denyRawMessage"`
	DenyContentType string                      `json:"denyContentType"`
	AuditSink       AISensitiveRuntimeAuditSink `json:"auditSink,omitempty"`
}

type AISensitiveStatus struct {
	DBEnabled                 bool       `json:"dbEnabled"`
	DetectRuleCount           int        `json:"detectRuleCount"`
	ReplaceRuleCount          int        `json:"replaceRuleCount"`
	AuditRecordCount          int        `json:"auditRecordCount"`
	SystemDenyEnabled         bool       `json:"systemDenyEnabled"`
	SystemDictionaryWordCount int        `json:"systemDictionaryWordCount"`
	SystemDictionaryUpdatedAt *time.Time `json:"systemDictionaryUpdatedAt,omitempty"`
	ProjectedInstanceCount    int        `json:"projectedInstanceCount"`
	EnabledRouteCount         int        `json:"enabledRouteCount"`
	EnabledRoutes             []string   `json:"enabledRoutes,omitempty"`
	LastReconciledAt          *time.Time `json:"lastReconciledAt,omitempty"`
	LastMigratedAt            *time.Time `json:"lastMigratedAt,omitempty"`
	LastError                 string     `json:"lastError,omitempty"`
}

type AISensitiveAuditQuery struct {
	ConsumerName string
	DisplayName  string
	RouteName    string
	MatchType    string
	StartTime    string
	EndTime      string
	Limit        int
}

func (s *Service) GetAISensitiveMenuState(ctx context.Context) (*AISensitiveMenuState, error) {
	status, err := s.GetAISensitiveStatus(ctx)
	if err != nil {
		return nil, err
	}
	return &AISensitiveMenuState{
		Enabled:           status.DetectRuleCount > 0 || status.ReplaceRuleCount > 0 || status.EnabledRouteCount > 0 || status.SystemDenyEnabled,
		EnabledRouteCount: status.EnabledRouteCount,
	}, nil
}

func (s *Service) ListAISensitiveDetectRules(ctx context.Context) ([]AISensitiveDetectRule, error) {
	db, err := s.db(ctx)
	if err != nil {
		return nil, err
	}
	items, err := newPortalStore(db, s.client.Driver()).listAISensitiveDetectRules(ctx)
	if err != nil {
		return nil, err
	}
	records := make([]AISensitiveDetectRule, 0, len(items))
	for _, item := range items {
		records = append(records, aiSensitiveDetectRuleFromEntity(item))
	}
	return records, nil
}

func (s *Service) SaveAISensitiveDetectRule(ctx context.Context, rule AISensitiveDetectRule) (*AISensitiveDetectRule, error) {
	db, err := s.db(ctx)
	if err != nil {
		return nil, err
	}
	rule.Pattern = strings.TrimSpace(rule.Pattern)
	rule.MatchType = strings.ToLower(strings.TrimSpace(rule.MatchType))
	if rule.Pattern == "" {
		return nil, errors.New("pattern cannot be blank")
	}
	if rule.MatchType == "" {
		rule.MatchType = "contains"
	}
	if rule.ID == 0 && !rule.Enabled {
		rule.Enabled = true
	}

	rule.ID, err = newPortalStore(db, s.client.Driver()).saveAISensitiveDetectRule(ctx, do.PortalAISensitiveDetectRule{
		Pattern:     rule.Pattern,
		MatchType:   rule.MatchType,
		Description: trimOrNil(rule.Description),
		Priority:    rule.Priority,
		Enabled:     rule.Enabled,
	}, rule.ID)
	if err != nil {
		return nil, err
	}
	if err := s.refreshAISensitiveProjection(ctx); err != nil {
		return nil, err
	}
	return s.getAISensitiveDetectRule(ctx, rule.ID)
}

func (s *Service) DeleteAISensitiveDetectRule(ctx context.Context, id int64) error {
	db, err := s.db(ctx)
	if err != nil {
		return err
	}
	deleted, err := newPortalStore(db, s.client.Driver()).deleteAISensitiveDetectRule(ctx, id)
	if err != nil {
		return err
	}
	if !deleted {
		return fmt.Errorf("detect rule not found: %d", id)
	}
	return s.refreshAISensitiveProjection(ctx)
}

func (s *Service) ListAISensitiveReplaceRules(ctx context.Context) ([]AISensitiveReplaceRule, error) {
	db, err := s.db(ctx)
	if err != nil {
		return nil, err
	}
	items, err := newPortalStore(db, s.client.Driver()).listAISensitiveReplaceRules(ctx)
	if err != nil {
		return nil, err
	}
	records := make([]AISensitiveReplaceRule, 0, len(items))
	for _, item := range items {
		records = append(records, aiSensitiveReplaceRuleFromEntity(item))
	}
	return records, nil
}

func (s *Service) SaveAISensitiveReplaceRule(ctx context.Context, rule AISensitiveReplaceRule) (*AISensitiveReplaceRule, error) {
	db, err := s.db(ctx)
	if err != nil {
		return nil, err
	}
	rule.Pattern = strings.TrimSpace(rule.Pattern)
	rule.ReplaceType = strings.ToLower(strings.TrimSpace(rule.ReplaceType))
	if rule.Pattern == "" {
		return nil, errors.New("pattern cannot be blank")
	}
	if rule.ReplaceType == "" {
		rule.ReplaceType = "replace"
	}
	if rule.ID == 0 && !rule.Enabled {
		rule.Enabled = true
	}

	rule.ID, err = newPortalStore(db, s.client.Driver()).saveAISensitiveReplaceRule(ctx, do.PortalAISensitiveReplaceRule{
		Pattern:      rule.Pattern,
		ReplaceType:  rule.ReplaceType,
		ReplaceValue: trimOrNil(rule.ReplaceValue),
		Restore:      rule.Restore,
		Description:  trimOrNil(rule.Description),
		Priority:     rule.Priority,
		Enabled:      rule.Enabled,
	}, rule.ID)
	if err != nil {
		return nil, err
	}
	if err := s.refreshAISensitiveProjection(ctx); err != nil {
		return nil, err
	}
	return s.getAISensitiveReplaceRule(ctx, rule.ID)
}

func (s *Service) DeleteAISensitiveReplaceRule(ctx context.Context, id int64) error {
	db, err := s.db(ctx)
	if err != nil {
		return err
	}
	deleted, err := newPortalStore(db, s.client.Driver()).deleteAISensitiveReplaceRule(ctx, id)
	if err != nil {
		return err
	}
	if !deleted {
		return fmt.Errorf("replace rule not found: %d", id)
	}
	return s.refreshAISensitiveProjection(ctx)
}

func (s *Service) ListAISensitiveAudits(ctx context.Context, query AISensitiveAuditQuery) ([]AISensitiveBlockAudit, error) {
	db, err := s.db(ctx)
	if err != nil {
		return nil, err
	}
	items, err := newPortalStore(db, s.client.Driver()).listAISensitiveAudits(ctx, query)
	if err != nil {
		return nil, err
	}
	records := make([]AISensitiveBlockAudit, 0, len(items))
	for _, item := range items {
		records = append(records, aiSensitiveAuditFromEntity(item))
	}
	return records, nil
}

func (s *Service) GetAISensitiveSystemConfig(ctx context.Context) (*AISensitiveSystemConfig, error) {
	db, err := s.db(ctx)
	if err != nil {
		return nil, err
	}
	item, err := newPortalStore(db, s.client.Driver()).getAISensitiveSystemConfig(ctx)
	if err != nil {
		return nil, err
	}
	defaultDictionary := DefaultAISensitiveDictionaryText()
	if item == nil {
		return &AISensitiveSystemConfig{
			DictionaryText: defaultDictionary,
		}, nil
	}
	record := aiSensitiveSystemConfigFromEntity(*item)
	if strings.TrimSpace(record.DictionaryText) == "" {
		record.DictionaryText = defaultDictionary
	}
	return record, nil
}

func (s *Service) SaveAISensitiveSystemConfig(ctx context.Context, config AISensitiveSystemConfig) (*AISensitiveSystemConfig, error) {
	db, err := s.db(ctx)
	if err != nil {
		return nil, err
	}
	if err := newPortalStore(db, s.client.Driver()).saveAISensitiveSystemConfig(ctx, do.PortalAISensitiveSystemConfig{
		SystemDenyEnabled: config.SystemDenyEnabled,
		DictionaryText:    config.DictionaryText,
		UpdatedBy:         nullIfEmpty(config.UpdatedBy),
	}); err != nil {
		return nil, err
	}
	if err := s.refreshAISensitiveProjection(ctx); err != nil {
		return nil, err
	}
	return s.GetAISensitiveSystemConfig(ctx)
}

func (s *Service) GetAISensitiveRuntimeConfig(ctx context.Context) (*AISensitiveRuntimeConfig, error) {
	config := defaultAISensitiveRuntimeConfig()
	if s.k8sClient == nil {
		return &config, nil
	}
	item, err := s.k8sClient.GetResource(ctx, "ai-sensitive-projections", "default")
	if err != nil {
		return &config, nil
	}
	payload := mapStringAny(item["runtimeConfig"])
	if len(payload) == 0 {
		return &config, nil
	}
	return normalizeAISensitiveRuntimeConfig(payload), nil
}

func (s *Service) SaveAISensitiveRuntimeConfig(ctx context.Context, config AISensitiveRuntimeConfig) (*AISensitiveRuntimeConfig, error) {
	if s.k8sClient == nil {
		return nil, errors.New("ai sensitive runtime config requires k8s client")
	}
	normalized := normalizeAISensitiveRuntimeConfig(map[string]any{
		"denyOpenai":      config.DenyOpenai,
		"denyJsonpath":    config.DenyJsonpath,
		"denyRaw":         config.DenyRaw,
		"denyCode":        config.DenyCode,
		"denyMessage":     config.DenyMessage,
		"denyRawMessage":  config.DenyRawMessage,
		"denyContentType": config.DenyContentType,
		"auditSink": map[string]any{
			"serviceName": config.AuditSink.ServiceName,
			"namespace":   config.AuditSink.Namespace,
			"port":        config.AuditSink.Port,
			"path":        config.AuditSink.Path,
			"timeoutMs":   config.AuditSink.TimeoutMs,
		},
	})
	if err := s.upsertAISensitiveProjection(ctx, *normalized); err != nil {
		return nil, err
	}
	return s.GetAISensitiveRuntimeConfig(ctx)
}

func (s *Service) GetAISensitiveStatus(ctx context.Context) (*AISensitiveStatus, error) {
	db, err := s.db(ctx)
	if err != nil {
		return nil, err
	}
	config, err := s.GetAISensitiveSystemConfig(ctx)
	if err != nil {
		return nil, err
	}
	status := &AISensitiveStatus{
		DBEnabled:                 true,
		DetectRuleCount:           newPortalStore(db, s.client.Driver()).countTable(ctx, dao.PortalAISensitiveDetectRule.Name),
		ReplaceRuleCount:          newPortalStore(db, s.client.Driver()).countTable(ctx, dao.PortalAISensitiveReplaceRule.Name),
		AuditRecordCount:          newPortalStore(db, s.client.Driver()).countTable(ctx, dao.PortalAISensitiveBlockAudit.Name),
		SystemDenyEnabled:         config.SystemDenyEnabled,
		SystemDictionaryWordCount: countDictionaryLines(config.DictionaryText),
		SystemDictionaryUpdatedAt: config.UpdatedAt,
		LastMigratedAt:            config.UpdatedAt,
	}

	if s.k8sClient != nil {
		items, err := s.k8sClient.ListResources(ctx, "ai-sensitive-projections")
		if err == nil {
			status.ProjectedInstanceCount = len(items)
		}
		item, err := s.k8sClient.GetResource(ctx, "ai-sensitive-projections", "default")
		if err == nil {
			status.LastReconciledAt = parseRFC3339Pointer(fmt.Sprint(item["projectedAt"]))
			status.LastError = strings.TrimSpace(fmt.Sprint(item["lastError"]))
		}
		boundRoutes, err := s.listAISensitiveBoundRoutes(ctx)
		if err == nil {
			status.EnabledRoutes = boundRoutes
			status.EnabledRouteCount = len(boundRoutes)
		}
	}
	return status, nil
}

func (s *Service) ReconcileAISensitive(ctx context.Context) (*AISensitiveStatus, error) {
	if _, err := s.db(ctx); err != nil {
		return nil, err
	}
	if s.k8sClient == nil {
		return nil, errors.New("ai sensitive projection requires k8s client")
	}
	runtimeConfig, err := s.GetAISensitiveRuntimeConfig(ctx)
	if err != nil {
		return nil, err
	}
	if err := s.upsertAISensitiveProjection(ctx, *runtimeConfig); err != nil {
		return nil, err
	}
	return s.GetAISensitiveStatus(ctx)
}

func (s *Service) buildAISensitiveProjectionPayload(ctx context.Context, runtimeConfig AISensitiveRuntimeConfig) (map[string]any, error) {
	detectRules, err := s.ListAISensitiveDetectRules(ctx)
	if err != nil {
		return nil, err
	}
	replaceRules, err := s.ListAISensitiveReplaceRules(ctx)
	if err != nil {
		return nil, err
	}
	config, err := s.GetAISensitiveSystemConfig(ctx)
	if err != nil {
		return nil, err
	}
	return map[string]any{
		"name":         "default",
		"detectRules":  detectRules,
		"replaceRules": replaceRules,
		"systemConfig": map[string]any{
			"systemDenyEnabled": config.SystemDenyEnabled,
			"updatedBy":         config.UpdatedBy,
			"updatedAt":         config.UpdatedAt,
		},
		"runtimeConfig": runtimeConfig,
		"projectedAt":   time.Now().UTC().Format(time.RFC3339),
	}, nil
}

func (s *Service) refreshAISensitiveProjection(ctx context.Context) error {
	if s.k8sClient == nil {
		return nil
	}
	runtimeConfig, err := s.GetAISensitiveRuntimeConfig(ctx)
	if err != nil {
		return err
	}
	return s.upsertAISensitiveProjection(ctx, *runtimeConfig)
}

func (s *Service) upsertAISensitiveProjection(ctx context.Context, runtimeConfig AISensitiveRuntimeConfig) error {
	payload, err := s.buildAISensitiveProjectionPayload(ctx, runtimeConfig)
	if err != nil {
		return err
	}
	_, err = s.k8sClient.UpsertResource(ctx, "ai-sensitive-projections", "default", payload)
	return err
}

func (s *Service) listAISensitiveBoundRoutes(ctx context.Context) ([]string, error) {
	if s.k8sClient == nil {
		return []string{}, nil
	}
	rawRoutes, err := s.k8sClient.ListResources(ctx, "ai-routes")
	if err != nil {
		return nil, err
	}
	items := make([]string, 0, len(rawRoutes))
	for _, route := range rawRoutes {
		routeName := strings.TrimSpace(fmt.Sprint(route["name"]))
		if routeName == "" {
			continue
		}
		if s.isAISensitiveRouteBound(ctx, routeName) {
			items = append(items, routeName)
		}
	}
	sort.Strings(items)
	return items, nil
}

func (s *Service) isAISensitiveRouteBound(ctx context.Context, routeName string) bool {
	targets := []string{
		fmt.Sprintf("ai-route-%s.internal", routeName),
		fmt.Sprintf("ai-route-%s.internal-internal", routeName),
		fmt.Sprintf("ai-route-%s.fallback.internal", routeName),
		fmt.Sprintf("ai-route-%s.fallback.internal-internal", routeName),
	}
	for _, target := range targets {
		pluginInstances, err := s.k8sClient.ListResources(ctx, fmt.Sprintf("route-plugin-instances:%s", target))
		if err != nil {
			continue
		}
		for _, item := range pluginInstances {
			pluginName := strings.TrimSpace(fmt.Sprint(item["pluginName"]))
			if pluginName == "" {
				pluginName = strings.TrimSpace(fmt.Sprint(item["name"]))
			}
			if pluginName != aiSensitivePluginName {
				continue
			}
			if readBool(item["enabled"], true) || readBool(item["runtimeEnabled"], false) {
				return true
			}
		}
	}
	return false
}

func (s *Service) getAISensitiveDetectRule(ctx context.Context, id int64) (*AISensitiveDetectRule, error) {
	db, err := s.db(ctx)
	if err != nil {
		return nil, err
	}
	row := db.QueryRowContext(ctx, `
		SELECT id, pattern, match_type, description, priority, enabled, created_at, updated_at
		FROM `+dao.PortalAISensitiveDetectRule.Name+`
		WHERE id = ?`, id)
	item, err := scanAISensitiveDetectRule(row)
	if err != nil {
		return nil, err
	}
	return &item, nil
}

func (s *Service) getAISensitiveReplaceRule(ctx context.Context, id int64) (*AISensitiveReplaceRule, error) {
	db, err := s.db(ctx)
	if err != nil {
		return nil, err
	}
	row := db.QueryRowContext(ctx, `
		SELECT id, pattern, replace_type, replace_value, restore, description, priority, enabled, created_at, updated_at
		FROM `+dao.PortalAISensitiveReplaceRule.Name+`
		WHERE id = ?`, id)
	item, err := scanAISensitiveReplaceRule(row)
	if err != nil {
		return nil, err
	}
	return &item, nil
}

func scanAISensitiveDetectRule(scanner interface{ Scan(...any) error }) (AISensitiveDetectRule, error) {
	var (
		item      AISensitiveDetectRule
		enabled   bool
		desc      sql.NullString
		createdAt sql.NullTime
		updatedAt sql.NullTime
	)
	if err := scanner.Scan(
		&item.ID,
		&item.Pattern,
		&item.MatchType,
		&desc,
		&item.Priority,
		&enabled,
		&createdAt,
		&updatedAt,
	); err != nil {
		return AISensitiveDetectRule{}, err
	}
	item.Enabled = enabled
	item.Description = desc.String
	if createdAt.Valid {
		item.CreatedAt = &createdAt.Time
	}
	if updatedAt.Valid {
		item.UpdatedAt = &updatedAt.Time
	}
	return item, nil
}

func aiSensitiveDetectRuleFromEntity(item entity.PortalAISensitiveDetectRule) AISensitiveDetectRule {
	record := AISensitiveDetectRule{
		ID:          item.Id,
		Pattern:     item.Pattern,
		MatchType:   item.MatchType,
		Description: item.Description,
		Priority:    item.Priority,
		Enabled:     item.Enabled > 0,
	}
	if item.CreatedAt != nil {
		value := item.CreatedAt.Time
		record.CreatedAt = &value
	}
	if item.UpdatedAt != nil {
		value := item.UpdatedAt.Time
		record.UpdatedAt = &value
	}
	return record
}

func aiSensitiveReplaceRuleFromEntity(item entity.PortalAISensitiveReplaceRule) AISensitiveReplaceRule {
	record := AISensitiveReplaceRule{
		ID:           item.Id,
		Pattern:      item.Pattern,
		ReplaceType:  item.ReplaceType,
		ReplaceValue: item.ReplaceValue,
		Restore:      item.Restore > 0,
		Description:  item.Description,
		Priority:     item.Priority,
		Enabled:      item.Enabled > 0,
	}
	if item.CreatedAt != nil {
		value := item.CreatedAt.Time
		record.CreatedAt = &value
	}
	if item.UpdatedAt != nil {
		value := item.UpdatedAt.Time
		record.UpdatedAt = &value
	}
	return record
}

func aiSensitiveAuditFromEntity(item entity.PortalAISensitiveBlockAudit) AISensitiveBlockAudit {
	record := AISensitiveBlockAudit{
		ID:                item.Id,
		RequestID:         item.RequestId,
		RouteName:         item.RouteName,
		ConsumerName:      item.ConsumerName,
		DisplayName:       item.DisplayName,
		BlockedBy:         item.BlockedBy,
		RequestPhase:      item.RequestPhase,
		BlockedReasonJSON: item.BlockedReasonJson,
		MatchType:         item.MatchType,
		MatchedRule:       item.MatchedRule,
		MatchedExcerpt:    item.MatchedExcerpt,
		CostUSD:           item.CostUsd,
	}
	if payload, ok := parseAISensitiveBlockedReason(item.BlockedReasonJson); ok {
		if payload.RequestID != "" {
			record.RequestID = payload.RequestID
		}
		if payload.GuardCode != nil {
			record.GuardCode = payload.GuardCode
		}
		if len(payload.BlockedDetails) > 0 {
			record.BlockedDetails = payload.BlockedDetails
		}
	}
	if item.BlockedAt != nil {
		value := item.BlockedAt.Time
		record.BlockedAt = &value
	}
	if item.ProviderId > 0 {
		value := item.ProviderId
		record.ProviderID = &value
	}
	return record
}

type aiSensitiveBlockedReasonPayload struct {
	RequestID      string                     `json:"requestId"`
	GuardCode      *int                       `json:"guardCode"`
	BlockedDetails []AISensitiveBlockedDetail `json:"blockedDetails"`
}

func parseAISensitiveBlockedReason(raw string) (aiSensitiveBlockedReasonPayload, bool) {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return aiSensitiveBlockedReasonPayload{}, false
	}
	var payload aiSensitiveBlockedReasonPayload
	if err := json.Unmarshal([]byte(trimmed), &payload); err != nil {
		return aiSensitiveBlockedReasonPayload{}, false
	}
	for index := range payload.BlockedDetails {
		payload.BlockedDetails[index].Type = strings.TrimSpace(payload.BlockedDetails[index].Type)
		payload.BlockedDetails[index].Level = strings.TrimSpace(payload.BlockedDetails[index].Level)
		payload.BlockedDetails[index].Suggestion = strings.TrimSpace(payload.BlockedDetails[index].Suggestion)
	}
	if payload.GuardCode == nil && payload.RequestID == "" && len(payload.BlockedDetails) == 0 {
		return aiSensitiveBlockedReasonPayload{}, false
	}
	return payload, true
}

func defaultAISensitiveRuntimeConfig() AISensitiveRuntimeConfig {
	return AISensitiveRuntimeConfig{
		DenyOpenai:      true,
		DenyJsonpath:    []string{"$.messages[*].content"},
		DenyRaw:         false,
		DenyCode:        200,
		DenyMessage:     "提问或回答中包含敏感词，已被屏蔽",
		DenyRawMessage:  "{\"errmsg\":\"提问或回答中包含敏感词，已被屏蔽\"}",
		DenyContentType: "application/json",
		AuditSink: AISensitiveRuntimeAuditSink{
			TimeoutMs: 2000,
		},
	}
}

func normalizeAISensitiveRuntimeConfig(payload map[string]any) *AISensitiveRuntimeConfig {
	config := defaultAISensitiveRuntimeConfig()
	if len(payload) == 0 {
		return &config
	}
	config.DenyOpenai = readBool(firstNonNil(payload["denyOpenai"], payload["deny_openai"]), config.DenyOpenai)
	config.DenyJsonpath = uniqueTrimmedStrings(firstNonNil(payload["denyJsonpath"], payload["deny_jsonpath"]))
	if len(config.DenyJsonpath) == 0 {
		config.DenyJsonpath = defaultAISensitiveRuntimeConfig().DenyJsonpath
	}
	config.DenyRaw = readBool(firstNonNil(payload["denyRaw"], payload["deny_raw"]), config.DenyRaw)
	config.DenyCode = readInt(firstNonNil(payload["denyCode"], payload["deny_code"]), config.DenyCode)
	config.DenyMessage = firstNonEmptyString(firstNonNil(payload["denyMessage"], payload["deny_message"]), config.DenyMessage)
	config.DenyRawMessage = firstNonEmptyString(firstNonNil(payload["denyRawMessage"], payload["deny_raw_message"]), config.DenyRawMessage)
	config.DenyContentType = firstNonEmptyString(firstNonNil(payload["denyContentType"], payload["deny_content_type"]), config.DenyContentType)
	auditSink := mapStringAny(firstNonNil(payload["auditSink"], payload["audit_sink"]))
	config.AuditSink = AISensitiveRuntimeAuditSink{
		ServiceName: nullableString(firstNonNil(auditSink["serviceName"], auditSink["service_name"])),
		Namespace:   nullableString(auditSink["namespace"]),
		Port:        readInt(auditSink["port"], 0),
		Path:        nullableString(auditSink["path"]),
		TimeoutMs:   readInt(firstNonNil(auditSink["timeoutMs"], auditSink["timeout_ms"]), 2000),
	}
	if config.AuditSink.TimeoutMs <= 0 {
		config.AuditSink.TimeoutMs = 2000
	}
	return &config
}

func mapStringAny(value any) map[string]any {
	if value == nil {
		return map[string]any{}
	}
	switch typed := value.(type) {
	case map[string]any:
		return typed
	default:
		return map[string]any{}
	}
}

func uniqueTrimmedStrings(value any) []string {
	items := make([]string, 0)
	seen := map[string]struct{}{}
	switch typed := value.(type) {
	case []string:
		for _, item := range typed {
			trimmed := strings.TrimSpace(item)
			if trimmed == "" {
				continue
			}
			if _, ok := seen[trimmed]; ok {
				continue
			}
			seen[trimmed] = struct{}{}
			items = append(items, trimmed)
		}
	case []any:
		for _, item := range typed {
			trimmed := strings.TrimSpace(fmt.Sprint(item))
			if trimmed == "" {
				continue
			}
			if _, ok := seen[trimmed]; ok {
				continue
			}
			seen[trimmed] = struct{}{}
			items = append(items, trimmed)
		}
	case string:
		for _, item := range strings.Split(typed, "\n") {
			trimmed := strings.TrimSpace(item)
			if trimmed == "" {
				continue
			}
			if _, ok := seen[trimmed]; ok {
				continue
			}
			seen[trimmed] = struct{}{}
			items = append(items, trimmed)
		}
	}
	return items
}

func readBool(value any, fallback bool) bool {
	switch typed := value.(type) {
	case bool:
		return typed
	case string:
		trimmed := strings.TrimSpace(strings.ToLower(typed))
		if trimmed == "true" || trimmed == "1" {
			return true
		}
		if trimmed == "false" || trimmed == "0" {
			return false
		}
	case int:
		return typed != 0
	case int64:
		return typed != 0
	case float64:
		return typed != 0
	}
	return fallback
}

func readInt(value any, fallback int) int {
	switch typed := value.(type) {
	case int:
		return typed
	case int64:
		return int(typed)
	case float64:
		return int(typed)
	case string:
		parsed, err := strconv.Atoi(strings.TrimSpace(typed))
		if err == nil {
			return parsed
		}
	}
	return fallback
}

func firstNonNil(values ...any) any {
	for _, value := range values {
		if value != nil {
			return value
		}
	}
	return nil
}

func firstNonEmptyString(value any, fallback string) string {
	trimmed := strings.TrimSpace(fmt.Sprint(value))
	if trimmed == "" || trimmed == "<nil>" {
		return fallback
	}
	return trimmed
}

func nullableString(value any) string {
	trimmed := strings.TrimSpace(fmt.Sprint(value))
	if trimmed == "" || trimmed == "<nil>" {
		return ""
	}
	return trimmed
}

func aiSensitiveSystemConfigFromEntity(item entity.PortalAISensitiveSystemConfig) *AISensitiveSystemConfig {
	record := &AISensitiveSystemConfig{
		SystemDenyEnabled: item.SystemDenyEnabled > 0,
		DictionaryText:    item.DictionaryText,
		UpdatedBy:         item.UpdatedBy,
	}
	if item.UpdatedAt != nil {
		value := item.UpdatedAt.Time
		record.UpdatedAt = &value
	}
	return record
}

func scanAISensitiveReplaceRule(scanner interface{ Scan(...any) error }) (AISensitiveReplaceRule, error) {
	var (
		item         AISensitiveReplaceRule
		restore      bool
		enabled      bool
		replaceValue sql.NullString
		desc         sql.NullString
		createdAt    sql.NullTime
		updatedAt    sql.NullTime
	)
	if err := scanner.Scan(
		&item.ID,
		&item.Pattern,
		&item.ReplaceType,
		&replaceValue,
		&restore,
		&desc,
		&item.Priority,
		&enabled,
		&createdAt,
		&updatedAt,
	); err != nil {
		return AISensitiveReplaceRule{}, err
	}
	item.ReplaceValue = replaceValue.String
	item.Restore = restore
	item.Enabled = enabled
	item.Description = desc.String
	if createdAt.Valid {
		item.CreatedAt = &createdAt.Time
	}
	if updatedAt.Valid {
		item.UpdatedAt = &updatedAt.Time
	}
	return item, nil
}

func scanAISensitiveAudit(scanner interface{ Scan(...any) error }) (AISensitiveBlockAudit, error) {
	var (
		item              AISensitiveBlockAudit
		requestID         sql.NullString
		routeName         sql.NullString
		consumerName      sql.NullString
		displayName       sql.NullString
		blockedAt         sql.NullTime
		blockedBy         sql.NullString
		requestPhase      sql.NullString
		blockedReasonJSON sql.NullString
		matchType         sql.NullString
		matchedRule       sql.NullString
		matchedExcerpt    sql.NullString
		providerID        sql.NullInt64
		costUSD           sql.NullString
	)
	if err := scanner.Scan(
		&item.ID,
		&requestID,
		&routeName,
		&consumerName,
		&displayName,
		&blockedAt,
		&blockedBy,
		&requestPhase,
		&blockedReasonJSON,
		&matchType,
		&matchedRule,
		&matchedExcerpt,
		&providerID,
		&costUSD,
	); err != nil {
		return AISensitiveBlockAudit{}, err
	}
	item.RequestID = requestID.String
	item.RouteName = routeName.String
	item.ConsumerName = consumerName.String
	item.DisplayName = displayName.String
	item.BlockedBy = blockedBy.String
	item.RequestPhase = requestPhase.String
	item.BlockedReasonJSON = blockedReasonJSON.String
	item.MatchType = matchType.String
	item.MatchedRule = matchedRule.String
	item.MatchedExcerpt = matchedExcerpt.String
	if blockedAt.Valid {
		item.BlockedAt = &blockedAt.Time
	}
	if providerID.Valid {
		value := providerID.Int64
		item.ProviderID = &value
	}
	item.CostUSD = costUSD.String
	return item, nil
}

func queryCount(ctx context.Context, db *sql.DB, statement string, args ...any) int {
	var count int
	_ = db.QueryRowContext(ctx, statement, args...).Scan(&count)
	return count
}

func parseRFC3339Pointer(value string) *time.Time {
	value = strings.TrimSpace(value)
	if value == "" {
		return nil
	}
	parsed, err := time.Parse(time.RFC3339, value)
	if err != nil {
		return nil
	}
	return &parsed
}

func countDictionaryLines(value string) int {
	count := 0
	for _, line := range strings.Split(value, "\n") {
		if strings.TrimSpace(line) != "" {
			count++
		}
	}
	return count
}
