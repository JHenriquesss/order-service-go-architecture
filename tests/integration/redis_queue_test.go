//go:build integration

package integration

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/google/uuid"
)

func TestRedisQueuePublishConsumeRoundTrip(t *testing.T) {
	redisAddr := requireRedisEnv(t)
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	client := connectRedis(t, ctx, redisAddr)
	queueKey := orderQueueName + ":test:" + uniqueSuffix()

	msg := map[string]any{
		"order_id":   uuid.New().String(),
		"event":      "ORDER_CREATED",
		"created_at": time.Now().UTC().Format(time.RFC3339Nano),
	}
	payload, err := json.Marshal(msg)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}

	if err := client.LPush(ctx, queueKey, payload).Err(); err != nil {
		t.Fatalf("lpush: %v", err)
	}

	res, err := client.BRPop(ctx, 2*time.Second, queueKey).Result()
	if err != nil {
		t.Fatalf("brpop: %v", err)
	}
	if len(res) != 2 {
		t.Fatalf("unexpected brpop result: %v", res)
	}

	var got map[string]any
	if err := json.Unmarshal([]byte(res[1]), &got); err != nil {
		t.Fatalf("unmarshal consumed message: %v", err)
	}
	if got["event"] != "ORDER_CREATED" {
		t.Fatalf("event mismatch: %v", got["event"])
	}
}
