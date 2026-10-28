package service

import (
	"fmt"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/sodikinnaa/rdp-generator/internal/model"
)

func TestDerivePasswordVector(t *testing.T) {
	secret := "0123456789abcdef0123456789abcdef"
	codespace := "curly-space-funicular-4jw56qxgjvrcvwpx"

	got := DerivePassword(secret, codespace)

	expected := "Aa1!89773ecdf58c68239ad0e8e3"

	if got != expected {
		t.Fatalf("expected %q, got %q", expected, got)
	}
	if !strings.HasPrefix(got, "Aa1!") {
		t.Fatalf("expected Aa1! prefix, got %q", got)
	}
	if len(got) != 28 {
		t.Fatalf("expected length 28, got %d (%q)", len(got), got)
	}
}

func TestInstanceViewHMACScheme(t *testing.T) {
	cs := model.Codespace{
		Name:        "curly-space-funicular-4jw56qxgjvrcvwpx",
		DisplayName: "rdp-v1-2022-my-workstation",
		State:       "Available",
		CreatedAt:   time.Now(),
		WebURL:      "https://github.com/codespaces/curly-space-funicular-4jw56qxgjvrcvwpx",
		Repository: model.CodespaceRepo{
			FullName: "octocat/hello-world",
		},
		Machine: model.CodespaceMachine{
			DisplayName:  "4-core · 16GB RAM · 64GB Storage",
			CPUs:         4,
			MemoryBytes:  16 * 1024 * 1024 * 1024,
			StorageBytes: 64 * 1024 * 1024 * 1024,
		},
	}

	vm := InstanceView(cs)

	if vm.CredentialScheme != CredentialSchemeHMAC {
		t.Fatalf("expected %s, got %s", CredentialSchemeHMAC, vm.CredentialScheme)
	}
	expectedHost := "winlab-curly-space-funicular-4jw56qxgjvrcvwpx"
	if vm.RDPAddress != expectedHost+":3389" {
		t.Fatalf("expected rdp address %s:3389, got %s", expectedHost, vm.RDPAddress)
	}
	if vm.WebConsoleURL != "http://"+expectedHost+":8006" {
		t.Fatalf("expected web console http://%s:8006, got %s", expectedHost, vm.WebConsoleURL)
	}
	if vm.Username != "WinAdmin" {
		t.Fatalf("expected WinAdmin username, got %s", vm.Username)
	}
	if vm.Password != "" {
		t.Fatalf("expected empty password, got %s", vm.Password)
	}
}

func TestInstanceViewLegacy(t *testing.T) {
	cs := model.Codespace{
		Name:        "legacy-codespace-12345",
		DisplayName: "win-2022-1024-0800",
		State:       "Available",
		CreatedAt:   time.Now(),
		WebURL:      "https://github.com/codespaces/legacy-codespace-12345",
		Repository: model.CodespaceRepo{
			FullName: "octocat/hello-world",
		},
	}

	vm := InstanceView(cs)

	if vm.CredentialScheme != "" {
		t.Fatalf("expected empty credential scheme for legacy, got %s", vm.CredentialScheme)
	}
	if vm.RDPAddress != "" {
		t.Fatalf("expected empty rdp address for legacy, got %s", vm.RDPAddress)
	}
	if vm.Password != "" {
		t.Fatalf("expected empty password for legacy, got %s", vm.Password)
	}
	expectedConsole := "https://legacy-codespace-12345-8006.app.github.dev"
	if vm.WebConsoleURL != expectedConsole {
		t.Fatalf("expected native console %s, got %s", expectedConsole, vm.WebConsoleURL)
	}
}

type rejectingTransport struct {
	t *testing.T
}

func (rt rejectingTransport) RoundTrip(*http.Request) (*http.Response, error) {
	rt.t.Error("unexpected network request for invalid input")
	return nil, fmt.Errorf("network disabled in validation test")
}

func TestProvisionInputValidation(t *testing.T) {
	original := http.DefaultTransport
	http.DefaultTransport = rejectingTransport{t}
	t.Cleanup(func() { http.DefaultTransport = original })
	orch := NewOrchestrator()

	tests := []struct {
		name    string
		req     model.ProvisionRequest
		wantErr string
	}{
		{
			name: "missing github token",
			req: model.ProvisionRequest{
				RepoOwner: "owner", RepoName: "repo", MasterSecret: strings.Repeat("x", 32), TailscaleKey: "tskey",
			},
			wantErr: "github_token is required",
		},
		{
			name: "missing repo owner",
			req: model.ProvisionRequest{
				GitHubToken: "token", RepoName: "repo", MasterSecret: strings.Repeat("x", 32), TailscaleKey: "tskey",
			},
			wantErr: "repo_owner and repo_name are required",
		},
		{
			name: "master secret too short",
			req: model.ProvisionRequest{
				GitHubToken: "token", RepoOwner: "owner", RepoName: "repo", MasterSecret: "too-short", TailscaleKey: "tskey",
			},
			wantErr: "master_secret must be at least 32 bytes",
		},
		{
			name: "missing tailscale key",
			req: model.ProvisionRequest{
				GitHubToken: "token", RepoOwner: "owner", RepoName: "repo", MasterSecret: strings.Repeat("x", 32),
			},
			wantErr: "tailscale_key is required",
		},
		{
			name: "unsupported windows version",
			req: model.ProvisionRequest{
				GitHubToken: "token", RepoOwner: "owner", RepoName: "repo", MasterSecret: strings.Repeat("x", 32), TailscaleKey: "tskey", WindowsVersion: "98",
			},
			wantErr: "unsupported windows_version \"98\"",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := orch.Provision(&tt.req)
			if err == nil || !strings.Contains(err.Error(), tt.wantErr) {
				t.Fatalf("expected error containing %q, got %v", tt.wantErr, err)
			}
		})
	}
}

func TestInstanceVersionMarkers(t *testing.T) {
	for _, version := range []string{"11", "2019", "2022", "2025"} {
		if got, ok := instanceVersion("rdp-v1-" + version + "-custom"); !ok || got != version {
			t.Fatalf("supported version not recognized: %s", version)
		}
	}
	for _, name := range []string{"rdp-v1-98-custom", "rdp-v1-20220-custom", "rdp-v1-2022", "prefix-rdp-v1-2022-custom", "rdp-v2-2022-custom"} {
		if vm := InstanceView(model.Codespace{Name: "legacy", DisplayName: name}); vm.CredentialScheme != "" || vm.RDPAddress != "" || vm.Password != "" {
			t.Fatalf("legacy instance mislabeled: %s", name)
		}
	}
}

func TestTailscaleHostnameTruncation(t *testing.T) {
	short := TailscaleHostname("my-cs")
	if short != "winlab-my-cs" {
		t.Fatalf("unexpected %s", short)
	}

	longCs := strings.Repeat("a", 100)
	long := TailscaleHostname(longCs)
	if len(long) > 63 {
		t.Fatalf("expected length <= 63, got %d (%s)", len(long), long)
	}
	if strings.HasSuffix(long, "-") {
		t.Fatalf("trailing dash not trimmed: %s", long)
	}
}
