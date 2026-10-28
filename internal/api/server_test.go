package api

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/sodikinnaa/rdp-generator/internal/service"
)

func TestHealthEndpoint(t *testing.T) {
	orch := service.NewOrchestrator()
	srv, err := NewServer(orch)
	if err != nil {
		t.Fatalf("NewServer failed: %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	rec := httptest.NewRecorder()

	srv.Routes().ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", rec.Code)
	}

	var res map[string]string
	if err := json.Unmarshal(rec.Body.Bytes(), &res); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if res["status"] != "ok" {
		t.Fatalf("expected status ok, got %s", res["status"])
	}
}

func TestLandingAndDashboardPages(t *testing.T) {
	orch := service.NewOrchestrator()
	srv, err := NewServer(orch)
	if err != nil {
		t.Fatalf("NewServer failed: %v", err)
	}

	for _, tc := range []struct {
		path   string
		marker string
		status int
	}{
		{path: "/", marker: "Komputer Windows.", status: http.StatusOK},
		{path: "/dashboard", marker: "rdpSimpleApp", status: http.StatusOK},
		{path: "/unknown-page", marker: "", status: http.StatusNotFound},
	} {
		req := httptest.NewRequest(http.MethodGet, tc.path, nil)
		rec := httptest.NewRecorder()
		srv.Routes().ServeHTTP(rec, req)

		if rec.Code != tc.status {
			t.Fatalf("expected status %d for %s, got %d", tc.status, tc.path, rec.Code)
		}
		if tc.marker != "" && !strings.Contains(rec.Body.String(), tc.marker) {
			t.Fatalf("expected %s to contain %q", tc.path, tc.marker)
		}
	}
}

func TestRemovedRoutes(t *testing.T) {
	srv, err := NewServer(service.NewOrchestrator())
	if err != nil {
		t.Fatal(err)
	}
	for _, route := range []struct{ method, path string }{
		{http.MethodGet, "/api/credentials"},
		{http.MethodPost, "/api/credentials"},
		{http.MethodPost, "/api/instances/example/heartbeat"},
		{http.MethodPost, "/api/vps/deploy-local"},
	} {
		rec := httptest.NewRecorder()
		srv.Routes().ServeHTTP(rec, httptest.NewRequest(route.method, route.path, nil))
		if rec.Code < 400 {
			t.Fatalf("removed route still available: %s %s", route.method, route.path)
		}
	}
}

func TestGenerateVPSScriptValidation(t *testing.T) {
	orch := service.NewOrchestrator()
	srv, err := NewServer(orch)
	if err != nil {
		t.Fatalf("NewServer failed: %v", err)
	}

	for _, url := range []string{
		"/api/vps/script?version=invalid",
		"/api/vps/script?name=bad%22%24%28whoami%29",
	} {
		req := httptest.NewRequest(http.MethodGet, url, nil)
		rec := httptest.NewRecorder()
		srv.Routes().ServeHTTP(rec, req)
		if rec.Code != http.StatusBadRequest {
			t.Fatalf("expected 400 for %s, got %d", url, rec.Code)
		}
	}
}
