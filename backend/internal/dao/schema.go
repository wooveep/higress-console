package dao

import "github.com/gogf/gf/v2/database/gdb"

type TableDAO struct {
	Name string
}

func (d TableDAO) Model(db gdb.DB) *gdb.Model {
	return db.Model(d.Name)
}

var (
	PortalUser                     = TableDAO{Name: "portal_user"}
	PortalInviteCode               = TableDAO{Name: "portal_invite_code"}
	OrgDepartment                  = TableDAO{Name: "org_department"}
	OrgAccountMembership           = TableDAO{Name: "org_account_membership"}
	AssetGrant                     = TableDAO{Name: "asset_grant"}
	QuotaPolicyUser                = TableDAO{Name: "quota_policy_user"}
	PortalModelAsset               = TableDAO{Name: "portal_model_asset"}
	PortalModelBinding             = TableDAO{Name: "portal_model_binding"}
	PortalModelBindingPriceVersion = TableDAO{Name: "portal_model_binding_price_version"}
	PortalAgentCatalog             = TableDAO{Name: "portal_agent_catalog"}
	PortalAISensitiveDetectRule    = TableDAO{Name: "portal_ai_sensitive_detect_rule"}
	PortalAISensitiveReplaceRule   = TableDAO{Name: "portal_ai_sensitive_replace_rule"}
	PortalAISensitiveSystemConfig  = TableDAO{Name: "portal_ai_sensitive_system_config"}
	PortalAISensitiveBlockAudit    = TableDAO{Name: "portal_ai_sensitive_block_audit"}
	PortalAIQuotaBalance           = TableDAO{Name: "portal_ai_quota_balance"}
	PortalAIQuotaScheduleRule      = TableDAO{Name: "portal_ai_quota_schedule_rule"}
	JobRunRecord                   = TableDAO{Name: "job_run_record"}
)
