package objectstore

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"testing"
	"time"
)

// TestStoreRoundTrip exercises put, presigned get, and delete against a real
// S3-compatible endpoint (the compose Garage service). It is skipped unless
// OBJECT_STORE_TEST_ENDPOINT is set.
func TestStoreRoundTrip(t *testing.T) {
	endpoint := os.Getenv("OBJECT_STORE_TEST_ENDPOINT")
	if endpoint == "" {
		t.Skip("OBJECT_STORE_TEST_ENDPOINT is not set")
	}
	store, err := New(Config{
		Endpoint:        endpoint,
		Region:          envOr("OBJECT_STORE_TEST_REGION", "garage"),
		Bucket:          os.Getenv("OBJECT_STORE_TEST_BUCKET"),
		AccessKeyID:     os.Getenv("OBJECT_STORE_TEST_ACCESS_KEY_ID"),
		SecretAccessKey: os.Getenv("OBJECT_STORE_TEST_SECRET_ACCESS_KEY"),
	})
	if err != nil {
		t.Fatal(err)
	}
	if store == nil {
		t.Fatal("store is unconfigured despite OBJECT_STORE_TEST_ENDPOINT being set")
	}
	ctx := context.Background()
	key := fmt.Sprintf("test/roundtrip-%d.txt", time.Now().UnixNano())
	payload := []byte("hello nutrition photo")

	if err := store.Put(ctx, key, "text/plain", bytes.NewReader(payload)); err != nil {
		t.Fatalf("put: %v", err)
	}

	url, err := store.PresignGet(ctx, key, time.Minute)
	if err != nil {
		t.Fatalf("presign get: %v", err)
	}
	got := httpGet(t, url)
	if !bytes.Equal(got, payload) {
		t.Fatalf("presigned GET returned %q, want %q", got, payload)
	}

	if err := store.Delete(ctx, key); err != nil {
		t.Fatalf("delete: %v", err)
	}

	// After deletion the presigned URL must no longer resolve to the object.
	afterURL, err := store.PresignGet(ctx, key, time.Minute)
	if err != nil {
		t.Fatalf("presign get after delete: %v", err)
	}
	if status := httpStatus(t, afterURL); status == http.StatusOK {
		t.Fatalf("object still readable after delete (status %d)", status)
	}
}

func httpGet(t *testing.T, url string) []byte {
	t.Helper()
	resp, err := http.Get(url)
	if err != nil {
		t.Fatalf("GET presigned URL: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("presigned GET status = %d", resp.StatusCode)
	}
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("read body: %v", err)
	}
	return body
}

func httpStatus(t *testing.T, url string) int {
	t.Helper()
	resp, err := http.Get(url)
	if err != nil {
		t.Fatalf("GET presigned URL: %v", err)
	}
	defer resp.Body.Close()
	_, _ = io.Copy(io.Discard, resp.Body)
	return resp.StatusCode
}

func envOr(key, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return fallback
}
