package httpapi

import (
	"encoding/json"
	"net/http"
	"strconv"
	"strings"
)

// typeBaseURL is the error catalog page; each code's lower-cased name is a
// heading anchor on that page.
const typeBaseURL = "https://github.com/isasumer/go-react-calculator/blob/main/docs/errors.md#"

// Code is a stable, machine-readable error code. Codes are listed in
// docs/errors.md and are never renamed once released.
type Code string

// Error codes.
const (
	CodeInvalidBody          Code = "INVALID_BODY"
	CodeValidationFailed     Code = "VALIDATION_FAILED"
	CodeUnsupportedOperation Code = "UNSUPPORTED_OPERATION"
	CodeUnexpectedOperand    Code = "UNEXPECTED_OPERAND"
	CodeDivisionByZero       Code = "DIVISION_BY_ZERO"
	CodeDomainError          Code = "DOMAIN_ERROR"
	CodeResultNotFinite      Code = "RESULT_NOT_FINITE"
	CodeUnsupportedMediaType Code = "UNSUPPORTED_MEDIA_TYPE"
	CodePayloadTooLarge      Code = "PAYLOAD_TOO_LARGE"
	CodeNotFound             Code = "NOT_FOUND"
	CodeMethodNotAllowed     Code = "METHOD_NOT_ALLOWED"
	CodeInternal             Code = "INTERNAL"
)

// meta returns the HTTP status and human-readable title for c. Unknown codes
// are reported as internal errors.
func (c Code) meta() (status int, title string) {
	switch c {
	case CodeInvalidBody:
		return http.StatusBadRequest, "Invalid request body"
	case CodeValidationFailed:
		return http.StatusBadRequest, "Validation failed"
	case CodeUnsupportedOperation:
		return http.StatusUnprocessableEntity, "Unsupported operation"
	case CodeUnexpectedOperand:
		return http.StatusUnprocessableEntity, "Unexpected operand"
	case CodeDivisionByZero:
		return http.StatusUnprocessableEntity, "Division by zero"
	case CodeDomainError:
		return http.StatusUnprocessableEntity, "Result is not a real number"
	case CodeResultNotFinite:
		return http.StatusUnprocessableEntity, "Result is not finite"
	case CodeUnsupportedMediaType:
		return http.StatusUnsupportedMediaType, "Unsupported media type"
	case CodePayloadTooLarge:
		return http.StatusRequestEntityTooLarge, "Payload too large"
	case CodeNotFound:
		return http.StatusNotFound, "Not found"
	case CodeMethodNotAllowed:
		return http.StatusMethodNotAllowed, "Method not allowed"
	default:
		return http.StatusInternalServerError, "Internal server error"
	}
}

// FieldError points at one invalid request field.
type FieldError struct {
	Field   string `json:"field"`
	Message string `json:"message"`
}

// Problem is an RFC 9457 problem details document with a stable Code.
// It implements error so decoding and validation can return it through
// mapError unchanged.
type Problem struct {
	Type      string       `json:"type"`
	Title     string       `json:"title"`
	Status    int          `json:"status"`
	Detail    string       `json:"detail"`
	Code      Code         `json:"code"`
	Instance  string       `json:"instance"`
	RequestID string       `json:"requestId,omitempty"`
	Errors    []FieldError `json:"errors,omitempty"`
}

// NewProblem builds the problem for code, filling type, title and status
// from the error catalog. Instance and RequestID are filled by [Write].
func NewProblem(code Code, detail string, errs ...FieldError) *Problem {
	status, title := code.meta()
	return &Problem{
		Type:   typeBaseURL + strings.ToLower(string(code)),
		Title:  title,
		Status: status,
		Detail: detail,
		Code:   code,
		Errors: errs,
	}
}

func (p *Problem) Error() string {
	return string(p.Code) + ": " + p.Detail
}

// Write sends p as application/problem+json. Instance defaults to the request
// path; RequestID is copied from an X-Request-ID response header when one has
// already been set. Headers set on w before the call (such as Allow) are kept.
func Write(w http.ResponseWriter, r *http.Request, p *Problem) {
	out := *p
	if out.Instance == "" {
		out.Instance = r.URL.Path
	}
	if out.RequestID == "" {
		out.RequestID = w.Header().Get("X-Request-ID")
	}
	body, err := json.Marshal(out)
	if err != nil {
		// Problem holds only strings and ints, so this cannot happen.
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}
	h := w.Header()
	h.Set("Content-Type", "application/problem+json")
	h.Set("Content-Length", strconv.Itoa(len(body)))
	w.WriteHeader(out.Status)
	_, _ = w.Write(body)
}
