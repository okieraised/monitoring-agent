package api_response

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/google/uuid"
	"github.com/okieraised/monitoring-agent/internal/constants"
)

type Response[T any] struct {
	RequestID     string         `json:"request_id"`
	Code          string         `json:"code"`
	Message       string         `json:"message"`
	ServerTime    int64          `json:"server_time"`
	ServerTimeISO string         `json:"server_time_iso"`
	Count         int            `json:"count,omitempty"`
	Data          T              `json:"data"`
	Agg           any            `json:"agg,omitempty"`
	Meta          map[string]any `json:"meta,omitempty"`
}

// Pagination is a common meta-payload for list endpoints.
type Pagination struct {
	Page       int   `json:"page"`
	PerPage    int   `json:"per_page"`
	Total      int64 `json:"total"`
	TotalPages int   `json:"total_pages"`
}

type BaseOutput struct {
	Status  int
	Code    string
	Message string
	Data    any
	Count   int
	Meta    any
}

func New[T any](ctx context.Context) *Response[T] {
	now := time.Now()
	return &Response[T]{
		RequestID:     requestIDFromContext(ctx),
		ServerTime:    now.UnixMilli(),
		ServerTimeISO: now.Format(time.RFC3339Nano),
	}
}

// OK constructs a success response with data.
func OK[T any](ctx context.Context, data T) *Response[T] {
	resp := New[T](ctx)
	resp.Data = data
	return resp
}

// Error constructs an error response with a code/message.
func Error[T any](ctx context.Context, code, message string) *Response[T] {
	resp := New[T](ctx)
	resp.Code = code
	resp.Message = message
	return resp
}

// FromError converts an error into a response. If err is nil, returns OK with zero T.
func FromError[T any](ctx context.Context, code string, err error) *Response[T] {
	if err == nil {
		return OK[T](ctx, *new(T))
	}
	resp := New[T](ctx)
	resp.Code = code
	resp.Message = err.Error()
	return resp
}

func (r *Response[T]) WithCode(c string) *Response[T]    { r.Code = c; return r }
func (r *Response[T]) WithMessage(m string) *Response[T] { r.Message = m; return r }
func (r *Response[T]) WithCount(n int) *Response[T]      { r.Count = n; return r }
func (r *Response[T]) WithAgg(agg any) *Response[T]      { r.Agg = agg; return r }
func (r *Response[T]) WithMetaKV(k string, v any) *Response[T] {
	if r.Meta == nil {
		r.Meta = make(map[string]any)
	}
	r.Meta[k] = v
	return r
}
func (r *Response[T]) WithMeta(m map[string]any) *Response[T] {
	if m == nil {
		return r
	}
	if r.Meta == nil {
		r.Meta = make(map[string]any, len(m))
	}
	for k, v := range m {
		r.Meta[k] = v
	}
	return r
}
func (r *Response[T]) WithPagination(p Pagination) *Response[T] {
	return r.WithMetaKV("pagination", p)
}

// Populate keeps parity with your original method signature.
// Prefer fluent setters in new code.
func (r *Response[T]) Populate(code, message string, data T, meta any, count any) *Response[T] {
	r.Code = code
	r.Message = message
	r.Data = data
	if meta != nil {
		switch m := meta.(type) {
		case map[string]any:
			r.WithMeta(m)
		default:
			r.WithMetaKV("meta", m)
		}
	}
	if count != nil {
		if total, ok := count.(int); ok {
			r.Count = total
		}
	}
	return r
}

// WriteJSON writes the response with the given HTTP status.
// It sets Content-Type and echoes X-Request-ID if present.
func WriteJSON[T any](w http.ResponseWriter, status int, resp *Response[T]) error {
	if resp == nil {
		return errors.New("nil response")
	}
	// headers
	if resp.RequestID != "" {
		w.Header().Set(constants.HeaderXRequestID, resp.RequestID)
	}
	w.Header().Set(constants.HeaderContentType, constants.ContentTypeJSON)
	w.WriteHeader(status)
	enc := json.NewEncoder(w)
	enc.SetEscapeHTML(true)
	return enc.Encode(resp)
}

// WriteProblemJSON is a tiny helper for problem+json style errors.
func WriteProblemJSON(w http.ResponseWriter, status int, typ, title, detail, instance string) error {
	w.Header().Set(constants.HeaderContentType, constants.ContentTypeProblemJSON)
	w.WriteHeader(status)
	payload := map[string]any{
		"type":            typ,
		"title":           title,
		"status":          status,
		"detail":          detail,
		"instance":        instance,
		"server_time":     time.Now().UTC().UnixMilli(),
		"server_time_iso": time.Now().UTC().Format(time.RFC3339Nano),
	}
	return json.NewEncoder(w).Encode(payload)
}

func requestIDFromContext(ctx context.Context) string {
	if ctx == nil {
		return uuid.New().String()
	}

	if v := ctx.Value(constants.APIFieldRequestID); v != nil {
		return fmt.Sprint(v)
	}
	return uuid.New().String()
}
