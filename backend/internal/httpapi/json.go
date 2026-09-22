package httpapi

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"mime"
	"net/http"
	"reflect"
	"strconv"
	"strings"
)

// maxBodyBytes caps request bodies; a calculate request is well under 100 B.
const maxBodyBytes = 4 << 10

// decodeJSON reads exactly one JSON object from the request body into a new
// T. Unknown fields, trailing data and non-object bodies are rejected. Every
// failure is a *Problem whose detail never contains Go's error text.
func decodeJSON[T any](w http.ResponseWriter, r *http.Request) (*T, error) {
	mediaType, _, err := mime.ParseMediaType(r.Header.Get("Content-Type"))
	if err != nil || mediaType != "application/json" {
		return nil, NewProblem(CodeUnsupportedMediaType, "Content-Type must be application/json")
	}

	dec := json.NewDecoder(http.MaxBytesReader(w, r.Body, maxBodyBytes))
	dec.DisallowUnknownFields()

	var v *T
	if err := dec.Decode(&v); err != nil {
		return nil, decodeProblem(err)
	}
	if v == nil {
		return nil, NewProblem(CodeInvalidBody, "request body must be a JSON object, got null")
	}
	if err := dec.Decode(&struct{}{}); !errors.Is(err, io.EOF) {
		if _, ok := errors.AsType[*http.MaxBytesError](err); ok {
			return nil, decodeProblem(err)
		}
		return nil, NewProblem(CodeInvalidBody, "request body must contain a single JSON object")
	}
	return v, nil
}

// decodeProblem translates a json.Decoder error into a problem naming the
// offending field or byte offset.
func decodeProblem(err error) *Problem {
	if e, ok := errors.AsType[*http.MaxBytesError](err); ok {
		return NewProblem(CodePayloadTooLarge,
			fmt.Sprintf("request body must not exceed %d bytes", e.Limit))
	}
	if errors.Is(err, io.EOF) {
		return NewProblem(CodeInvalidBody, "request body is empty")
	}
	if errors.Is(err, io.ErrUnexpectedEOF) {
		return NewProblem(CodeInvalidBody, "request body is truncated JSON")
	}
	if e, ok := errors.AsType[*json.SyntaxError](err); ok {
		return NewProblem(CodeInvalidBody,
			fmt.Sprintf("request body is not valid JSON (syntax error at byte %d)", e.Offset))
	}
	if e, ok := errors.AsType[*json.UnmarshalTypeError](err); ok {
		if e.Field == "" {
			return NewProblem(CodeInvalidBody,
				fmt.Sprintf("request body must be a JSON object, got %s", e.Value))
		}
		msg := "must be " + jsonTypeName(e.Type)
		if strings.HasPrefix(e.Value, "number") {
			// A number that does not fit, e.g. 1e400 into a float64.
			msg += " within the 64-bit floating-point range"
		}
		return NewProblem(CodeInvalidBody,
			fmt.Sprintf("field %q %s (byte %d)", e.Field, msg, e.Offset),
			FieldError{Field: e.Field, Message: msg})
	}
	// encoding/json has no typed error for unknown fields.
	if name, ok := strings.CutPrefix(err.Error(), "json: unknown field "); ok {
		if field, uerr := strconv.Unquote(name); uerr == nil {
			return NewProblem(CodeInvalidBody,
				fmt.Sprintf("field %q is not allowed", field),
				FieldError{Field: field, Message: "is not allowed"})
		}
	}
	return NewProblem(CodeInvalidBody, "request body could not be read")
}

// jsonTypeName names the JSON type that decodes into t.
func jsonTypeName(t reflect.Type) string {
	switch t.Kind() {
	case reflect.Float32, reflect.Float64,
		reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64,
		reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		return "a number"
	case reflect.String:
		return "a string"
	case reflect.Bool:
		return "a boolean"
	case reflect.Slice, reflect.Array:
		return "an array"
	default:
		return "an object"
	}
}

// writeJSON encodes v into memory first, so an encoding failure becomes a
// 500 problem instead of a half-written 200.
func (h *Handler) writeJSON(w http.ResponseWriter, r *http.Request, status int, v any) {
	body, err := json.Marshal(v)
	if err != nil {
		Write(w, r, h.mapError(fmt.Errorf("encode response: %w", err)))
		return
	}
	hdr := w.Header()
	hdr.Set("Content-Type", "application/json; charset=utf-8")
	hdr.Set("Content-Length", strconv.Itoa(len(body)))
	w.WriteHeader(status)
	_, _ = w.Write(body)
}
