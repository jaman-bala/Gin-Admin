package postgres

import (
	"context"
	"gin_auth_service/internal/domain/auditlog"
	"time"

	"uuid"
	"github.com/jmoiron/sqlx"
)

// AuditLogDB represents the audit log record in the database.
type AuditLogDB struct {
	ID        uint      `db:"id"`
	UserID    uuid.UUID `db:"user_id"`
	Action    string    `db:"action"`
	Entity    string    `db:"entity"`
	EntityID  uuid.UUID `db:"entity_id"`
	ClientIP  string    `db:"client_ip"`
	UserAgent string    `db:"user_agent"`
	Data      string    `db:"data"`
	Status    int       `db:"status"`
	CreatedAt time.Time `db:"created_at"`
}

func (m *AuditLogDB) ToEntity() auditlog.AuditLog {
	return auditlog.AuditLog{
		ID:        m.ID,
		UserID:    m.UserID,
		Action:    m.Action,
		Entity:    m.Entity,
		EntityID:  m.EntityID,
		ClientIP:  m.ClientIP,
		UserAgent: m.UserAgent,
		Data:      m.Data,
		Status:    m.Status,
		CreatedAt: m.CreatedAt,
	}
}

func fromAuditLogEntity(e *auditlog.AuditLog) *AuditLogDB {
	return &AuditLogDB{
		ID:        e.ID,
		UserID:    e.UserID,
		Action:    e.Action,
		Entity:    e.Entity,
		EntityID:  e.EntityID,
		ClientIP:  e.ClientIP,
		UserAgent: e.UserAgent,
		Data:      e.Data,
		Status:    e.Status,
		CreatedAt: e.CreatedAt,
	}
}

type auditRepository struct {
	base
}

// NewAuditRepository creates a new instance of AuditRepository.
func NewAuditRepository(db *sqlx.DB) auditlog.Repository {
	return &auditRepository{base{db: db}}
}

func (r *auditRepository) Create(ctx context.Context, log *auditlog.AuditLog) error {
	query := `
		INSERT INTO audit_logs (user_id, action, entity, entity_id, client_ip, user_agent, data, status, created_at)
		VALUES (:user_id, :action, :entity, :entity_id, :client_ip, :user_agent, :data, :status, :created_at)
	`
	dbModel := fromAuditLogEntity(log)
	_, err := sqlx.NamedExecContext(ctx, r.q(ctx), query, dbModel)
	return err
}

// CreateBatch inserts multiple audit entries in a single multi-row statement.
// Used by the async recorder to keep audit writes off the request hot path.
func (r *auditRepository) CreateBatch(ctx context.Context, logs []auditlog.AuditLog) error {
	if len(logs) == 0 {
		return nil
	}
	rows := make([]AuditLogDB, 0, len(logs))
	for i := range logs {
		rows = append(rows, *fromAuditLogEntity(&logs[i]))
	}
	query := `
		INSERT INTO audit_logs (user_id, action, entity, entity_id, client_ip, user_agent, data, status, created_at)
		VALUES (:user_id, :action, :entity, :entity_id, :client_ip, :user_agent, :data, :status, :created_at)
	`
	_, err := sqlx.NamedExecContext(ctx, r.q(ctx), query, rows)
	return err
}

func (r *auditRepository) GetAll(ctx context.Context) ([]auditlog.AuditLog, error) {
	var logsDB []AuditLogDB
	query := `SELECT * FROM audit_logs ORDER BY created_at DESC`
	err := sqlx.SelectContext(ctx, r.q(ctx), &logsDB, query)
	if err != nil {
		return nil, err
	}
	return mapSlice(logsDB, (*AuditLogDB).ToEntity), nil
}

func (r *auditRepository) GetAllPaginated(ctx context.Context, page, limit int, entityID *uuid.UUID) ([]auditlog.AuditLog, int64, error) {
	where := ""
	var args []any
	if entityID != nil {
		where = " WHERE entity_id = $1"
		args = append(args, *entityID)
	}

	logsDB, total, err := r.selectPage[AuditLogDB](ctx, "audit_logs", where, "created_at DESC", page, limit, args...)
	if err != nil {
		return nil, 0, err
	}
	return mapSlice(logsDB, (*AuditLogDB).ToEntity), total, nil
}

func (r *auditRepository) GetByUserID(ctx context.Context, userID uuid.UUID) ([]auditlog.AuditLog, error) {
	var logsDB []AuditLogDB
	query := `SELECT * FROM audit_logs WHERE user_id = $1 ORDER BY created_at DESC`
	err := sqlx.SelectContext(ctx, r.q(ctx), &logsDB, query, userID)
	if err != nil {
		return nil, err
	}
	return mapSlice(logsDB, (*AuditLogDB).ToEntity), nil
}

func (r *auditRepository) GetByEntity(ctx context.Context, entity string) ([]auditlog.AuditLog, error) {
	var logsDB []AuditLogDB
	query := `SELECT * FROM audit_logs WHERE entity = $1 ORDER BY created_at DESC`
	err := sqlx.SelectContext(ctx, r.q(ctx), &logsDB, query, entity)
	if err != nil {
		return nil, err
	}
	return mapSlice(logsDB, (*AuditLogDB).ToEntity), nil
}
