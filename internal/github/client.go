package github

import (
	"bytes"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/sodikinnaa/rdp-generator/internal/model"
	"golang.org/x/crypto/nacl/box"
)

const (
	defaultBaseURL = "https://api.github.com"
	apiVersion     = "2022-11-28"
)

// Client handles GitHub REST API operations for Codespaces.
type Client struct {
	token      string
	baseURL    string
	httpClient *http.Client
}

// NewClient instantiates a GitHub API client using a Personal Access Token.
func NewClient(token string) *Client {
	return &Client{
		token:   token,
		baseURL: defaultBaseURL,
		httpClient: &http.Client{
			Timeout: 45 * time.Second,
		},
	}
}

// request sends an authenticated HTTP request to GitHub API.
func (c *Client) request(method, path string, body any, out any) error {
	var bodyReader io.Reader
	if body != nil {
		data, err := json.Marshal(body)
		if err != nil {
			return fmt.Errorf("marshal request body: %w", err)
		}
		bodyReader = bytes.NewReader(data)
	}

	req, err := http.NewRequest(method, c.baseURL+path, bodyReader)
	if err != nil {
		return fmt.Errorf("create request: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+c.token)
	req.Header.Set("Accept", "application/vnd.github+json")
	req.Header.Set("X-GitHub-Api-Version", apiVersion)
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	res, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("perform request: %w", err)
	}
	defer res.Body.Close()

	resBody, err := io.ReadAll(res.Body)
	if err != nil {
		return fmt.Errorf("read response body: %w", err)
	}

	if res.StatusCode < 200 || res.StatusCode >= 300 {
		return c.parseAPIError(res.StatusCode, resBody)
	}

	if out != nil && len(resBody) > 0 {
		if err := json.Unmarshal(resBody, out); err != nil {
			return fmt.Errorf("unmarshal response (%s): %w", string(resBody), err)
		}
	}
	return nil
}

func (c *Client) parseAPIError(code int, body []byte) error {
	var ghErr struct {
		Message          string `json:"message"`
		DocumentationURL string `json:"documentation_url"`
	}
	_ = json.Unmarshal(body, &ghErr)
	msg := strings.ToLower(ghErr.Message)

	if strings.Contains(msg, "spending limit") || strings.Contains(msg, "billing") {
		return fmt.Errorf("KUOTA_HABIS: Kuota gratis Codespace akun Anda habis atau spending limit $0. Naikkan limit di Settings -> Billing GitHub, atau alihkan deployment ke VPS.")
	}
	if strings.Contains(msg, "maximum number of codespaces") || strings.Contains(msg, "limit of") {
		return fmt.Errorf("LIMIT_CODESPACE: Batas maksimal Codespace aktif telah tercapai. Hentikan (Stop) atau hapus (Delete) instance yang tidak terpakai.")
	}
	if strings.Contains(msg, "machine") && (strings.Contains(msg, "not allowed") || strings.Contains(msg, "unavailable") || strings.Contains(msg, "policy")) {
		return fmt.Errorf("LIMIT_TIER: Machine tier 4 vCPU / 16GB tidak diizinkan pada repo/organisasi ini. Gunakan basicLinux32gb (2 vCPU) atau deploy di VPS.")
	}
	if strings.Contains(msg, "policy") || strings.Contains(msg, "organization") || strings.Contains(msg, "disabled") {
		return fmt.Errorf("ORGANIZATION_BLOCKED: Fitur Codespaces dibatasi oleh policy organisasi pemilik repo. Gunakan repository personal Anda.")
	}
	if ghErr.Message != "" {
		return fmt.Errorf("GitHub API (%d): %s", code, ghErr.Message)
	}
	return fmt.Errorf("GitHub API error (%d): %s", code, string(body))
}

// GetUser retrieves the authenticated user's profile.
func (c *Client) GetUser() (*model.UserProfile, error) {
	var user model.UserProfile
	if err := c.request(http.MethodGet, "/user", nil, &user); err != nil {
		return nil, err
	}
	return &user, nil
}

// GetCodespacesPublicKey fetches the public key used to encrypt Codespaces secrets.
func (c *Client) GetCodespacesPublicKey() (*model.PublicKeyResponse, error) {
	var pk model.PublicKeyResponse
	if err := c.request(http.MethodGet, "/user/codespaces/secrets/public-key", nil, &pk); err != nil {
		return nil, err
	}
	return &pk, nil
}

// EncryptSecret encrypts a plaintext secret using LibSodium sealed box (X25519-XSalsa20-Poly1305).
func EncryptSecret(plainSecret, base64PubKey string) (string, error) {
	pubKeyBytes, err := base64.StdEncoding.DecodeString(base64PubKey)
	if err != nil {
		return "", fmt.Errorf("decode public key: %w", err)
	}
	if len(pubKeyBytes) != 32 {
		return "", fmt.Errorf("invalid public key length: expected 32, got %d", len(pubKeyBytes))
	}

	var recipientKey [32]byte
	copy(recipientKey[:], pubKeyBytes)

	encrypted, err := box.SealAnonymous(nil, []byte(plainSecret), &recipientKey, rand.Reader)
	if err != nil {
		return "", fmt.Errorf("seal anonymous: %w", err)
	}

	return base64.StdEncoding.EncodeToString(encrypted), nil
}

// GetUserWithScopes retrieves the authenticated user's profile and token scopes.
func (c *Client) GetUserWithScopes() (*model.UserProfile, []string, error) {
	req, err := http.NewRequest(http.MethodGet, c.baseURL+"/user", nil)
	if err != nil {
		return nil, nil, fmt.Errorf("create request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+c.token)
	req.Header.Set("Accept", "application/vnd.github+json")
	req.Header.Set("X-GitHub-Api-Version", apiVersion)

	res, err := c.httpClient.Do(req)
	if err != nil {
		return nil, nil, fmt.Errorf("perform request: %w", err)
	}
	defer res.Body.Close()

	if res.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(res.Body)
		return nil, nil, fmt.Errorf("github api error (%d): %s", res.StatusCode, string(body))
	}

	var user model.UserProfile
	if err := json.NewDecoder(res.Body).Decode(&user); err != nil {
		return nil, nil, fmt.Errorf("decode user response: %w", err)
	}

	rawScopes := res.Header.Get("X-OAuth-Scopes")
	var scopes []string
	for _, s := range bytes.Split([]byte(rawScopes), []byte(",")) {
		trimmed := string(bytes.TrimSpace(s))
		if trimmed != "" {
			scopes = append(scopes, trimmed)
		}
	}

	return &user, scopes, nil
}

// ListUserRepos retrieves list of repositories accessible to user.
func (c *Client) ListUserRepos() ([]model.RepositoryItem, error) {
	var repos []model.RepositoryItem
	path := "/user/repos?per_page=100&sort=updated&affiliation=owner,collaborator"
	if err := c.request(http.MethodGet, path, nil, &repos); err != nil {
		return nil, err
	}
	return repos, nil
}

// CheckRepoDevContainer checks if .devcontainer/devcontainer.json exists.
func (c *Client) CheckRepoDevContainer(owner, repo string) (bool, error) {
	path := fmt.Sprintf("/repos/%s/%s/contents/.devcontainer/devcontainer.json", owner, repo)
	req, err := http.NewRequest(http.MethodGet, c.baseURL+path, nil)
	if err != nil {
		return false, err
	}
	req.Header.Set("Authorization", "Bearer "+c.token)
	req.Header.Set("Accept", "application/vnd.github+json")
	req.Header.Set("X-GitHub-Api-Version", apiVersion)

	res, err := c.httpClient.Do(req)
	if err != nil {
		return false, err
	}
	defer res.Body.Close()

	if res.StatusCode == http.StatusOK {
		return true, nil
	}
	return false, nil
}

// GetRepoDetails gets repo id and default branch.
func (c *Client) GetRepoDetails(owner, repo string) (*model.RepositoryItem, error) {
	var item model.RepositoryItem
	path := fmt.Sprintf("/repos/%s/%s", owner, repo)
	if err := c.request(http.MethodGet, path, nil, &item); err != nil {
		return nil, err
	}
	hasDc, _ := c.CheckRepoDevContainer(owner, repo)
	item.HasDevContainer = hasDc
	return &item, nil
}

// SetUserCodespaceSecret sets a user-level Codespace secret with repository scope.
func (c *Client) SetUserCodespaceSecret(secretName, plainValue string, selectedRepoIDs []int64) error {
	pk, err := c.GetCodespacesPublicKey()
	if err != nil {
		return fmt.Errorf("fetch public key: %w", err)
	}

	encryptedVal, err := EncryptSecret(plainValue, pk.Key)
	if err != nil {
		return fmt.Errorf("encrypt secret: %w", err)
	}

	payload := map[string]any{
		"encrypted_value": encryptedVal,
		"key_id":          pk.KeyID,
	}

	if len(selectedRepoIDs) > 0 {
		var ids []string
		for _, id := range selectedRepoIDs {
			ids = append(ids, fmt.Sprintf("%d", id))
		}
		payload["selected_repository_ids"] = ids
	}

	path := fmt.Sprintf("/user/codespaces/secrets/%s", secretName)
	return c.request(http.MethodPut, path, payload, nil)
}

// ListCodespaces lists the authenticated user's Codespaces.
func (c *Client) ListCodespaces() (*model.CodespaceListResponse, error) {
	var list model.CodespaceListResponse
	if err := c.request(http.MethodGet, "/user/codespaces", nil, &list); err != nil {
		return nil, err
	}
	return &list, nil
}

// GetCodespace retrieves details of a specific codespace by name.
func (c *Client) GetCodespace(name string) (*model.Codespace, error) {
	var cs model.Codespace
	path := fmt.Sprintf("/user/codespaces/%s", name)
	if err := c.request(http.MethodGet, path, nil, &cs); err != nil {
		return nil, err
	}
	return &cs, nil
}

// CreateCodespace requests creation of a new codespace in the specified repository.
func (c *Client) CreateCodespace(owner, repo, branch, machineType, displayName string, idleTimeoutMins int) (*model.Codespace, error) {
	if branch == "" {
		branch = "main"
	}
	if machineType == "" {
		machineType = "standardLinux32gb"
	}
	if idleTimeoutMins <= 0 {
		idleTimeoutMins = 240
	}

	payload := map[string]any{
		"ref":                  branch,
		"machine":              machineType,
		"display_name":         displayName,
		"idle_timeout_minutes": idleTimeoutMins,
		"geo":                  "SoutheastAsia",
	}

	var created model.Codespace
	path := fmt.Sprintf("/repos/%s/%s/codespaces", owner, repo)
	if err := c.request(http.MethodPost, path, payload, &created); err != nil {
		return nil, err
	}
	return &created, nil
}

// StartCodespace starts a stopped codespace.
func (c *Client) StartCodespace(name string) error {
	path := fmt.Sprintf("/user/codespaces/%s/start", name)
	return c.request(http.MethodPost, path, nil, nil)
}

// StopCodespace stops an active codespace.
func (c *Client) StopCodespace(name string) error {
	path := fmt.Sprintf("/user/codespaces/%s/stop", name)
	return c.request(http.MethodPost, path, nil, nil)
}

// PutFile creates or updates a file in a repository via GitHub Contents API.
func (c *Client) PutFile(owner, repo, path, message, contentBase64, sha string) error {
	payload := map[string]string{
		"message": message,
		"content": contentBase64,
	}
	if sha != "" {
		payload["sha"] = sha
	}
	apiPath := fmt.Sprintf("/repos/%s/%s/contents/%s", owner, repo, path)
	return c.request(http.MethodPut, apiPath, payload, nil)
}

// GetFileSHA fetches the blob SHA for an existing file if present.
func (c *Client) GetFileSHA(owner, repo, path string) (string, error) {
	var fileInfo struct {
		SHA string `json:"sha"`
	}
	apiPath := fmt.Sprintf("/repos/%s/%s/contents/%s", owner, repo, path)
	if err := c.request(http.MethodGet, apiPath, nil, &fileInfo); err != nil {
		return "", err
	}
	return fileInfo.SHA, nil
}

// DeleteCodespace permanently deletes a codespace.
func (c *Client) DeleteCodespace(name string) error {
	path := fmt.Sprintf("/user/codespaces/%s", name)
	return c.request(http.MethodDelete, path, nil, nil)
}
