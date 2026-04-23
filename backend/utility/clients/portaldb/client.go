package portaldb

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"

	_ "github.com/jackc/pgx/v5/stdlib"
	portalshared "higress-portal-backend/schema/shared"
)

var ErrUnavailable = errors.New("portal database is unavailable")

type Config struct {
	Enabled     bool
	Driver      string
	DSN         string
	AutoMigrate bool
}

type Client interface {
	Healthy(ctx context.Context) error
	Enabled() bool
	DB() *sql.DB
	Driver() string
	EnsureSchema(ctx context.Context) error
	MigrateLegacyData(ctx context.Context) error
}

type FakeClient struct {
	config Config
}

type SQLClient struct {
	config Config
	db     *sql.DB
	err    error
}

func New(cfg Config) Client {
	if !cfg.Enabled || strings.TrimSpace(cfg.DSN) == "" {
		return &FakeClient{config: cfg}
	}

	driver := normalizeDriver(cfg.Driver, cfg.DSN)
	db, err := sql.Open(driver, cfg.DSN)
	if err != nil {
		return &SQLClient{config: cfg, err: err}
	}
	db.SetConnMaxLifetime(30 * time.Minute)
	db.SetMaxIdleConns(4)
	db.SetMaxOpenConns(8)

	client := &SQLClient{
		config: cfg,
		db:     db,
	}
	client.config.Driver = driver
	if cfg.AutoMigrate {
		client.err = client.EnsureSchema(context.Background())
	}
	return client
}

func NewFromDB(cfg Config, db *sql.DB) Client {
	return &SQLClient{
		config: Config{
			Enabled:     true,
			Driver:      normalizeDriver(cfg.Driver, cfg.DSN),
			DSN:         cfg.DSN,
			AutoMigrate: cfg.AutoMigrate,
		},
		db: db,
	}
}

func (c *FakeClient) Healthy(ctx context.Context) error { return ErrUnavailable }
func (c *FakeClient) Enabled() bool                     { return false }
func (c *FakeClient) DB() *sql.DB                       { return nil }
func (c *FakeClient) Driver() string                    { return normalizeDriver(c.config.Driver, c.config.DSN) }
func (c *FakeClient) EnsureSchema(ctx context.Context) error {
	return ErrUnavailable
}
func (c *FakeClient) MigrateLegacyData(ctx context.Context) error { return ErrUnavailable }

func (c *SQLClient) Healthy(ctx context.Context) error {
	if c.err != nil {
		return c.err
	}
	if c.db == nil {
		return ErrUnavailable
	}
	return c.db.PingContext(ctx)
}

func (c *SQLClient) Enabled() bool {
	return c.db != nil && (c.config.Enabled || strings.TrimSpace(c.config.DSN) != "")
}

func (c *SQLClient) DB() *sql.DB {
	return c.db
}

func (c *SQLClient) Driver() string {
	return c.config.Driver
}

func (c *SQLClient) EnsureSchema(ctx context.Context) error {
	if c.db == nil {
		return ErrUnavailable
	}
	if c.config.AutoMigrate {
		if err := portalshared.ApplyToSQLWithDriver(ctx, c.db, c.config.Driver); err != nil {
			return WrapExecError("apply shared schema", err)
		}
	}
	if err := c.ensureSharedSchemaAvailable(ctx); err != nil {
		return err
	}
	if !c.config.AutoMigrate {
		return nil
	}

	for _, statement := range ensureSchemaDDLs(c.config.Driver) {
		if _, err := ExecContext(ctx, c.db, c.config.Driver, statement); err != nil {
			return err
		}
	}
	return c.MigrateLegacyData(ctx)
}

func (c *SQLClient) MigrateLegacyData(ctx context.Context) error {
	if c.db == nil {
		return ErrUnavailable
	}
	if err := c.ensureSharedSchemaAvailable(ctx); err != nil {
		return err
	}

	if exists, err := c.tableExists(ctx, "portal_users"); err != nil {
		return err
	} else if exists {
		if err := c.migrateLegacyUsers(ctx); err != nil {
			return err
		}
	}
	if exists, err := c.tableExists(ctx, "portal_departments"); err != nil {
		return err
	} else if exists {
		if err := c.migrateLegacyDepartments(ctx); err != nil {
			return err
		}
	}
	if exists, err := c.tableExists(ctx, "portal_asset_grant"); err != nil {
		return err
	} else if exists {
		if err := c.migrateLegacyAssetGrants(ctx); err != nil {
			return err
		}
	}
	if exists, err := c.tableExists(ctx, "portal_ai_quota_user_policy"); err != nil {
		return err
	} else if exists {
		if err := c.migrateLegacyQuotaPolicies(ctx); err != nil {
			return err
		}
	}
	if exists, err := c.tableExists(ctx, "ai_sensitive_detect_rule"); err != nil {
		return err
	} else if exists {
		if err := c.migrateLegacyAISensitiveDetectRules(ctx); err != nil {
			return err
		}
	}
	if exists, err := c.tableExists(ctx, "ai_sensitive_replace_rule"); err != nil {
		return err
	} else if exists {
		if err := c.migrateLegacyAISensitiveReplaceRules(ctx); err != nil {
			return err
		}
	}
	if exists, err := c.tableExists(ctx, "ai_sensitive_system_config"); err != nil {
		return err
	} else if exists {
		if err := c.migrateLegacyAISensitiveSystemConfig(ctx); err != nil {
			return err
		}
	}
	if exists, err := c.tableExists(ctx, "ai_sensitive_block_audit"); err != nil {
		return err
	} else if exists {
		if err := c.migrateLegacyAISensitiveAudits(ctx); err != nil {
			return err
		}
	}
	return nil
}

func (c *SQLClient) ensureSharedSchemaAvailable(ctx context.Context) error {
	for _, table := range portalshared.RequiredTables() {
		ok, err := c.tableExists(ctx, table)
		if err != nil {
			return WrapExecError("check shared schema", err)
		}
		if !ok {
			return fmt.Errorf("shared portal schema table is missing: %s", table)
		}
	}
	return nil
}

func (c *SQLClient) tableExists(ctx context.Context, table string) (bool, error) {
	var count int
	err := QueryRowContext(ctx, c.db, c.config.Driver, tableExistenceQuery(c.config.Driver), table).Scan(&count)
	return count > 0, err
}

func (c *SQLClient) migrateLegacyUsers(ctx context.Context) error {
	statement := legacyUsersMigrationSQL(c.config.Driver)
	if _, err := c.db.ExecContext(ctx, statement); err != nil {
		return WrapExecError("migrate legacy users", err)
	}

	membershipStatement := legacyMembershipMigrationSQL(c.config.Driver)
	_, err := ExecContext(ctx, c.db, c.config.Driver, membershipStatement)
	return WrapExecError("migrate legacy memberships", err)
}

func (c *SQLClient) migrateLegacyDepartments(ctx context.Context) error {
	_, err := ExecContext(ctx, c.db, c.config.Driver, legacyDepartmentMigrationSQL(c.config.Driver))
	return WrapExecError("migrate legacy departments", err)
}

func (c *SQLClient) migrateLegacyAssetGrants(ctx context.Context) error {
	_, err := ExecContext(ctx, c.db, c.config.Driver, legacyAssetGrantMigrationSQL(c.config.Driver))
	return WrapExecError("migrate legacy asset grants", err)
}

func (c *SQLClient) migrateLegacyQuotaPolicies(ctx context.Context) error {
	return WrapExecError("migrate legacy quota policies", c.migrateLegacyQuotaPoliciesByRows(ctx))
}

func (c *SQLClient) migrateLegacyAISensitiveDetectRules(ctx context.Context) error {
	_, err := ExecContext(ctx, c.db, c.config.Driver, legacyAISensitiveDetectRuleMigrationSQL(c.config.Driver))
	return WrapExecError("migrate legacy ai sensitive detect rules", err)
}

func (c *SQLClient) migrateLegacyAISensitiveReplaceRules(ctx context.Context) error {
	_, err := ExecContext(ctx, c.db, c.config.Driver, legacyAISensitiveReplaceRuleMigrationSQL(c.config.Driver))
	return WrapExecError("migrate legacy ai sensitive replace rules", err)
}

func (c *SQLClient) migrateLegacyAISensitiveSystemConfig(ctx context.Context) error {
	return WrapExecError("migrate legacy ai sensitive system config", c.migrateLegacyAISensitiveSystemConfigByRow(ctx))
}

func (c *SQLClient) migrateLegacyAISensitiveAudits(ctx context.Context) error {
	_, err := ExecContext(ctx, c.db, c.config.Driver, legacyAISensitiveAuditMigrationSQL(c.config.Driver))
	return WrapExecError("migrate legacy ai sensitive audits", err)
}

func normalizeDriver(driver, dsn string) string {
	_ = dsn
	return pgxRebindDriverName
}

func MustDB(client Client) (*sql.DB, error) {
	if client == nil || !client.Enabled() || client.DB() == nil {
		return nil, ErrUnavailable
	}
	return client.DB(), nil
}

func WrapExecError(action string, err error) error {
	if err == nil {
		return nil
	}
	return fmt.Errorf("%s: %w", action, err)
}
