package httpapi

import (
	"net/http"
	"strconv"

	"github.com/isasumer/go-react-calculator/backend/api"
)

// specContentType is the media type registered for OpenAPI documents
// (RFC 9512). Redoc, Swagger UI and the contract tests all accept it.
const specContentType = "application/yaml"

// openapiSpec serves the embedded contract at GET /api/v1/openapi.yaml.
//
// It is served by the process that implements it, so the description a client
// fetches at run time is the one this binary was built from; there is no
// second copy to drift. no-store for the same reason: a cached spec would
// outlive the deployment it describes.
func (h *Handler) openapiSpec(w http.ResponseWriter, _ *http.Request) {
	body := api.Spec()
	hdr := w.Header()
	hdr.Set("Content-Type", specContentType)
	hdr.Set("Cache-Control", noStore)
	hdr.Set("Content-Length", strconv.Itoa(len(body)))
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(body)
}
