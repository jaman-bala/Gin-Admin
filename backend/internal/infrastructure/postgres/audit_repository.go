package postgres

import (
	"context"
	"fmt"
	"gin_auth_service/internal/domain/auditlog"
	"time"

	"github.com/google/uuid"
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
	db *sqlx.DB
}

// NewAuditRepository creates a new instance of AuditRepository.
func NewAuditRepository(db *sqlx.DB) auditlog.Repository {
	return &auditRepository{db: db}
}

func (r *auditRepository) Create(ctx context.Context, log *auditlog.AuditLog) error {
	query := `
		INSERT INTO audit_logs (user_id, action, entity, entity_id, client_ip, user_agent, data, status, created_at)
		VALUES (:user_id, :action, :entity, :entity_id, :client_ip, :user_agent, :data, :status, :created_at)
	`
	dbModel := fromAuditLogEntity(log)
	_, err := r.db.NamedExecContext(ctx, query, dbModel)
	return err
}

func (r *auditRepository) GetAll(ctx context.Context) ([]auditlog.AuditLog, error) {
	var logsDB []AuditLogDB
	query := `SELECT * FROM audit_logs ORDER BY created_at DESC`
	err := r.db.SelectContext(ctx, &logsDB, query)
	if err != nil {
		return nil, err
	}

	var logs []auditlog.AuditLog
	for _, lDB := range logsDB {
		logs = append(logs, lDB.ToEntity())
	}
	return logs, nil
}

func (r *auditRepository) GetAllPaginated(ctx context.Context, page, limit int, entityID *uuid.UUID) ([]auditlog.AuditLog, int64, error) {
	var logsDB []AuditLogDB
	var total int64

	where := ""
	var args []interface{}
	if entityID != nil {
		where = " WHERE entity_id = $1"
		args = append(args, entityID)
	}

	// Count total records
	countQuery := `SELECT COUNT(*) FROM audit_logs` + where
	err := r.db.GetContext(ctx, &total, countQuery, args...)
	if err != nil {
		return nil, 0, err
	}

	// Get paginated records
	offset := (page - 1) * limit
	limitIdx := len(args) + 1
	offsetIdx := len(args) + 2
	query := `SELECT * FROM audit_logs` + where + fmt.Sprintf(" ORDER BY created_at DESC LIMIT $%d OFFSET $%d", limitIdx, offsetIdx)
	
	args = append(args, limit, offset)
	err = r.db.SelectContext(ctx, &logsDB, query, args...)
	if err != nil {
		return nil, 0, err
	}

	var logs []auditlog.AuditLog
	for _, lDB := range logsDB {
		logs = append(logs, lDB.ToEntity())
	}
	return logs, total, nil
}

func (r *auditRepository) GetByUserID(ctx context.Context, userID uuid.UUID) ([]auditlog.AuditLog, error) {
	var logsDB []AuditLogDB
	query := `SELECT * FROM audit_logs WHERE user_id = $1 ORDER BY created_at DESC`
	err := r.db.SelectContext(ctx, &logsDB, query, userID)
	if err != nil {
		return nil, err
	}

	var logs []auditlog.AuditLog
	for _, lDB := range logsDB {
		logs = append(logs, lDB.ToEntity())
	}
	return logs, nil
}

func (r *auditRepository) GetByEntity(ctx context.Context, entity string) ([]auditlog.AuditLog, error) {
	var logsDB []AuditLogDB
	query := `SELECT * FROM audit_logs WHERE entity = $1 ORDER BY created_at DESC`
	err := r.db.SelectContext(ctx, &logsDB, query, entity)
	if err != nil {
		return nil, err
	}

	var logs []auditlog.AuditLog
	for _, lDB := range logsDB {
		logs = append(logs, lDB.ToEntity())
	}
	return logs, nil
}
