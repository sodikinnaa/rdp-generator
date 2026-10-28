package api

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"html/template"
	"log"
	"net/http"
	"regexp"
	"strings"
	"time"

	"github.com/sodikinnaa/rdp-generator/internal/github"
	"github.com/sodikinnaa/rdp-generator/internal/model"
	"github.com/sodikinnaa/rdp-generator/internal/service"
	"github.com/sodikinnaa/rdp-generator/internal/web"
)

// Server handles HTTP web UI and JSON API requests.
type Server struct {
	orch *service.Orchestrator
	tmpl *template.Template
}

func NewServer(orch *service.Orchestrator) (*Server, error) {
	tmpl, err := template.ParseFS(web.TemplateFS, "templates/*.html")
	if err != nil {
		return nil, fmt.Errorf("parse embedded templates: %w", err)
	}

	return &Server{
		orch: orch,
		tmpl: tmpl,
	}, nil
}

func (s *Server) Routes() http.Handler {
	mux := http.NewServeMux()

	mux.HandleFunc("GET /{$}", s.handleLanding)
	mux.HandleFunc("GET /dashboard", s.handleDashboard)
	mux.HandleFunc("GET /download/exe", s.handleDownloadExe)
	mux.HandleFunc("GET /health", s.handleHealth)
	mux.HandleFunc("POST /api/auth/verify", s.handleVerifyToken)
	mux.HandleFunc("GET /api/repos", s.handleListRepos)
	mux.HandleFunc("GET /api/repos/overview", s.handleRepoOverview)
	mux.HandleFunc("POST /api/repos/setup-devcontainer", s.handleSetupDevContainer)
	mux.HandleFunc("GET /api/vps/script", s.handleGenerateVPSScript)
	mux.HandleFunc("POST /api/repos/validate", s.handleValidateRepo)
	mux.HandleFunc("POST /api/provision", s.handleProvision)
	mux.HandleFunc("GET /api/instances", s.handleListInstances)
	mux.HandleFunc("POST /api/instances/{name}/start", s.handleStartInstance)
	mux.HandleFunc("POST /api/instances/{name}/stop", s.handleStopInstance)
	mux.HandleFunc("DELETE /api/instances/{name}", s.handleDeleteInstance)

	return s.corsMiddleware(mux)
}

func (s *Server) corsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusOK)
			return
		}
		next.ServeHTTP(w, r)
	})
}

func (s *Server) handleLanding(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := s.tmpl.ExecuteTemplate(w, "landing.html", nil); err != nil {
		log.Printf("render landing: %v", err)
	}
}

func (s *Server) handleDashboard(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := s.tmpl.ExecuteTemplate(w, "index.html", nil); err != nil {
		log.Printf("render dashboard: %v", err)
	}
}

func (s *Server) handleDownloadExe(w http.ResponseWriter, r *http.Request) {
	exePath := "rdp-generator.exe"
	w.Header().Set("Content-Disposition", "attachment; filename=rdp-generator.exe")
	w.Header().Set("Content-Type", "application/octet-stream")
	http.ServeFile(w, r, exePath)
}

func (s *Server) handleHealth(w http.ResponseWriter, r *http.Request) {
	s.jsonResponse(w, http.StatusOK, map[string]string{"status": "ok"})
}

func (s *Server) handleVerifyToken(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Token string `json:"token"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil || body.Token == "" {
		s.jsonError(w, http.StatusBadRequest, "Missing token")
		return
	}

	ghClient := github.NewClient(body.Token)
	user, scopes, err := ghClient.GetUserWithScopes()
	if err != nil {
		s.jsonError(w, http.StatusUnauthorized, fmt.Sprintf("Authentication failed: %v", err))
		return
	}

	hasScope := false
	for _, sc := range scopes {
		if sc == "codespace" || sc == "admin:org" || sc == "repo" {
			hasScope = true
			break
		}
	}

	s.jsonResponse(w, http.StatusOK, model.VerifyTokenResponse{
		Valid:    true,
		User:     *user,
		Scopes:   scopes,
		HasScope: hasScope,
	})
}

func (s *Server) handleListRepos(w http.ResponseWriter, r *http.Request) {
	token := s.extractBearerToken(r)
	if token == "" {
		s.jsonError(w, http.StatusUnauthorized, "Missing Authorization Bearer token")
		return
	}

	ghClient := github.NewClient(token)
	repos, err := ghClient.ListUserRepos()
	if err != nil {
		s.jsonError(w, http.StatusInternalServerError, err.Error())
		return
	}

	s.jsonResponse(w, http.StatusOK, repos)
}

type RepoOverviewItem struct {
	ID                   int64                     `json:"id"`
	FullName             string                    `json:"full_name"`
	DefaultBranch        string                    `json:"default_branch"`
	HasDevContainer      bool                      `json:"has_devcontainer"`
	ActiveCodespaceCount int                       `json:"active_codespace_count"`
	Codespaces           []model.InstanceViewModel `json:"codespaces"`
}

func (s *Server) handleRepoOverview(w http.ResponseWriter, r *http.Request) {
	token := s.extractBearerToken(r)
	if token == "" {
		s.jsonError(w, http.StatusUnauthorized, "Missing Authorization Bearer token")
		return
	}

	ghClient := github.NewClient(token)
	repos, err := ghClient.ListUserRepos()
	if err != nil {
		s.jsonError(w, http.StatusInternalServerError, err.Error())
		return
	}

	csList, err := ghClient.ListCodespaces()
	if err != nil {
		s.jsonError(w, http.StatusInternalServerError, err.Error())
		return
	}

	codespaceMap := make(map[string][]model.InstanceViewModel)
	for _, cs := range csList.Codespaces {
		repoKey := cs.Repository.FullName
		codespaceMap[repoKey] = append(codespaceMap[repoKey], service.InstanceView(cs))
	}

	var overview []RepoOverviewItem
	for _, repo := range repos {
		item := RepoOverviewItem{
			ID:                   repo.ID,
			FullName:             repo.FullName,
			DefaultBranch:        repo.DefaultBranch,
			HasDevContainer:      repo.HasDevContainer,
			Codespaces:           codespaceMap[repo.FullName],
			ActiveCodespaceCount: len(codespaceMap[repo.FullName]),
		}
		overview = append(overview, item)
	}

	s.jsonResponse(w, http.StatusOK, overview)
}

func (s *Server) handleSetupDevContainer(w http.ResponseWriter, r *http.Request) {
	token := s.extractBearerToken(r)
	if token == "" {
		s.jsonError(w, http.StatusUnauthorized, "Missing Authorization Bearer token")
		return
	}

	var body struct {
		Owner string `json:"owner"`
		Repo  string `json:"repo"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil || body.Owner == "" || body.Repo == "" {
		s.jsonError(w, http.StatusBadRequest, "Missing owner and repo")
		return
	}

	devcontainerJSON := `{
  "name": "RDP Windows Server Lab",
  "image": "mcr.microsoft.com/devcontainers/universal:2",
  "runArgs": [
    "--device=/dev/kvm",
    "--device=/dev/net/tun",
    "--cap-add=NET_ADMIN"
  ],
  "forwardPorts": [8006, 3389],
  "portsAttributes": {
    "8006": {
      "label": "Windows Web Console",
      "onAutoForward": "silent",
      "visibility": "private"
    },
    "3389": {
      "label": "RDP Service",
      "onAutoForward": "silent",
      "visibility": "private"
    }
  },
  "overrideCommand": false,
  "postStartCommand": "/bin/bash .devcontainer/start-worker.sh"
}
`

	startWorkerSH := `#!/usr/bin/env bash
set -euo pipefail
umask 077
WORKSPACE_DIR="${WORKSPACE_DIR:-$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")/.." && pwd)}"
STATE_DIR="$WORKSPACE_DIR/.rdp-generator"
mkdir -p "$STATE_DIR"
chmod 700 "$STATE_DIR"
LOG_FILE="$STATE_DIR/worker.log"
exec >> "$LOG_FILE" 2>&1
echo "=== [$(date -u '+%Y-%m-%dT%H:%M:%SZ')] Starting RDP Worker ==="

WIN_VERSION="${WIN_VERSION:-2022}"
WIN_RAM="${WIN_RAM:-8G}"
WIN_CPUS="${WIN_CPUS:-2}"
WIN_DISK="${WIN_DISK:-32G}"
STORAGE_PATH="/tmp/windows"
CREDS_FILE="$STATE_DIR/credentials"
STATUS_FILE="$STATE_DIR/status.json"
TS_STATE_DIR="$STATE_DIR/tailscale"

mkdir -p "$TS_STATE_DIR"
chmod 700 "$TS_STATE_DIR"
chmod 600 "$CREDS_FILE" "$STATUS_FILE" "$LOG_FILE" 2>/dev/null || true
sudo find "$TS_STATE_DIR" -type f -exec chmod 600 {} +
sudo mkdir -p "$STORAGE_PATH" /var/run/tailscale

RDP_MASTER_SECRET="${RDP_MASTER_SECRET:-}"
TS_AUTHKEY="${TS_AUTHKEY:-}"
if [ -f "/workspaces/.codespaces/shared/user-secrets-envs.json" ]; then
    TS_AUTHKEY_SECRET=$(python3 -c "import json; print(json.load(open('/workspaces/.codespaces/shared/user-secrets-envs.json')).get('TS_AUTHKEY', ''))" 2>/dev/null || true)
    WIN_VERSION_SECRET=$(python3 -c "import json; print(json.load(open('/workspaces/.codespaces/shared/user-secrets-envs.json')).get('WIN_VERSION', ''))" 2>/dev/null || true)
    MASTER_SECRET=$(python3 -c "import json; print(json.load(open('/workspaces/.codespaces/shared/user-secrets-envs.json')).get('RDP_MASTER_SECRET', ''))" 2>/dev/null || true)
    WIN_PASSWORD_SECRET=$(python3 -c "import json; print(json.load(open('/workspaces/.codespaces/shared/user-secrets-envs.json')).get('WIN_PASSWORD', ''))" 2>/dev/null || true)
    if [ -n "$TS_AUTHKEY_SECRET" ] && [ -z "$TS_AUTHKEY" ]; then
        TS_AUTHKEY="$TS_AUTHKEY_SECRET"
    fi
    if [ -n "$WIN_VERSION_SECRET" ]; then
        WIN_VERSION="$WIN_VERSION_SECRET"
    fi
    if [ -n "$MASTER_SECRET" ] && [ -z "$RDP_MASTER_SECRET" ]; then
        RDP_MASTER_SECRET="$MASTER_SECRET"
    fi
    if [ -n "$WIN_PASSWORD_SECRET" ] && [ -z "${WIN_PASSWORD:-}" ]; then
        WIN_PASSWORD="$WIN_PASSWORD_SECRET"
    fi
fi

if [ ! -f "$CREDS_FILE" ] && [ -f "/tmp/rdp-creds" ]; then
    cp /tmp/rdp-creds "$CREDS_FILE"
    chmod 600 "$CREDS_FILE"
fi

if [ ! -f "$CREDS_FILE" ]; then
    WIN_USER="${WIN_USER:-WinAdmin}"
    WIN_PASS=""
    if [ -n "$RDP_MASTER_SECRET" ]; then
        if [ -z "${CODESPACE_NAME:-}" ]; then
            echo "Error: CODESPACE_NAME missing with RDP_MASTER_SECRET" >&2
            exit 1
        fi
        export RDP_MASTER_SECRET CODESPACE_NAME
        WIN_PASS=$(python3 -c "import hmac,hashlib,os; ms=os.environ['RDP_MASTER_SECRET'].encode('utf-8'); cs=os.environ['CODESPACE_NAME'].encode('utf-8'); assert len(ms) >= 32, 'RDP_MASTER_SECRET must be at least 32 bytes'; print('Aa1!' + hmac.new(ms, cs, hashlib.sha256).hexdigest()[:24])")
    elif [ -n "${WIN_PASSWORD:-}" ]; then
        WIN_PASS="$WIN_PASSWORD"
    else
        WIN_PASS=$(python3 -c 'import secrets,string; chars=string.ascii_letters+string.digits+"!@#%_-"; print("".join(secrets.choice(chars) for _ in range(16)))')
    fi
    printf "%s\n%s\n" "$WIN_USER" "$WIN_PASS" > "$CREDS_FILE"
    chmod 600 "$CREDS_FILE"
fi

WIN_USER=$(sed -n 1p "$CREDS_FILE")
WIN_PASS=$(sed -n 2p "$CREDS_FILE")

sudo rm -f /etc/apt/sources.list.d/yarn.list 2>/dev/null || true

if ! command -v tailscale >/dev/null 2>&1; then
    curl -fsSL https://pkgs.tailscale.com/stable/tailscale_1.76.6_amd64.tgz -o /tmp/ts.tgz
    tar -C /tmp -xzf /tmp/ts.tgz
    sudo cp /tmp/tailscale_*/tailscale* /usr/local/bin/
    rm -rf /tmp/ts.tgz /tmp/tailscale_*
fi

if ! pgrep -x tailscaled >/dev/null 2>&1; then
    sudo nohup /usr/local/bin/tailscaled --state="$TS_STATE_DIR/tailscaled.state" --socket=/var/run/tailscale/tailscaled.sock > "$STATE_DIR/tailscaled.log" 2>&1 &
    sleep 3
fi

TS_IP=""
TS_HOST=""
if [ -n "$TS_AUTHKEY" ]; then
    NODE_ID="${CODESPACE_NAME:-$(cat /etc/hostname | tr -dc 'a-zA-Z0-9')}"
    TS_HOST="winlab-${NODE_ID}"
    if [ ${#TS_HOST} -gt 63 ]; then
        TS_HOST="${TS_HOST:0:63}"
        while [[ "$TS_HOST" == *- ]]; do TS_HOST="${TS_HOST%-}"; done
    fi
    sudo /usr/local/bin/tailscale up --auth-key="$TS_AUTHKEY" --hostname="$TS_HOST" --accept-dns=true --reset || true
    TS_IP=$(sudo /usr/local/bin/tailscale ip -4 || true)
fi

if ! docker ps -a --format '{{.Names}}' | grep -q '^windows$'; then
    docker run -d --name windows \
        --restart unless-stopped \
        --device=/dev/kvm --device=/dev/net/tun --cap-add NET_ADMIN \
        -e VERSION="$WIN_VERSION" -e RAM_SIZE="$WIN_RAM" -e CPU_CORES="$WIN_CPUS" -e DISK_SIZE="$WIN_DISK" \
        -e USERNAME="$WIN_USER" -e PASSWORD="$WIN_PASS" \
        -p 127.0.0.1:8006:8006 -p 127.0.0.1:3389:3389/tcp \
        -v "$STORAGE_PATH:/storage" --stop-timeout 120 ghcr.io/dockur/windows
else
    docker start windows || true
fi

if [ -n "$TS_IP" ]; then
    sudo pkill -f "start_server.*$TS_IP" 2>/dev/null || true
    sudo nohup python3 -c "
import socket, threading

def relay(client, target_host, target_port):
    try:
        server = socket.socket(socket.AF_INET, socket.SOCK_STREAM)
        server.connect((target_host, target_port))
        def forward(src, dst):
            try:
                while True:
                    data = src.recv(4096)
                    if not data: break
                    dst.sendall(data)
            except: pass
            finally:
                try: src.close()
                except: pass
                try: dst.close()
                except: pass
        threading.Thread(target=forward, args=(client, server), daemon=True).start()
        threading.Thread(target=forward, args=(server, client), daemon=True).start()
    except:
        client.close()

def start_server(bind_ip, bind_port, target_ip, target_port):
    s = socket.socket(socket.AF_INET, socket.SOCK_STREAM)
    s.setsockopt(socket.SOL_SOCKET, socket.SO_REUSEADDR, 1)
    s.bind((bind_ip, bind_port))
    s.listen(50)
    while True:
        c, _ = s.accept()
        relay(c, target_ip, target_port)

threading.Thread(target=start_server, args=('$TS_IP', 3389, '127.0.0.1', 3389), daemon=True).start()
start_server('$TS_IP', 8006, '127.0.0.1', 8006)
" > "$STATE_DIR/pyrelay.log" 2>&1 &
fi

export STATE_DIR WIN_VERSION WIN_RAM WIN_CPUS WIN_DISK WIN_USER WIN_PASS TS_IP TS_HOST
python3 - <<'PY'
import datetime, json, os
state = os.environ['STATE_DIR']
config = {key: os.environ[key] for key in ('WIN_VERSION', 'WIN_RAM', 'WIN_CPUS', 'WIN_DISK')}
status = {
    'updated_at': datetime.datetime.now(datetime.timezone.utc).isoformat(),
    'status': 'running',
    'tailscale_ip': os.environ['TS_IP'],
    'tailscale_host': os.environ['TS_HOST'],
    'rdp_host': (os.environ['TS_IP'] or '127.0.0.1') + ':3389',
    'web_console': 'http://' + (os.environ['TS_IP'] or '127.0.0.1') + ':8006',
    'username': os.environ['WIN_USER'],
    'password': os.environ['WIN_PASS'],
    'windows_version': os.environ['WIN_VERSION'],
}
for name, data in (('config.json', config), ('status.json', status)):
    path = os.path.join(state, name)
    with open(path, 'w') as f:
        json.dump(data, f)
    os.chmod(path, 0o600)
PY
`

	ghClient := github.NewClient(token)

	devSha, _ := ghClient.GetFileSHA(body.Owner, body.Repo, ".devcontainer/devcontainer.json")
	if err := ghClient.PutFile(body.Owner, body.Repo, ".devcontainer/devcontainer.json", "feat: setup Windows RDP devcontainer", base64Encode(devcontainerJSON), devSha); err != nil {
		s.jsonError(w, http.StatusInternalServerError, fmt.Sprintf("write devcontainer.json: %v", err))
		return
	}

	workerSha, _ := ghClient.GetFileSHA(body.Owner, body.Repo, ".devcontainer/start-worker.sh")
	if err := ghClient.PutFile(body.Owner, body.Repo, ".devcontainer/start-worker.sh", "feat: setup Windows RDP worker bootstrapper", base64Encode(startWorkerSH), workerSha); err != nil {
		s.jsonError(w, http.StatusInternalServerError, fmt.Sprintf("write start-worker.sh: %v", err))
		return
	}

	s.jsonResponse(w, http.StatusOK, map[string]string{
		"status":  "success",
		"message": fmt.Sprintf("Berhasil mengonfigurasi template Windows RDP pada repo %s/%s", body.Owner, body.Repo),
	})
}

var safeInstanceName = regexp.MustCompile(`^[A-Za-z0-9_-]{1,48}$`)

func (s *Server) handleGenerateVPSScript(w http.ResponseWriter, r *http.Request) {
	tsKey := "${TS_AUTHKEY:?Set TS_AUTHKEY before running this script}"

	version := r.URL.Query().Get("version")
	if version == "" {
		version = "2022"
	}
	if !service.SupportedWindowsVersion(version) {
		s.jsonError(w, http.StatusBadRequest, fmt.Sprintf("unsupported version %q", version))
		return
	}
	name := r.URL.Query().Get("name")
	if name == "" {
		name = fmt.Sprintf("vps-win-%s", time.Now().Format("0102-1504"))
	}
	if !safeInstanceName.MatchString(name) {
		s.jsonError(w, http.StatusBadRequest, "name must match [A-Za-z0-9_-]{1,48}")
		return
	}

	dockerCmd := fmt.Sprintf(`docker run -d \
  --name "%s" \
  --restart unless-stopped \
  --device=/dev/kvm \
  --device=/dev/net/tun \
  --cap-add=NET_ADMIN \
  -e VERSION="%s" \
  -e RAM_SIZE="6G" \
  -e CPU_CORES="2" \
  -e DISK_SIZE="32G" \
  -e USERNAME="WinAdmin" \
  -e PASSWORD="WinPassword%s!" \
  -p 8006:8006 \
  -p 3389:3389/tcp \
  -v /tmp/windows-vps:/storage \
  ghcr.io/dockur/windows`, name, version, time.Now().Format("1504"))

	fullScript := fmt.Sprintf(`#!/usr/bin/env bash
# ====================================================================
# Windows Server Automated VPS Installer via Tailscale + Docker KVM
# ====================================================================
set -euo pipefail

echo "=== 1. Checking Virtualization (KVM) ==="
if [ ! -e /dev/kvm ]; then
    echo "Warning: /dev/kvm not found! Enabling software emulation or check host virtualization."
fi

echo "=== 2. Installing Tailscale & Joining Tailnet ==="
if ! command -v tailscale >/dev/null 2>&1; then
    curl -fsSL https://tailscale.com/install.sh | sh
fi
sudo tailscale up --auth-key="%s" --hostname="%s" --accept-dns=false --reset
TS_IP=$(tailscale ip -4)
echo "Joined Tailscale with IP: $TS_IP"

echo "=== 3. Starting Windows VM in Docker ==="
%s

echo "=== Deployment Completed! Access RDP at $TS_IP:3389 or Console at http://$TS_IP:8006 ==="
`, tsKey, name, dockerCmd)

	s.jsonResponse(w, http.StatusOK, map[string]string{
		"instance_name": name,
		"docker_cmd":    dockerCmd,
		"install_sh":    fullScript,
		"tailscale_key": tsKey,
	})
}

func base64Encode(s string) string {
	return strings.TrimSpace(base64.StdEncoding.EncodeToString([]byte(s)))
}

func (s *Server) handleValidateRepo(w http.ResponseWriter, r *http.Request) {
	token := s.extractBearerToken(r)
	if token == "" {
		s.jsonError(w, http.StatusUnauthorized, "Missing Authorization Bearer token")
		return
	}

	var body struct {
		Owner string `json:"owner"`
		Repo  string `json:"repo"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil || body.Owner == "" || body.Repo == "" {
		s.jsonError(w, http.StatusBadRequest, "Missing owner and repo")
		return
	}

	ghClient := github.NewClient(token)
	details, err := ghClient.GetRepoDetails(body.Owner, body.Repo)
	if err != nil {
		s.jsonError(w, http.StatusBadRequest, fmt.Sprintf("Repository inaccessible or not found: %v", err))
		return
	}

	msg := "Repository is valid and ready."
	if !details.HasDevContainer {
		msg = "Repository has no .devcontainer. Windows and Tailscale will NOT start automatically unless you use sodikinnaa/rdp-generator or add .devcontainer."
	}

	s.jsonResponse(w, http.StatusOK, model.ValidateRepoResponse{
		Valid:           true,
		RepoID:          details.ID,
		FullName:        details.FullName,
		DefaultBranch:   details.DefaultBranch,
		HasDevContainer: details.HasDevContainer,
		Message:         msg,
	})
}

func (s *Server) handleProvision(w http.ResponseWriter, r *http.Request) {
	var req model.ProvisionRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		s.jsonError(w, http.StatusBadRequest, "Invalid request payload")
		return
	}

	vm, err := s.orch.Provision(&req)
	if err != nil {
		s.jsonError(w, http.StatusInternalServerError, err.Error())
		return
	}

	s.jsonResponse(w, http.StatusCreated, vm)
}

func (s *Server) handleListInstances(w http.ResponseWriter, r *http.Request) {
	token := s.extractBearerToken(r)
	if token == "" {
		s.jsonError(w, http.StatusUnauthorized, "Missing Authorization Bearer token")
		return
	}

	ghClient := github.NewClient(token)
	list, err := ghClient.ListCodespaces()
	if err != nil {
		s.jsonError(w, http.StatusInternalServerError, fmt.Sprintf("fetch codespaces: %v", err))
		return
	}

	var viewModels []model.InstanceViewModel
	for _, cs := range list.Codespaces {
		viewModels = append(viewModels, service.InstanceView(cs))
	}

	s.jsonResponse(w, http.StatusOK, viewModels)
}

func (s *Server) handleStartInstance(w http.ResponseWriter, r *http.Request) {
	token := s.extractBearerToken(r)
	if token == "" {
		s.jsonError(w, http.StatusUnauthorized, "Missing Authorization Bearer token")
		return
	}

	name := r.PathValue("name")
	if name == "" {
		s.jsonError(w, http.StatusBadRequest, "Missing instance name")
		return
	}

	ghClient := github.NewClient(token)
	if err := ghClient.StartCodespace(name); err != nil {
		s.jsonError(w, http.StatusInternalServerError, err.Error())
		return
	}

	s.jsonResponse(w, http.StatusOK, map[string]string{"status": "starting", "name": name})
}

func (s *Server) handleStopInstance(w http.ResponseWriter, r *http.Request) {
	token := s.extractBearerToken(r)
	if token == "" {
		s.jsonError(w, http.StatusUnauthorized, "Missing Authorization Bearer token")
		return
	}

	name := r.PathValue("name")
	if name == "" {
		s.jsonError(w, http.StatusBadRequest, "Missing instance name")
		return
	}

	ghClient := github.NewClient(token)
	if err := ghClient.StopCodespace(name); err != nil {
		s.jsonError(w, http.StatusInternalServerError, err.Error())
		return
	}

	s.jsonResponse(w, http.StatusOK, map[string]string{"status": "stopping", "name": name})
}

func (s *Server) handleDeleteInstance(w http.ResponseWriter, r *http.Request) {
	token := s.extractBearerToken(r)
	if token == "" {
		s.jsonError(w, http.StatusUnauthorized, "Missing Authorization Bearer token")
		return
	}

	name := r.PathValue("name")
	if name == "" {
		s.jsonError(w, http.StatusBadRequest, "Missing instance name")
		return
	}

	ghClient := github.NewClient(token)
	if err := ghClient.DeleteCodespace(name); err != nil {
		s.jsonError(w, http.StatusInternalServerError, err.Error())
		return
	}

	s.jsonResponse(w, http.StatusOK, map[string]string{"status": "deleted", "name": name})
}

func (s *Server) extractBearerToken(r *http.Request) string {
	auth := r.Header.Get("Authorization")
	if strings.HasPrefix(auth, "Bearer ") {
		return strings.TrimPrefix(auth, "Bearer ")
	}
	return ""
}

func (s *Server) jsonResponse(w http.ResponseWriter, code int, data any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	_ = json.NewEncoder(w).Encode(data)
}

func (s *Server) jsonError(w http.ResponseWriter, code int, message string) {
	s.jsonResponse(w, code, map[string]string{"error": message})
}
