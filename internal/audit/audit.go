// Package audit registra eventos relevantes de segurança e de negócio.
// Nunca registra senhas, tokens ou outras credenciais.
package audit

import (
	"context"
	"database/sql"
	"encoding/json"
)

type Recorder struct {
	db *sql.DB
}

func NewRecorder(db *sql.DB) *Recorder {
	return &Recorder{db: db}
}

// Record grava um evento de auditoria. Falhas ao gravar são deliberadamente
// não propagadas para não bloquear a operação de negócio principal — a
// auditoria é um registro complementar, não uma fonte de verdade financeira.
func (r *Recorder) Record(ctx context.Context, userID *string, action, entityType, entityID string, metadata map[string]any) {
	var metaJSON []byte
	if metadata != nil {
		metaJSON, _ = json.Marshal(metadata)
	}
	const q = `
		INSERT INTO audit_events (user_id, action, entity_type, entity_id, metadata)
		VALUES ($1, $2, $3, $4, $5)
	`
	_, _ = r.db.ExecContext(ctx, q, userID, action, entityType, entityID, metaJSON)
}
