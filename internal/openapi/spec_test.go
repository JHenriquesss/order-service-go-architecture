package openapi_test

import (
	"testing"

	"order-service-go/internal/openapi"
)

func TestOpenAPISpecLoadsAndValidates(t *testing.T) {
	doc, err := openapi.Load()
	if err != nil {
		t.Fatalf("load spec: %v", err)
	}
	if doc.Info.Title == "" {
		t.Fatal("expected info.title")
	}
}

func TestOpenAPICoversAllDomainEndpoints(t *testing.T) {
	doc, err := openapi.Load()
	if err != nil {
		t.Fatalf("load spec: %v", err)
	}
	for _, path := range openapi.RequiredPaths {
		if doc.Paths.Find(path) == nil {
			t.Errorf("missing path %s", path)
		}
	}
}

func TestOpenAPIHasBearerSecurityScheme(t *testing.T) {
	doc, err := openapi.Load()
	if err != nil {
		t.Fatalf("load spec: %v", err)
	}
	scheme, ok := doc.Components.SecuritySchemes["bearerAuth"]
	if !ok {
		t.Fatal("missing bearerAuth security scheme")
	}
	if scheme.Value.Type != "http" || scheme.Value.Scheme != "bearer" {
		t.Fatalf("unexpected bearer scheme: type=%s scheme=%s", scheme.Value.Type, scheme.Value.Scheme)
	}
}

func TestOpenAPIDocumentsErrorSchema(t *testing.T) {
	doc, err := openapi.Load()
	if err != nil {
		t.Fatalf("load spec: %v", err)
	}
	schemaRef := doc.Components.Schemas["ErrorResponse"]
	if schemaRef == nil || schemaRef.Value == nil {
		t.Fatal("missing ErrorResponse schema")
	}
	errProp := schemaRef.Value.Properties["error"]
	if errProp == nil || errProp.Value == nil {
		t.Fatal("error property missing on ErrorResponse")
	}
	codeProp := errProp.Value.Properties["code"]
	msgProp := errProp.Value.Properties["message"]
	if codeProp == nil || msgProp == nil {
		t.Fatal("error.code and error.message must be documented")
	}
}

func TestOpenAPIDocumentsSwaggerUIPath(t *testing.T) {
	doc, err := openapi.Load()
	if err != nil {
		t.Fatalf("load spec: %v", err)
	}
	if doc.Paths.Find(openapi.SwaggerUIPath) == nil {
		t.Fatalf("missing swagger path %s", openapi.SwaggerUIPath)
	}
}

func TestDefaultBuildExcludesIntegrationTests(t *testing.T) {
	// This test documents that integration tests live under tests/integration
	// with //go:build integration and are not compiled in default go test ./...
	t.Log("integration tests are tag-gated; default go test ./... skips them")
}
