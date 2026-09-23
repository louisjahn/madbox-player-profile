package main

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
)

type TestServer struct {
	s  *httptest.Server
	db *mongo.Database
}

// decode unmarshals body into T, failing the test on malformed JSON.
func decode[T any](t *testing.T, body []byte) T {
	t.Helper()
	var v T
	if err := json.Unmarshal(body, &v); err != nil {
		t.Fatalf("decode %T: %v\nbody: %s", v, err, body)
	}
	return v
}

func uniqueID(t *testing.T) string {
	t.Helper()
	id := "test_" + strings.ToLower(strings.NewReplacer("/", "_", " ", "_").Replace(t.Name()))
	return id + "_" + bson.NewObjectID().Hex()
}

// do sends req, asserts the status code and returns the body bytes.
// On mismatch it fails the test with the body, which is where the
// error message you actually need lives.
func (ts *TestServer) do(t *testing.T, req *http.Request, wantStatus int) []byte {
	t.Helper()

	resp, err := ts.s.Client().Do(req)
	if err != nil {
		t.Fatalf("%s %s: %v", req.Method, req.URL.Path, err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("%s %s: read body: %v", req.Method, req.URL.Path, err)
	}

	if resp.StatusCode != wantStatus {
		t.Fatalf("%s %s: status = %d, want %d\nbody: %s",
			req.Method, req.URL.Path, resp.StatusCode, wantStatus, body)
	}
	return body
}

func (ts *TestServer) request(t *testing.T, method, path, body string) *http.Request {
	t.Helper()

	var rdr io.Reader
	if body != "" {
		rdr = strings.NewReader(body)
	}

	req, err := http.NewRequest(method, ts.s.URL+path, rdr)
	if err != nil {
		t.Fatalf("build %s %s: %v", method, path, err)
	}
	if body != "" {
		req.Header.Set("Content-Type", "application/json")
	}
	return req
}

func (ts *TestServer) get(t *testing.T, path string, wantStatus int) []byte {
	t.Helper()
	return ts.do(t, ts.request(t, http.MethodGet, path, ""), wantStatus)
}

func (ts *TestServer) put(t *testing.T, path, body string, wantStatus int) []byte {
	t.Helper()
	return ts.do(t, ts.request(t, http.MethodPut, path, body), wantStatus)
}

func (ts *TestServer) post(t *testing.T, path, body string, wantStatus int) []byte {
	t.Helper()
	return ts.do(t, ts.request(t, http.MethodPost, path, body), wantStatus)
}

func (ts *TestServer) patch(t *testing.T, path, body string, wantStatus int) []byte {
	t.Helper()
	return ts.do(t, ts.request(t, http.MethodPatch, path, body), wantStatus)
}

func (ts *TestServer) deletePlayerData(t *testing.T, playerID string) {
	t.Helper()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if _, err := ts.db.Collection("players").DeleteOne(ctx, bson.M{"_id": playerID}); err != nil {
		t.Errorf("cleanup player %s: %v", playerID, err)
	}
	if _, err := ts.db.Collection("events").DeleteMany(ctx, bson.M{"player_id": playerID}); err != nil {
		t.Errorf("cleanup events %s: %v", playerID, err)
	}
}
