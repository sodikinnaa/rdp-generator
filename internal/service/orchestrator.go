package service

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"strings"
	"time"

	"github.com/sodikinnaa/rdp-generator/internal/github"
	"github.com/sodikinnaa/rdp-generator/internal/model"
)

const CredentialSchemeHMAC = "hmac-sha256-v1"

var supportedWindowsVersions = map[string]bool{
	"11": true, "2019": true, "2022": true, "2025": true,
}

func SupportedWindowsVersion(version string) bool {
	return supportedWindowsVersions[version]
}

type Orchestrator struct{}

func NewOrchestrator() *Orchestrator {
	return &Orchestrator{}
}

func DerivePassword(masterSecret, codespaceName string) string {
	mac := hmac.New(sha256.New, []byte(masterSecret))
	mac.Write([]byte(codespaceName))
	return "Aa1!" + hex.EncodeToString(mac.Sum(nil))[:24]
}

func TailscaleHostname(codespaceName string) string {
	hostname := "winlab-" + codespaceName
	if len(hostname) > 63 {
		hostname = hostname[:63]
	}
	return strings.TrimRight(hostname, "-")
}

func instanceVersion(displayName string) (string, bool) {
	for version := range supportedWindowsVersions {
		if strings.HasPrefix(displayName, "rdp-v1-"+version+"-") {
			return version, true
		}
	}
	return "", false
}

func InstanceView(cs model.Codespace) model.InstanceViewModel {
	vm := model.InstanceViewModel{
		Name:          cs.Name,
		DisplayName:   cs.DisplayName,
		State:         cs.State,
		Machine:       cs.Machine.DisplayName,
		CPUs:          cs.Machine.CPUs,
		MemoryGB:      cs.Machine.MemoryBytes / (1024 * 1024 * 1024),
		StorageGB:     cs.Machine.StorageBytes / (1024 * 1024 * 1024),
		RepoFullName:  cs.Repository.FullName,
		CreatedAt:     cs.CreatedAt.Format(time.RFC3339),
		WebURL:        cs.WebURL,
		WebConsoleURL: fmt.Sprintf("https://%s-8006.app.github.dev", cs.Name),
		Username:      "WinAdmin",
	}
	if _, ok := instanceVersion(cs.DisplayName); ok {
		hostname := TailscaleHostname(cs.Name)
		vm.RDPAddress = hostname + ":3389"
		vm.WebConsoleURL = "http://" + hostname + ":8006"
		vm.Username = "WinAdmin"
		vm.CredentialScheme = CredentialSchemeHMAC
	}
	return vm
}

func (o *Orchestrator) Provision(req *model.ProvisionRequest) (*model.InstanceViewModel, error) {
	if req.GitHubToken == "" {
		return nil, fmt.Errorf("github_token is required")
	}
	if req.RepoOwner == "" || req.RepoName == "" {
		return nil, fmt.Errorf("repo_owner and repo_name are required")
	}
	if len([]byte(req.MasterSecret)) < 32 {
		return nil, fmt.Errorf("master_secret must be at least 32 bytes")
	}
	if req.TailscaleKey == "" {
		return nil, fmt.Errorf("tailscale_key is required")
	}
	if req.WindowsVersion == "" {
		req.WindowsVersion = "2022"
	}
	if !supportedWindowsVersions[req.WindowsVersion] {
		return nil, fmt.Errorf("unsupported windows_version %q", req.WindowsVersion)
	}

	ghClient := github.NewClient(req.GitHubToken)
	if _, err := ghClient.GetUser(); err != nil {
		return nil, fmt.Errorf("validate github token: %w", err)
	}
	repoDetails, err := ghClient.GetRepoDetails(req.RepoOwner, req.RepoName)
	if err != nil {
		return nil, fmt.Errorf("fetch target repo details (%s/%s): %w", req.RepoOwner, req.RepoName, err)
	}
	repoIDs := []int64{repoDetails.ID}
	if err := ghClient.SetUserCodespaceSecret("RDP_MASTER_SECRET", req.MasterSecret, repoIDs); err != nil {
		return nil, fmt.Errorf("configure master secret: %w", err)
	}
	if err := ghClient.SetUserCodespaceSecret("TS_AUTHKEY", req.TailscaleKey, repoIDs); err != nil {
		return nil, fmt.Errorf("configure tailscale secret: %w", err)
	}
	if err := ghClient.SetUserCodespaceSecret("WIN_VERSION", req.WindowsVersion, repoIDs); err != nil {
		return nil, fmt.Errorf("configure windows version secret: %w", err)
	}

	name := req.DisplayName
	if name == "" {
		name = "win-" + time.Now().Format("0102-1504")
	}
	displayName := "rdp-v1-" + req.WindowsVersion + "-" + name
	targetBranch := repoDetails.DefaultBranch
	if targetBranch == "" {
		targetBranch = "main"
	}
	cs, err := ghClient.CreateCodespace(req.RepoOwner, req.RepoName, targetBranch, req.MachineType, displayName, req.IdleTimeoutMins)
	if err != nil {
		return nil, fmt.Errorf("create codespace: %w", err)
	}
	vm := InstanceView(*cs)
	return &vm, nil
}
