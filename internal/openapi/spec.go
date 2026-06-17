// Package openapi embeds and serves the API contract (architecture §32) and
// exposes a loader used to validate it.
package openapi

import (
	_ "embed"
	"fmt"
	"net/http"

	"github.com/getkin/kin-openapi/openapi3"
)

// SwaggerUIPath is the HTTP path where Swagger UI is mounted.
const SwaggerUIPath = "/swagger/index.html"

// specYAML is the canonical OpenAPI document, embedded so it is available in a
// built binary without any filesystem dependency.
//
//go:embed openapi.yaml
var specYAML []byte

// RequiredPaths lists every auth/customer/product/order endpoint that must
// appear in the spec.
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

// Load parses and validates the embedded OpenAPI document.
func Load() (*openapi3.T, error) {
	loader := openapi3.NewLoader()
	doc, err := loader.LoadFromData(specYAML)
	if err != nil {
		return nil, fmt.Errorf("parse openapi spec: %w", err)
	}
	if err := doc.Validate(loader.Context); err != nil {
		return nil, fmt.Errorf("validate openapi spec: %w", err)
	}
	return doc, nil
}

const swaggerHTML = `<!DOCTYPE html>
<html lang="en">
<head>
  <meta charset="UTF-8">
  <title>Order Service API</title>
  <link rel="stylesheet" href="https://unpkg.com/swagger-ui-dist@5/swagger-ui.css">
</head>
<body>
  <div id="swagger-ui"></div>
  <script src="https://unpkg.com/swagger-ui-dist@5/swagger-ui-bundle.js"></script>
  <script>
    window.onload = function () {
      window.ui = SwaggerUIBundle({ url: "/swagger/openapi.yaml", dom_id: "#swagger-ui" });
    };
  </script>
</body>
</html>`

// SwaggerUIHandler serves the Swagger UI HTML page.
func SwaggerUIHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		_, _ = w.Write([]byte(swaggerHTML))
	}
}

// SpecHandler serves the raw OpenAPI YAML document.
func SpecHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/yaml")
		_, _ = w.Write(specYAML)
	}
}
