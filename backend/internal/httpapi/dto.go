package httpapi

// CalculateRequest is the body of POST /api/v1/calculate. Operands are
// pointers so a missing field is distinguishable from zero; a JSON null is
// treated as missing.
type CalculateRequest struct {
	Operation string   `json:"operation"`
	A         *float64 `json:"a"`
	B         *float64 `json:"b"`
}

// CalculateResponse echoes the evaluated request with its result. B is
// omitted for unary operations.
type CalculateResponse struct {
	Operation string   `json:"operation"`
	A         float64  `json:"a"`
	B         *float64 `json:"b,omitempty"`
	Result    float64  `json:"result"`
}

// OperationsResponse is the body of GET /api/v1/operations.
type OperationsResponse struct {
	Operations []OperationInfo `json:"operations"`
}

// OperationInfo describes one supported operation.
type OperationInfo struct {
	Name   string `json:"name"`
	Symbol string `json:"symbol"`
	Arity  int    `json:"arity"`
}
