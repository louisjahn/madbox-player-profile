package main

import (
	"bytes"
	"context"
	"encoding/json"
	"log"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"github.com/testcontainers/testcontainers-go/modules/mongodb"
	"go.mongodb.org/mongo-driver/v2/mongo"

	mongocli "madbox-player-profile/internal/mongo"
	"madbox-player-profile/internal/player"
)

// Shared state for integration tests. TestMain sets these up once; each
// test function reads them. Namespace your collections per test (or
// clean up in t.Cleanup) so tests don't see each other's writes.
var (
	testMongo  *mongo.Client
	testServer *httptest.Server
)

// TestMain boots a real MongoDB container via testcontainers-go, connects
// through the Mongo helper, wires the client into newMux, and exposes the
// client and an httptest server as package-level globals for your tests
// to use.
//
// One container is shared across all tests — startup is ~5-15s, so paying
// that per-test would hurt.
func TestMain(m *testing.M) {
	ctx, cancel := context.WithTimeout(context.Background(), 90*time.Second)
	defer cancel()

	mongoC, err := mongodb.Run(ctx, "mongo:7")
	if err != nil {
		log.Fatalf("start mongo container: %v", err)
	}

	uri, err := mongoC.ConnectionString(ctx)
	if err != nil {
		_ = mongoC.Terminate(context.Background())
		log.Fatalf("get connection string: %v", err)
	}

	testMongo, err = mongocli.Connect(ctx, uri)
	if err != nil {
		_ = mongoC.Terminate(context.Background())
		log.Fatalf("connect mongo: %v", err)
	}

	testServer = httptest.NewServer(newMux(testMongo))

	// os.Exit skips deferred cleanup, so tear down explicitly here.
	code := m.Run()
	testServer.Close()
	_ = testMongo.Disconnect(context.Background())
	_ = mongoC.Terminate(context.Background())
	os.Exit(code)
}

// TestHealthz is a minimal example test showing how to use testServer.
// Replace or delete it as you add tests for your own handlers.
func TestHealthz(t *testing.T) {
	resp, err := http.Get(testServer.URL + "/healthz")
	if err != nil {
		t.Fatalf("get healthz: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status: %d", resp.StatusCode)
	}
}

func TestCreateInvalid(t *testing.T) {
	req, err := http.NewRequest(http.MethodPut, testServer.URL+"/v1/players/!!!", nil)
	if err != nil {
		t.Fatalf("failed creating first put req")
	}

	client := &http.Client{Timeout: time.Second * 5}
	resp, err := client.Do(req)
	if err != nil {
		t.Fatalf("failed sending first put req")
	}

	resp.Body.Close()
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected 400 (bad request), got: %v", resp.StatusCode)
	}
}

// Test creating a new player profile
func TestCreate(t *testing.T) {
	testID := t.Name()
	req, err := http.NewRequest(http.MethodPut, testServer.URL+"/v1/players/"+testID, nil)
	if err != nil {
		t.Fatalf("failed creating first put req")
	}

	client := &http.Client{Timeout: time.Second * 5}
	resp, err := client.Do(req)
	if err != nil {
		t.Fatalf("failed sending first put req")
	}

	resp.Body.Close()
	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("profile creation received code: %v, expected 201 StatusCreated", resp.StatusCode)
	}

	req, err = http.NewRequest(http.MethodGet, testServer.URL+"/v1/players/"+testID, nil)
	if err != nil {
		t.Fatalf("failed creating get req")
	}
	resp, err = client.Do(req)
	if err != nil {
		t.Fatalf("failed sending get req")
	}
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected get response to be OK, got: %v", resp.StatusCode)
		resp.Body.Close()
	}
	var playerBefore player.Player
	err = json.NewDecoder(resp.Body).Decode(&playerBefore)
	if err != nil {
		t.Fatalf("error while reading get response: %v", err)
	}
	resp.Body.Close()

	req, err = http.NewRequest(http.MethodPut, testServer.URL+"/v1/players/"+testID, nil)
	if err != nil {
		t.Fatalf("failed creating second req")
	}

	resp, err = client.Do(req)
	if err != nil {
		t.Fatalf("failed sending second put req")
	}
	resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("profile creation received code: %v, expected 200 StatusOk", resp.StatusCode)
	}

	req, err = http.NewRequest(http.MethodGet, testServer.URL+"/v1/players/"+testID, nil)
	if err != nil {
		t.Fatalf("failed creating get req")
	}
	resp, err = client.Do(req)
	if err != nil {
		t.Fatalf("failed sending get req")
	}
	var playerAfter player.Player
	err = json.NewDecoder(resp.Body).Decode(&playerAfter)
	resp.Body.Close()
	if err != nil {
		t.Fatalf("error while reading get response: %v", err)
	}

	if !playerBefore.CreatedAt.Equal(playerAfter.CreatedAt) {
		t.Fatalf("idempotency fail, before: %v, after: %v", playerBefore.CreatedAt, playerAfter.CreatedAt)
	}
	t.Cleanup(func() {
		testMongo.Database("profiles").Collection("player").Drop(context.Background())
	})
}

func TestUpdate(t *testing.T) {
	testID := t.Name()
	req, err := http.NewRequest(http.MethodPut, testServer.URL+"/v1/players/"+testID, nil)
	if err != nil {
		t.Fatalf("failed creating first put req")
	}

	client := &http.Client{Timeout: time.Second * 5}
	resp, err := client.Do(req)
	if err != nil {
		t.Fatalf("failed sending first put req")
	}
	resp.Body.Close()

	name := "patch"
	patch := player.Patch{
		DisplayName: &name,
	}
	bodyBytes, err := json.Marshal(patch)
	if err != nil {
		t.Fatalf("failed serializing patch: %v", err)
	}
	req, err = http.NewRequest(http.MethodPatch, testServer.URL+"/v1/players/"+testID, bytes.NewBuffer(bodyBytes))
	if err != nil {
		t.Fatalf("failed creating patch req")
	}
	resp, err = client.Do(req)
	if err != nil {
		t.Fatalf("failed sending patch req")
	}
	var playerAfter player.Player
	err = json.NewDecoder(resp.Body).Decode(&playerAfter)
	resp.Body.Close()
	if err != nil {
		t.Fatalf("error while reading patch response: %v", err)
	}

	if playerAfter.DisplayName != name {
		t.Fatalf("update failed, want: %s, have: %s", name, playerAfter.DisplayName)
	}

	t.Cleanup(func() {
		testMongo.Database("profiles").Collection("player").Drop(context.Background())
	})
}
