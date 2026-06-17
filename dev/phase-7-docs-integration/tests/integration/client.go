//go:build integration

package integration

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/redis/go-redis/v9"
)

type apiClient struct {
	base   string
	token  string
	client *http.Client
}

func newAPIClient(base string) *apiClient {
	return &apiClient{
		base: base,
		client: &http.Client{
			Timeout: 15 * time.Second,
		},
	}
}

func (c *apiClient) login(t *testing.T, email, password string) {
	t.Helper()
	body := map[string]string{"email": email, "password": password}
	var resp tokenResponse
	c.doJSONAuth(t, http.MethodPost, "/api/auth/login", body, "", http.StatusOK, &resp)
	if resp.AccessToken == "" {
		t.Fatal("login returned empty access_token")
	}
	c.token = resp.AccessToken
}

func (c *apiClient) doJSON(t *testing.T, method, path string, reqBody any, wantStatus int, out any) {
	t.Helper()
	c.doJSONAuth(t, method, path, reqBody, c.token, wantStatus, out)
}

func (c *apiClient) doJSONAuth(t *testing.T, method, path string, reqBody any, token string, wantStatus int, out any) {
	t.Helper()
	var body io.Reader
	if reqBody != nil {
		b, err := json.Marshal(reqBody)
		if err != nil {
			t.Fatalf("marshal request: %v", err)
		}
		body = bytes.NewReader(b)
	}
	req, err := http.NewRequest(method, c.base+path, body)
	if err != nil {
		t.Fatalf("new request: %v", err)
	}
	if reqBody != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	res, err := c.client.Do(req)
	if err != nil {
		t.Fatalf("%s %s: %v", method, path, err)
	}
	defer res.Body.Close()
	raw, err := io.ReadAll(res.Body)
	if err != nil {
		t.Fatalf("read body: %v", err)
	}
	if res.StatusCode != wantStatus {
		t.Fatalf("%s %s: status %d, want %d, body=%s", method, path, res.StatusCode, wantStatus, string(raw))
	}
	if out != nil && len(raw) > 0 && res.StatusCode != http.StatusNoContent {
		if err := json.Unmarshal(raw, out); err != nil {
			t.Fatalf("decode response: %v, body=%s", err, string(raw))
		}
	}
}

func (c *apiClient) getStatus(t *testing.T, path string, token string) int {
	t.Helper()
	req, err := http.NewRequest(http.MethodGet, c.base+path, nil)
	if err != nil {
		t.Fatalf("new request: %v", err)
	}
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	res, err := c.client.Do(req)
	if err != nil {
		t.Fatalf("GET %s: %v", path, err)
	}
	defer res.Body.Close()
	_, _ = io.Copy(io.Discard, res.Body)
	return res.StatusCode
}

func (c *apiClient) fetchMetrics(t *testing.T) string {
	t.Helper()
	req, err := http.NewRequest(http.MethodGet, c.base+"/metrics", nil)
	if err != nil {
		t.Fatalf("new request: %v", err)
	}
	res, err := c.client.Do(req)
	if err != nil {
		t.Fatalf("GET /metrics: %v", err)
	}
	defer res.Body.Close()
	raw, err := io.ReadAll(res.Body)
	if err != nil {
		t.Fatalf("read metrics: %v", err)
	}
	if res.StatusCode != http.StatusOK {
		t.Fatalf("GET /metrics: status %d", res.StatusCode)
	}
	return string(raw)
}

func waitForOrderStatus(t *testing.T, c *apiClient, orderID string, want string, timeout time.Duration) orderResponse {
	t.Helper()
	deadline := time.Now().Add(timeout)
	var last orderResponse
	for time.Now().Before(deadline) {
		c.doJSON(t, http.MethodGet, "/api/orders/"+orderID, nil, http.StatusOK, &last)
		if last.Status == want {
			return last
		}
		time.Sleep(200 * time.Millisecond)
	}
	t.Fatalf("order %s did not reach status %s within %s (last=%s)", orderID, want, timeout, last.Status)
	return last
}

func parseMetricValue(metrics, name string) (int64, bool) {
	prefix := name + " "
	for _, line := range bytes.Split([]byte(metrics), []byte("\n")) {
		if bytes.HasPrefix(line, []byte(prefix)) {
			var v int64
			_, err := fmt.Sscanf(string(line), name+" %d", &v)
			if err != nil {
				return 0, false
			}
			return v, true
		}
	}
	return 0, false
}

type tokenResponse struct {
	AccessToken string `json:"access_token"`
	TokenType   string `json:"token_type"`
	ExpiresIn   int    `json:"expires_in"`
}

type customerResponse struct {
	ID       string `json:"id"`
	Name     string `json:"name"`
	Document string `json:"document"`
	Email    string `json:"email"`
	Phone    string `json:"phone"`
	Active   bool   `json:"active"`
}

type productResponse struct {
	ID    string  `json:"id"`
	Name  string  `json:"name"`
	SKU   string  `json:"sku"`
	Price float64 `json:"price"`
}

type orderResponse struct {
	ID          string          `json:"id"`
	CustomerID  string          `json:"customer_id"`
	Status      string          `json:"status"`
	TotalAmount float64         `json:"total_amount"`
	Items       []orderItemResp `json:"items"`
}

type orderItemResp struct {
	ProductID  string  `json:"product_id"`
	Quantity   int     `json:"quantity"`
	UnitPrice  float64 `json:"unit_price"`
	TotalPrice float64 `json:"total_price"`
}

func connectPostgres(t *testing.T, ctx context.Context, dsn string) *pgxpool.Pool {
	t.Helper()
	pool, err := pgxpool.New(ctx, dsn)
	if err != nil {
		t.Fatalf("postgres connect: %v", err)
	}
	if err := pool.Ping(ctx); err != nil {
		pool.Close()
		t.Fatalf("postgres ping: %v", err)
	}
	t.Cleanup(pool.Close)
	return pool
}

func connectRedis(t *testing.T, ctx context.Context, addr string) *redis.Client {
	t.Helper()
	client := redis.NewClient(&redis.Options{
		Addr:     addr,
		Password: os.Getenv("REDIS_PASSWORD"),
	})
	if err := client.Ping(ctx).Err(); err != nil {
		_ = client.Close()
		t.Fatalf("redis ping: %v", err)
	}
	t.Cleanup(func() { _ = client.Close() })
	return client
}
