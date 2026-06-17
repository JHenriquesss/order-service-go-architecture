//go:build integration

package integration

import (
	"net/http"
	"testing"
)

func TestUnauthenticatedRequestRejectedWith401(t *testing.T) {
	apiBase := requireAPIEnv(t)
	client := newAPIClient(apiBase)

	status := client.getStatus(t, "/api/customers", "")
	if status != http.StatusUnauthorized {
		t.Fatalf("GET /api/customers without token: status %d, want 401", status)
	}
}
