package entity

import "github.com/gogf/gf/v2/os/gtime"

type PortalUser struct {
	Id           int64       `json:"id" orm:"id"`
	ConsumerName string      `json:"consumerName" orm:"consumer_name"`
	DisplayName  string      `json:"displayName" orm:"display_name"`
	Email        string      `json:"email" orm:"email"`
	PasswordHash string      `json:"passwordHash" orm:"password_hash"`
	Status       string      `json:"status" orm:"status"`
	Source       string      `json:"source" orm:"source"`
	UserLevel    string      `json:"userLevel" orm:"user_level"`
	IsDeleted    int         `json:"isDeleted" orm:"is_deleted"`
	DeletedAt    *gtime.Time `json:"deletedAt" orm:"deleted_at"`
	LastLoginAt  *gtime.Time `json:"lastLoginAt" orm:"last_login_at"`
	CreatedAt    *gtime.Time `json:"createdAt" orm:"created_at"`
	UpdatedAt    *gtime.Time `json:"updatedAt" orm:"updated_at"`
}

type PortalInviteCode struct {
	Id             int64       `json:"id" orm:"id"`
	InviteCode     string      `json:"inviteCode" orm:"invite_code"`
	Status         string      `json:"status" orm:"status"`
	ExpiresAt      *gtime.Time `json:"expiresAt" orm:"expires_at"`
	UsedByConsumer string      `json:"usedByConsumer" orm:"used_by_consumer"`
	UsedAt         *gtime.Time `json:"usedAt" orm:"used_at"`
	CreatedAt      *gtime.Time `json:"createdAt" orm:"created_at"`
	UpdatedAt      *gtime.Time `json:"updatedAt" orm:"updated_at"`
}

type OrgDepartment struct {
	Id                 int64       `json:"id" orm:"id"`
	DepartmentId       string      `json:"departmentId" orm:"department_id"`
	Name               string      `json:"name" orm:"name"`
	ParentDepartmentId string      `json:"parentDepartmentId" orm:"parent_department_id"`
	AdminConsumerName  string      `json:"adminConsumerName" orm:"admin_consumer_name"`
	Path               string      `json:"path" orm:"path"`
	Level              int         `json:"level" orm:"level"`
	SortOrder          int         `json:"sortOrder" orm:"sort_order"`
	Status             string      `json:"status" orm:"status"`
	CreatedAt          *gtime.Time `json:"createdAt" orm:"created_at"`
	UpdatedAt          *gtime.Time `json:"updatedAt" orm:"updated_at"`
}

type OrgAccountMembership struct {
	Id                 int64       `json:"id" orm:"id"`
	ConsumerName       string      `json:"consumerName" orm:"consumer_name"`
	DepartmentId       string      `json:"departmentId" orm:"department_id"`
	ParentConsumerName string      `json:"parentConsumerName" orm:"parent_consumer_name"`
	CreatedAt          *gtime.Time `json:"createdAt" orm:"created_at"`
	UpdatedAt          *gtime.Time `json:"updatedAt" orm:"updated_at"`
}

type AssetGrant struct {
	Id          int64       `json:"id" orm:"id"`
	AssetType   string      `json:"assetType" orm:"asset_type"`
	AssetId     string      `json:"assetId" orm:"asset_id"`
	SubjectType string      `json:"subjectType" orm:"subject_type"`
	SubjectId   string      `json:"subjectId" orm:"subject_id"`
	CreatedAt   *gtime.Time `json:"createdAt" orm:"created_at"`
	UpdatedAt   *gtime.Time `json:"updatedAt" orm:"updated_at"`
}

type QuotaPolicyUser struct {
	Id                    int64       `json:"id" orm:"id"`
	ConsumerName          string      `json:"consumerName" orm:"consumer_name"`
	LimitTotalMicroYuan   int64       `json:"limitTotalMicroYuan" orm:"limit_total_micro_yuan"`
	Limit5hMicroYuan      int64       `json:"limit5hMicroYuan" orm:"limit_5h_micro_yuan"`
	LimitDailyMicroYuan   int64       `json:"limitDailyMicroYuan" orm:"limit_daily_micro_yuan"`
	DailyResetMode        string      `json:"dailyResetMode" orm:"daily_reset_mode"`
	DailyResetTime        string      `json:"dailyResetTime" orm:"daily_reset_time"`
	LimitWeeklyMicroYuan  int64       `json:"limitWeeklyMicroYuan" orm:"limit_weekly_micro_yuan"`
	LimitMonthlyMicroYuan int64       `json:"limitMonthlyMicroYuan" orm:"limit_monthly_micro_yuan"`
	CostResetAt           *gtime.Time `json:"costResetAt" orm:"cost_reset_at"`
	CreatedAt             *gtime.Time `json:"createdAt" orm:"created_at"`
	UpdatedAt             *gtime.Time `json:"updatedAt" orm:"updated_at"`
}

type PortalModelAsset struct {
	Id                   int64       `json:"id" orm:"id"`
	AssetId              string      `json:"assetId" orm:"asset_id"`
	CanonicalName        string      `json:"canonicalName" orm:"canonical_name"`
	DisplayName          string      `json:"displayName" orm:"display_name"`
	Intro                string      `json:"intro" orm:"intro"`
	ModelType            string      `json:"modelType" orm:"model_type"`
	TagsJson             string      `json:"tagsJson" orm:"tags_json"`
	InputModalitiesJson  string      `json:"inputModalitiesJson" orm:"input_modalities_json"`
	OutputModalitiesJson string      `json:"outputModalitiesJson" orm:"output_modalities_json"`
	FeatureFlagsJson     string      `json:"featureFlagsJson" orm:"feature_flags_json"`
	ModalitiesJson       string      `json:"modalitiesJson" orm:"modalities_json"`
	FeaturesJson         string      `json:"featuresJson" orm:"features_json"`
	RequestKindsJson     string      `json:"requestKindsJson" orm:"request_kinds_json"`
	CreatedAt            *gtime.Time `json:"createdAt" orm:"created_at"`
	UpdatedAt            *gtime.Time `json:"updatedAt" orm:"updated_at"`
}

type PortalModelBinding struct {
	Id            int64       `json:"id" orm:"id"`
	BindingId     string      `json:"bindingId" orm:"binding_id"`
	AssetId       string      `json:"assetId" orm:"asset_id"`
	ModelId       string      `json:"modelId" orm:"model_id"`
	ProviderName  string      `json:"providerName" orm:"provider_name"`
	TargetModel   string      `json:"targetModel" orm:"target_model"`
	Protocol      string      `json:"protocol" orm:"protocol"`
	Endpoint      string      `json:"endpoint" orm:"endpoint"`
	PricingJson   string      `json:"pricingJson" orm:"pricing_json"`
	LimitsJson    string      `json:"limitsJson" orm:"limits_json"`
	Rpm           int64       `json:"rpm" orm:"rpm"`
	Tpm           int64       `json:"tpm" orm:"tpm"`
	ContextWindow int64       `json:"contextWindow" orm:"context_window"`
	Status        string      `json:"status" orm:"status"`
	PublishedAt   *gtime.Time `json:"publishedAt" orm:"published_at"`
	UnpublishedAt *gtime.Time `json:"unpublishedAt" orm:"unpublished_at"`
	CreatedAt     *gtime.Time `json:"createdAt" orm:"created_at"`
	UpdatedAt     *gtime.Time `json:"updatedAt" orm:"updated_at"`
}

type PortalModelBindingPriceVersion struct {
	VersionId     int64       `json:"versionId" orm:"version_id"`
	AssetId       string      `json:"assetId" orm:"asset_id"`
	BindingId     string      `json:"bindingId" orm:"binding_id"`
	Status        string      `json:"status" orm:"status"`
	Active        int         `json:"active" orm:"active"`
	EffectiveFrom *gtime.Time `json:"effectiveFrom" orm:"effective_from"`
	EffectiveTo   *gtime.Time `json:"effectiveTo" orm:"effective_to"`
	PricingJson   string      `json:"pricingJson" orm:"pricing_json"`
	CreatedAt     *gtime.Time `json:"createdAt" orm:"created_at"`
	UpdatedAt     *gtime.Time `json:"updatedAt" orm:"updated_at"`
}

type PortalAgentCatalog struct {
	Id                 int64       `json:"id" orm:"id"`
	AgentId            string      `json:"agentId" orm:"agent_id"`
	CanonicalName      string      `json:"canonicalName" orm:"canonical_name"`
	DisplayName        string      `json:"displayName" orm:"display_name"`
	Intro              string      `json:"intro" orm:"intro"`
	Description        string      `json:"description" orm:"description"`
	IconUrl            string      `json:"iconUrl" orm:"icon_url"`
	TagsJson           string      `json:"tagsJson" orm:"tags_json"`
	McpServerName      string      `json:"mcpServerName" orm:"mcp_server_name"`
	ToolCount          int64       `json:"toolCount" orm:"tool_count"`
	TransportTypesJson string      `json:"transportTypesJson" orm:"transport_types_json"`
	ResourceSummary    string      `json:"resourceSummary" orm:"resource_summary"`
	PromptSummary      string      `json:"promptSummary" orm:"prompt_summary"`
	Status             string      `json:"status" orm:"status"`
	PublishedAt        *gtime.Time `json:"publishedAt" orm:"published_at"`
	UnpublishedAt      *gtime.Time `json:"unpublishedAt" orm:"unpublished_at"`
	CreatedAt          *gtime.Time `json:"createdAt" orm:"created_at"`
	UpdatedAt          *gtime.Time `json:"updatedAt" orm:"updated_at"`
}

type PortalAISensitiveDetectRule struct {
	Id          int64       `json:"id" orm:"id"`
	Pattern     string      `json:"pattern" orm:"pattern"`
	MatchType   string      `json:"matchType" orm:"match_type"`
	Description string      `json:"description" orm:"description"`
	Priority    int         `json:"priority" orm:"priority"`
	Enabled     int         `json:"enabled" orm:"enabled"`
	CreatedAt   *gtime.Time `json:"createdAt" orm:"created_at"`
	UpdatedAt   *gtime.Time `json:"updatedAt" orm:"updated_at"`
}

type PortalAISensitiveReplaceRule struct {
	Id           int64       `json:"id" orm:"id"`
	Pattern      string      `json:"pattern" orm:"pattern"`
	ReplaceType  string      `json:"replaceType" orm:"replace_type"`
	ReplaceValue string      `json:"replaceValue" orm:"replace_value"`
	Restore      int         `json:"restore" orm:"restore"`
	Description  string      `json:"description" orm:"description"`
	Priority     int         `json:"priority" orm:"priority"`
	Enabled      int         `json:"enabled" orm:"enabled"`
	CreatedAt    *gtime.Time `json:"createdAt" orm:"created_at"`
	UpdatedAt    *gtime.Time `json:"updatedAt" orm:"updated_at"`
}

type PortalAISensitiveSystemConfig struct {
	ConfigKey         string      `json:"configKey" orm:"config_key"`
	SystemDenyEnabled int         `json:"systemDenyEnabled" orm:"system_deny_enabled"`
	DictionaryText    string      `json:"dictionaryText" orm:"dictionary_text"`
	UpdatedBy         string      `json:"updatedBy" orm:"updated_by"`
	UpdatedAt         *gtime.Time `json:"updatedAt" orm:"updated_at"`
}

type PortalAISensitiveBlockAudit struct {
	Id                int64       `json:"id" orm:"id"`
	RequestId         string      `json:"requestId" orm:"request_id"`
	RouteName         string      `json:"routeName" orm:"route_name"`
	ConsumerName      string      `json:"consumerName" orm:"consumer_name"`
	DisplayName       string      `json:"displayName" orm:"display_name"`
	BlockedAt         *gtime.Time `json:"blockedAt" orm:"blocked_at"`
	BlockedBy         string      `json:"blockedBy" orm:"blocked_by"`
	RequestPhase      string      `json:"requestPhase" orm:"request_phase"`
	BlockedReasonJson string      `json:"blockedReasonJson" orm:"blocked_reason_json"`
	MatchType         string      `json:"matchType" orm:"match_type"`
	MatchedRule       string      `json:"matchedRule" orm:"matched_rule"`
	MatchedExcerpt    string      `json:"matchedExcerpt" orm:"matched_excerpt"`
	ProviderId        int64       `json:"providerId" orm:"provider_id"`
	CostUsd           string      `json:"costUsd" orm:"cost_usd"`
}

type PortalAIQuotaBalance struct {
	RouteName    string      `json:"routeName" orm:"route_name"`
	ConsumerName string      `json:"consumerName" orm:"consumer_name"`
	Quota        int64       `json:"quota" orm:"quota"`
	CreatedAt    *gtime.Time `json:"createdAt" orm:"created_at"`
	UpdatedAt    *gtime.Time `json:"updatedAt" orm:"updated_at"`
}

type PortalAIQuotaScheduleRule struct {
	Id            string      `json:"id" orm:"id"`
	RouteName     string      `json:"routeName" orm:"route_name"`
	ConsumerName  string      `json:"consumerName" orm:"consumer_name"`
	Action        string      `json:"action" orm:"action"`
	Cron          string      `json:"cron" orm:"cron"`
	Value         int64       `json:"value" orm:"value"`
	Enabled       int         `json:"enabled" orm:"enabled"`
	LastAppliedAt *gtime.Time `json:"lastAppliedAt" orm:"last_applied_at"`
	LastError     string      `json:"lastError" orm:"last_error"`
	CreatedAt     *gtime.Time `json:"createdAt" orm:"created_at"`
	UpdatedAt     *gtime.Time `json:"updatedAt" orm:"updated_at"`
}

type JobRunRecord struct {
	Id             int64       `json:"id" orm:"id"`
	JobName        string      `json:"jobName" orm:"job_name"`
	TriggerSource  string      `json:"triggerSource" orm:"trigger_source"`
	TriggerId      string      `json:"triggerId" orm:"trigger_id"`
	Status         string      `json:"status" orm:"status"`
	IdempotencyKey string      `json:"idempotencyKey" orm:"idempotency_key"`
	TargetVersion  string      `json:"targetVersion" orm:"target_version"`
	Message        string      `json:"message" orm:"message"`
	ErrorText      string      `json:"errorText" orm:"error_text"`
	StartedAt      *gtime.Time `json:"startedAt" orm:"started_at"`
	FinishedAt     *gtime.Time `json:"finishedAt" orm:"finished_at"`
	DurationMs     int64       `json:"durationMs" orm:"duration_ms"`
}
