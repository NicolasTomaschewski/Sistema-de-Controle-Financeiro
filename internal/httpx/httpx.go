// Package httpx padroniza respostas JSON, decodificação de entrada e erros
// de domínio para todos os handlers HTTP da API.
package httpx

import (
	"encoding/json"
	"errors"
	"net/http"
)

type envelope struct {
	Data any `json:"data,omitempty"`
	Meta any `json:"meta,omitempty"`
}

type errorEnvelope struct {
	Error errorBody `json:"error"`
}

type errorBody struct {
	Code    string            `json:"code"`
	Message string            `json:"message"`
	Fields  map[string]string `json:"fields,omitempty"`
}

// JSON escreve uma resposta de sucesso padronizada.
func JSON(w http.ResponseWriter, status int, data any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(envelope{Data: data})
}

// JSONWithMeta escreve uma resposta de sucesso com metadados (ex.: paginação).
func JSONWithMeta(w http.ResponseWriter, status int, data, meta any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(envelope{Data: data, Meta: meta})
}

// Error escreve uma resposta de erro padronizada, sem expor detalhes internos.
func Error(w http.ResponseWriter, status int, code, message string) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(errorEnvelope{Error: errorBody{Code: code, Message: message}})
}

// ValidationError escreve uma resposta 422 com os campos inválidos indicados.
func ValidationError(w http.ResponseWriter, fields map[string]string) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(http.StatusUnprocessableEntity)
	_ = json.NewEncoder(w).Encode(errorEnvelope{Error: errorBody{
		Code:    "validation_error",
		Message: "Um ou mais campos são inválidos.",
		Fields:  fields,
	}})
}

// DecodeJSON decodifica o corpo JSON da requisição de forma estrita,
// rejeitando campos desconhecidos e corpos malformados.
func DecodeJSON(r *http.Request, dst any) error {
	dec := json.NewDecoder(r.Body)
	dec.DisallowUnknownFields()
	if err := dec.Decode(dst); err != nil {
		return err
	}
	return nil
}

// Sentinel errors de domínio reconhecidos genericamente pelos handlers.
var (
	ErrNotFound     = errors.New("recurso não encontrado")
	ErrForbidden    = errors.New("acesso não autorizado ao recurso")
	ErrConflict     = errors.New("conflito de estado do recurso")
	ErrInvalidInput = errors.New("dados de entrada inválidos")
)

// NotFoundOrForbidden responde de forma consistente para recursos
// inexistentes ou não pertencentes ao usuário autenticado, sem revelar qual
// dos dois casos ocorreu.
func NotFoundOrForbidden(w http.ResponseWriter) {
	Error(w, http.StatusNotFound, "not_found", "Recurso não encontrado.")
}
