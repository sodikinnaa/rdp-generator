package model

import "time"

// ProvisionRequest defines user parameters for creating a Windows lab codespace.
type ProvisionRequest struct {
	GitHubToken     string `json:"github_token"`
	RepoOwner       string `json:"repo_owner"`
	RepoName        string `json:"repo_name"`
	TailscaleKey    string `json:"tailscale_key"`
	WindowsVersion  string `json:"windows_version"` // "2022", "2025", "2019", "11"
	MachineType     string `json:"machine_type"`    // "standardLinux32gb" (4-core), etc.
	IdleTimeoutMins int    `json:"idle_timeout_mins"`
	DisplayName     string `json:"display_name"`
	MasterSecret    string `json:"master_secret"`
}

// UserProfile describes the authenticated GitHub user.
type UserProfile struct {
	Login     string `json:"login"`
	ID        int64  `json:"id"`
	AvatarURL string `json:"avatar_url"`
	HTMLURL   string `json:"html_url"`
}

// PublicKeyResponse represents the encryption public key for Codespaces secrets.
type PublicKeyResponse struct {
	KeyID string `json:"key_id"`
	Key   string `json:"key"` // Base64 encoded
}

// CodespaceMachine describes VM hardware specs.
type CodespaceMachine struct {
	Name         string `json:"name"`
	DisplayName  string `json:"display_name"`
	CPUs         int    `json:"cpus"`
	MemoryBytes  int64  `json:"memory_in_bytes"`
	StorageBytes int64  `json:"storage_in_bytes"`
	OperatingSys string `json:"operating_system"`
}

// Codespace represents a GitHub Codespace instance.
type Codespace struct {
	ID                 int64            `json:"id"`
	Name               string           `json:"name"`
	DisplayName        string           `json:"display_name"`
	State              string           `json:"state"` // "Available", "Provisioning", "Shutdown", etc.
	WebURL             string           `json:"web_url"`
	Location           string           `json:"location"`
	IdleTimeoutMinutes int              `json:"idle_timeout_minutes"`
	CreatedAt          time.Time        `json:"created_at"`
	UpdatedAt          time.Time        `json:"updated_at"`
	LastUsedAt         time.Time        `json:"last_used_at"`
	Machine            CodespaceMachine `json:"machine"`
	Repository         CodespaceRepo    `json:"repository"`
}

// CodespaceRepo holds basic repo metadata attached to a codespace.
type CodespaceRepo struct {
	ID       int64  `json:"id"`
	Name     string `json:"name"`
	FullName string `json:"full_name"`
	Private  bool   `json:"private"`
	HTMLURL  string `json:"html_url"`
}

// CodespaceListResponse wraps the API response for listing codespaces.
type CodespaceListResponse struct {
	TotalCount int         `json:"total_count"`
	Codespaces []Codespace `json:"codespaces"`
}

// RepositoryItem represents a repository in user's GitHub account.
type RepositoryItem struct {
	ID              int64  `json:"id"`
	Name            string `json:"name"`
	FullName        string `json:"full_name"`
	Private         bool   `json:"private"`
	DefaultBranch   string `json:"default_branch"`
	HasDevContainer bool   `json:"has_devcontainer"`
}

// ValidateRepoResponse returns verification results of a repository for Windows devcontainer.
type ValidateRepoResponse struct {
	Valid           bool   `json:"valid"`
	RepoID          int64  `json:"repo_id"`
	FullName        string `json:"full_name"`
	DefaultBranch   string `json:"default_branch"`
	HasDevContainer bool   `json:"has_devcontainer"`
	Message         string `json:"message"`
}

// VerifyTokenResponse provides verification information for Step 1.
type VerifyTokenResponse struct {
	Valid    bool        `json:"valid"`
	User     UserProfile `json:"user"`
	Scopes   []string    `json:"scopes"`
	HasScope bool        `json:"has_codespace_scope"`
}

// InstanceViewModel represents data prepared for UI display.
type InstanceViewModel struct {
	Name             string `json:"name"`
	DisplayName      string `json:"display_name"`
	State            string `json:"state"`
	Machine          string `json:"machine"`
	CPUs             int    `json:"cpus"`
	MemoryGB         int64  `json:"memory_gb"`
	StorageGB        int64  `json:"storage_gb"`
	RepoFullName     string `json:"repo_full_name"`
	CreatedAt        string `json:"created_at"`
	WebURL           string `json:"web_url"`
	TailscaleIP      string `json:"tailscale_ip,omitempty"`
	RDPAddress       string `json:"rdp_address,omitempty"`
	WebConsoleURL    string `json:"web_console_url,omitempty"`
	Username         string `json:"username,omitempty"`
	Password         string `json:"password,omitempty"`
	CredentialScheme string `json:"credential_scheme,omitempty"`
}
