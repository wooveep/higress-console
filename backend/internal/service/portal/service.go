package portal

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"slices"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/gogf/gf/v2/os/gtime"
	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"

	"github.com/wooveep/aigateway-console/backend/internal/model/do"
	"github.com/wooveep/aigateway-console/backend/internal/model/entity"
	k8sclient "github.com/wooveep/aigateway-console/backend/utility/clients/k8s"
	portaldbclient "github.com/wooveep/aigateway-console/backend/utility/clients/portaldb"
)

var (
	errPortalUnavailable = errors.New("portal database is unavailable")
	allowedStatuses      = []string{"active", "disabled", "pending"}
	allowedUserLevels    = []string{"normal", "plus", "pro", "ultra"}
)

const orgRootDepartmentID = "root"

type departmentRef struct {
	name   string
	parent string
}

type Hook interface {
	AfterWrite(ctx context.Context, trigger string) error
}

type noopHook struct{}

type Service struct {
	client    portaldbclient.Client
	k8sClient k8sclient.Client
	hook      Hook

	schemaMu      sync.Mutex
	schemaChecked bool
}

type ConsumerRecord struct {
	Name               string     `json:"name"`
	Department         string     `json:"department,omitempty"`
	DepartmentID       string     `json:"departmentId,omitempty"`
	DepartmentPath     string     `json:"departmentPath,omitempty"`
	PortalStatus       string     `json:"portalStatus,omitempty"`
	PortalDisplayName  string     `json:"portalDisplayName,omitempty"`
	PortalEmail        string     `json:"portalEmail,omitempty"`
	PortalUserSource   string     `json:"portalUserSource,omitempty"`
	PortalUserLevel    string     `json:"portalUserLevel,omitempty"`
	PortalTempPassword string     `json:"portalTempPassword,omitempty"`
	Credentials        []string   `json:"credentials,omitempty"`
	CreatedAt          *time.Time `json:"createdAt,omitempty"`
	UpdatedAt          *time.Time `json:"updatedAt,omitempty"`
	LastLoginAt        *time.Time `json:"lastLoginAt,omitempty"`
}

type ConsumerMutation struct {
	Name              string `json:"name"`
	Department        string `json:"department"`
	PortalStatus      string `json:"portalStatus"`
	PortalDisplayName string `json:"portalDisplayName"`
	PortalEmail       string `json:"portalEmail"`
	PortalUserLevel   string `json:"portalUserLevel"`
	PortalPassword    string `json:"portalPassword"`
}

type ResetPasswordResult struct {
	ConsumerName string    `json:"consumerName"`
	TempPassword string    `json:"tempPassword"`
	UpdatedAt    time.Time `json:"updatedAt"`
}

type OrgDepartmentNode struct {
	DepartmentID       string               `json:"departmentId"`
	Name               string               `json:"name"`
	ParentDepartmentID string               `json:"parentDepartmentId,omitempty"`
	AdminConsumerName  string               `json:"adminConsumerName,omitempty"`
	Level              int                  `json:"level,omitempty"`
	MemberCount        int                  `json:"memberCount,omitempty"`
	Children           []*OrgDepartmentNode `json:"children,omitempty"`
}

type DepartmentMutation struct {
	Name               string `json:"name"`
	ParentDepartmentID string `json:"parentDepartmentId"`
	AdminConsumerName  string `json:"adminConsumerName"`
	AdminDisplayName   string `json:"adminDisplayName"`
	AdminEmail         string `json:"adminEmail"`
	AdminUserLevel     string `json:"adminUserLevel"`
	AdminPassword      string `json:"adminPassword"`
}

type OrgAccountRecord struct {
	ConsumerName       string     `json:"consumerName"`
	DisplayName        string     `json:"displayName,omitempty"`
	Email              string     `json:"email,omitempty"`
	Status             string     `json:"status,omitempty"`
	UserLevel          string     `json:"userLevel,omitempty"`
	Source             string     `json:"source,omitempty"`
	DepartmentID       string     `json:"departmentId,omitempty"`
	DepartmentName     string     `json:"departmentName,omitempty"`
	DepartmentPath     string     `json:"departmentPath,omitempty"`
	ParentConsumerName string     `json:"parentConsumerName,omitempty"`
	IsDepartmentAdmin  bool       `json:"isDepartmentAdmin,omitempty"`
	LastLoginAt        *time.Time `json:"lastLoginAt,omitempty"`
	TempPassword       string     `json:"tempPassword,omitempty"`
}

type AccountMutation struct {
	ConsumerName       string `json:"consumerName"`
	DisplayName        string `json:"displayName"`
	Email              string `json:"email"`
	UserLevel          string `json:"userLevel"`
	Password           string `json:"password"`
	Status             string `json:"status"`
	DepartmentID       string `json:"departmentId"`
	ParentConsumerName string `json:"parentConsumerName"`
}

type InviteCodeRecord struct {
	InviteCode     string     `json:"inviteCode"`
	Status         string     `json:"status"`
	ExpiresAt      *time.Time `json:"expiresAt,omitempty"`
	UsedByConsumer string     `json:"usedByConsumer,omitempty"`
	UsedAt         *time.Time `json:"usedAt,omitempty"`
	CreatedAt      *time.Time `json:"createdAt,omitempty"`
}

type InviteCodeQuery struct {
	PageNum  int
	PageSize int
	Status   string
}

func New(client portaldbclient.Client, k8sClients ...k8sclient.Client) *Service {
	var k8sSvc k8sclient.Client
	if len(k8sClients) > 0 {
		k8sSvc = k8sClients[0]
	}
	return &Service{
		client:    client,
		k8sClient: k8sSvc,
		hook:      noopHook{},
	}
}

func NewWithHook(client portaldbclient.Client, hook Hook, k8sClients ...k8sclient.Client) *Service {
	svc := New(client, k8sClients...)
	if hook != nil {
		svc.hook = hook
	}
	return svc
}

func (noopHook) AfterWrite(ctx context.Context, trigger string) error { return nil }

func (s *Service) SetHook(hook Hook) {
	if hook == nil {
		s.hook = noopHook{}
		return
	}
	s.hook = hook
}

func (s *Service) Enabled() bool {
	return s.client != nil && s.client.Enabled() && s.client.DB() != nil
}

func (s *Service) ListConsumers(ctx context.Context) ([]ConsumerRecord, error) {
	db, err := s.db(ctx)
	if err != nil {
		return nil, err
	}

	rows, err := db.QueryContext(ctx, `
		SELECT u.consumer_name, u.display_name, u.email, u.status, u.user_level, u.source, d.department_id, d.name, d.path
		FROM portal_user u
		LEFT JOIN org_account_membership m ON m.consumer_name = u.consumer_name
		LEFT JOIN org_department d ON d.department_id = m.department_id AND d.status = 'active'
		WHERE COALESCE(u.is_deleted, FALSE) = FALSE
		ORDER BY u.consumer_name ASC`)
	if err != nil {
		return nil, portaldbclient.WrapExecError("list consumers", err)
	}
	defer rows.Close()

	items := make([]ConsumerRecord, 0)
	for rows.Next() {
		var item ConsumerRecord
		if err := rows.Scan(
			&item.Name,
			&item.PortalDisplayName,
			&item.PortalEmail,
			&item.PortalStatus,
			&item.PortalUserLevel,
			&item.PortalUserSource,
			&item.DepartmentID,
			&item.Department,
			&item.DepartmentPath,
		); err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	return items, rows.Err()
}

func (s *Service) GetConsumer(ctx context.Context, consumerName string) (*ConsumerRecord, error) {
	db, err := s.db(ctx)
	if err != nil {
		return nil, err
	}
	name := strings.TrimSpace(consumerName)
	if name == "" {
		return nil, errors.New("consumerName cannot be blank")
	}

	var item ConsumerRecord
	err = db.QueryRowContext(ctx, `
		SELECT u.consumer_name, u.display_name, u.email, u.status, u.user_level, u.source,
			d.department_id, d.name, d.path, u.created_at, u.updated_at, u.last_login_at
		FROM portal_user u
		LEFT JOIN org_account_membership m ON m.consumer_name = u.consumer_name
		LEFT JOIN org_department d ON d.department_id = m.department_id AND d.status = 'active'
		WHERE COALESCE(u.is_deleted, FALSE) = FALSE AND u.consumer_name = ?`, name).
		Scan(
			&item.Name,
			&item.PortalDisplayName,
			&item.PortalEmail,
			&item.PortalStatus,
			&item.PortalUserLevel,
			&item.PortalUserSource,
			&item.DepartmentID,
			&item.Department,
			&item.DepartmentPath,
			&item.CreatedAt,
			&item.UpdatedAt,
			&item.LastLoginAt,
		)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, portaldbclient.WrapExecError("get consumer", err)
	}
	item.Credentials = s.listMaskedConsumerCredentials(ctx, db, name)
	return &item, nil
}

func (s *Service) listMaskedConsumerCredentials(ctx context.Context, db *sql.DB, consumerName string) []string {
	rows, err := db.QueryContext(ctx, `
		SELECT raw_key
		FROM portal_api_key
		WHERE consumer_name = ? AND status = 'active' AND deleted_at IS NULL
		ORDER BY id ASC`, consumerName)
	if err != nil {
		return nil
	}
	defer rows.Close()

	items := make([]string, 0)
	for rows.Next() {
		var rawKey string
		if err := rows.Scan(&rawKey); err != nil {
			return items
		}
		if masked := maskPortalCredential(rawKey); masked != "" {
			items = append(items, masked)
		}
	}
	return items
}

func maskPortalCredential(raw string) string {
	normalized := strings.TrimSpace(raw)
	if normalized == "" {
		return ""
	}
	if len(normalized) <= 8 {
		return "****"
	}
	return normalized[:4] + "****" + normalized[len(normalized)-4:]
}

func (s *Service) SaveConsumer(ctx context.Context, mutation ConsumerMutation, create bool) (*ConsumerRecord, error) {
	db, err := s.db(ctx)
	if err != nil {
		return nil, err
	}

	name := strings.TrimSpace(mutation.Name)
	if name == "" {
		return nil, errors.New("name cannot be blank")
	}
	status := normalizeStatus(mutation.PortalStatus, create)
	userLevel := normalizeUserLevel(mutation.PortalUserLevel)

	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()

	exists, err := s.consumerExists(ctx, tx, name)
	if err != nil {
		return nil, err
	}
	if create && exists {
		return nil, fmt.Errorf("consumer already exists: %s", name)
	}
	if !create && !exists {
		return nil, fmt.Errorf("consumer not found: %s", name)
	}

	departmentID, err := s.lookupDepartmentIDByName(ctx, tx, mutation.Department)
	if err != nil {
		return nil, err
	}
	passwordHash := ""
	if strings.TrimSpace(mutation.PortalPassword) != "" {
		passwordHash, err = hashPassword(mutation.PortalPassword)
		if err != nil {
			return nil, err
		}
	} else if create {
		passwordHash, err = hashPassword(newTempPassword())
		if err != nil {
			return nil, err
		}
	}
	insertPasswordHash := passwordHash
	if create && insertPasswordHash == "" {
		insertPasswordHash, err = hashPassword(newTempPassword())
		if err != nil {
			return nil, err
		}
	}

	if create {
		_, err = tx.ExecContext(ctx, `
			INSERT INTO portal_user (
				consumer_name, display_name, email, status, user_level, source, password_hash
			) VALUES (?, ?, ?, ?, ?, 'console', ?)`,
			name,
			firstNonEmpty(strings.TrimSpace(mutation.PortalDisplayName), name),
			firstNonEmpty(strings.TrimSpace(mutation.PortalEmail), ""),
			status,
			userLevel,
			insertPasswordHash,
		)
	} else {
		query := `
			UPDATE portal_user
			SET display_name = COALESCE(?, display_name),
				email = COALESCE(?, email),
				status = ?,
				user_level = ?,
				updated_at = CURRENT_TIMESTAMP`
		args := []any{
			trimOrNil(mutation.PortalDisplayName),
			trimOrNil(mutation.PortalEmail),
			status,
			userLevel,
		}
		if passwordHash != "" {
			query += `, password_hash = ?`
			args = append(args, passwordHash)
		}
		query += ` WHERE consumer_name = ? AND COALESCE(is_deleted, FALSE) = FALSE`
		args = append(args, name)
		_, err = tx.ExecContext(ctx, query, args...)
	}
	if err != nil {
		return nil, portaldbclient.WrapExecError("save consumer", err)
	}
	if _, err := tx.ExecContext(ctx, `
		INSERT INTO org_account_membership (consumer_name, department_id, parent_consumer_name)
		VALUES (?, ?, NULL)
		`+portaldbclient.UpsertClause(s.client.Driver(), []string{"consumer_name"},
		portaldbclient.AssignValue(s.client.Driver(), "department_id"),
		`updated_at = CURRENT_TIMESTAMP`)+``,
		name,
		nullIfEmpty(departmentID),
	); err != nil {
		return nil, portaldbclient.WrapExecError("save consumer membership", err)
	}
	if err := tx.Commit(); err != nil {
		return nil, err
	}
	if err := s.hook.AfterWrite(ctx, "consumer-upsert"); err != nil {
		return nil, err
	}
	return s.GetConsumer(ctx, name)
}

func (s *Service) DeleteConsumer(ctx context.Context, consumerName string) error {
	db, err := s.db(ctx)
	if err != nil {
		return err
	}
	name := strings.TrimSpace(consumerName)
	if name == "" {
		return errors.New("consumerName cannot be blank")
	}

	if admin, err := s.isDepartmentAdmin(ctx, db, name); err != nil {
		return err
	} else if admin {
		return errors.New("department administrator cannot be deleted")
	}

	result, err := db.ExecContext(ctx, `
		UPDATE portal_user SET is_deleted = 1, deleted_at = CURRENT_TIMESTAMP, updated_at = CURRENT_TIMESTAMP
		WHERE consumer_name = ? AND COALESCE(is_deleted, FALSE) = FALSE`, name)
	if err != nil {
		return portaldbclient.WrapExecError("delete consumer", err)
	}
	affected, _ := result.RowsAffected()
	if affected == 0 {
		return fmt.Errorf("consumer not found: %s", name)
	}
	return s.hook.AfterWrite(ctx, "consumer-delete")
}

func (s *Service) UpdateConsumerStatus(ctx context.Context, consumerName, status string) (*ConsumerRecord, error) {
	db, err := s.db(ctx)
	if err != nil {
		return nil, err
	}
	name := strings.TrimSpace(consumerName)
	if name == "" {
		return nil, errors.New("consumerName cannot be blank")
	}
	normalized := normalizeStatus(status, false)

	result, err := db.ExecContext(ctx, `
		UPDATE portal_user SET status = ?, updated_at = CURRENT_TIMESTAMP
		WHERE consumer_name = ? AND COALESCE(is_deleted, FALSE) = FALSE`, normalized, name)
	if err != nil {
		return nil, err
	}
	affected, _ := result.RowsAffected()
	if affected == 0 {
		return nil, fmt.Errorf("consumer not found: %s", name)
	}
	if err := s.hook.AfterWrite(ctx, "consumer-status"); err != nil {
		return nil, err
	}
	return s.GetConsumer(ctx, name)
}

func (s *Service) ResetPassword(ctx context.Context, consumerName string) (*ResetPasswordResult, error) {
	db, err := s.db(ctx)
	if err != nil {
		return nil, err
	}
	name := strings.TrimSpace(consumerName)
	if name == "" {
		return nil, errors.New("consumerName cannot be blank")
	}

	tempPassword := newTempPassword()
	passwordHash, err := hashPassword(tempPassword)
	if err != nil {
		return nil, err
	}
	result, err := db.ExecContext(ctx, `
		UPDATE portal_user
		SET password_hash = ?, updated_at = CURRENT_TIMESTAMP
		WHERE consumer_name = ? AND COALESCE(is_deleted, FALSE) = FALSE`,
		passwordHash,
		name,
	)
	if err != nil {
		return nil, err
	}
	affected, _ := result.RowsAffected()
	if affected == 0 {
		return nil, fmt.Errorf("consumer not found: %s", name)
	}
	return &ResetPasswordResult{
		ConsumerName: name,
		TempPassword: tempPassword,
		UpdatedAt:    time.Now(),
	}, nil
}

func (s *Service) ListConsumerDepartments(ctx context.Context) ([]string, error) {
	nodes, err := s.ListDepartmentTree(ctx)
	if err != nil {
		return nil, err
	}
	result := make([]string, 0)
	var walk func(items []*OrgDepartmentNode)
	walk = func(items []*OrgDepartmentNode) {
		for _, item := range items {
			result = append(result, item.Name)
			walk(item.Children)
		}
	}
	walk(nodes)
	return result, nil
}

func (s *Service) AddDepartmentCompat(ctx context.Context, name string) error {
	_, err := s.CreateDepartment(ctx, DepartmentMutation{Name: name})
	return err
}

func (s *Service) ListDepartmentTree(ctx context.Context) ([]*OrgDepartmentNode, error) {
	db, err := s.db(ctx)
	if err != nil {
		return nil, err
	}

	rows, err := db.QueryContext(ctx, `
		SELECT department_id, name, parent_department_id, admin_consumer_name
		FROM org_department
		WHERE status = 'active' AND department_id <> ?
		ORDER BY name ASC`, orgRootDepartmentID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	nodes := make(map[string]*OrgDepartmentNode)
	order := make([]string, 0)
	for rows.Next() {
		node := &OrgDepartmentNode{}
		var (
			parentID          sql.NullString
			adminConsumerName sql.NullString
		)
		if err := rows.Scan(&node.DepartmentID, &node.Name, &parentID, &adminConsumerName); err != nil {
			return nil, err
		}
		node.ParentDepartmentID = parentID.String
		node.AdminConsumerName = adminConsumerName.String
		nodes[node.DepartmentID] = node
		order = append(order, node.DepartmentID)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	counts, err := s.departmentMemberCounts(ctx, db)
	if err != nil {
		return nil, err
	}
	for id, count := range counts {
		if node := nodes[id]; node != nil {
			node.MemberCount = count
		}
	}

	roots := make([]*OrgDepartmentNode, 0)
	for _, id := range order {
		node := nodes[id]
		node.Level = s.resolveDepartmentLevel(nodes, node.DepartmentID)
		parent := nodes[node.ParentDepartmentID]
		if parent == nil {
			roots = append(roots, node)
			continue
		}
		parent.Children = append(parent.Children, node)
	}
	sortDepartmentTree(roots)
	return roots, nil
}

func (s *Service) CreateDepartment(ctx context.Context, mutation DepartmentMutation) (*OrgDepartmentNode, error) {
	db, err := s.db(ctx)
	if err != nil {
		return nil, err
	}
	name := strings.TrimSpace(mutation.Name)
	if name == "" {
		return nil, errors.New("department name cannot be blank")
	}

	id := "dept-" + strings.ToLower(strings.ReplaceAll(uuid.NewString()[:8], "-", ""))
	path, level, err := s.resolveDepartmentPlacement(ctx, db, mutation.ParentDepartmentID, name)
	if err != nil {
		return nil, err
	}
	if _, err := db.ExecContext(ctx, `
		INSERT INTO org_department (department_id, name, parent_department_id, admin_consumer_name, path, level, sort_order, status)
		VALUES (?, ?, ?, ?, ?, ?, 0, 'active')`,
		id,
		name,
		nullIfEmpty(strings.TrimSpace(mutation.ParentDepartmentID)),
		trimOrNil(mutation.AdminConsumerName),
		path,
		level,
	); err != nil {
		return nil, err
	}
	if err := s.hook.AfterWrite(ctx, "org-department-create"); err != nil {
		return nil, err
	}
	return s.getDepartment(ctx, id)
}

func (s *Service) UpdateDepartment(ctx context.Context, departmentID string, mutation DepartmentMutation) (*OrgDepartmentNode, error) {
	db, err := s.db(ctx)
	if err != nil {
		return nil, err
	}
	id := strings.TrimSpace(departmentID)
	if id == "" {
		return nil, errors.New("departmentId cannot be blank")
	}

	result, err := db.ExecContext(ctx, `
		UPDATE org_department
		SET name = COALESCE(?, name),
			admin_consumer_name = COALESCE(?, admin_consumer_name),
			updated_at = CURRENT_TIMESTAMP
		WHERE department_id = ? AND status = 'active'`,
		trimOrNil(mutation.Name),
		trimOrNil(mutation.AdminConsumerName),
		id,
	)
	if err != nil {
		return nil, err
	}
	affected, _ := result.RowsAffected()
	if affected == 0 {
		return nil, fmt.Errorf("department not found: %s", id)
	}
	if err := s.hook.AfterWrite(ctx, "org-department-update"); err != nil {
		return nil, err
	}
	return s.getDepartment(ctx, id)
}

func (s *Service) MoveDepartment(ctx context.Context, departmentID, parentDepartmentID string) (*OrgDepartmentNode, error) {
	db, err := s.db(ctx)
	if err != nil {
		return nil, err
	}
	id := strings.TrimSpace(departmentID)
	parentID := strings.TrimSpace(parentDepartmentID)
	if id == "" {
		return nil, errors.New("departmentId cannot be blank")
	}
	if id == parentID && parentID != "" {
		return nil, errors.New("department cannot be its own parent")
	}
	if parentID != "" {
		parentNode, err := s.getDepartment(ctx, parentID)
		if err != nil {
			return nil, err
		}
		if parentNode == nil {
			return nil, fmt.Errorf("parent department not found: %s", parentID)
		}
		if s.isDepartmentDescendant(ctx, db, parentID, id) {
			return nil, errors.New("department move would create a cycle")
		}
	}
	current, err := s.getDepartment(ctx, id)
	if err != nil {
		return nil, err
	}
	if current == nil {
		return nil, fmt.Errorf("department not found: %s", id)
	}
	path, level, err := s.resolveDepartmentPlacement(ctx, db, parentID, current.Name)
	if err != nil {
		return nil, err
	}

	result, err := db.ExecContext(ctx, `
		UPDATE org_department
		SET parent_department_id = ?, path = ?, level = ?, updated_at = CURRENT_TIMESTAMP
		WHERE department_id = ? AND status = 'active'`,
		nullIfEmpty(parentID),
		path,
		level,
		id,
	)
	if err != nil {
		return nil, err
	}
	affected, _ := result.RowsAffected()
	if affected == 0 {
		return nil, fmt.Errorf("department not found: %s", id)
	}
	if err := s.hook.AfterWrite(ctx, "org-department-move"); err != nil {
		return nil, err
	}
	return s.getDepartment(ctx, id)
}

func (s *Service) DeleteDepartment(ctx context.Context, departmentID string) error {
	db, err := s.db(ctx)
	if err != nil {
		return err
	}
	id := strings.TrimSpace(departmentID)
	if id == "" {
		return errors.New("departmentId cannot be blank")
	}
	var childCount int
	if err := db.QueryRowContext(ctx, `
		SELECT COUNT(1) FROM org_department
		WHERE parent_department_id = ? AND status = 'active'`, id).Scan(&childCount); err != nil {
		return err
	}
	if childCount > 0 {
		return errors.New("department has child departments")
	}
	var memberCount int
	if err := db.QueryRowContext(ctx, `
		SELECT COUNT(1)
		FROM org_account_membership m
		INNER JOIN portal_user u ON u.consumer_name = m.consumer_name
		WHERE m.department_id = ? AND COALESCE(u.is_deleted, FALSE) = FALSE`, id).Scan(&memberCount); err != nil {
		return err
	}
	if memberCount > 0 {
		return errors.New("department still has assigned accounts")
	}

	result, err := db.ExecContext(ctx, `
		UPDATE org_department
		SET status = 'deleted', updated_at = CURRENT_TIMESTAMP
		WHERE department_id = ? AND status = 'active'`, id)
	if err != nil {
		return err
	}
	affected, _ := result.RowsAffected()
	if affected == 0 {
		return fmt.Errorf("department not found: %s", id)
	}
	return s.hook.AfterWrite(ctx, "org-department-delete")
}

func (s *Service) ListAccounts(ctx context.Context) ([]OrgAccountRecord, error) {
	db, err := s.db(ctx)
	if err != nil {
		return nil, err
	}
	departmentNames, departmentPaths, err := s.departmentMaps(ctx, db)
	if err != nil {
		return nil, err
	}

	rows, err := db.QueryContext(ctx, `
		SELECT u.consumer_name, u.display_name, u.email, u.status, u.user_level, u.source, m.department_id,
			m.parent_consumer_name, CASE WHEN d.admin_consumer_name = u.consumer_name THEN 1 ELSE 0 END, u.last_login_at
		FROM portal_user u
		LEFT JOIN org_account_membership m ON m.consumer_name = u.consumer_name
		LEFT JOIN org_department d ON d.department_id = m.department_id
		WHERE COALESCE(u.is_deleted, FALSE) = FALSE
		ORDER BY consumer_name ASC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	items := make([]OrgAccountRecord, 0)
	for rows.Next() {
		var item OrgAccountRecord
		var departmentID sql.NullString
		var parentName sql.NullString
		var lastLogin sql.NullTime
		if err := rows.Scan(
			&item.ConsumerName,
			&item.DisplayName,
			&item.Email,
			&item.Status,
			&item.UserLevel,
			&item.Source,
			&departmentID,
			&parentName,
			&item.IsDepartmentAdmin,
			&lastLogin,
		); err != nil {
			return nil, err
		}
		item.DepartmentID = departmentID.String
		item.ParentConsumerName = parentName.String
		if lastLogin.Valid {
			item.LastLoginAt = &lastLogin.Time
		}
		item.DepartmentName = departmentNames[item.DepartmentID]
		item.DepartmentPath = departmentPaths[item.DepartmentID]
		items = append(items, item)
	}
	return items, rows.Err()
}

func (s *Service) CreateAccount(ctx context.Context, mutation AccountMutation) (*OrgAccountRecord, error) {
	db, err := s.db(ctx)
	if err != nil {
		return nil, err
	}

	name := strings.TrimSpace(mutation.ConsumerName)
	if name == "" {
		return nil, errors.New("consumerName cannot be blank")
	}
	if exists, err := s.consumerExists(ctx, db, name); err != nil {
		return nil, err
	} else if exists {
		return nil, fmt.Errorf("consumer already exists: %s", name)
	}

	status := normalizeStatus(mutation.Status, true)
	userLevel := normalizeUserLevel(mutation.UserLevel)
	password := strings.TrimSpace(mutation.Password)
	if password == "" {
		password = newTempPassword()
	}
	passwordHash, err := hashPassword(password)
	if err != nil {
		return nil, err
	}
	if mutation.DepartmentID != "" {
		if dept, err := s.getDepartment(ctx, mutation.DepartmentID); err != nil {
			return nil, err
		} else if dept == nil {
			return nil, fmt.Errorf("department not found: %s", mutation.DepartmentID)
		}
	}

	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()

	_, err = tx.ExecContext(ctx, `
		INSERT INTO portal_user (
			consumer_name, display_name, email, status, user_level, source, password_hash
		) VALUES (?, ?, ?, ?, ?, 'console', ?)`,
		name,
		firstNonEmpty(strings.TrimSpace(mutation.DisplayName), name),
		firstNonEmpty(strings.TrimSpace(mutation.Email), ""),
		status,
		userLevel,
		passwordHash,
	)
	if err != nil {
		return nil, err
	}
	if _, err := tx.ExecContext(ctx, `
		INSERT INTO org_account_membership (consumer_name, department_id, parent_consumer_name)
		VALUES (?, ?, ?)
		`+portaldbclient.UpsertClause(s.client.Driver(), []string{"consumer_name"},
		portaldbclient.AssignValue(s.client.Driver(), "department_id"),
		portaldbclient.AssignValue(s.client.Driver(), "parent_consumer_name"),
		`updated_at = CURRENT_TIMESTAMP`)+``,
		name,
		nullIfEmpty(strings.TrimSpace(mutation.DepartmentID)),
		nullIfEmpty(strings.TrimSpace(mutation.ParentConsumerName)),
	); err != nil {
		return nil, err
	}
	if err := tx.Commit(); err != nil {
		return nil, err
	}
	if err := s.hook.AfterWrite(ctx, "org-account-create"); err != nil {
		return nil, err
	}
	return s.getAccount(ctx, name)
}

func (s *Service) UpdateAccount(ctx context.Context, consumerName string, mutation AccountMutation) (*OrgAccountRecord, error) {
	db, err := s.db(ctx)
	if err != nil {
		return nil, err
	}
	name := strings.TrimSpace(consumerName)
	if name == "" {
		return nil, errors.New("consumerName cannot be blank")
	}

	query := `
		UPDATE portal_user
		SET display_name = COALESCE(?, display_name),
			email = COALESCE(?, email),
			status = ?,
			user_level = ?,
			updated_at = CURRENT_TIMESTAMP`
	args := []any{
		trimOrNil(mutation.DisplayName),
		trimOrNil(mutation.Email),
		normalizeStatus(mutation.Status, false),
		normalizeUserLevel(mutation.UserLevel),
	}
	if strings.TrimSpace(mutation.Password) != "" {
		passwordHash, hashErr := hashPassword(mutation.Password)
		if hashErr != nil {
			return nil, hashErr
		}
		query += `, password_hash = ?`
		args = append(args, passwordHash)
	}
	query += ` WHERE consumer_name = ? AND COALESCE(is_deleted, FALSE) = FALSE`
	args = append(args, name)

	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()

	result, err := tx.ExecContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	affected, _ := result.RowsAffected()
	if affected == 0 {
		return nil, fmt.Errorf("consumer not found: %s", name)
	}
	if _, err := tx.ExecContext(ctx, `
		INSERT INTO org_account_membership (consumer_name, department_id, parent_consumer_name)
		VALUES (?, ?, ?)
		`+portaldbclient.UpsertClause(s.client.Driver(), []string{"consumer_name"},
		portaldbclient.AssignValue(s.client.Driver(), "department_id"),
		portaldbclient.AssignValue(s.client.Driver(), "parent_consumer_name"),
		`updated_at = CURRENT_TIMESTAMP`)+``,
		name,
		nullIfEmpty(strings.TrimSpace(mutation.DepartmentID)),
		nullIfEmpty(strings.TrimSpace(mutation.ParentConsumerName)),
	); err != nil {
		return nil, err
	}
	if err := tx.Commit(); err != nil {
		return nil, err
	}
	if err := s.hook.AfterWrite(ctx, "org-account-update"); err != nil {
		return nil, err
	}
	return s.getAccount(ctx, name)
}

func (s *Service) UpdateAccountAssignment(ctx context.Context, consumerName, departmentID, parentConsumerName string) (*OrgAccountRecord, error) {
	db, err := s.db(ctx)
	if err != nil {
		return nil, err
	}
	name := strings.TrimSpace(consumerName)
	if name == "" {
		return nil, errors.New("consumerName cannot be blank")
	}
	if strings.TrimSpace(departmentID) != "" {
		if dept, err := s.getDepartment(ctx, departmentID); err != nil {
			return nil, err
		} else if dept == nil {
			return nil, fmt.Errorf("department not found: %s", departmentID)
		}
	}

	result, err := db.ExecContext(ctx, `
		INSERT INTO org_account_membership (consumer_name, department_id, parent_consumer_name)
		VALUES (?, ?, ?)
		`+portaldbclient.UpsertClause(s.client.Driver(), []string{"consumer_name"},
		portaldbclient.AssignValue(s.client.Driver(), "department_id"),
		portaldbclient.AssignValue(s.client.Driver(), "parent_consumer_name"),
		`updated_at = CURRENT_TIMESTAMP`)+``,
		name,
		nullIfEmpty(strings.TrimSpace(departmentID)),
		nullIfEmpty(strings.TrimSpace(parentConsumerName)),
	)
	if err != nil {
		return nil, err
	}
	_, _ = result.RowsAffected()
	if err := s.hook.AfterWrite(ctx, "org-account-assignment"); err != nil {
		return nil, err
	}
	return s.getAccount(ctx, name)
}

func (s *Service) UpdateAccountStatus(ctx context.Context, consumerName, status string) (*OrgAccountRecord, error) {
	db, err := s.db(ctx)
	if err != nil {
		return nil, err
	}
	name := strings.TrimSpace(consumerName)
	if name == "" {
		return nil, errors.New("consumerName cannot be blank")
	}

	result, err := db.ExecContext(ctx, `
		UPDATE portal_user
		SET status = ?, updated_at = CURRENT_TIMESTAMP
		WHERE consumer_name = ? AND COALESCE(is_deleted, FALSE) = FALSE`,
		normalizeStatus(status, false),
		name,
	)
	if err != nil {
		return nil, err
	}
	affected, _ := result.RowsAffected()
	if affected == 0 {
		return nil, fmt.Errorf("consumer not found: %s", name)
	}
	if err := s.hook.AfterWrite(ctx, "org-account-status"); err != nil {
		return nil, err
	}
	return s.getAccount(ctx, name)
}

func (s *Service) CreateInviteCode(ctx context.Context, expiresInDays int) (*InviteCodeRecord, error) {
	db, err := s.db(ctx)
	if err != nil {
		return nil, err
	}
	if expiresInDays <= 0 {
		expiresInDays = 7
	}
	if expiresInDays > 365 {
		return nil, errors.New("expiresInDays must be between 1 and 365")
	}

	record := &InviteCodeRecord{
		InviteCode: randomInviteCode(),
		Status:     "active",
	}
	expiresAt := time.Now().Add(time.Duration(expiresInDays) * 24 * time.Hour)
	record.ExpiresAt = &expiresAt
	if err := newPortalStore(db, s.client.Driver()).insertInviteCode(ctx, do.PortalInviteCode{
		InviteCode: record.InviteCode,
		Status:     record.Status,
		ExpiresAt:  wrapGTime(expiresAt),
	}); err != nil {
		return nil, err
	}
	return s.getInviteCode(ctx, record.InviteCode)
}

func (s *Service) ListInviteCodes(ctx context.Context, query InviteCodeQuery) ([]InviteCodeRecord, error) {
	db, err := s.db(ctx)
	if err != nil {
		return nil, err
	}
	items, err := newPortalStore(db, s.client.Driver()).listInviteCodes(ctx)
	if err != nil {
		return nil, err
	}

	records := make([]InviteCodeRecord, 0, len(items))
	filter := strings.TrimSpace(strings.ToLower(query.Status))
	for _, item := range items {
		record := inviteCodeRecordFromEntity(item)
		if filter != "" && strings.ToLower(item.Status) != filter {
			continue
		}
		records = append(records, record)
	}

	pageNum := query.PageNum
	pageSize := query.PageSize
	if pageNum <= 0 {
		pageNum = 1
	}
	if pageSize <= 0 {
		pageSize = len(records)
	}
	start := (pageNum - 1) * pageSize
	if start >= len(records) {
		return []InviteCodeRecord{}, nil
	}
	end := min(start+pageSize, len(records))
	return records[start:end], nil
}

func (s *Service) UpdateInviteCodeStatus(ctx context.Context, inviteCode, status string) (*InviteCodeRecord, error) {
	db, err := s.db(ctx)
	if err != nil {
		return nil, err
	}
	code := strings.TrimSpace(inviteCode)
	if code == "" {
		return nil, errors.New("inviteCode cannot be blank")
	}
	normalized := strings.TrimSpace(strings.ToLower(status))
	if normalized != "active" && normalized != "disabled" {
		return nil, errors.New("status must be 'active' or 'disabled'")
	}

	updated, err := newPortalStore(db, s.client.Driver()).updateInviteCodeStatus(ctx, code, normalized)
	if err != nil {
		return nil, err
	}
	if !updated {
		return nil, fmt.Errorf("invite code not found: %s", code)
	}
	return s.getInviteCode(ctx, code)
}

func (s *Service) db(ctx context.Context) (*sql.DB, error) {
	if s.client == nil || !s.client.Enabled() || s.client.DB() == nil {
		return nil, errPortalUnavailable
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

func (s *Service) consumerExists(ctx context.Context, queryable interface {
	QueryRowContext(context.Context, string, ...any) *sql.Row
}, consumerName string) (bool, error) {
	var count int
	if err := queryable.QueryRowContext(ctx,
		`SELECT COUNT(1) FROM portal_user WHERE consumer_name = ? AND COALESCE(is_deleted, FALSE) = FALSE`,
		consumerName,
	).Scan(&count); err != nil {
		return false, err
	}
	return count > 0, nil
}

func (s *Service) lookupDepartmentIDByName(ctx context.Context, queryable interface {
	QueryRowContext(context.Context, string, ...any) *sql.Row
}, departmentName string) (string, error) {
	name := strings.TrimSpace(departmentName)
	if name == "" {
		return "", nil
	}
	var departmentID string
	err := queryable.QueryRowContext(ctx,
		`SELECT department_id FROM org_department WHERE name = ? AND status = 'active' LIMIT 1`,
		name,
	).Scan(&departmentID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return "", fmt.Errorf("department not found: %s", name)
		}
		return "", err
	}
	return departmentID, nil
}

func (s *Service) getDepartment(ctx context.Context, departmentID string) (*OrgDepartmentNode, error) {
	nodes, err := s.ListDepartmentTree(ctx)
	if err != nil {
		return nil, err
	}
	var walk func(items []*OrgDepartmentNode) *OrgDepartmentNode
	walk = func(items []*OrgDepartmentNode) *OrgDepartmentNode {
		for _, item := range items {
			if item.DepartmentID == departmentID {
				return item
			}
			if found := walk(item.Children); found != nil {
				return found
			}
		}
		return nil
	}
	return walk(nodes), nil
}

func (s *Service) getAccount(ctx context.Context, consumerName string) (*OrgAccountRecord, error) {
	items, err := s.ListAccounts(ctx)
	if err != nil {
		return nil, err
	}
	for _, item := range items {
		if item.ConsumerName == consumerName {
			return &item, nil
		}
	}
	return nil, nil
}

func (s *Service) isDepartmentAdmin(ctx context.Context, db *sql.DB, consumerName string) (bool, error) {
	var count int
	if err := db.QueryRowContext(ctx, `
		SELECT COUNT(1) FROM org_department
		WHERE admin_consumer_name = ? AND status = 'active'`, consumerName).Scan(&count); err != nil {
		return false, err
	}
	return count > 0, nil
}

func (s *Service) departmentMemberCounts(ctx context.Context, db *sql.DB) (map[string]int, error) {
	rows, err := db.QueryContext(ctx, `
		SELECT m.department_id, COUNT(1)
		FROM org_account_membership m
		INNER JOIN portal_user u ON u.consumer_name = m.consumer_name
		WHERE COALESCE(u.is_deleted, FALSE) = FALSE AND m.department_id IS NOT NULL AND m.department_id <> '' AND m.department_id <> ?
		GROUP BY m.department_id`, orgRootDepartmentID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	counts := map[string]int{}
	for rows.Next() {
		var departmentID string
		var count int
		if err := rows.Scan(&departmentID, &count); err != nil {
			return nil, err
		}
		counts[departmentID] = count
	}
	return counts, rows.Err()
}

func (s *Service) resolveDepartmentLevel(nodes map[string]*OrgDepartmentNode, departmentID string) int {
	level := 1
	current := nodes[departmentID]
	for current != nil && current.ParentDepartmentID != "" {
		level++
		current = nodes[current.ParentDepartmentID]
	}
	return level
}

func (s *Service) departmentMaps(ctx context.Context, db *sql.DB) (map[string]string, map[string]string, error) {
	rows, err := db.QueryContext(ctx, `
		SELECT department_id, name, parent_department_id, path
		FROM org_department
		WHERE status = 'active' AND department_id <> ?`, orgRootDepartmentID)
	if err != nil {
		return nil, nil, err
	}
	defer rows.Close()

	departments := map[string]departmentRef{}
	paths := map[string]string{}
	for rows.Next() {
		var (
			id     string
			name   string
			parent sql.NullString
			path   string
		)
		if err := rows.Scan(&id, &name, &parent, &path); err != nil {
			return nil, nil, err
		}
		departments[id] = departmentRef{name: name, parent: parent.String}
		paths[id] = strings.TrimSpace(path)
	}
	if err := rows.Err(); err != nil {
		return nil, nil, err
	}

	names := map[string]string{}
	for id, item := range departments {
		names[id] = item.name
		if strings.TrimSpace(paths[id]) == "" {
			paths[id] = buildDepartmentPath(departments, id)
		}
	}
	return names, paths, nil
}

func (s *Service) resolveDepartmentPlacement(ctx context.Context, db *sql.DB, parentDepartmentID, name string) (string, int, error) {
	parentID := strings.TrimSpace(parentDepartmentID)
	if parentID == "" || parentID == orgRootDepartmentID {
		return strings.TrimSpace(name), 1, nil
	}
	var (
		parentPath  string
		parentLevel int
	)
	if err := db.QueryRowContext(ctx, `
		SELECT path, level
		FROM org_department
		WHERE department_id = ? AND status = 'active'`,
		parentID,
	).Scan(&parentPath, &parentLevel); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return "", 0, fmt.Errorf("parent department not found: %s", parentID)
		}
		return "", 0, err
	}
	parentPath = strings.TrimSpace(parentPath)
	if parentPath == "" {
		parentPath = parentID
	}
	return parentPath + " / " + strings.TrimSpace(name), parentLevel + 1, nil
}

func (s *Service) isDepartmentDescendant(ctx context.Context, db *sql.DB, candidateParentID, departmentID string) bool {
	current := candidateParentID
	for current != "" {
		if current == departmentID {
			return true
		}
		var next sql.NullString
		if err := db.QueryRowContext(ctx, `
			SELECT parent_department_id
			FROM org_department
			WHERE department_id = ? AND status = 'active'`, current).Scan(&next); err != nil {
			return false
		}
		current = next.String
	}
	return false
}

func (s *Service) getInviteCode(ctx context.Context, inviteCode string) (*InviteCodeRecord, error) {
	db, err := s.db(ctx)
	if err != nil {
		return nil, err
	}
	item, err := newPortalStore(db, s.client.Driver()).getInviteCode(ctx, inviteCode)
	if err != nil {
		return nil, err
	}
	if item == nil {
		return nil, nil
	}
	record := inviteCodeRecordFromEntity(*item)
	return &record, nil
}

func scanInviteCode(scanner interface {
	Scan(...any) error
}) (InviteCodeRecord, error) {
	var item InviteCodeRecord
	var expiresAt sql.NullTime
	var usedAt sql.NullTime
	var createdAt sql.NullTime
	var usedByConsumer sql.NullString
	err := scanner.Scan(
		&item.InviteCode,
		&item.Status,
		&expiresAt,
		&usedByConsumer,
		&usedAt,
		&createdAt,
	)
	if err != nil {
		return InviteCodeRecord{}, err
	}
	if expiresAt.Valid {
		item.ExpiresAt = &expiresAt.Time
	}
	if usedAt.Valid {
		item.UsedAt = &usedAt.Time
	}
	if createdAt.Valid {
		item.CreatedAt = &createdAt.Time
	}
	item.UsedByConsumer = usedByConsumer.String
	return item, nil
}

func inviteCodeRecordFromEntity(item entity.PortalInviteCode) InviteCodeRecord {
	record := InviteCodeRecord{
		InviteCode:     item.InviteCode,
		Status:         item.Status,
		UsedByConsumer: item.UsedByConsumer,
	}
	if item.ExpiresAt != nil {
		value := item.ExpiresAt.Time
		record.ExpiresAt = &value
	}
	if item.UsedAt != nil {
		value := item.UsedAt.Time
		record.UsedAt = &value
	}
	if item.CreatedAt != nil {
		value := item.CreatedAt.Time
		record.CreatedAt = &value
	}
	return record
}

func normalizeStatus(status string, create bool) string {
	normalized := strings.ToLower(strings.TrimSpace(status))
	if normalized == "" {
		if create {
			return "active"
		}
		return "pending"
	}
	if !slices.Contains(allowedStatuses, normalized) {
		return "pending"
	}
	return normalized
}

func normalizeUserLevel(userLevel string) string {
	normalized := strings.ToLower(strings.TrimSpace(userLevel))
	if !slices.Contains(allowedUserLevels, normalized) {
		return "normal"
	}
	return normalized
}

func trimOrNil(value string) any {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return nil
	}
	return trimmed
}

func wrapGTime(value time.Time) *gtime.Time {
	return gtime.NewFromTime(value)
}

func nullIfEmpty(value string) any {
	if strings.TrimSpace(value) == "" {
		return nil
	}
	return value
}

func hashPassword(password string) (string, error) {
	bytes, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return "", err
	}
	return string(bytes), nil
}

func newTempPassword() string {
	return strings.ToUpper(strings.ReplaceAll(uuid.NewString()[:8], "-", ""))
}

func randomInviteCode() string {
	return strings.ReplaceAll(strings.ToUpper(uuid.NewString()[:16]), "-", "")
}

func buildDepartmentPath(departments map[string]departmentRef, departmentID string) string {
	if departmentID == "" {
		return ""
	}
	parts := make([]string, 0)
	current := departmentID
	for current != "" {
		item, ok := departments[current]
		if !ok {
			break
		}
		parts = append([]string{item.name}, parts...)
		current = item.parent
	}
	return strings.Join(parts, " / ")
}

func sortDepartmentTree(items []*OrgDepartmentNode) {
	sort.Slice(items, func(i, j int) bool {
		return items[i].Name < items[j].Name
	})
	for _, item := range items {
		sortDepartmentTree(item.Children)
	}
}
