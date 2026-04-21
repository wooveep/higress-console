package jobs

import (
	"context"
	"database/sql"
	"regexp"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/stretchr/testify/require"

	portalsvc "github.com/wooveep/aigateway-console/backend/internal/service/portal"
	k8sclient "github.com/wooveep/aigateway-console/backend/utility/clients/k8s"
)

type stubPortal struct {
	consumers   []portalsvc.ConsumerRecord
	accounts    []portalsvc.OrgAccountRecord
	departments []*portalsvc.OrgDepartmentNode
}

type stubPortalDBClient struct {
	db *sql.DB
}

func (c stubPortalDBClient) Healthy(ctx context.Context) error { return nil }
func (c stubPortalDBClient) Enabled() bool                     { return true }
func (c stubPortalDBClient) DB() *sql.DB                       { return c.db }
func (c stubPortalDBClient) Driver() string                    { return "postgres" }
func (c stubPortalDBClient) EnsureSchema(ctx context.Context) error {
	return nil
}
func (c stubPortalDBClient) MigrateLegacyData(ctx context.Context) error { return nil }

func (s stubPortal) ListConsumers(ctx context.Context) ([]portalsvc.ConsumerRecord, error) {
	return append([]portalsvc.ConsumerRecord{}, s.consumers...), nil
}

func (s stubPortal) ListAccounts(ctx context.Context) ([]portalsvc.OrgAccountRecord, error) {
	return append([]portalsvc.OrgAccountRecord{}, s.accounts...), nil
}

func (s stubPortal) ListDepartmentTree(ctx context.Context) ([]*portalsvc.OrgDepartmentNode, error) {
	return append([]*portalsvc.OrgDepartmentNode{}, s.departments...), nil
}

func TestTriggerPortalConsumerProjectionAndSkipDuplicateSnapshot(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()
	mock.MatchExpectationsInOrder(false)
	mock.ExpectQuery(regexp.QuoteMeta(`
		SELECT id, job_name, trigger_source, trigger_id, status, idempotency_key, target_version, message,
			error_text, started_at, finished_at, duration_ms
		FROM job_run_record
		WHERE job_name = ? AND idempotency_key = ? AND status IN (?, ?)
		ORDER BY id DESC
		LIMIT 1`)).
		WithArgs("portal-consumer-projection", sqlmock.AnyArg(), RunStatusSuccess, RunStatusSkipped).
		WillReturnRows(sqlmock.NewRows([]string{
			"id", "job_name", "trigger_source", "trigger_id", "status", "idempotency_key", "target_version", "message",
			"error_text", "started_at", "finished_at", "duration_ms",
		}))
	mock.ExpectQuery(regexp.QuoteMeta(`
		INSERT INTO job_run_record (
			job_name, trigger_source, trigger_id, status, idempotency_key, target_version, message, started_at
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?) RETURNING id`)).
		WithArgs(
			"portal-consumer-projection",
			"manual",
			"manual-1",
			RunStatusRunning,
			sqlmock.AnyArg(),
			sqlmock.AnyArg(),
			"job started",
			sqlmock.AnyArg(),
		).
		WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(1))
	mock.ExpectExec(regexp.QuoteMeta(`
		UPDATE job_run_record
		SET status = ?, message = ?, error_text = ?, finished_at = ?, duration_ms = ?
		WHERE id = ?`)).
		WithArgs(RunStatusSuccess, sqlmock.AnyArg(), nil, sqlmock.AnyArg(), sqlmock.AnyArg(), int64(1)).
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectQuery(regexp.QuoteMeta(`
		SELECT id, job_name, trigger_source, trigger_id, status, idempotency_key, target_version, message,
			error_text, started_at, finished_at, duration_ms
		FROM job_run_record
		WHERE id = ?`)).
		WithArgs(int64(1)).
		WillReturnRows(sqlmock.NewRows([]string{
			"id", "job_name", "trigger_source", "trigger_id", "status", "idempotency_key", "target_version", "message",
			"error_text", "started_at", "finished_at", "duration_ms",
		}).AddRow(
			1, "portal-consumer-projection", "manual", "manual-1", RunStatusSuccess, "same", "same", "projected 1 portal consumers, cleaned 0 stale resources",
			nil, time.Now(), time.Now(), int64(1),
		))

	mock.ExpectQuery(regexp.QuoteMeta(`
		SELECT id, job_name, trigger_source, trigger_id, status, idempotency_key, target_version, message,
			error_text, started_at, finished_at, duration_ms
		FROM job_run_record
		WHERE job_name = ? AND idempotency_key = ? AND status IN (?, ?)
		ORDER BY id DESC
		LIMIT 1`)).
		WithArgs("portal-consumer-projection", sqlmock.AnyArg(), RunStatusSuccess, RunStatusSkipped).
		WillReturnRows(sqlmock.NewRows([]string{
			"id", "job_name", "trigger_source", "trigger_id", "status", "idempotency_key", "target_version", "message",
			"error_text", "started_at", "finished_at", "duration_ms",
		}).AddRow(
			1, "portal-consumer-projection", "manual", "manual-1", RunStatusSuccess, "same", "same", "projected 1 portal consumers, cleaned 0 stale resources",
			nil, time.Now(), time.Now(), int64(1),
		))
	mock.ExpectQuery(regexp.QuoteMeta(`
		INSERT INTO job_run_record (
			job_name, trigger_source, trigger_id, status, idempotency_key, target_version, message, error_text, started_at, finished_at, duration_ms
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?) RETURNING id`)).
		WithArgs(
			"portal-consumer-projection",
			"manual",
			"manual-2",
			RunStatusSkipped,
			sqlmock.AnyArg(),
			sqlmock.AnyArg(),
			"snapshot unchanged",
			nil,
			sqlmock.AnyArg(),
			sqlmock.AnyArg(),
			int64(0),
		).
		WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(2))
	mock.ExpectQuery(regexp.QuoteMeta(`
		SELECT id, job_name, trigger_source, trigger_id, status, idempotency_key, target_version, message,
			error_text, started_at, finished_at, duration_ms
		FROM job_run_record
		WHERE id = ?`)).
		WithArgs(int64(2)).
		WillReturnRows(sqlmock.NewRows([]string{
			"id", "job_name", "trigger_source", "trigger_id", "status", "idempotency_key", "target_version", "message",
			"error_text", "started_at", "finished_at", "duration_ms",
		}).AddRow(
			2, "portal-consumer-projection", "manual", "manual-2", RunStatusSkipped, "same", "same", "snapshot unchanged",
			nil, time.Now(), time.Now(), int64(0),
		))

	service := New(
		stubPortalDBClient{db: db},
		stubPortal{
			consumers: []portalsvc.ConsumerRecord{{
				Name:              "demo",
				PortalDisplayName: "Demo",
				PortalEmail:       "demo@example.com",
				PortalStatus:      "active",
				PortalUserLevel:   "pro",
			}},
		},
		nil,
		k8sclient.NewMemoryClient(),
	)

	first, err := service.Trigger(context.Background(), "portal-consumer-projection", TriggerInput{
		Source:    "manual",
		TriggerID: "manual-1",
	})
	require.NoError(t, err)
	require.Equal(t, RunStatusSuccess, first.Status)

	second, err := service.Trigger(context.Background(), "portal-consumer-projection", TriggerInput{
		Source:    "manual",
		TriggerID: "manual-2",
	})
	require.NoError(t, err)
	require.Equal(t, RunStatusSkipped, second.Status)

	item, err := service.k8s.GetResource(context.Background(), "consumers", "demo")
	require.NoError(t, err)
	require.Equal(t, "pro", item["userLevel"])
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestExpandAllowedConsumersWithDepartmentsIncludesDescendantsAndSkipsDisabled(t *testing.T) {
	portal := stubPortal{
		accounts: []portalsvc.OrgAccountRecord{
			{ConsumerName: "parent-user", DepartmentID: "dept-parent", Status: "active", UserLevel: "normal"},
			{ConsumerName: "child-user", DepartmentID: "dept-child", Status: "active", UserLevel: "plus"},
			{ConsumerName: "pro-user", DepartmentID: "dept-other", Status: "active", UserLevel: "pro"},
			{ConsumerName: "disabled-user", DepartmentID: "dept-child", Status: "disabled", UserLevel: "pro"},
		},
	}

	result := expandAllowedConsumersWithDepartments(
		context.Background(),
		[]string{"explicit-user"},
		[]string{"pro"},
		[]string{"dept-parent"},
		map[string][]string{"pro": {"pro-user"}},
		map[string][]string{"dept-parent": {"dept-child"}},
		portal,
	)
	require.ElementsMatch(t, []string{"explicit-user", "parent-user", "child-user", "pro-user"}, result)
}

func TestIndexDepartmentDescendantsCollectsNestedChildren(t *testing.T) {
	index := indexDepartmentDescendants([]*portalsvc.OrgDepartmentNode{
		{
			DepartmentID: "dept-parent",
			Children: []*portalsvc.OrgDepartmentNode{
				{
					DepartmentID: "dept-child",
					Children: []*portalsvc.OrgDepartmentNode{
						{DepartmentID: "dept-grandchild"},
					},
				},
			},
		},
	})

	require.ElementsMatch(t, []string{"dept-child", "dept-grandchild"}, index["dept-parent"])
	require.ElementsMatch(t, []string{"dept-grandchild"}, index["dept-child"])
}
