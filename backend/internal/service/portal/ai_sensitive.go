package portal

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
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

type AISensitiveStatus struct {
	DBEnabled                 bool       `json:"dbEnabled"`
	DetectRuleCount           int        `json:"detectRuleCount"`
	ReplaceRuleCount          int        `json:"replaceRuleCount"`
	AuditRecordCount          int        `json:"auditRecordCount"`
	SystemDenyEnabled         bool       `json:"systemDenyEnabled"`
	SystemDictionaryWordCount int        `json:"systemDictionaryWordCount"`
	SystemDictionaryUpdatedAt *time.Time `json:"systemDictionaryUpdatedAt,omitempty"`
	ProjectedInstanceCount    int        `json:"projectedInstanceCount"`
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
		Enabled:           status.DetectRuleCount > 0 || status.ReplaceRuleCount > 0 || status.ProjectedInstanceCount > 0 || status.SystemDenyEnabled,
		EnabledRouteCount: status.ProjectedInstanceCount,
	}, nil
}

func (s *Service) ListAISensitiveDetectRules(ctx context.Context) ([]AISensitiveDetectRule, error) {
	db, err := s.db(ctx)
	if err != nil {
		return nil, err
	}
	items, err := newPortalStore(db).listAISensitiveDetectRules(ctx)
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
	enabled := 0
	if rule.Enabled {
		enabled = 1
	}

	rule.ID, err = newPortalStore(db).saveAISensitiveDetectRule(ctx, do.PortalAISensitiveDetectRule{
		Pattern:     rule.Pattern,
		MatchType:   rule.MatchType,
		Description: trimOrNil(rule.Description),
		Priority:    rule.Priority,
		Enabled:     enabled,
	}, rule.ID)
	if err != nil {
		return nil, err
	}
	return s.getAISensitiveDetectRule(ctx, rule.ID)
}

func (s *Service) DeleteAISensitiveDetectRule(ctx context.Context, id int64) error {
	db, err := s.db(ctx)
	if err != nil {
		return err
	}
	deleted, err := newPortalStore(db).deleteAISensitiveDetectRule(ctx, id)
	if err != nil {
		return err
	}
	if !deleted {
		return fmt.Errorf("detect rule not found: %d", id)
	}
	return nil
}

func (s *Service) ListAISensitiveReplaceRules(ctx context.Context) ([]AISensitiveReplaceRule, error) {
	db, err := s.db(ctx)
	if err != nil {
		return nil, err
	}
	items, err := newPortalStore(db).listAISensitiveReplaceRules(ctx)
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
	enabled := 0
	if rule.Enabled {
		enabled = 1
	}
	restore := 0
	if rule.Restore {
		restore = 1
	}

	rule.ID, err = newPortalStore(db).saveAISensitiveReplaceRule(ctx, do.PortalAISensitiveReplaceRule{
		Pattern:      rule.Pattern,
		ReplaceType:  rule.ReplaceType,
		ReplaceValue: trimOrNil(rule.ReplaceValue),
		Restore:      restore,
		Description:  trimOrNil(rule.Description),
		Priority:     rule.Priority,
		Enabled:      enabled,
	}, rule.ID)
	if err != nil {
		return nil, err
	}
	return s.getAISensitiveReplaceRule(ctx, rule.ID)
}

func (s *Service) DeleteAISensitiveReplaceRule(ctx context.Context, id int64) error {
	db, err := s.db(ctx)
	if err != nil {
		return err
	}
	deleted, err := newPortalStore(db).deleteAISensitiveReplaceRule(ctx, id)
	if err != nil {
		return err
	}
	if !deleted {
		return fmt.Errorf("replace rule not found: %d", id)
	}
	return nil
}

func (s *Service) ListAISensitiveAudits(ctx context.Context, query AISensitiveAuditQuery) ([]AISensitiveBlockAudit, error) {
	db, err := s.db(ctx)
	if err != nil {
		return nil, err
	}
	items, err := newPortalStore(db).listAISensitiveAudits(ctx, query)
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
	item, err := newPortalStore(db).getAISensitiveSystemConfig(ctx)
	if err != nil {
		return nil, err
	}
	if item == nil {
		return &AISensitiveSystemConfig{}, nil
	}
	return aiSensitiveSystemConfigFromEntity(*item), nil
}

func (s *Service) SaveAISensitiveSystemConfig(ctx context.Context, config AISensitiveSystemConfig) (*AISensitiveSystemConfig, error) {
	db, err := s.db(ctx)
	if err != nil {
		return nil, err
	}
	enabledInt := 0
	if config.SystemDenyEnabled {
		enabledInt = 1
	}
	if err := newPortalStore(db).saveAISensitiveSystemConfig(ctx, do.PortalAISensitiveSystemConfig{
		SystemDenyEnabled: enabledInt,
		DictionaryText:    config.DictionaryText,
		UpdatedBy:         nullIfEmpty(config.UpdatedBy),
	}); err != nil {
		return nil, err
	}
	return s.GetAISensitiveSystemConfig(ctx)
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
		DetectRuleCount:           newPortalStore(db).countTable(ctx, dao.PortalAISensitiveDetectRule.Name),
		ReplaceRuleCount:          newPortalStore(db).countTable(ctx, dao.PortalAISensitiveReplaceRule.Name),
		AuditRecordCount:          newPortalStore(db).countTable(ctx, dao.PortalAISensitiveBlockAudit.Name),
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
	payload := map[string]any{
		"name":         "default",
		"detectRules":  detectRules,
		"replaceRules": replaceRules,
		"systemConfig": config,
		"projectedAt":  time.Now().UTC().Format(time.RFC3339),
	}
	if _, err := s.k8sClient.UpsertResource(ctx, "ai-sensitive-projections", "default", payload); err != nil {
		return nil, err
	}
	return s.GetAISensitiveStatus(ctx)
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
		enabled   int
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
	item.Enabled = enabled > 0
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
		restoreInt   int
		enabledInt   int
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
		&restoreInt,
		&desc,
		&item.Priority,
		&enabledInt,
		&createdAt,
		&updatedAt,
	); err != nil {
		return AISensitiveReplaceRule{}, err
	}
	item.ReplaceValue = replaceValue.String
	item.Restore = restoreInt > 0
	item.Enabled = enabledInt > 0
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
