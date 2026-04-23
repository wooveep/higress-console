package do

import (
	"github.com/gogf/gf/v2/frame/g"
	"github.com/gogf/gf/v2/os/gtime"
)

type PortalUser struct {
	g.Meta       `orm:"table:portal_user, do:true"`
	Id           any
	ConsumerName any
	DisplayName  any
	Email        any
	PasswordHash any
	Status       any
	Source       any
	UserLevel    any
	IsDeleted    any
	DeletedAt    *gtime.Time
	LastLoginAt  *gtime.Time
	CreatedAt    *gtime.Time
	UpdatedAt    *gtime.Time
}

type PortalInviteCode struct {
	g.Meta         `orm:"table:portal_invite_code, do:true"`
	Id             any
	InviteCode     any
	Status         any
	ExpiresAt      *gtime.Time
	UsedByConsumer any
	UsedAt         *gtime.Time
	CreatedAt      *gtime.Time
	UpdatedAt      *gtime.Time
}

type OrgDepartment struct {
	g.Meta             `orm:"table:org_department, do:true"`
	Id                 any
	DepartmentId       any
	Name               any
	ParentDepartmentId any
	AdminConsumerName  any
	Path               any
	Level              any
	SortOrder          any
	Status             any
	CreatedAt          *gtime.Time
	UpdatedAt          *gtime.Time
}

type OrgAccountMembership struct {
	g.Meta             `orm:"table:org_account_membership, do:true"`
	Id                 any
	ConsumerName       any
	DepartmentId       any
	ParentConsumerName any
	CreatedAt          *gtime.Time
	UpdatedAt          *gtime.Time
}

type AssetGrant struct {
	g.Meta      `orm:"table:asset_grant, do:true"`
	Id          any
	AssetType   any
	AssetId     any
	SubjectType any
	SubjectId   any
	CreatedAt   *gtime.Time
	UpdatedAt   *gtime.Time
}

type QuotaPolicyUser struct {
	g.Meta                `orm:"table:quota_policy_user, do:true"`
	Id                    any
	ConsumerName          any
	LimitTotalMicroYuan   any
	Limit5hMicroYuan      any
	LimitDailyMicroYuan   any
	DailyResetMode        any
	DailyResetTime        any
	LimitWeeklyMicroYuan  any
	LimitMonthlyMicroYuan any
	CostResetAt           *gtime.Time
	CreatedAt             *gtime.Time
	UpdatedAt             *gtime.Time
}

type PortalModelAsset struct {
	g.Meta           `orm:"table:portal_model_asset, do:true"`
	Id               any
	AssetId          any
	CanonicalName    any
	DisplayName      any
	Intro            any
	TagsJson         any
	ModalitiesJson   any
	FeaturesJson     any
	RequestKindsJson any
	CreatedAt        *gtime.Time
	UpdatedAt        *gtime.Time
}

type PortalModelBinding struct {
	g.Meta        `orm:"table:portal_model_binding, do:true"`
	Id            any
	BindingId     any
	AssetId       any
	ModelId       any
	ProviderName  any
	TargetModel   any
	Protocol      any
	Endpoint      any
	PricingJson   any
	Rpm           any
	Tpm           any
	ContextWindow any
	Status        any
	PublishedAt   *gtime.Time
	UnpublishedAt *gtime.Time
	CreatedAt     *gtime.Time
	UpdatedAt     *gtime.Time
}

type PortalModelBindingPriceVersion struct {
	g.Meta        `orm:"table:portal_model_binding_price_version, do:true"`
	VersionId     any
	AssetId       any
	BindingId     any
	Status        any
	Active        any
	EffectiveFrom *gtime.Time
	EffectiveTo   *gtime.Time
	PricingJson   any
	CreatedAt     *gtime.Time
	UpdatedAt     *gtime.Time
}

type PortalAgentCatalog struct {
	g.Meta             `orm:"table:portal_agent_catalog, do:true"`
	Id                 any
	AgentId            any
	CanonicalName      any
	DisplayName        any
	Intro              any
	Description        any
	IconUrl            any
	TagsJson           any
	McpServerName      any
	ToolCount          any
	TransportTypesJson any
	ResourceSummary    any
	PromptSummary      any
	Status             any
	PublishedAt        *gtime.Time
	UnpublishedAt      *gtime.Time
	CreatedAt          *gtime.Time
	UpdatedAt          *gtime.Time
}

type PortalAISensitiveDetectRule struct {
	g.Meta      `orm:"table:portal_ai_sensitive_detect_rule, do:true"`
	Id          any
	Pattern     any
	MatchType   any
	Description any
	Priority    any
	Enabled     any
	CreatedAt   *gtime.Time
	UpdatedAt   *gtime.Time
}

type PortalAISensitiveReplaceRule struct {
	g.Meta       `orm:"table:portal_ai_sensitive_replace_rule, do:true"`
	Id           any
	Pattern      any
	ReplaceType  any
	ReplaceValue any
	Restore      any
	Description  any
	Priority     any
	Enabled      any
	CreatedAt    *gtime.Time
	UpdatedAt    *gtime.Time
}

type PortalAISensitiveSystemConfig struct {
	g.Meta            `orm:"table:portal_ai_sensitive_system_config, do:true"`
	ConfigKey         any
	SystemDenyEnabled any
	DictionaryText    any
	UpdatedBy         any
	UpdatedAt         *gtime.Time
}

type PortalAISensitiveBlockAudit struct {
	g.Meta            `orm:"table:portal_ai_sensitive_block_audit, do:true"`
	Id                any
	RequestId         any
	RouteName         any
	ConsumerName      any
	DisplayName       any
	BlockedAt         *gtime.Time
	BlockedBy         any
	RequestPhase      any
	BlockedReasonJson any
	MatchType         any
	MatchedRule       any
	MatchedExcerpt    any
	ProviderId        any
	CostUsd           any
}

type PortalAIQuotaBalance struct {
	g.Meta       `orm:"table:portal_ai_quota_balance, do:true"`
	RouteName    any
	ConsumerName any
	Quota        any
	CreatedAt    *gtime.Time
	UpdatedAt    *gtime.Time
}

type PortalAIQuotaScheduleRule struct {
	g.Meta        `orm:"table:portal_ai_quota_schedule_rule, do:true"`
	Id            any
	RouteName     any
	ConsumerName  any
	Action        any
	Cron          any
	Value         any
	Enabled       any
	LastAppliedAt *gtime.Time
	LastError     any
	CreatedAt     *gtime.Time
	UpdatedAt     *gtime.Time
}

type JobRunRecord struct {
	g.Meta         `orm:"table:job_run_record, do:true"`
	Id             any
	JobName        any
	TriggerSource  any
	TriggerId      any
	Status         any
	IdempotencyKey any
	TargetVersion  any
	Message        any
	ErrorText      any
	StartedAt      *gtime.Time
	FinishedAt     *gtime.Time
	DurationMs     any
}
