package auth

import (
	"context"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"
)

func TestClientCachesValidation(t *testing.T) {
	var calls int32
	server := httptest.NewServer(http.HandlerFunc(func(response http.ResponseWriter, _ *http.Request) {
		atomic.AddInt32(&calls, 1)
		response.Header().Set("Content-Type", "application/json")
		_, _ = response.Write([]byte(`{"sub":"11111111-1111-1111-1111-111111111111","permissions":["catalog.read"],"type":"access","exp":4102444800}`))
	}))
	defer server.Close()
	client := NewClient(server.URL, 30*time.Second)
	for range 2 {
		user, err := client.Validate(context.Background(), "token")
		if err != nil || !user.HasPermission("catalog.read") {
			t.Fatalf("Validate() user=%v error=%v", user, err)
		}
	}
	if atomic.LoadInt32(&calls) != 1 {
		t.Fatalf("expected one call, got %d", calls)
	}
}
