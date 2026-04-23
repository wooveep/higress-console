package jobs

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"slices"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/gogf/gf/v2/os/gcron"

	"github.com/wooveep/aigateway-console/backend/internal/consts"
	internaljob "github.com/wooveep/aigateway-console/backend/internal/job"
	gatewaysvc "github.com/wooveep/aigateway-console/backend/internal/service/gateway"
	portalsvc "github.com/wooveep/aigateway-console/backend/internal/service/portal"
	k8sclient "github.com/wooveep/aigateway-console/backend/utility/clients/k8s"
	portaldbclient "github.com/wooveep/aigateway-console/backend/utility/clients/portaldb"
)

const (
	RunStatusRunning = "running"
	RunStatusSuccess = "success"
	RunStatusFailed  = "failed"
	RunStatusSkipped = "skipped"

	modelRateLimitProjectionKind    = "ai-model-rate-limit-projections"
	modelRateLimitProjectionName    = "default"
	modelRateLimitPluginRPM         = "cluster-key-rate-limit"
	modelRateLimitPluginTPM         = "ai-token-ratelimit"
	modelRateLimitRuleNameRPMPrefix = "model-rate-rpm:"
	modelRateLimitRuleNameTPMPrefix = "model-rate-tpm:"
)

type TriggerInput struct {
	Source    string `json:"source,omitempty"`
	TriggerID string `json:"triggerId,omitempty"`
}

type RunRecord struct {
	ID             int64      `json:"id"`
	JobName        string     `json:"jobName"`
	TriggerSource  string     `json:"triggerSource"`
	TriggerID      string     `json:"triggerId"`
	Status         string     `json:"status"`
	IdempotencyKey string     `json:"idempotencyKey,omitempty"`
	TargetVersion  string     `json:"targetVersion,omitempty"`
	Message        string     `json:"message,omitempty"`
	ErrorText      string     `json:"errorText,omitempty"`
	StartedAt      *time.Time `json:"startedAt,omitempty"`
	FinishedAt     *time.Time `json:"finishedAt,omitempty"`
	DurationMs     *int64     `json:"durationMs,omitempty"`
}

type JobSummary struct {
	Name        string     `json:"name"`
	Description string     `json:"description"`
	Schedule    string     `json:"schedule,omitempty"`
	ManualOnly  bool       `json:"manualOnly"`
	Running     bool       `json:"running"`
	LastRun     *RunRecord `json:"lastRun,omitempty"`
}

type JobDetail struct {
	JobSummary
	RecentRuns []RunRecord `json:"recentRuns"`
}

type Service struct {
	client  portaldbclient.Client
	portal  portalReader
	gateway *gatewaysvc.Service
	k8s     k8sclient.Client

	cron *gcron.Cron

	schemaMu      sync.Mutex
	schemaChecked bool

	startMu sync.Mutex
	started bool

	runMu   sync.RWMutex
	running map[string]bool

	definitions map[string]internaljob.Definition
}

type portalReader interface {
	ListConsumers(ctx context.Context) ([]portalsvc.ConsumerRecord, error)
	ListAccounts(ctx context.Context) ([]portalsvc.OrgAccountRecord, error)
	ListDepartmentTree(ctx context.Context) ([]*portalsvc.OrgDepartmentNode, error)
}

func New(
	client portaldbclient.Client,
	portal portalReader,
	gateway *gatewaysvc.Service,
	k8s k8sclient.Client,
) *Service {
	definitions := make(map[string]internaljob.Definition, len(internaljob.PlannedJobs))
	for _, item := range internaljob.PlannedJobs {
		definitions[item.Name] = item
	}
	return &Service{
		client:      client,
		portal:      portal,
		gateway:     gateway,
		k8s:         k8s,
		cron:        gcron.New(),
		running:     map[string]bool{},
		definitions: definitions,
	}
}

func (s *Service) Start(ctx context.Context) error {
	s.startMu.Lock()
	defer s.startMu.Unlock()

	if s.started {
		return nil
	}
	for _, definition := range internaljob.PlannedJobs {
		if definition.ManualOnly || strings.TrimSpace(definition.Schedule) == "" {
			continue
		}
		jobName := definition.Name
		if _, err := s.cron.AddSingleton(ctx, definition.Schedule, func(ctx context.Context) {
			_, _ = s.Trigger(ctx, jobName, TriggerInput{
				Source:    "scheduled",
				TriggerID: fmt.Sprintf("scheduled:%s", jobName),
			})
		}, jobName); err != nil {
			return err
		}
	}
	s.started = true
	return nil
}

func (s *Service) AfterWrite(ctx context.Context, trigger string) error {
	relevant := s.jobsForHook(trigger)
	if len(relevant) == 0 {
		return nil
	}
	for _, jobName := range relevant {
		jobName := jobName
		go func() {
			_, _ = s.Trigger(context.Background(), jobName, TriggerInput{
				Source:    "hook",
				TriggerID: trigger,
			})
		}()
	}
	return nil
}

func (s *Service) ListJobs(ctx context.Context) ([]JobSummary, error) {
	items := make([]JobSummary, 0, len(internaljob.PlannedJobs))
	for _, definition := range internaljob.PlannedJobs {
		lastRun, err := s.latestRun(ctx, definition.Name)
		if err != nil {
			return nil, err
		}
		items = append(items, JobSummary{
			Name:        definition.Name,
			Description: definition.Description,
			Schedule:    definition.Schedule,
			ManualOnly:  definition.ManualOnly,
			Running:     s.isRunning(definition.Name),
			LastRun:     lastRun,
		})
	}
	return items, nil
}

func (s *Service) GetJob(ctx context.Context, name string) (*JobDetail, error) {
	definition, ok := s.definitions[strings.TrimSpace(name)]
	if !ok {
		return nil, fmt.Errorf("job not found: %s", name)
	}
	lastRun, err := s.latestRun(ctx, definition.Name)
	if err != nil {
		return nil, err
	}
	recentRuns, err := s.listRuns(ctx, definition.Name, 10)
	if err != nil {
		return nil, err
	}
	return &JobDetail{
		JobSummary: JobSummary{
			Name:        definition.Name,
			Description: definition.Description,
			Schedule:    definition.Schedule,
			ManualOnly:  definition.ManualOnly,
			Running:     s.isRunning(definition.Name),
			LastRun:     lastRun,
		},
		RecentRuns: recentRuns,
	}, nil
}

func (s *Service) Trigger(ctx context.Context, name string, input TriggerInput) (*RunRecord, error) {
	definition, ok := s.definitions[strings.TrimSpace(name)]
	if !ok {
		return nil, fmt.Errorf("job not found: %s", name)
	}
	source := strings.TrimSpace(input.Source)
	if source == "" {
		source = "manual"
	}
	triggerID := strings.TrimSpace(input.TriggerID)
	if triggerID == "" {
		triggerID = fmt.Sprintf("%s:%d", definition.Name, time.Now().UnixNano())
	}

	targetVersion, err := s.targetVersion(ctx, definition.Name)
	if err != nil {
		return nil, err
	}
	idempotencyKey := definition.Name + ":" + targetVersion

	if previous, err := s.findLatestSuccessfulByKey(ctx, definition.Name, idempotencyKey); err == nil && previous != nil {
		return s.createTerminalRun(ctx, definition.Name, source, triggerID, RunStatusSkipped, idempotencyKey, targetVersion,
			"snapshot unchanged", "")
	}

	if !s.tryMarkRunning(definition.Name) {
		return s.createTerminalRun(ctx, definition.Name, source, triggerID, RunStatusSkipped, idempotencyKey, targetVersion,
			"job is already running", "")
	}
	defer s.markDone(definition.Name)

	runID, startedAt, err := s.createRunningRun(ctx, definition.Name, source, triggerID, idempotencyKey, targetVersion)
	if err != nil {
		return nil, err
	}

	message, execErr := s.execute(ctx, definition.Name)
	status := RunStatusSuccess
	errorText := ""
	if execErr != nil {
		status = RunStatusFailed
		errorText = execErr.Error()
		if strings.TrimSpace(message) == "" {
			message = "job execution failed"
		}
	}
	return s.finishRun(ctx, runID, startedAt, status, message, errorText)
}

func (s *Service) jobsForHook(trigger string) []string {
	trigger = strings.TrimSpace(strings.ToLower(trigger))
	if strings.HasPrefix(trigger, "consumer") || strings.HasPrefix(trigger, "org-") {
		return []string{"portal-consumer-projection", "portal-consumer-level-auth-reconcile"}
	}
	if strings.HasPrefix(trigger, "model-binding-") || trigger == "ai-route-save" || trigger == "ai-route-delete" {
		return []string{"ai-model-rate-limit-reconcile"}
	}
	return nil
}

func (s *Service) targetVersion(ctx context.Context, name string) (string, error) {
	var snapshot any
	var err error
	switch name {
	case "portal-consumer-projection":
		snapshot, err = s.snapshotPortalConsumers(ctx)
	case "portal-consumer-level-auth-reconcile":
		snapshot, err = s.snapshotConsumerLevelAuth(ctx)
	case "ai-sensitive-projection":
		snapshot, err = s.snapshotAISensitive(ctx)
	case "ai-model-rate-limit-reconcile":
		snapshot, err = s.snapshotAIModelRateLimitProjection(ctx)
	case "ai-plugin-execution-order-reconcile":
		snapshot, err = s.snapshotPluginOrder(ctx)
	default:
		return "", fmt.Errorf("unsupported job: %s", name)
	}
	if err != nil {
		return "", err
	}
	return hashJSON(snapshot), nil
}

func (s *Service) execute(ctx context.Context, name string) (string, error) {
	switch name {
	case "portal-consumer-projection":
		return s.executePortalConsumerProjection(ctx)
	case "portal-consumer-level-auth-reconcile":
		return s.executePortalConsumerLevelAuthReconcile(ctx)
	case "ai-sensitive-projection":
		return s.executeAISensitiveProjection(ctx)
	case "ai-model-rate-limit-reconcile":
		return s.executeAIModelRateLimitReconcile(ctx)
	case "ai-plugin-execution-order-reconcile":
		return s.executePluginExecutionOrderReconcile(ctx)
	default:
		return "", fmt.Errorf("unsupported job: %s", name)
	}
}

func (s *Service) executePortalConsumerProjection(ctx context.Context) (string, error) {
	if s.portal == nil || s.k8s == nil {
		return "", errors.New("portal consumer projection dependencies are unavailable")
	}
	consumers, err := s.portal.ListConsumers(ctx)
	if err != nil {
		return "", err
	}
	desired := map[string]map[string]any{}
	for _, consumer := range consumers {
		desired[consumer.Name] = map[string]any{
			"name":             consumer.Name,
			"displayName":      consumer.PortalDisplayName,
			"email":            consumer.PortalEmail,
			"department":       consumer.Department,
			"status":           consumer.PortalStatus,
			"userLevel":        consumer.PortalUserLevel,
			"source":           consumer.PortalUserSource,
			"tempPassword":     consumer.PortalTempPassword,
			"projectionSource": "portaldb",
		}
	}

	existing, err := s.k8s.ListResources(ctx, "consumers")
	if err != nil {
		return "", err
	}
	existingNames := map[string]struct{}{}
	for _, item := range existing {
		name := strings.TrimSpace(fmt.Sprint(item["name"]))
		if name == "" {
			continue
		}
		existingNames[name] = struct{}{}
	}

	updated := 0
	for name, payload := range desired {
		if _, err := s.k8s.UpsertResource(ctx, "consumers", name, payload); err != nil {
			return "", err
		}
		updated++
		delete(existingNames, name)
	}

	deleted := 0
	for name := range existingNames {
		if err := s.k8s.DeleteResource(ctx, "consumers", name); err != nil && !errors.Is(err, k8sclient.ErrNotFound) {
			return "", err
		}
		deleted++
	}
	return fmt.Sprintf("projected %d portal consumers, cleaned %d stale resources", updated, deleted), nil
}

func (s *Service) executePortalConsumerLevelAuthReconcile(ctx context.Context) (string, error) {
	if s.portal == nil || s.k8s == nil {
		return "", errors.New("consumer level auth reconcile dependencies are unavailable")
	}
	accounts, err := s.portal.ListAccounts(ctx)
	if err != nil {
		return "", err
	}
	consumersByLevel := map[string][]string{}
	for _, account := range accounts {
		if strings.EqualFold(account.Status, "disabled") || strings.TrimSpace(account.ConsumerName) == "" {
			continue
		}
		level := strings.TrimSpace(strings.ToLower(account.UserLevel))
		if level == "" {
			continue
		}
		consumersByLevel[level] = append(consumersByLevel[level], account.ConsumerName)
	}
	for level := range consumersByLevel {
		consumersByLevel[level] = normalizeUniqueStrings(consumersByLevel[level])
	}

	departmentTree, err := s.portal.ListDepartmentTree(ctx)
	if err != nil {
		return "", err
	}

	routeUpdated, err := s.reconcileResourceAuth(ctx, "routes", "authConfig", consumersByLevel, departmentTree)
	if err != nil {
		return "", err
	}
	aiRouteUpdated, err := s.reconcileResourceAuth(ctx, "ai-routes", "authConfig", consumersByLevel, departmentTree)
	if err != nil {
		return "", err
	}
	mcpUpdated, err := s.reconcileMCPAuth(ctx, consumersByLevel)
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("reconciled routeUpdated=%d aiRouteUpdated=%d mcpUpdated=%d", routeUpdated, aiRouteUpdated, mcpUpdated), nil
}

func (s *Service) executeAISensitiveProjection(ctx context.Context) (string, error) {
	db, err := s.db(ctx)
	if err != nil {
		return "", err
	}
	if s.k8s == nil {
		return "", errors.New("ai sensitive projection requires k8s client")
	}

	detectRules, err := queryRows(ctx, db, `
		SELECT id, pattern, match_type, description, priority, enabled, created_at, updated_at
		FROM portal_ai_sensitive_detect_rule
		ORDER BY priority DESC, id ASC`)
	if err != nil {
		return "", err
	}
	replaceRules, err := queryRows(ctx, db, `
		SELECT id, pattern, replace_type, replace_value, restore, description, priority, enabled, created_at, updated_at
		FROM portal_ai_sensitive_replace_rule
		ORDER BY priority DESC, id ASC`)
	if err != nil {
		return "", err
	}
	systemConfig, err := queryRows(ctx, db, `
		SELECT config_key, system_deny_enabled, dictionary_text, updated_by, updated_at
		FROM portal_ai_sensitive_system_config
		ORDER BY config_key`)
	if err != nil {
		return "", err
	}
	systemProjection := map[string]any{
		"systemDenyEnabled": false,
	}
	if len(systemConfig) == 0 {
		systemProjection["updatedBy"] = ""
		systemProjection["updatedAt"] = nil
	} else {
		item := systemConfig[0]
		systemProjection["systemDenyEnabled"] = toInt(item["system_deny_enabled"]) != 0
		systemProjection["updatedBy"] = strings.TrimSpace(fmt.Sprint(item["updated_by"]))
		systemProjection["updatedAt"] = item["updated_at"]
	}

	runtimeConfig := map[string]any{}
	if existing, err := s.k8s.GetResource(ctx, "ai-sensitive-projections", "default"); err == nil {
		if raw, ok := existing["runtimeConfig"].(map[string]any); ok {
			runtimeConfig = map[string]any{}
			for key, value := range raw {
				runtimeConfig[key] = value
			}
		}
	}

	payload := map[string]any{
		"name":          "default",
		"detectRules":   detectRules,
		"replaceRules":  replaceRules,
		"systemConfig":  systemProjection,
		"runtimeConfig": runtimeConfig,
		"projectedAt":   time.Now().UTC().Format(time.RFC3339),
	}
	if _, err := s.k8s.UpsertResource(ctx, "ai-sensitive-projections", "default", payload); err != nil {
		return "", err
	}
	return fmt.Sprintf("projected detectRules=%d replaceRules=%d systemConfigs=%d", len(detectRules), len(replaceRules), len(systemConfig)), nil
}

type modelBindingRateLimit struct {
	BindingID    string
	AssetID      string
	ModelID      string
	Status       string
	RPM          int
	TPM          int
	EffectiveRPM int
	EffectiveTPM int
}

func (s *Service) executeAIModelRateLimitReconcile(ctx context.Context) (string, error) {
	db, err := s.db(ctx)
	if err != nil {
		return "", err
	}
	if s.k8s == nil {
		return "", errors.New("ai model rate limit reconcile requires k8s client")
	}

	bindings, duplicates, err := queryPublishedModelBindingRateLimits(ctx, db)
	if err != nil {
		return "", err
	}
	routes, err := s.listGatewayKind(ctx, "ai-routes")
	if err != nil {
		return "", err
	}

	rules := make([]map[string]any, 0)
	skipped := make([]map[string]any, 0)
	projectedRoutes := 0
	projectedRPMRules := 0
	projectedTPMRules := 0

	for _, route := range routes {
		routeName := strings.TrimSpace(fmt.Sprint(route["name"]))
		if routeName == "" {
			continue
		}
		modelID, skipReason := resolveAIRouteModelRateLimitEligibility(route, bindings, duplicates)
		if skipReason != "" {
			skipped = append(skipped, map[string]any{
				"routeName": routeName,
				"reason":    skipReason,
			})
			continue
		}
		binding := bindings[modelID]
		targets := aiRouteModelRateLimitRuntimeTargets(route)
		if len(targets) == 0 {
			skipped = append(skipped, map[string]any{
				"routeName": routeName,
				"modelId":   modelID,
				"reason":    "route has no runtime ingress targets",
			})
			continue
		}
		if binding.EffectiveRPM <= 0 && binding.EffectiveTPM <= 0 {
			skipped = append(skipped, map[string]any{
				"routeName": routeName,
				"modelId":   modelID,
				"reason":    "published binding has no positive rpm/tpm limit",
			})
			continue
		}
		projectedRoutes++
		for _, ingressName := range targets {
			if binding.EffectiveRPM > 0 {
				rules = append(rules, map[string]any{
					"pluginName": modelRateLimitPluginRPM,
					"ingress":    ingressName,
					"config": buildAIModelRPMRuleConfig(
						routeName,
						modelID,
						binding.EffectiveRPM,
						s.k8s,
						ctx,
					),
				})
				projectedRPMRules++
			}
			if binding.EffectiveTPM > 0 {
				rules = append(rules, map[string]any{
					"pluginName": modelRateLimitPluginTPM,
					"ingress":    ingressName,
					"config": buildAIModelTPMRuleConfig(
						routeName,
						modelID,
						binding.EffectiveTPM,
						s.k8s,
						ctx,
					),
				})
				projectedTPMRules++
			}
		}
	}

	published := make([]map[string]any, 0, len(bindings))
	modelIDs := make([]string, 0, len(bindings))
	for modelID := range bindings {
		modelIDs = append(modelIDs, modelID)
	}
	sort.Strings(modelIDs)
	for _, modelID := range modelIDs {
		binding := bindings[modelID]
		published = append(published, map[string]any{
			"bindingId":    binding.BindingID,
			"assetId":      binding.AssetID,
			"modelId":      binding.ModelID,
			"status":       binding.Status,
			"rpm":          binding.RPM,
			"tpm":          binding.TPM,
			"effectiveRPM": binding.EffectiveRPM,
			"effectiveTPM": binding.EffectiveTPM,
		})
	}

	payload := map[string]any{
		"name":             modelRateLimitProjectionName,
		"rules":            rules,
		"publishedModels":  published,
		"duplicatedModels": sortedKeys(duplicates),
		"skippedRoutes":    skipped,
		"projectedAt":      time.Now().UTC().Format(time.RFC3339),
	}
	if _, err := s.k8s.UpsertResource(ctx, modelRateLimitProjectionKind, modelRateLimitProjectionName, payload); err != nil {
		return "", err
	}
	return fmt.Sprintf(
		"projected routes=%d rpmRules=%d tpmRules=%d skipped=%d",
		projectedRoutes,
		projectedRPMRules,
		projectedTPMRules,
		len(skipped),
	), nil
}

func (s *Service) executePluginExecutionOrderReconcile(ctx context.Context) (string, error) {
	if s.k8s == nil {
		return "", errors.New("plugin execution order reconcile requires k8s client")
	}
	type desiredPlugin struct {
		Phase    string
		Priority int
	}
	desired := map[string]desiredPlugin{
		"ai-statistics":   {Phase: "STATS", Priority: 900},
		"ai-data-masking": {Phase: "AUTHN", Priority: 100},
	}

	updated := 0
	for name, config := range desired {
		item, err := s.k8s.GetResource(ctx, "wasm-plugins", name)
		if err != nil && !errors.Is(err, k8sclient.ErrNotFound) {
			return "", err
		}
		if item == nil {
			item = map[string]any{"name": name, "builtIn": true}
		}
		currentPhase := strings.TrimSpace(fmt.Sprint(item["phase"]))
		currentPriority := toInt(item["priority"])
		if currentPhase == config.Phase && currentPriority == config.Priority {
			continue
		}
		item["phase"] = config.Phase
		item["priority"] = config.Priority
		item["builtIn"] = true
		if _, err := s.k8s.UpsertResource(ctx, "wasm-plugins", name, item); err != nil {
			return "", err
		}
		updated++
	}
	return fmt.Sprintf("reconciled %d built-in plugin execution orders", updated), nil
}

func (s *Service) reconcileResourceAuth(
	ctx context.Context,
	kind string,
	authField string,
	consumersByLevel map[string][]string,
	departmentTree []*portalsvc.OrgDepartmentNode,
) (int, error) {
	items, err := s.listGatewayKind(ctx, kind)
	if err != nil {
		return 0, err
	}
	departmentDescendants := indexDepartmentDescendants(departmentTree)
	updated := 0
	for _, item := range items {
		name := strings.TrimSpace(fmt.Sprint(item["name"]))
		if name == "" {
			continue
		}
		authConfig, _ := item[authField].(map[string]any)
		if len(authConfig) == 0 {
			continue
		}
		levels := normalizeUniqueStrings(readStringList(authConfig["allowedConsumerLevels"]))
		departments := normalizeUniqueStrings(readStringList(authConfig["allowedDepartments"]))
		if len(levels) == 0 && len(departments) == 0 {
			continue
		}
		resolvedConsumers := normalizeUniqueStrings(expandAllowedConsumersWithDepartments(
			ctx,
			readStringList(authConfig["allowedConsumers"]),
			levels,
			departments,
			consumersByLevel,
			departmentDescendants,
			s.portal,
		))
		if slices.Equal(readStringList(authConfig["allowedConsumers"]), resolvedConsumers) {
			continue
		}
		authConfig["allowedConsumers"] = resolvedConsumers
		item[authField] = authConfig
		if _, err := s.k8s.UpsertResource(ctx, kind, name, item); err != nil {
			return updated, err
		}
		updated++
	}
	return updated, nil
}

func (s *Service) reconcileMCPAuth(ctx context.Context, consumersByLevel map[string][]string) (int, error) {
	items, err := s.listGatewayKind(ctx, "mcp-servers")
	if err != nil {
		return 0, err
	}
	updated := 0
	for _, item := range items {
		name := strings.TrimSpace(fmt.Sprint(item["name"]))
		if name == "" {
			continue
		}
		authInfo, _ := item["consumerAuthInfo"].(map[string]any)
		if len(authInfo) == 0 {
			continue
		}
		levels := normalizeUniqueStrings(readStringList(authInfo["allowedConsumerLevels"]))
		if len(levels) == 0 {
			continue
		}
		resolvedConsumers := normalizeUniqueStrings(expandAllowedConsumers(readStringList(authInfo["allowedConsumers"]), levels, consumersByLevel))
		if slices.Equal(readStringList(authInfo["allowedConsumers"]), resolvedConsumers) {
			continue
		}
		authInfo["allowedConsumers"] = resolvedConsumers
		item["consumerAuthInfo"] = authInfo
		if _, err := s.k8s.UpsertResource(ctx, "mcp-servers", name, item); err != nil {
			return updated, err
		}
		updated++
	}
	return updated, nil
}

func expandAllowedConsumers(existing []string, levels []string, consumersByLevel map[string][]string) []string {
	return expandAllowedConsumersWithDepartments(context.Background(), existing, levels, nil, consumersByLevel, nil, nil)
}

func expandAllowedConsumersWithDepartments(
	ctx context.Context,
	existing []string,
	levels []string,
	departments []string,
	consumersByLevel map[string][]string,
	departmentDescendants map[string][]string,
	portal portalReader,
) []string {
	result := append([]string{}, existing...)
	for _, level := range levels {
		result = append(result, consumersByLevel[strings.TrimSpace(strings.ToLower(level))]...)
	}
	if len(departments) == 0 || portal == nil {
		return result
	}

	accounts, err := portal.ListAccounts(ctx)
	if err != nil {
		return result
	}
	allowedDepartments := map[string]struct{}{}
	for _, departmentID := range departments {
		trimmed := strings.TrimSpace(departmentID)
		if trimmed == "" {
			continue
		}
		allowedDepartments[trimmed] = struct{}{}
		for _, childID := range departmentDescendants[trimmed] {
			allowedDepartments[childID] = struct{}{}
		}
	}
	for _, account := range accounts {
		if strings.EqualFold(strings.TrimSpace(account.Status), "disabled") {
			continue
		}
		if _, ok := allowedDepartments[strings.TrimSpace(account.DepartmentID)]; ok {
			result = append(result, strings.TrimSpace(account.ConsumerName))
		}
	}
	return result
}

func indexDepartmentDescendants(nodes []*portalsvc.OrgDepartmentNode) map[string][]string {
	result := map[string][]string{}
	var collect func(items []*portalsvc.OrgDepartmentNode) []string
	collect = func(items []*portalsvc.OrgDepartmentNode) []string {
		ids := make([]string, 0)
		for _, item := range items {
			if item == nil {
				continue
			}
			childIDs := collect(item.Children)
			result[item.DepartmentID] = append([]string{}, childIDs...)
			ids = append(ids, item.DepartmentID)
			ids = append(ids, childIDs...)
		}
		return ids
	}
	collect(nodes)
	return result
}

func readStringList(value any) []string {
	switch typed := value.(type) {
	case []string:
		return normalizeUniqueStrings(typed)
	case []any:
		items := make([]string, 0, len(typed))
		for _, item := range typed {
			items = append(items, strings.TrimSpace(fmt.Sprint(item)))
		}
		return normalizeUniqueStrings(items)
	default:
		return []string{}
	}
}

func normalizeUniqueStrings(items []string) []string {
	set := map[string]struct{}{}
	result := make([]string, 0, len(items))
	for _, item := range items {
		trimmed := strings.TrimSpace(item)
		if trimmed == "" {
			continue
		}
		if _, ok := set[trimmed]; ok {
			continue
		}
		set[trimmed] = struct{}{}
		result = append(result, trimmed)
	}
	sort.Strings(result)
	return result
}

func (s *Service) snapshotPortalConsumers(ctx context.Context) (any, error) {
	consumers, err := s.portal.ListConsumers(ctx)
	if err != nil {
		return nil, err
	}
	return consumers, nil
}

func (s *Service) snapshotConsumerLevelAuth(ctx context.Context) (any, error) {
	accounts, err := s.portal.ListAccounts(ctx)
	if err != nil {
		return nil, err
	}
	routes, err := s.listGatewayKind(ctx, "routes")
	if err != nil {
		return nil, err
	}
	aiRoutes, err := s.listGatewayKind(ctx, "ai-routes")
	if err != nil {
		return nil, err
	}
	mcpServers, err := s.listGatewayKind(ctx, "mcp-servers")
	if err != nil {
		return nil, err
	}
	return map[string]any{
		"accounts":   accounts,
		"routes":     routes,
		"aiRoutes":   aiRoutes,
		"mcpServers": mcpServers,
	}, nil
}

func (s *Service) snapshotAISensitive(ctx context.Context) (any, error) {
	db, err := s.db(ctx)
	if err != nil {
		return nil, err
	}
	return map[string]any{
		"detectCount":  queryInt(ctx, db, `SELECT COUNT(1) FROM portal_ai_sensitive_detect_rule`),
		"replaceCount": queryInt(ctx, db, `SELECT COUNT(1) FROM portal_ai_sensitive_replace_rule`),
		"configCount":  queryInt(ctx, db, `SELECT COUNT(1) FROM portal_ai_sensitive_system_config`),
		"auditCount":   queryInt(ctx, db, `SELECT COUNT(1) FROM portal_ai_sensitive_block_audit`),
	}, nil
}

func (s *Service) snapshotAIModelRateLimitProjection(ctx context.Context) (any, error) {
	db, err := s.db(ctx)
	if err != nil {
		return nil, err
	}
	bindings, duplicates, err := queryPublishedModelBindingRateLimits(ctx, db)
	if err != nil {
		return nil, err
	}
	routes, err := s.listGatewayKind(ctx, "ai-routes")
	if err != nil {
		return nil, err
	}
	items := make([]map[string]any, 0, len(routes))
	for _, route := range routes {
		routeName := strings.TrimSpace(fmt.Sprint(route["name"]))
		if routeName == "" {
			continue
		}
		modelID, reason := resolveAIRouteModelRateLimitEligibility(route, bindings, duplicates)
		items = append(items, map[string]any{
			"name":          routeName,
			"modelId":       modelID,
			"targetIngress": aiRouteModelRateLimitRuntimeTargets(route),
			"skipReason":    reason,
		})
	}
	sort.Slice(items, func(i, j int) bool {
		return strings.TrimSpace(fmt.Sprint(items[i]["name"])) < strings.TrimSpace(fmt.Sprint(items[j]["name"]))
	})
	published := make([]map[string]any, 0, len(bindings))
	for _, modelID := range sortedKeys(bindings) {
		binding := bindings[modelID]
		published = append(published, map[string]any{
			"modelId":      binding.ModelID,
			"bindingId":    binding.BindingID,
			"rpm":          binding.RPM,
			"tpm":          binding.TPM,
			"effectiveRPM": binding.EffectiveRPM,
			"effectiveTPM": binding.EffectiveTPM,
		})
	}
	return map[string]any{
		"publishedModels":  published,
		"duplicatedModels": sortedKeys(duplicates),
		"routes":           items,
	}, nil
}

func (s *Service) snapshotPluginOrder(ctx context.Context) (any, error) {
	items, err := s.listGatewayKind(ctx, "wasm-plugins")
	if err != nil {
		return nil, err
	}
	filtered := make([]map[string]any, 0)
	for _, item := range items {
		name := strings.TrimSpace(fmt.Sprint(item["name"]))
		if name == "ai-statistics" || name == "ai-data-masking" {
			filtered = append(filtered, map[string]any{
				"name":     name,
				"phase":    item["phase"],
				"priority": item["priority"],
			})
		}
	}
	sort.Slice(filtered, func(i, j int) bool {
		return fmt.Sprint(filtered[i]["name"]) < fmt.Sprint(filtered[j]["name"])
	})
	return filtered, nil
}

func queryPublishedModelBindingRateLimits(ctx context.Context, db *sql.DB) (map[string]modelBindingRateLimit, map[string]struct{}, error) {
	rows, err := queryRows(ctx, db, `
		SELECT binding_id, asset_id, model_id, status, rpm, tpm
		FROM portal_model_binding
		WHERE status = 'published'
		ORDER BY model_id ASC, binding_id ASC`)
	if err != nil {
		return nil, nil, err
	}
	result := make(map[string]modelBindingRateLimit, len(rows))
	duplicates := map[string]struct{}{}
	for _, row := range rows {
		modelID := strings.TrimSpace(fmt.Sprint(row["model_id"]))
		if modelID == "" {
			continue
		}
		if _, exists := result[modelID]; exists {
			duplicates[modelID] = struct{}{}
			continue
		}
		rpm := toInt(row["rpm"])
		tpm := toInt(row["tpm"])
		result[modelID] = modelBindingRateLimit{
			BindingID:    strings.TrimSpace(fmt.Sprint(row["binding_id"])),
			AssetID:      strings.TrimSpace(fmt.Sprint(row["asset_id"])),
			ModelID:      modelID,
			Status:       strings.TrimSpace(fmt.Sprint(row["status"])),
			RPM:          rpm,
			TPM:          tpm,
			EffectiveRPM: effectiveModelRateLimit(rpm),
			EffectiveTPM: effectiveModelRateLimit(tpm),
		}
	}
	return result, duplicates, nil
}

func resolveAIRouteModelRateLimitEligibility(
	route map[string]any,
	bindings map[string]modelBindingRateLimit,
	duplicates map[string]struct{},
) (string, string) {
	modelPredicates := toMapSlice(route["modelPredicates"])
	switch len(modelPredicates) {
	case 0:
		return "", "modelPredicates is empty"
	case 1:
	default:
		return "", "modelPredicates must contain exactly one predicate"
	}
	predicate := modelPredicates[0]
	matchType := strings.ToUpper(strings.TrimSpace(fmt.Sprint(predicate["matchType"])))
	if matchType != "EQUAL" {
		return "", "modelPredicates only supports EQUAL for automatic rate limit projection"
	}
	modelID := strings.TrimSpace(fmt.Sprint(predicate["matchValue"]))
	if modelID == "" {
		return "", "modelPredicates.matchValue is empty"
	}
	if _, duplicated := duplicates[modelID]; duplicated {
		return "", "modelId matches multiple published bindings"
	}
	if _, ok := bindings[modelID]; !ok {
		return "", "modelId does not match any published binding"
	}
	return modelID, ""
}

func aiRouteModelRateLimitRuntimeTargets(route map[string]any) []string {
	name := strings.TrimSpace(fmt.Sprint(route["name"]))
	if name == "" {
		return nil
	}
	targets := []string{
		"ai-route-" + name + consts.InternalResourceNameSuffix,
		"ai-route-" + name + consts.InternalResourceNameSuffix + "-internal",
	}
	fallback := mapValue(route["fallbackConfig"])
	if boolValue(firstNonNil(fallback["enabled"], fallback["enable"])) && len(toMapSlice(fallback["upstreams"])) > 0 {
		fallbackName := "ai-route-" + name + ".fallback" + consts.InternalResourceNameSuffix
		targets = append(targets, fallbackName, fallbackName+"-internal")
	}
	return targets
}

func buildAIModelRPMRuleConfig(routeName, modelID string, rpm int, client k8sclient.Client, ctx context.Context) map[string]any {
	redisServiceName, redisPassword := resolveRedisRuntimeSettings(client, ctx)
	return map[string]any{
		"rule_name": modelRateLimitRuleNameRPMPrefix + strings.TrimSpace(routeName) + ":" + strings.TrimSpace(modelID),
		"rule_items": []map[string]any{{
			"limit_by_per_consumer": "",
			"limit_keys": []map[string]any{{
				"key":              "*",
				"query_per_minute": rpm,
			}},
		}},
		"redis": buildRedisRuntimeConfigPayload(redisServiceName, redisPassword),
	}
}

func buildAIModelTPMRuleConfig(routeName, modelID string, tpm int, client k8sclient.Client, ctx context.Context) map[string]any {
	redisServiceName, redisPassword := resolveRedisRuntimeSettings(client, ctx)
	return map[string]any{
		"rule_name":  modelRateLimitRuleNameTPMPrefix + strings.TrimSpace(routeName) + ":" + strings.TrimSpace(modelID),
		"limit_unit": "token",
		"rule_items": []map[string]any{{
			"limit_by_per_consumer": "",
			"limit_keys": []map[string]any{{
				"key":              "*",
				"token_per_minute": tpm,
			}},
		}},
		"redis": buildRedisRuntimeConfigPayload(redisServiceName, redisPassword),
	}
}

func resolveRedisRuntimeSettings(client k8sclient.Client, ctx context.Context) (string, string) {
	if client == nil {
		return "redis-server.aigateway-system.svc.cluster.local", "aigateway-redis"
	}
	return client.ResolveAIQuotaRedisServiceName(ctx), client.ResolveAIQuotaRedisPassword(ctx)
}

func buildRedisRuntimeConfigPayload(serviceName, password string) map[string]any {
	config := map[string]any{
		"service_name": strings.TrimSpace(serviceName),
		"service_port": 6379,
		"timeout":      1000,
		"database":     0,
	}
	if strings.TrimSpace(password) != "" {
		config["password"] = strings.TrimSpace(password)
	}
	return config
}

func effectiveModelRateLimit(value int) int {
	if value <= 0 {
		return 0
	}
	effective := (value * 70) / 100
	if effective < 1 {
		return 1
	}
	return effective
}

func sortedKeys[T any](items map[string]T) []string {
	result := make([]string, 0, len(items))
	for key := range items {
		result = append(result, key)
	}
	sort.Strings(result)
	return result
}

func queryInt(ctx context.Context, db *sql.DB, statement string, args ...any) int {
	var count int
	_ = db.QueryRowContext(ctx, statement, args...).Scan(&count)
	return count
}

func queryRows(ctx context.Context, db *sql.DB, statement string, args ...any) ([]map[string]any, error) {
	rows, err := db.QueryContext(ctx, statement, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	columns, err := rows.Columns()
	if err != nil {
		return nil, err
	}
	result := make([]map[string]any, 0)
	for rows.Next() {
		values := make([]any, len(columns))
		scanArgs := make([]any, len(columns))
		for i := range values {
			scanArgs[i] = &values[i]
		}
		if err := rows.Scan(scanArgs...); err != nil {
			return nil, err
		}
		record := map[string]any{}
		for i, column := range columns {
			record[column] = normalizeDBValue(values[i])
		}
		result = append(result, record)
	}
	return result, rows.Err()
}

func normalizeDBValue(value any) any {
	switch typed := value.(type) {
	case nil:
		return nil
	case []byte:
		return string(typed)
	case time.Time:
		return typed.UTC().Format(time.RFC3339)
	default:
		return typed
	}
}

func mapValue(value any) map[string]any {
	typed, _ := value.(map[string]any)
	if typed == nil {
		return map[string]any{}
	}
	return typed
}

func toMapSlice(value any) []map[string]any {
	switch typed := value.(type) {
	case []map[string]any:
		result := make([]map[string]any, 0, len(typed))
		for _, item := range typed {
			result = append(result, mapValue(item))
		}
		return result
	case []any:
		result := make([]map[string]any, 0, len(typed))
		for _, item := range typed {
			if mapped, ok := item.(map[string]any); ok {
				result = append(result, mapped)
			}
		}
		return result
	default:
		return nil
	}
}

func firstNonNil(values ...any) any {
	for _, value := range values {
		if value != nil {
			return value
		}
	}
	return nil
}

func boolValue(value any) bool {
	switch typed := value.(type) {
	case bool:
		return typed
	case string:
		return strings.EqualFold(strings.TrimSpace(typed), "true")
	default:
		return false
	}
}

func hashJSON(value any) string {
	bytes, _ := json.Marshal(value)
	sum := sha256.Sum256(bytes)
	return hex.EncodeToString(sum[:])
}

func toInt(value any) int {
	switch typed := value.(type) {
	case int:
		return typed
	case int32:
		return int(typed)
	case int64:
		return int(typed)
	case float64:
		return int(typed)
	case json.Number:
		i, _ := typed.Int64()
		return int(i)
	default:
		return 0
	}
}

func (s *Service) isRunning(jobName string) bool {
	s.runMu.RLock()
	defer s.runMu.RUnlock()
	return s.running[jobName]
}

func (s *Service) tryMarkRunning(jobName string) bool {
	s.runMu.Lock()
	defer s.runMu.Unlock()
	if s.running[jobName] {
		return false
	}
	s.running[jobName] = true
	return true
}

func (s *Service) markDone(jobName string) {
	s.runMu.Lock()
	defer s.runMu.Unlock()
	delete(s.running, jobName)
}

func (s *Service) createRunningRun(
	ctx context.Context,
	jobName, source, triggerID, idempotencyKey, targetVersion string,
) (int64, time.Time, error) {
	db, err := s.db(ctx)
	if err != nil {
		return 0, time.Time{}, err
	}
	startedAt := time.Now().UTC()
	id, err := portaldbclient.InsertReturningID(ctx, db, s.client.Driver(), `
		INSERT INTO job_run_record (
			job_name, trigger_source, trigger_id, status, idempotency_key, target_version, message, started_at
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?)`,
		jobName,
		source,
		triggerID,
		RunStatusRunning,
		nullIfEmpty(idempotencyKey),
		nullIfEmpty(targetVersion),
		"job started",
		startedAt,
	)
	if err != nil {
		return 0, time.Time{}, err
	}
	return id, startedAt, nil
}

func (s *Service) createTerminalRun(
	ctx context.Context,
	jobName, source, triggerID, status, idempotencyKey, targetVersion, message, errorText string,
) (*RunRecord, error) {
	db, err := s.db(ctx)
	if err != nil {
		return nil, err
	}
	startedAt := time.Now().UTC()
	finishedAt := startedAt
	duration := int64(0)
	id, err := portaldbclient.InsertReturningID(ctx, db, s.client.Driver(), `
		INSERT INTO job_run_record (
			job_name, trigger_source, trigger_id, status, idempotency_key, target_version, message, error_text, started_at, finished_at, duration_ms
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		jobName,
		source,
		triggerID,
		status,
		nullIfEmpty(idempotencyKey),
		nullIfEmpty(targetVersion),
		nullIfEmpty(message),
		nullIfEmpty(errorText),
		startedAt,
		finishedAt,
		duration,
	)
	if err != nil {
		return nil, err
	}
	return s.getRun(ctx, id)
}

func (s *Service) finishRun(
	ctx context.Context,
	id int64,
	startedAt time.Time,
	status, message, errorText string,
) (*RunRecord, error) {
	db, err := s.db(ctx)
	if err != nil {
		return nil, err
	}
	finishedAt := time.Now().UTC()
	duration := finishedAt.Sub(startedAt).Milliseconds()
	if _, err := db.ExecContext(ctx, `
		UPDATE job_run_record
		SET status = ?, message = ?, error_text = ?, finished_at = ?, duration_ms = ?
		WHERE id = ?`,
		status,
		nullIfEmpty(message),
		nullIfEmpty(errorText),
		finishedAt,
		duration,
		id,
	); err != nil {
		return nil, err
	}
	return s.getRun(ctx, id)
}

func (s *Service) latestRun(ctx context.Context, jobName string) (*RunRecord, error) {
	runs, err := s.listRuns(ctx, jobName, 1)
	if err != nil {
		return nil, err
	}
	if len(runs) == 0 {
		return nil, nil
	}
	return &runs[0], nil
}

func (s *Service) findLatestSuccessfulByKey(ctx context.Context, jobName, idempotencyKey string) (*RunRecord, error) {
	db, err := s.db(ctx)
	if err != nil {
		return nil, err
	}
	row := db.QueryRowContext(ctx, `
		SELECT id, job_name, trigger_source, trigger_id, status, idempotency_key, target_version, message,
			error_text, started_at, finished_at, duration_ms
		FROM job_run_record
		WHERE job_name = ? AND idempotency_key = ? AND status IN (?, ?)
		ORDER BY id DESC
		LIMIT 1`,
		jobName,
		idempotencyKey,
		RunStatusSuccess,
		RunStatusSkipped,
	)
	record, err := scanRunRecord(row)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	return &record, nil
}

func (s *Service) listRuns(ctx context.Context, jobName string, limit int) ([]RunRecord, error) {
	db, err := s.db(ctx)
	if err != nil {
		return nil, err
	}
	rows, err := db.QueryContext(ctx, `
		SELECT id, job_name, trigger_source, trigger_id, status, idempotency_key, target_version, message,
			error_text, started_at, finished_at, duration_ms
		FROM job_run_record
		WHERE job_name = ?
		ORDER BY id DESC
		LIMIT ?`,
		jobName,
		limit,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	result := make([]RunRecord, 0)
	for rows.Next() {
		record, err := scanRunRecord(rows)
		if err != nil {
			return nil, err
		}
		result = append(result, record)
	}
	return result, rows.Err()
}

func (s *Service) getRun(ctx context.Context, id int64) (*RunRecord, error) {
	db, err := s.db(ctx)
	if err != nil {
		return nil, err
	}
	row := db.QueryRowContext(ctx, `
		SELECT id, job_name, trigger_source, trigger_id, status, idempotency_key, target_version, message,
			error_text, started_at, finished_at, duration_ms
		FROM job_run_record
		WHERE id = ?`,
		id,
	)
	record, err := scanRunRecord(row)
	if err != nil {
		return nil, err
	}
	return &record, nil
}

func scanRunRecord(scanner interface{ Scan(...any) error }) (RunRecord, error) {
	var (
		record         RunRecord
		idempotencyKey sql.NullString
		targetVersion  sql.NullString
		message        sql.NullString
		errorText      sql.NullString
		startedAt      sql.NullTime
		finishedAt     sql.NullTime
		durationMs     sql.NullInt64
	)
	if err := scanner.Scan(
		&record.ID,
		&record.JobName,
		&record.TriggerSource,
		&record.TriggerID,
		&record.Status,
		&idempotencyKey,
		&targetVersion,
		&message,
		&errorText,
		&startedAt,
		&finishedAt,
		&durationMs,
	); err != nil {
		return RunRecord{}, err
	}
	record.IdempotencyKey = idempotencyKey.String
	record.TargetVersion = targetVersion.String
	record.Message = message.String
	record.ErrorText = errorText.String
	if startedAt.Valid {
		record.StartedAt = &startedAt.Time
	}
	if finishedAt.Valid {
		record.FinishedAt = &finishedAt.Time
	}
	if durationMs.Valid {
		value := durationMs.Int64
		record.DurationMs = &value
	}
	return record, nil
}

func (s *Service) db(ctx context.Context) (*sql.DB, error) {
	if s.client == nil || !s.client.Enabled() || s.client.DB() == nil {
		return nil, portaldbclient.ErrUnavailable
	}
	s.schemaMu.Lock()
	defer s.schemaMu.Unlock()
	if !s.schemaChecked {
		if err := s.client.EnsureSchema(ctx); err != nil {
			return nil, err
		}
		s.schemaChecked = true
	}
	return s.client.DB(), nil
}

func (s *Service) listGatewayKind(ctx context.Context, kind string) ([]map[string]any, error) {
	if s.gateway != nil {
		return s.gateway.List(ctx, kind)
	}
	if s.k8s == nil {
		return nil, errors.New("k8s client is unavailable")
	}
	return s.k8s.ListResources(ctx, kind)
}

func nullIfEmpty(value string) any {
	if strings.TrimSpace(value) == "" {
		return nil
	}
	return strings.TrimSpace(value)
}
