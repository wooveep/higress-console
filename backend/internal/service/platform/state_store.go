package platform

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"errors"
	"strings"

	"golang.org/x/crypto/bcrypt"

	"github.com/wooveep/aigateway-console/backend/internal/consts"
	"github.com/wooveep/aigateway-console/backend/internal/model/response"
	k8sclient "github.com/wooveep/aigateway-console/backend/utility/clients/k8s"
	portaldbclient "github.com/wooveep/aigateway-console/backend/utility/clients/portaldb"
)

const consoleStateTableName = "console_system_state"

type persistedStateRecord struct {
	Initialized       bool
	AdminUsername     string
	AdminDisplayName  string
	AdminPasswordHash string
	Configs           map[string]any
}

func defaultUserConfigs(dashboardBuiltIn bool) map[string]any {
	return map[string]any{
		"system.initialized":             false,
		"route.default.initialized":      false,
		"dashboard.builtin":              dashboardBuiltIn,
		"login.prompt":                   "",
		"index.redirect-target":          "/dashboard",
		"admin.password-change.disabled": false,
		"chat.enabled":                   false,
	}
}

func (s *Service) ensurePersistedStateLoaded(ctx context.Context) error {
	if s.portalClient == nil || !s.portalClient.Enabled() || s.portalClient.DB() == nil {
		return portaldbclient.ErrUnavailable
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	if s.stateLoaded {
		return nil
	}
	return s.loadPersistedStateLocked(ctx)
}

func (s *Service) loadPersistedStateLocked(ctx context.Context) error {
	db, err := portaldbclient.MustDB(s.portalClient)
	if err != nil {
		return err
	}
	if err := ensureConsoleStateTable(ctx, db, s.portalClient.Driver()); err != nil {
		return err
	}

	record, found, err := loadPersistedStateRecord(ctx, db)
	if err != nil {
		return err
	}
	if !found {
		migrated, migrateErr := s.migrateLegacySecretStateLocked(ctx, db)
		if migrateErr != nil {
			return migrateErr
		}
		if migrated != nil {
			record = migrated
		}
	}
	if record != nil {
		s.applyPersistedStateLocked(record)
		s.ensureDefaultGatewayResourcesLocked(ctx)
	}
	s.stateLoaded = true
	return nil
}

func (s *Service) persistStateLocked(ctx context.Context) error {
	db, err := portaldbclient.MustDB(s.portalClient)
	if err != nil {
		return nil
	}
	if err := ensureConsoleStateTable(ctx, db, s.portalClient.Driver()); err != nil {
		return err
	}

	configBytes, err := json.Marshal(s.userConfigs)
	if err != nil {
		return err
	}
	initialized, _ := s.userConfigs["system.initialized"].(bool)
	adminUsername := ""
	adminDisplayName := ""
	if s.adminUser != nil {
		adminUsername = firstNonEmpty(s.adminUser.Username, s.adminUser.Name)
		adminDisplayName = s.adminUser.DisplayName
	}

	_, err = db.ExecContext(ctx, `
		INSERT INTO console_system_state (
			state_key, initialized, admin_username, admin_display_name, admin_password_hash, configs_json
		)
		VALUES (?, ?, ?, ?, ?, ?)
		`+portaldbclient.UpsertClause(s.portalClient.Driver(), []string{"state_key"},
		portaldbclient.AssignValue(s.portalClient.Driver(), "initialized"),
		portaldbclient.AssignValue(s.portalClient.Driver(), "admin_username"),
		portaldbclient.AssignValue(s.portalClient.Driver(), "admin_display_name"),
		portaldbclient.AssignValue(s.portalClient.Driver(), "admin_password_hash"),
		portaldbclient.AssignValue(s.portalClient.Driver(), "configs_json"),
		`updated_at = CURRENT_TIMESTAMP`)+``,
		consts.DefaultAdminStateKey,
		initialized,
		adminUsername,
		adminDisplayName,
		s.adminHash,
		string(configBytes),
	)
	return err
}

func (s *Service) applyPersistedStateLocked(record *persistedStateRecord) {
	configs := defaultUserConfigs(s.grafanaClient.IsBuiltIn())
	for key, value := range record.Configs {
		configs[key] = value
	}
	configs["dashboard.builtin"] = s.grafanaClient.IsBuiltIn()
	configs["system.initialized"] = record.Initialized
	if record.Initialized {
		configs["route.default.initialized"] = true
	}
	s.userConfigs = configs

	if record.AdminUsername != "" {
		s.adminUser = &response.User{
			Name:        record.AdminUsername,
			Username:    record.AdminUsername,
			DisplayName: firstNonEmpty(record.AdminDisplayName, consts.DefaultAdminDisplayName),
			Type:        "admin",
		}
	} else {
		s.adminUser = nil
	}
	s.adminHash = strings.TrimSpace(record.AdminPasswordHash)
}

func (s *Service) migrateLegacySecretStateLocked(ctx context.Context, db *sql.DB) (*persistedStateRecord, error) {
	if s.k8sClient == nil {
		return nil, nil
	}
	secret, err := s.k8sClient.ReadSecret(ctx, consts.DefaultSecretName)
	if err != nil {
		if errors.Is(err, k8sclient.ErrNotFound) {
			return nil, nil
		}
		return nil, err
	}
	username := strings.TrimSpace(secret["adminUsername"])
	password := secret["adminPassword"]
	displayName := firstNonEmpty(secret["adminDisplayName"], consts.DefaultAdminDisplayName)
	if username == "" || strings.TrimSpace(password) == "" {
		return nil, nil
	}

	passwordHash, err := hashAdminPassword(password)
	if err != nil {
		return nil, err
	}
	record := &persistedStateRecord{
		Initialized:       true,
		AdminUsername:     username,
		AdminDisplayName:  displayName,
		AdminPasswordHash: passwordHash,
		Configs:           defaultUserConfigs(s.grafanaClient.IsBuiltIn()),
	}
	record.Configs["system.initialized"] = true
	record.Configs["route.default.initialized"] = true

	configBytes, err := json.Marshal(record.Configs)
	if err != nil {
		return nil, err
	}
	if _, err := db.ExecContext(ctx, `
		INSERT INTO console_system_state (
			state_key, initialized, admin_username, admin_display_name, admin_password_hash, configs_json
		)
		VALUES (?, ?, ?, ?, ?, ?)
		`+portaldbclient.UpsertClause(s.portalClient.Driver(), []string{"state_key"},
		portaldbclient.AssignValue(s.portalClient.Driver(), "initialized"),
		portaldbclient.AssignValue(s.portalClient.Driver(), "admin_username"),
		portaldbclient.AssignValue(s.portalClient.Driver(), "admin_display_name"),
		portaldbclient.AssignValue(s.portalClient.Driver(), "admin_password_hash"),
		portaldbclient.AssignValue(s.portalClient.Driver(), "configs_json"),
		`updated_at = CURRENT_TIMESTAMP`)+``,
		consts.DefaultAdminStateKey,
		true,
		username,
		displayName,
		passwordHash,
		string(configBytes),
	); err != nil {
		return nil, err
	}
	return record, nil
}

func loadPersistedStateRecord(ctx context.Context, db *sql.DB) (*persistedStateRecord, bool, error) {
	var (
		record      persistedStateRecord
		configsJSON sql.NullString
	)
	err := db.QueryRowContext(ctx, `
		SELECT initialized, admin_username, admin_display_name, admin_password_hash, configs_json
		FROM console_system_state
		WHERE state_key = ?`,
		consts.DefaultAdminStateKey,
	).Scan(
		&record.Initialized,
		&record.AdminUsername,
		&record.AdminDisplayName,
		&record.AdminPasswordHash,
		&configsJSON,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, false, nil
		}
		return nil, false, err
	}
	record.Configs = map[string]any{}
	if strings.TrimSpace(configsJSON.String) != "" {
		if err := json.Unmarshal([]byte(configsJSON.String), &record.Configs); err != nil {
			return nil, false, err
		}
	}
	return &record, true, nil
}

func ensureConsoleStateTable(ctx context.Context, db *sql.DB, driver string) error {
	_ = driver
	statement := `
		CREATE TABLE IF NOT EXISTS console_system_state (
			state_key VARCHAR(64) PRIMARY KEY,
			initialized BOOLEAN NOT NULL DEFAULT FALSE,
			admin_username VARCHAR(128) NOT NULL DEFAULT '',
			admin_display_name VARCHAR(255) NOT NULL DEFAULT '',
			admin_password_hash VARCHAR(255) NOT NULL DEFAULT '',
			configs_json TEXT NULL,
			created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
			updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
		)`
	_, err := db.ExecContext(ctx, statement)
	return err
}

func hashAdminPassword(password string) (string, error) {
	bytes, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return "", err
	}
	return string(bytes), nil
}

func compareAdminPassword(passwordHash, password string) bool {
	if strings.TrimSpace(passwordHash) == "" || password == "" {
		return false
	}
	return bcrypt.CompareHashAndPassword([]byte(passwordHash), []byte(password)) == nil
}

func signSessionToken(username, issuedAt, passwordHash string) string {
	sum := sha256.Sum256([]byte(username + ":" + issuedAt + ":" + passwordHash))
	return hex.EncodeToString(sum[:])
}
