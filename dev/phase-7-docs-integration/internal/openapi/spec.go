// Package openapi loads and validates the API contract for this phase.
package openapi

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"

	"github.com/getkin/kin-openapi/openapi3"
)

// SwaggerUIPath is the HTTP path where Swagger UI is mounted at root integration.
const SwaggerUIPath = "/swagger/index.html"

// RequiredPaths lists every auth/customer/product/order endpoint that must appear in the spec.
var RequiredPaths = []string{
	"/api/auth/login",
	"/api/auth/register",
	"/api/customers",
	"/api/customers/{id}",
	"/api/customers/{id}/deactivate",
	"/api/products",
	"/api/products/{id}",
	"/api/products/{id}/deactivate",
	"/api/orders",
	"/api/orders/{id}",
	"/api/orders/{id}/cancel",
	"/api/orders/{id}/ship",
}

// Load reads and validates docs/openapi.yaml.
func Load() (*openapi3.T, error) {
	data, err := os.ReadFile(specFilePath())
	if err != nil {
		return nil, fmt.Errorf("read openapi spec: %w", err)
	}
	loader := openapi3.NewLoader()
	doc, err := loader.LoadFromData(data)
	if err != nil {
		return nil, fmt.Errorf("parse openapi spec: %w", err)
	}
	if err := doc.Validate(loader.Context); err != nil {
		return nil, fmt.Errorf("validate openapi spec: %w", err)
	}
	return doc, nil
}

func specFilePath() string {
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		return filepath.Join("docs", "openapi.yaml")
	}
	return filepath.Join(filepath.Dir(file), "..", "..", "docs", "openapi.yaml")
}
