package portal

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/gogf/gf/v2/os/gtime"
	"github.com/google/uuid"

	"github.com/wooveep/aigateway-console/backend/internal/model/do"
	"github.com/wooveep/aigateway-console/backend/internal/model/entity"
)

const builtinQuotaAdminConsumer = "administrator"

type AIQuotaMenuState struct {
	Enabled           bool `json:"enabled"`
	EnabledRouteCount int  `json:"enabledRouteCount"`
}

type AIQuotaRouteSummary struct {
	RouteName         string   `json:"routeName"`
	Domains           []string `json:"domains,omitempty"`
	Path              string   `json:"path,omitempty"`
	RedisKeyPrefix    string   `json:"redisKeyPrefix"`
	AdminConsumer     string   `json:"adminConsumer"`
	AdminPath         string   `json:"adminPath"`
	QuotaUnit         string   `json:"quotaUnit,omitempty"`
	ScheduleRuleCount int      `json:"scheduleRuleCount"`
}

type AIQuotaConsumerQuota struct {
	ConsumerName string `json:"consumerName"`
	Quota        int64  `json:"quota"`
}

type AIQuotaUserPolicy struct {
	ConsumerName   string     `json:"consumerName"`
	LimitTotal     int64      `json:"limitTotal"`
	Limit5h        int64      `json:"limit5h"`
	LimitDaily     int64      `json:"limitDaily"`
	DailyResetMode string     `json:"dailyResetMode"`
	DailyResetTime string     `json:"dailyResetTime"`
	LimitWeekly    int64      `json:"limitWeekly"`
	LimitMonthly   int64      `json:"limitMonthly"`
	CostResetAt    *time.Time `json:"costResetAt,omitempty"`
}

type AIQuotaUserPolicyRequest struct {
	LimitTotal     int64      `json:"limitTotal"`
	Limit5h        int64      `json:"limit5h"`
	LimitDaily     int64      `json:"limitDaily"`
	DailyResetMode string     `json:"dailyResetMode"`
	DailyResetTime string     `json:"dailyResetTime"`
	LimitWeekly    int64      `json:"limitWeekly"`
	LimitMonthly   int64      `json:"limitMonthly"`
	CostResetAt    *time.Time `json:"costResetAt"`
}

type AIQuotaScheduleRule struct {
	ID            string     `json:"id"`
	ConsumerName  string     `json:"consumerName"`
	Action        string     `json:"action"`
	Cron          string     `json:"cron"`
	Value         int64      `json:"value"`
	Enabled       bool       `json:"enabled"`
	CreatedAt     *time.Time `json:"createdAt,omitempty"`
	UpdatedAt     *time.Time `json:"updatedAt,omitempty"`
	LastAppliedAt *time.Time `json:"lastAppliedAt,omitempty"`
	LastError     string     `json:"lastError,omitempty"`
}

type AIQuotaScheduleRuleRequest struct {
	ID           string `json:"id"`
	ConsumerName string `json:"consumerName"`
	Action       string `json:"action"`
	Cron         string `json:"cron"`
	Value        int64  `json:"value"`
	Enabled      *bool  `json:"enabled"`
}

func (s *Service) GetAIQuotaMenuState(ctx context.Context) (*AIQuotaMenuState, error) {
	routes, err := s.ListAIQuotaRoutes(ctx)
	if err != nil {
		return nil, err
	}
	return &AIQuotaMenuState{
		Enabled:           len(routes) > 0,
		EnabledRouteCount: len(routes),
	}, nil
}

func (s *Service) ListAIQuotaRoutes(ctx context.Context) ([]AIQuotaRouteSummary, error) {
	if s.k8sClient == nil {
		return []AIQuotaRouteSummary{}, nil
	}
	rawRoutes, err := s.k8sClient.ListResources(ctx, "ai-routes")
	if err != nil {
		return nil, err
	}

	scheduleCounts := map[string]int{}
	if db, err := s.db(ctx); err == nil {
		items, queryErr := newPortalStore(db).listAIQuotaScheduleCounts(ctx)
		if queryErr == nil {
			scheduleCounts = items
		}
	}

	items := make([]AIQuotaRouteSummary, 0, len(rawRoutes))
	for _, item := range rawRoutes {
		name := stringValue(item["name"])
		if name == "" {
			continue
		}
		summary := AIQuotaRouteSummary{
			RouteName:         name,
			Domains:           readStringSlice(item["domains"]),
			Path:              firstNonEmpty(extractMatchValue(item["pathPredicate"]), extractMatchValue(item["path"]), "/"+name),
			RedisKeyPrefix:    firstNonEmpty(stringValue(item["redisKeyPrefix"]), "aigateway:quota:"+name),
			AdminConsumer:     firstNonEmpty(stringValue(item["adminConsumer"]), builtinQuotaAdminConsumer),
			AdminPath:         firstNonEmpty(stringValue(item["adminPath"]), "/v1/ai/quotas/routes/"+name+"/consumers"),
			QuotaUnit:         firstNonEmpty(stringValue(item["quotaUnit"]), "amount"),
			ScheduleRuleCount: scheduleCounts[name],
		}
		items = append(items, summary)
	}
	sort.Slice(items, func(i, j int) bool {
		return items[i].RouteName < items[j].RouteName
	})
	return items, nil
}

func (s *Service) ListAIQuotaConsumers(ctx context.Context, routeName string) ([]AIQuotaConsumerQuota, error) {
	db, err := s.db(ctx)
	if err != nil {
		return nil, err
	}
	routeName = strings.TrimSpace(routeName)
	if routeName == "" {
		return nil, errors.New("routeName cannot be blank")
	}

	balances := map[string]int64{}
	balanceItems, err := newPortalStore(db).listAIQuotaBalances(ctx, routeName)
	if err != nil {
		return nil, err
	}
	for _, item := range balanceItems {
		balances[item.ConsumerName] = item.Quota
	}

	names := map[string]struct{}{
		builtinQuotaAdminConsumer: {},
	}
	users, err := newPortalStore(db).listActivePortalUsers(ctx)
	if err != nil {
		return nil, err
	}
	for _, item := range users {
		names[item.ConsumerName] = struct{}{}
	}
	for consumerName := range balances {
		names[consumerName] = struct{}{}
	}

	consumers := make([]AIQuotaConsumerQuota, 0, len(names))
	for consumerName := range names {
		consumers = append(consumers, AIQuotaConsumerQuota{
			ConsumerName: consumerName,
			Quota:        balances[consumerName],
		})
	}
	sort.Slice(consumers, func(i, j int) bool {
		return consumers[i].ConsumerName < consumers[j].ConsumerName
	})
	return consumers, nil
}

func (s *Service) RefreshAIQuota(ctx context.Context, routeName, consumerName string, value int64) (*AIQuotaConsumerQuota, error) {
	db, err := s.db(ctx)
	if err != nil {
		return nil, err
	}
	routeName = strings.TrimSpace(routeName)
	consumerName = strings.TrimSpace(consumerName)
	if routeName == "" || consumerName == "" {
		return nil, errors.New("routeName and consumerName are required")
	}
	if err := newPortalStore(db).saveAIQuotaBalance(ctx, do.PortalAIQuotaBalance{
		RouteName:    routeName,
		ConsumerName: consumerName,
		Quota:        value,
	}); err != nil {
		return nil, err
	}
	return &AIQuotaConsumerQuota{ConsumerName: consumerName, Quota: value}, nil
}

func (s *Service) DeltaAIQuota(ctx context.Context, routeName, consumerName string, delta int64) (*AIQuotaConsumerQuota, error) {
	current, err := s.getAIQuotaBalance(ctx, routeName, consumerName)
	if err != nil {
		return nil, err
	}
	return s.RefreshAIQuota(ctx, routeName, consumerName, current+delta)
}

func (s *Service) GetAIQuotaUserPolicy(ctx context.Context, routeName, consumerName string) (*AIQuotaUserPolicy, error) {
	db, err := s.db(ctx)
	if err != nil {
		return nil, err
	}
	routeName = strings.TrimSpace(routeName)
	consumerName = strings.TrimSpace(consumerName)
	if routeName == "" || consumerName == "" {
		return nil, errors.New("routeName and consumerName are required")
	}

	policy := defaultAIQuotaPolicy(consumerName)
	item, err := newPortalStore(db).getAIQuotaUserPolicy(ctx, consumerName)
	if err != nil {
		return nil, err
	}
	if item == nil {
		return policy, nil
	}
	policy.LimitTotal = item.LimitTotalMicroYuan
	policy.Limit5h = item.Limit5hMicroYuan
	policy.LimitDaily = item.LimitDailyMicroYuan
	policy.DailyResetMode = item.DailyResetMode
	policy.DailyResetTime = item.DailyResetTime
	policy.LimitWeekly = item.LimitWeeklyMicroYuan
	policy.LimitMonthly = item.LimitMonthlyMicroYuan
	if item.CostResetAt != nil {
		value := item.CostResetAt.Time
		policy.CostResetAt = &value
	}
	return policy, nil
}

func (s *Service) SaveAIQuotaUserPolicy(
	ctx context.Context,
	routeName, consumerName string,
	request AIQuotaUserPolicyRequest,
) (*AIQuotaUserPolicy, error) {
	db, err := s.db(ctx)
	if err != nil {
		return nil, err
	}
	routeName = strings.TrimSpace(routeName)
	consumerName = strings.TrimSpace(consumerName)
	if routeName == "" || consumerName == "" {
		return nil, errors.New("routeName and consumerName are required")
	}

	mode := firstNonEmpty(strings.TrimSpace(request.DailyResetMode), "fixed")
	resetTime := firstNonEmpty(strings.TrimSpace(request.DailyResetTime), "00:00")
	var costResetAt *gtime.Time
	if request.CostResetAt != nil {
		costResetAt = wrapGTime(*request.CostResetAt)
	}
	if err := newPortalStore(db).saveAIQuotaUserPolicy(ctx, do.QuotaPolicyUser{
		ConsumerName:          consumerName,
		LimitTotalMicroYuan:   request.LimitTotal,
		Limit5hMicroYuan:      request.Limit5h,
		LimitDailyMicroYuan:   request.LimitDaily,
		DailyResetMode:        mode,
		DailyResetTime:        resetTime,
		LimitWeeklyMicroYuan:  request.LimitWeekly,
		LimitMonthlyMicroYuan: request.LimitMonthly,
		CostResetAt:           costResetAt,
	}); err != nil {
		return nil, err
	}
	return s.GetAIQuotaUserPolicy(ctx, routeName, consumerName)
}

func (s *Service) ListAIQuotaScheduleRules(ctx context.Context, routeName, consumerName string) ([]AIQuotaScheduleRule, error) {
	db, err := s.db(ctx)
	if err != nil {
		return nil, err
	}
	routeName = strings.TrimSpace(routeName)
	if routeName == "" {
		return nil, errors.New("routeName cannot be blank")
	}

	rows, err := newPortalStore(db).listAIQuotaScheduleRules(ctx, routeName, strings.TrimSpace(consumerName))
	if err != nil {
		return nil, err
	}
	items := make([]AIQuotaScheduleRule, 0, len(rows))
	for _, item := range rows {
		items = append(items, aiQuotaScheduleRuleFromEntity(item))
	}
	return items, nil
}

func (s *Service) SaveAIQuotaScheduleRule(
	ctx context.Context,
	routeName string,
	request AIQuotaScheduleRuleRequest,
) (*AIQuotaScheduleRule, error) {
	db, err := s.db(ctx)
	if err != nil {
		return nil, err
	}
	routeName = strings.TrimSpace(routeName)
	consumerName := strings.TrimSpace(request.ConsumerName)
	if routeName == "" || consumerName == "" {
		return nil, errors.New("routeName and consumerName are required")
	}
	action := strings.ToUpper(strings.TrimSpace(request.Action))
	if action != "REFRESH" && action != "DELTA" {
		return nil, errors.New("action must be REFRESH or DELTA")
	}
	cron := strings.TrimSpace(request.Cron)
	if cron == "" {
		return nil, errors.New("cron cannot be blank")
	}
	ruleID := strings.TrimSpace(request.ID)
	if ruleID == "" {
		ruleID = "quota-rule-" + uuid.NewString()[:8]
	}
	enabled := true
	if request.Enabled != nil {
		enabled = *request.Enabled
	}
	enabledValue := 0
	if enabled {
		enabledValue = 1
	}

	if err := newPortalStore(db).saveAIQuotaScheduleRule(ctx, do.PortalAIQuotaScheduleRule{
		Id:           ruleID,
		RouteName:    routeName,
		ConsumerName: consumerName,
		Action:       action,
		Cron:         cron,
		Value:        request.Value,
		Enabled:      enabledValue,
	}); err != nil {
		return nil, err
	}
	return s.getAIQuotaScheduleRule(ctx, routeName, ruleID)
}

func (s *Service) DeleteAIQuotaScheduleRule(ctx context.Context, routeName, ruleID string) error {
	db, err := s.db(ctx)
	if err != nil {
		return err
	}
	routeName = strings.TrimSpace(routeName)
	ruleID = strings.TrimSpace(ruleID)
	if routeName == "" || ruleID == "" {
		return errors.New("routeName and ruleId are required")
	}
	deleted, err := newPortalStore(db).deleteAIQuotaScheduleRule(ctx, routeName, ruleID)
	if err != nil {
		return err
	}
	if !deleted {
		return fmt.Errorf("schedule rule not found: %s", ruleID)
	}
	return nil
}

func (s *Service) getAIQuotaBalance(ctx context.Context, routeName, consumerName string) (int64, error) {
	db, err := s.db(ctx)
	if err != nil {
		return 0, err
	}
	return newPortalStore(db).getAIQuotaBalance(ctx, strings.TrimSpace(routeName), strings.TrimSpace(consumerName))
}

func (s *Service) getAIQuotaScheduleRule(ctx context.Context, routeName, ruleID string) (*AIQuotaScheduleRule, error) {
	items, err := s.ListAIQuotaScheduleRules(ctx, routeName, "")
	if err != nil {
		return nil, err
	}
	for _, item := range items {
		if item.ID == ruleID {
			return &item, nil
		}
	}
	return nil, fmt.Errorf("schedule rule not found: %s", ruleID)
}

func defaultAIQuotaPolicy(consumerName string) *AIQuotaUserPolicy {
	return &AIQuotaUserPolicy{
		ConsumerName:   consumerName,
		DailyResetMode: "fixed",
		DailyResetTime: "00:00",
	}
}

func extractMatchValue(value any) string {
	record, ok := value.(map[string]any)
	if !ok {
		return ""
	}
	return stringValue(record["matchValue"])
}

func readStringSlice(value any) []string {
	switch typed := value.(type) {
	case []string:
		result := append([]string{}, typed...)
		sort.Strings(result)
		return result
	case []any:
		items := make([]string, 0, len(typed))
		for _, item := range typed {
			trimmed := strings.TrimSpace(fmt.Sprint(item))
			if trimmed == "" {
				continue
			}
			items = append(items, trimmed)
		}
		sort.Strings(items)
		return items
	default:
		return []string{}
	}
}

func stringValue(value any) string {
	if value == nil {
		return ""
	}
	trimmed := strings.TrimSpace(fmt.Sprint(value))
	if trimmed == "" || trimmed == "<nil>" {
		return ""
	}
	return trimmed
}

func aiQuotaScheduleRuleFromEntity(item entity.PortalAIQuotaScheduleRule) AIQuotaScheduleRule {
	record := AIQuotaScheduleRule{
		ID:           item.Id,
		ConsumerName: item.ConsumerName,
		Action:       item.Action,
		Cron:         item.Cron,
		Value:        item.Value,
		Enabled:      item.Enabled > 0,
		LastError:    item.LastError,
	}
	if item.CreatedAt != nil {
		value := item.CreatedAt.Time
		record.CreatedAt = &value
	}
	if item.UpdatedAt != nil {
		value := item.UpdatedAt.Time
		record.UpdatedAt = &value
	}
	if item.LastAppliedAt != nil {
		value := item.LastAppliedAt.Time
		record.LastAppliedAt = &value
	}
	return record
}
