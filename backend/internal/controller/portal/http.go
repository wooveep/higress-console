package portal

import (
	"errors"
	"io"
	"strings"

	"github.com/gogf/gf/v2/frame/g"
	"github.com/gogf/gf/v2/net/ghttp"

	portalsvc "github.com/wooveep/aigateway-console/backend/internal/service/portal"
)

func Bind(group *ghttp.RouterGroup, portalService *portalsvc.Service) {
	group.GET("/v1/consumers", func(r *ghttp.Request) {
		items, err := portalService.ListConsumers(r.Context())
		writeJSON(r, items, err, 200)
	})
	group.POST("/v1/consumers", func(r *ghttp.Request) {
		var req portalsvc.ConsumerMutation
		if err := r.Parse(&req); err != nil {
			writeJSON(r, nil, err, 400)
			return
		}
		item, err := portalService.SaveConsumer(r.Context(), req, true)
		writeJSON(r, item, err, 200)
	})
	group.GET("/v1/consumers/:name", func(r *ghttp.Request) {
		item, err := portalService.GetConsumer(r.Context(), r.GetRouter("name").String())
		if err == nil && item == nil {
			writeJSON(r, nil, errors.New("consumer not found"), 404)
			return
		}
		writeJSON(r, item, err, 200)
	})
	group.PUT("/v1/consumers/:name", func(r *ghttp.Request) {
		var req portalsvc.ConsumerMutation
		if err := r.Parse(&req); err != nil {
			writeJSON(r, nil, err, 400)
			return
		}
		req.Name = firstNonEmpty(req.Name, r.GetRouter("name").String())
		item, err := portalService.SaveConsumer(r.Context(), req, false)
		writeJSON(r, item, err, 200)
	})
	group.DELETE("/v1/consumers/:name", func(r *ghttp.Request) {
		if err := portalService.DeleteConsumer(r.Context(), r.GetRouter("name").String()); err != nil {
			writeJSON(r, nil, err, 400)
			return
		}
		r.Response.WriteStatus(204)
		r.ExitAll()
	})
	group.GET("/v1/consumers/departments", func(r *ghttp.Request) {
		items, err := portalService.ListConsumerDepartments(r.Context())
		writeJSON(r, items, err, 200)
	})
	group.POST("/v1/consumers/departments", func(r *ghttp.Request) {
		var req struct {
			Name string `json:"name"`
		}
		if err := r.Parse(&req); err != nil {
			writeJSON(r, nil, err, 400)
			return
		}
		if err := portalService.AddDepartmentCompat(r.Context(), req.Name); err != nil {
			writeJSON(r, nil, err, 400)
			return
		}
		r.Response.WriteStatus(204)
		r.ExitAll()
	})
	group.PATCH("/v1/consumers/:name/status", func(r *ghttp.Request) {
		var req struct {
			Status string `json:"status"`
		}
		if err := r.Parse(&req); err != nil {
			writeJSON(r, nil, err, 400)
			return
		}
		item, err := portalService.UpdateConsumerStatus(r.Context(), r.GetRouter("name").String(), req.Status)
		writeJSON(r, item, err, 200)
	})
	group.POST("/v1/consumers/:name/password/reset", func(r *ghttp.Request) {
		item, err := portalService.ResetPassword(r.Context(), r.GetRouter("name").String())
		writeJSON(r, item, err, 200)
	})

	group.GET("/v1/assets/:assetType/:assetId/grants", func(r *ghttp.Request) {
		items, err := portalService.ListAssetGrants(
			r.Context(),
			r.GetRouter("assetType").String(),
			r.GetRouter("assetId").String(),
		)
		writeJSON(r, items, err, 200)
	})
	group.PUT("/v1/assets/:assetType/:assetId/grants", func(r *ghttp.Request) {
		var req struct {
			Grants []portalsvc.AssetGrantRecord `json:"grants"`
		}
		if err := r.Parse(&req); err != nil {
			writeJSON(r, nil, err, 400)
			return
		}
		items, err := portalService.ReplaceAssetGrants(
			r.Context(),
			r.GetRouter("assetType").String(),
			r.GetRouter("assetId").String(),
			req.Grants,
		)
		writeJSON(r, items, err, 200)
	})

	group.GET("/v1/ai/model-assets", func(r *ghttp.Request) {
		items, err := portalService.ListModelAssets(r.Context())
		writeJSON(r, items, err, 200)
	})
	group.GET("/v1/ai/model-assets/options", func(r *ghttp.Request) {
		item, err := portalService.GetModelAssetOptions(r.Context())
		writeJSON(r, item, err, 200)
	})
	group.GET("/v1/ai/model-assets/:assetId", func(r *ghttp.Request) {
		item, err := portalService.GetModelAsset(r.Context(), r.GetRouter("assetId").String())
		if err == nil && item == nil {
			writeJSON(r, nil, errors.New("model asset not found"), 404)
			return
		}
		writeJSON(r, item, err, 200)
	})
	group.POST("/v1/ai/model-assets", func(r *ghttp.Request) {
		var req portalsvc.ModelAsset
		if err := r.Parse(&req); err != nil {
			writeJSON(r, nil, err, 400)
			return
		}
		item, err := portalService.CreateModelAsset(r.Context(), req)
		writeJSON(r, item, err, 200)
	})
	group.PUT("/v1/ai/model-assets/:assetId", func(r *ghttp.Request) {
		var req portalsvc.ModelAsset
		if err := r.Parse(&req); err != nil {
			writeJSON(r, nil, err, 400)
			return
		}
		req.AssetID = firstNonEmpty(req.AssetID, r.GetRouter("assetId").String())
		item, err := portalService.UpdateModelAsset(r.Context(), r.GetRouter("assetId").String(), req)
		writeJSON(r, item, err, 200)
	})
	group.POST("/v1/ai/model-assets/:assetId/bindings", func(r *ghttp.Request) {
		var req portalsvc.ModelAssetBinding
		if err := r.Parse(&req); err != nil {
			writeJSON(r, nil, err, 400)
			return
		}
		item, err := portalService.CreateModelBinding(r.Context(), r.GetRouter("assetId").String(), req)
		writeJSON(r, item, err, 200)
	})
	group.PUT("/v1/ai/model-assets/:assetId/bindings/:bindingId", func(r *ghttp.Request) {
		var req portalsvc.ModelAssetBinding
		if err := r.Parse(&req); err != nil {
			writeJSON(r, nil, err, 400)
			return
		}
		req.BindingID = firstNonEmpty(req.BindingID, r.GetRouter("bindingId").String())
		item, err := portalService.UpdateModelBinding(
			r.Context(),
			r.GetRouter("assetId").String(),
			r.GetRouter("bindingId").String(),
			req,
		)
		writeJSON(r, item, err, 200)
	})
	group.POST("/v1/ai/model-assets/:assetId/bindings/:bindingId/publish", func(r *ghttp.Request) {
		item, err := portalService.PublishModelBinding(
			r.Context(),
			r.GetRouter("assetId").String(),
			r.GetRouter("bindingId").String(),
		)
		writeJSON(r, item, err, 200)
	})
	group.POST("/v1/ai/model-assets/:assetId/bindings/:bindingId/unpublish", func(r *ghttp.Request) {
		item, err := portalService.UnpublishModelBinding(
			r.Context(),
			r.GetRouter("assetId").String(),
			r.GetRouter("bindingId").String(),
		)
		writeJSON(r, item, err, 200)
	})
	group.GET("/v1/ai/model-assets/:assetId/bindings/:bindingId/price-versions", func(r *ghttp.Request) {
		items, err := portalService.ListBindingPriceVersions(
			r.Context(),
			r.GetRouter("assetId").String(),
			r.GetRouter("bindingId").String(),
		)
		writeJSON(r, items, err, 200)
	})
	group.POST("/v1/ai/model-assets/:assetId/bindings/:bindingId/price-versions/:versionId/restore", func(r *ghttp.Request) {
		item, err := portalService.RestoreBindingPriceVersion(
			r.Context(),
			r.GetRouter("assetId").String(),
			r.GetRouter("bindingId").String(),
			r.GetRouter("versionId").Int64(),
		)
		writeJSON(r, item, err, 200)
	})

	group.GET("/v1/ai/agent-catalog", func(r *ghttp.Request) {
		items, err := portalService.ListAgentCatalogs(r.Context())
		writeJSON(r, items, err, 200)
	})
	group.GET("/v1/ai/agent-catalog/options", func(r *ghttp.Request) {
		item, err := portalService.GetAgentCatalogOptions(r.Context())
		writeJSON(r, item, err, 200)
	})
	group.GET("/v1/ai/agent-catalog/:agentId", func(r *ghttp.Request) {
		item, err := portalService.GetAgentCatalog(r.Context(), r.GetRouter("agentId").String())
		if err == nil && item == nil {
			writeJSON(r, nil, errors.New("agent catalog not found"), 404)
			return
		}
		writeJSON(r, item, err, 200)
	})
	group.POST("/v1/ai/agent-catalog", func(r *ghttp.Request) {
		var req portalsvc.AgentCatalogRecord
		if err := r.Parse(&req); err != nil {
			writeJSON(r, nil, err, 400)
			return
		}
		item, err := portalService.CreateAgentCatalog(r.Context(), req)
		writeJSON(r, item, err, 200)
	})
	group.PUT("/v1/ai/agent-catalog/:agentId", func(r *ghttp.Request) {
		var req portalsvc.AgentCatalogRecord
		if err := r.Parse(&req); err != nil {
			writeJSON(r, nil, err, 400)
			return
		}
		req.AgentID = firstNonEmpty(req.AgentID, r.GetRouter("agentId").String())
		item, err := portalService.UpdateAgentCatalog(r.Context(), r.GetRouter("agentId").String(), req)
		writeJSON(r, item, err, 200)
	})
	group.POST("/v1/ai/agent-catalog/:agentId/publish", func(r *ghttp.Request) {
		item, err := portalService.PublishAgentCatalog(r.Context(), r.GetRouter("agentId").String())
		writeJSON(r, item, err, 200)
	})
	group.POST("/v1/ai/agent-catalog/:agentId/unpublish", func(r *ghttp.Request) {
		item, err := portalService.UnpublishAgentCatalog(r.Context(), r.GetRouter("agentId").String())
		writeJSON(r, item, err, 200)
	})

	group.GET("/v1/ai/quotas/menu-state", func(r *ghttp.Request) {
		item, err := portalService.GetAIQuotaMenuState(r.Context())
		writeJSON(r, item, err, 200)
	})
	group.GET("/v1/ai/quotas/routes", func(r *ghttp.Request) {
		items, err := portalService.ListAIQuotaRoutes(r.Context())
		writeJSON(r, items, err, 200)
	})
	group.GET("/v1/ai/quotas/routes/:routeName/consumers", func(r *ghttp.Request) {
		items, err := portalService.ListAIQuotaConsumers(r.Context(), r.GetRouter("routeName").String())
		writeJSON(r, items, err, 200)
	})
	group.PUT("/v1/ai/quotas/routes/:routeName/consumers/:consumerName/quota", func(r *ghttp.Request) {
		var req struct {
			Value int64 `json:"value"`
		}
		if err := r.Parse(&req); err != nil {
			writeJSON(r, nil, err, 400)
			return
		}
		item, err := portalService.RefreshAIQuota(r.Context(), r.GetRouter("routeName").String(), r.GetRouter("consumerName").String(), req.Value)
		writeJSON(r, item, err, 200)
	})
	group.POST("/v1/ai/quotas/routes/:routeName/consumers/:consumerName/delta", func(r *ghttp.Request) {
		var req struct {
			Value int64 `json:"value"`
		}
		if err := r.Parse(&req); err != nil {
			writeJSON(r, nil, err, 400)
			return
		}
		item, err := portalService.DeltaAIQuota(r.Context(), r.GetRouter("routeName").String(), r.GetRouter("consumerName").String(), req.Value)
		writeJSON(r, item, err, 200)
	})
	group.GET("/v1/ai/quotas/routes/:routeName/consumers/:consumerName/policy", func(r *ghttp.Request) {
		item, err := portalService.GetAIQuotaUserPolicy(r.Context(), r.GetRouter("routeName").String(), r.GetRouter("consumerName").String())
		writeJSON(r, item, err, 200)
	})
	group.PUT("/v1/ai/quotas/routes/:routeName/consumers/:consumerName/policy", func(r *ghttp.Request) {
		var req portalsvc.AIQuotaUserPolicyRequest
		if err := r.Parse(&req); err != nil {
			writeJSON(r, nil, err, 400)
			return
		}
		item, err := portalService.SaveAIQuotaUserPolicy(r.Context(), r.GetRouter("routeName").String(), r.GetRouter("consumerName").String(), req)
		writeJSON(r, item, err, 200)
	})
	group.GET("/v1/ai/quotas/routes/:routeName/schedules", func(r *ghttp.Request) {
		items, err := portalService.ListAIQuotaScheduleRules(r.Context(), r.GetRouter("routeName").String(), r.GetQuery("consumerName").String())
		writeJSON(r, items, err, 200)
	})
	group.PUT("/v1/ai/quotas/routes/:routeName/schedules", func(r *ghttp.Request) {
		var req portalsvc.AIQuotaScheduleRuleRequest
		if err := r.Parse(&req); err != nil {
			writeJSON(r, nil, err, 400)
			return
		}
		item, err := portalService.SaveAIQuotaScheduleRule(r.Context(), r.GetRouter("routeName").String(), req)
		writeJSON(r, item, err, 200)
	})
	group.DELETE("/v1/ai/quotas/routes/:routeName/schedules/:ruleId", func(r *ghttp.Request) {
		if err := portalService.DeleteAIQuotaScheduleRule(r.Context(), r.GetRouter("routeName").String(), r.GetRouter("ruleId").String()); err != nil {
			writeJSON(r, nil, err, 400)
			return
		}
		r.Response.WriteStatus(204)
		r.ExitAll()
	})

	group.GET("/v1/ai/sensitive-words/menu-state", func(r *ghttp.Request) {
		item, err := portalService.GetAISensitiveMenuState(r.Context())
		writeJSON(r, item, err, 200)
	})
	group.GET("/v1/ai/sensitive-words/status", func(r *ghttp.Request) {
		item, err := portalService.GetAISensitiveStatus(r.Context())
		writeJSON(r, item, err, 200)
	})
	group.POST("/v1/ai/sensitive-words/reconcile", func(r *ghttp.Request) {
		item, err := portalService.ReconcileAISensitive(r.Context())
		writeJSON(r, item, err, 200)
	})
	group.GET("/v1/ai/sensitive-words/detect-rules", func(r *ghttp.Request) {
		items, err := portalService.ListAISensitiveDetectRules(r.Context())
		writeJSON(r, items, err, 200)
	})
	group.POST("/v1/ai/sensitive-words/detect-rules", func(r *ghttp.Request) {
		var req portalsvc.AISensitiveDetectRule
		if err := r.Parse(&req); err != nil {
			writeJSON(r, nil, err, 400)
			return
		}
		item, err := portalService.SaveAISensitiveDetectRule(r.Context(), req)
		writeJSON(r, item, err, 200)
	})
	group.PUT("/v1/ai/sensitive-words/detect-rules/:id", func(r *ghttp.Request) {
		var req portalsvc.AISensitiveDetectRule
		if err := r.Parse(&req); err != nil {
			writeJSON(r, nil, err, 400)
			return
		}
		req.ID = r.GetRouter("id").Int64()
		item, err := portalService.SaveAISensitiveDetectRule(r.Context(), req)
		writeJSON(r, item, err, 200)
	})
	group.DELETE("/v1/ai/sensitive-words/detect-rules/:id", func(r *ghttp.Request) {
		if err := portalService.DeleteAISensitiveDetectRule(r.Context(), r.GetRouter("id").Int64()); err != nil {
			writeJSON(r, nil, err, 400)
			return
		}
		r.Response.WriteStatus(204)
		r.ExitAll()
	})
	group.GET("/v1/ai/sensitive-words/replace-rules", func(r *ghttp.Request) {
		items, err := portalService.ListAISensitiveReplaceRules(r.Context())
		writeJSON(r, items, err, 200)
	})
	group.POST("/v1/ai/sensitive-words/replace-rules", func(r *ghttp.Request) {
		var req portalsvc.AISensitiveReplaceRule
		if err := r.Parse(&req); err != nil {
			writeJSON(r, nil, err, 400)
			return
		}
		item, err := portalService.SaveAISensitiveReplaceRule(r.Context(), req)
		writeJSON(r, item, err, 200)
	})
	group.PUT("/v1/ai/sensitive-words/replace-rules/:id", func(r *ghttp.Request) {
		var req portalsvc.AISensitiveReplaceRule
		if err := r.Parse(&req); err != nil {
			writeJSON(r, nil, err, 400)
			return
		}
		req.ID = r.GetRouter("id").Int64()
		item, err := portalService.SaveAISensitiveReplaceRule(r.Context(), req)
		writeJSON(r, item, err, 200)
	})
	group.DELETE("/v1/ai/sensitive-words/replace-rules/:id", func(r *ghttp.Request) {
		if err := portalService.DeleteAISensitiveReplaceRule(r.Context(), r.GetRouter("id").Int64()); err != nil {
			writeJSON(r, nil, err, 400)
			return
		}
		r.Response.WriteStatus(204)
		r.ExitAll()
	})
	group.GET("/v1/ai/sensitive-words/audits", func(r *ghttp.Request) {
		items, err := portalService.ListAISensitiveAudits(r.Context(), portalsvc.AISensitiveAuditQuery{
			ConsumerName: r.GetQuery("consumerName").String(),
			DisplayName:  r.GetQuery("displayName").String(),
			RouteName:    r.GetQuery("routeName").String(),
			MatchType:    r.GetQuery("matchType").String(),
			StartTime:    r.GetQuery("startTime").String(),
			EndTime:      r.GetQuery("endTime").String(),
			Limit:        r.GetQuery("limit").Int(),
		})
		writeJSON(r, items, err, 200)
	})
	group.GET("/v1/ai/sensitive-words/system-config", func(r *ghttp.Request) {
		item, err := portalService.GetAISensitiveSystemConfig(r.Context())
		writeJSON(r, item, err, 200)
	})
	group.PUT("/v1/ai/sensitive-words/system-config", func(r *ghttp.Request) {
		var req portalsvc.AISensitiveSystemConfig
		if err := r.Parse(&req); err != nil {
			writeJSON(r, nil, err, 400)
			return
		}
		item, err := portalService.SaveAISensitiveSystemConfig(r.Context(), req)
		writeJSON(r, item, err, 200)
	})

	group.GET("/v1/org/departments/tree", func(r *ghttp.Request) {
		items, err := portalService.ListDepartmentTree(r.Context())
		writeJSON(r, items, err, 200)
	})
	group.POST("/v1/org/departments", func(r *ghttp.Request) {
		var req portalsvc.DepartmentMutation
		if err := r.Parse(&req); err != nil {
			writeJSON(r, nil, err, 400)
			return
		}
		item, err := portalService.CreateDepartment(r.Context(), req)
		writeJSON(r, item, err, 200)
	})
	group.PUT("/v1/org/departments/:departmentId", func(r *ghttp.Request) {
		var req portalsvc.DepartmentMutation
		if err := r.Parse(&req); err != nil {
			writeJSON(r, nil, err, 400)
			return
		}
		item, err := portalService.UpdateDepartment(r.Context(), r.GetRouter("departmentId").String(), req)
		writeJSON(r, item, err, 200)
	})
	group.PATCH("/v1/org/departments/:departmentId/move", func(r *ghttp.Request) {
		var req struct {
			ParentDepartmentID string `json:"parentDepartmentId"`
		}
		if err := r.Parse(&req); err != nil {
			writeJSON(r, nil, err, 400)
			return
		}
		item, err := portalService.MoveDepartment(r.Context(), r.GetRouter("departmentId").String(), req.ParentDepartmentID)
		writeJSON(r, item, err, 200)
	})
	group.DELETE("/v1/org/departments/:departmentId", func(r *ghttp.Request) {
		if err := portalService.DeleteDepartment(r.Context(), r.GetRouter("departmentId").String()); err != nil {
			writeJSON(r, nil, err, 400)
			return
		}
		r.Response.WriteStatus(204)
		r.ExitAll()
	})

	group.GET("/v1/org/accounts", func(r *ghttp.Request) {
		items, err := portalService.ListAccounts(r.Context())
		writeJSON(r, items, err, 200)
	})
	group.POST("/v1/org/accounts", func(r *ghttp.Request) {
		var req portalsvc.AccountMutation
		if err := r.Parse(&req); err != nil {
			writeJSON(r, nil, err, 400)
			return
		}
		item, err := portalService.CreateAccount(r.Context(), req)
		writeJSON(r, item, err, 200)
	})
	group.PUT("/v1/org/accounts/:consumerName", func(r *ghttp.Request) {
		var req portalsvc.AccountMutation
		if err := r.Parse(&req); err != nil {
			writeJSON(r, nil, err, 400)
			return
		}
		item, err := portalService.UpdateAccount(r.Context(), r.GetRouter("consumerName").String(), req)
		writeJSON(r, item, err, 200)
	})
	group.PATCH("/v1/org/accounts/:consumerName/assignment", func(r *ghttp.Request) {
		var req struct {
			DepartmentID       string `json:"departmentId"`
			ParentConsumerName string `json:"parentConsumerName"`
		}
		if err := r.Parse(&req); err != nil {
			writeJSON(r, nil, err, 400)
			return
		}
		item, err := portalService.UpdateAccountAssignment(r.Context(), r.GetRouter("consumerName").String(), req.DepartmentID, req.ParentConsumerName)
		writeJSON(r, item, err, 200)
	})
	group.PATCH("/v1/org/accounts/:consumerName/status", func(r *ghttp.Request) {
		var req struct {
			Status string `json:"status"`
		}
		if err := r.Parse(&req); err != nil {
			writeJSON(r, nil, err, 400)
			return
		}
		item, err := portalService.UpdateAccountStatus(r.Context(), r.GetRouter("consumerName").String(), req.Status)
		writeJSON(r, item, err, 200)
	})
	group.GET("/v1/org/template", func(r *ghttp.Request) {
		content, err := portalService.DownloadOrgTemplate(r.Context())
		if err != nil {
			writeJSON(r, nil, err, 400)
			return
		}
		writeWorkbook(r, "organization-template.xlsx", content)
	})
	group.GET("/v1/org/export", func(r *ghttp.Request) {
		content, err := portalService.ExportOrganizationWorkbook(r.Context())
		if err != nil {
			writeJSON(r, nil, err, 400)
			return
		}
		writeWorkbook(r, "organization-export.xlsx", content)
	})
	group.POST("/v1/org/import", func(r *ghttp.Request) {
		file := r.GetUploadFile("file")
		if file == nil {
			writeJSON(r, nil, errors.New("import file cannot be empty"), 400)
			return
		}
		reader, err := file.Open()
		if err != nil {
			writeJSON(r, nil, err, 400)
			return
		}
		defer reader.Close()
		content, err := io.ReadAll(reader)
		if err != nil {
			writeJSON(r, nil, err, 400)
			return
		}
		item, err := portalService.ImportOrganizationWorkbook(r.Context(), content)
		writeJSON(r, item, err, 200)
	})

	group.POST("/v1/portal/invite-codes", func(r *ghttp.Request) {
		var req struct {
			ExpiresInDays int `json:"expiresInDays"`
		}
		if err := r.Parse(&req); err != nil {
			writeJSON(r, nil, err, 400)
			return
		}
		item, err := portalService.CreateInviteCode(r.Context(), req.ExpiresInDays)
		writeJSON(r, item, err, 200)
	})
	group.GET("/v1/portal/invite-codes", func(r *ghttp.Request) {
		items, err := portalService.ListInviteCodes(r.Context(), portalsvc.InviteCodeQuery{
			PageNum:  r.GetQuery("pageNum").Int(),
			PageSize: r.GetQuery("pageSize").Int(),
			Status:   r.GetQuery("status").String(),
		})
		writeJSON(r, items, err, 200)
	})
	group.PATCH("/v1/portal/invite-codes/:code", func(r *ghttp.Request) {
		var req struct {
			Status string `json:"status"`
		}
		if err := r.Parse(&req); err != nil {
			writeJSON(r, nil, err, 400)
			return
		}
		item, err := portalService.UpdateInviteCodeStatus(r.Context(), r.GetRouter("code").String(), req.Status)
		writeJSON(r, item, err, 200)
	})
	group.GET("/v1/portal/stats/usage", func(r *ghttp.Request) {
		items, err := portalService.ListUsageStats(r.Context(), portalsvc.UsageStatsQuery{
			From: optionalInt64Query(r, "from"),
			To:   optionalInt64Query(r, "to"),
		})
		writeJSON(r, items, err, 200)
	})
	group.GET("/v1/portal/stats/usage-trend", func(r *ghttp.Request) {
		items, err := portalService.ListUsageTrend(r.Context(), portalsvc.UsageTrendQuery{
			From:            optionalInt64Query(r, "from"),
			To:              optionalInt64Query(r, "to"),
			Bucket:          r.GetQuery("bucket").String(),
			ConsumerName:    r.GetQuery("consumerName").String(),
			DepartmentID:    r.GetQuery("departmentId").String(),
			IncludeChildren: optionalBoolQuery(r, "includeChildren"),
			ModelID:         r.GetQuery("modelId").String(),
			RouteName:       r.GetQuery("routeName").String(),
		})
		writeJSON(r, items, err, 200)
	})
	group.GET("/v1/portal/stats/usage-events", func(r *ghttp.Request) {
		items, err := portalService.ListUsageEvents(r.Context(), portalsvc.UsageEventsQuery{
			From:            optionalInt64Query(r, "from"),
			To:              optionalInt64Query(r, "to"),
			ConsumerNames:   optionalCSVQuery(r, "consumerNames"),
			DepartmentIDs:   optionalCSVQuery(r, "departmentIds"),
			IncludeChildren: optionalBoolQuery(r, "includeChildren"),
			APIKeyIDs:       optionalCSVQuery(r, "apiKeyIds"),
			ModelIDs:        optionalCSVQuery(r, "modelIds"),
			RouteNames:      optionalCSVQuery(r, "routeNames"),
			RequestStatuses: optionalCSVQuery(r, "requestStatuses"),
			UsageStatuses:   optionalCSVQuery(r, "usageStatuses"),
			PageNum:         r.GetQuery("pageNum").Int(),
			PageSize:        r.GetQuery("pageSize").Int(),
		})
		writeJSON(r, items, err, 200)
	})
	group.GET("/v1/portal/stats/usage-event-options", func(r *ghttp.Request) {
		item, err := portalService.ListUsageEventFilterOptions(r.Context(), portalsvc.UsageEventsQuery{
			From:            optionalInt64Query(r, "from"),
			To:              optionalInt64Query(r, "to"),
			ConsumerNames:   optionalCSVQuery(r, "consumerNames"),
			DepartmentIDs:   optionalCSVQuery(r, "departmentIds"),
			IncludeChildren: optionalBoolQuery(r, "includeChildren"),
			APIKeyIDs:       optionalCSVQuery(r, "apiKeyIds"),
			ModelIDs:        optionalCSVQuery(r, "modelIds"),
			RouteNames:      optionalCSVQuery(r, "routeNames"),
			RequestStatuses: optionalCSVQuery(r, "requestStatuses"),
			UsageStatuses:   optionalCSVQuery(r, "usageStatuses"),
		})
		writeJSON(r, item, err, 200)
	})
	group.GET("/v1/portal/stats/department-bills", func(r *ghttp.Request) {
		items, err := portalService.ListDepartmentBills(r.Context(), portalsvc.DepartmentBillsQuery{
			From:            optionalInt64Query(r, "from"),
			To:              optionalInt64Query(r, "to"),
			DepartmentIDs:   optionalCSVQuery(r, "departmentIds"),
			IncludeChildren: optionalBoolQuery(r, "includeChildren"),
		})
		writeJSON(r, items, err, 200)
	})
}

func writeJSON(r *ghttp.Request, data any, err error, successStatus int) {
	if err != nil {
		status := 400
		message := err.Error()
		if strings.Contains(strings.ToLower(message), "not found") {
			status = 404
		} else if strings.Contains(strings.ToLower(message), "unavailable") {
			status = 503
		}
		r.Response.WriteStatusExit(status, g.Map{"success": false, "message": message})
		return
	}
	if successStatus > 0 && successStatus != 200 {
		r.Response.WriteStatus(successStatus)
	}
	r.Response.WriteJsonExit(g.Map{"success": true, "data": data})
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return strings.TrimSpace(value)
		}
	}
	return ""
}

func optionalInt64Query(r *ghttp.Request, key string) *int64 {
	raw := strings.TrimSpace(r.GetQuery(key).String())
	if raw == "" {
		return nil
	}
	value := r.GetQuery(key).Int64()
	return &value
}

func optionalBoolQuery(r *ghttp.Request, key string) *bool {
	raw := strings.TrimSpace(r.GetQuery(key).String())
	if raw == "" {
		return nil
	}
	value := r.GetQuery(key).Bool()
	return &value
}

func optionalCSVQuery(r *ghttp.Request, key string) []string {
	raw := strings.TrimSpace(r.GetQuery(key).String())
	if raw == "" {
		return nil
	}
	parts := strings.Split(raw, ",")
	items := make([]string, 0, len(parts))
	for _, part := range parts {
		if trimmed := strings.TrimSpace(part); trimmed != "" {
			items = append(items, trimmed)
		}
	}
	if len(items) == 0 {
		return nil
	}
	return items
}

func writeWorkbook(r *ghttp.Request, filename string, content []byte) {
	r.Response.Header().Set("Content-Type", "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet")
	r.Response.Header().Set("Content-Disposition", "attachment; filename=\""+filename+"\"")
	r.Response.WriteExit(content)
}
