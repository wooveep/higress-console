package portal_test

import (
	"context"
	"database/sql"
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	portalshared "higress-portal-backend/schema/shared"

	jobssvc "github.com/wooveep/aigateway-console/backend/internal/service/jobs"
	portalsvc "github.com/wooveep/aigateway-console/backend/internal/service/portal"
	k8sclient "github.com/wooveep/aigateway-console/backend/utility/clients/k8s"
	portaldbclient "github.com/wooveep/aigateway-console/backend/utility/clients/portaldb"

	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/wait"
)

func TestPortalServiceUsesSharedSchemaAndConsoleOwnedTables(t *testing.T) {
	ctx := context.Background()
	db := openPortalPostgres(t, ctx, "console_portal_service_it")

	require.NoError(t, portalshared.ApplyToSQLWithDriver(ctx, db, "postgres"))
	_, err := db.ExecContext(ctx, `
		INSERT INTO org_department (department_id, name, parent_department_id, admin_consumer_name, path, level, sort_order, status)
		VALUES ('root', 'Root', NULL, NULL, 'Root', 0, 0, 'active')`)
	require.NoError(t, err)

	client := portaldbclient.NewFromDB(portaldbclient.Config{Enabled: true, Driver: "postgres", AutoMigrate: true}, db)
	k8s := k8sclient.NewMemoryClient()
	_, err = k8s.UpsertResource(ctx, "mcp-servers", "mcp-demo", map[string]any{
		"name": "mcp-demo",
	})
	require.NoError(t, err)
	svc := portalsvc.New(client, k8s)

	department, err := svc.CreateDepartment(ctx, portalsvc.DepartmentMutation{Name: "Engineering", ParentDepartmentID: "root"})
	require.NoError(t, err)
	require.Equal(t, "Engineering", department.Name)

	account, err := svc.CreateAccount(ctx, portalsvc.AccountMutation{
		ConsumerName: "demo",
		DisplayName:  "Demo",
		Email:        "demo@example.com",
		UserLevel:    "pro",
		Status:       "active",
		DepartmentID: department.DepartmentID,
		Password:     "secret",
	})
	require.NoError(t, err)
	require.Equal(t, department.DepartmentID, account.DepartmentID)

	consumer, err := svc.SaveConsumer(ctx, portalsvc.ConsumerMutation{
		Name:              "consumer-a",
		Department:        "Engineering",
		PortalStatus:      "active",
		PortalDisplayName: "Consumer A",
		PortalEmail:       "consumer-a@example.com",
		PortalUserLevel:   "plus",
		PortalPassword:    "secret",
	}, true)
	require.NoError(t, err)
	require.Equal(t, "consumer-a", consumer.Name)

	invite, err := svc.CreateInviteCode(ctx, 7)
	require.NoError(t, err)
	require.NotEmpty(t, invite.InviteCode)

	asset, err := svc.CreateModelAsset(ctx, portalsvc.ModelAsset{
		AssetID:       "qwen-plus",
		CanonicalName: "qwen-plus",
		DisplayName:   "Qwen Plus",
		Intro:         "test model",
		Tags:          []string{"chat"},
		Capabilities: &portalsvc.ModelAssetCapabilities{
			Modalities:   []string{"text"},
			Features:     []string{"reasoning"},
			RequestKinds: []string{"chat_completions"},
		},
	})
	require.NoError(t, err)
	require.Equal(t, "qwen-plus", asset.AssetID)

	binding, err := svc.CreateModelBinding(ctx, asset.AssetID, portalsvc.ModelAssetBinding{
		BindingID:    "binding-qwen-plus",
		ModelID:      "qwen-plus-model",
		ProviderName: "aliyun",
		TargetModel:  "qwen-plus",
		Protocol:     "openai/v1",
		Endpoint:     "/v1/chat/completions",
		Pricing:      &portalsvc.ModelBindingPricing{Currency: "CNY"},
		Limits:       &portalsvc.ModelBindingLimits{RPM: intPtr(60), TPM: intPtr(120000), ContextWindow: intPtr(8192)},
	})
	require.NoError(t, err)
	require.Equal(t, "binding-qwen-plus", binding.BindingID)

	binding, err = svc.PublishModelBinding(ctx, asset.AssetID, binding.BindingID)
	require.NoError(t, err)
	require.Equal(t, "published", binding.Status)

	grants, err := svc.ReplaceAssetGrants(ctx, "model_binding", binding.BindingID, []portalsvc.AssetGrantRecord{
		{SubjectType: "consumer", SubjectID: "demo"},
	})
	require.NoError(t, err)
	require.Len(t, grants, 1)

	agent, err := svc.CreateAgentCatalog(ctx, portalsvc.AgentCatalogRecord{
		AgentID:       "agent-demo",
		CanonicalName: "agent-demo",
		DisplayName:   "Agent Demo",
		Intro:         "agent intro",
		Description:   "agent description",
		McpServerName: "mcp-demo",
	})
	require.NoError(t, err)
	require.Equal(t, "agent-demo", agent.AgentID)

	agent, err = svc.PublishAgentCatalog(ctx, agent.AgentID)
	require.NoError(t, err)
	require.Equal(t, "published", agent.Status)

	policy, err := svc.SaveAIQuotaUserPolicy(ctx, "route-a", "demo", portalsvc.AIQuotaUserPolicyRequest{
		LimitTotal:     1000,
		Limit5h:        200,
		LimitDaily:     300,
		DailyResetMode: "fixed",
		DailyResetTime: "08:00",
		LimitWeekly:    400,
		LimitMonthly:   500,
	})
	require.NoError(t, err)
	require.EqualValues(t, 1000, policy.LimitTotal)

	config, err := svc.SaveAISensitiveSystemConfig(ctx, portalsvc.AISensitiveSystemConfig{
		SystemDenyEnabled: true,
		DictionaryText:    "alpha\nbeta",
		UpdatedBy:         "tester",
	})
	require.NoError(t, err)
	require.True(t, config.SystemDenyEnabled)

	jobs := jobssvc.New(client, svc, nil, k8sclient.NewMemoryClient())
	run, err := jobs.Trigger(ctx, "portal-consumer-projection", jobssvc.TriggerInput{
		Source:    "manual",
		TriggerID: "integration-test",
	})
	require.NoError(t, err)
	require.Equal(t, jobssvc.RunStatusSuccess, run.Status)

	var count int
	err = db.QueryRowContext(ctx, `SELECT COUNT(1) FROM portal_user WHERE consumer_name = 'demo'`).Scan(&count)
	require.NoError(t, err)
	require.Equal(t, 1, count)
	err = db.QueryRowContext(ctx, `SELECT COUNT(1) FROM quota_policy_user WHERE consumer_name = 'demo'`).Scan(&count)
	require.NoError(t, err)
	require.Equal(t, 1, count)
	err = db.QueryRowContext(ctx, `SELECT COUNT(1) FROM job_run_record WHERE job_name = 'portal-consumer-projection'`).Scan(&count)
	require.NoError(t, err)
	require.Equal(t, 1, count)
}

func openPortalPostgres(t *testing.T, ctx context.Context, databaseName string) *sql.DB {
	t.Helper()

	container, err := testcontainers.GenericContainer(ctx, testcontainers.GenericContainerRequest{
		ContainerRequest: testcontainers.ContainerRequest{
			Image:        "postgres:16-alpine",
			ExposedPorts: []string{"5432/tcp"},
			Env: map[string]string{
				"POSTGRES_PASSWORD": "postgres",
				"POSTGRES_USER":     "postgres",
				"POSTGRES_DB":       databaseName,
			},
			WaitingFor: wait.ForListeningPort("5432/tcp").WithStartupTimeout(90 * time.Second),
		},
		Started: true,
	})
	require.NoError(t, err)
	t.Cleanup(func() {
		_ = container.Terminate(context.Background())
	})

	host, err := container.Host(ctx)
	require.NoError(t, err)
	port, err := container.MappedPort(ctx, "5432/tcp")
	require.NoError(t, err)

	dsn := fmt.Sprintf("host=%s port=%s user=postgres password=postgres dbname=%s sslmode=disable", host, port.Port(), databaseName)
	db, err := sql.Open("pgx-rebind", dsn)
	require.NoError(t, err)
	t.Cleanup(func() {
		_ = db.Close()
	})

	require.Eventually(t, func() bool {
		return db.PingContext(ctx) == nil
	}, 30*time.Second, 500*time.Millisecond)
	return db
}

func intPtr(v int) *int {
	return &v
}
