# RDP Generator — Windows Server on GitHub Codespaces

A lightweight, single-binary automation tool and modern web dashboard to provision and manage cloud-hosted Windows Server virtual machines inside GitHub Codespaces, secured via Tailscale mesh VPN with zero public port exposure.

---

## 🏗️ Architecture

```text
┌────────────────────────────────────────────────────────────────────────┐
│                          Operator Workstation                          │
│                                                                        │
│   ┌──────────────────────────────────────────────────────────────┐     │
│   │   rdp-generator (Go Single Binary + Embedded Web UI)         │     │
│   │   http://127.0.0.1:8787                                      │     │
│   └───────────────┬──────────────────────────────────────────────┘     │
│                   │                                                    │
│                   │ 1. Encrypt Secret via NaCl Sealed Box              │
│                   │ 2. Create/Control Codespace via REST API           │
│                   ▼                                                    │
│         GitHub Codespaces API                                          │
└───────────────────┬────────────────────────────────────────────────────┘
                    │
                    ▼ (Spawns dedicated VM)
┌────────────────────────────────────────────────────────────────────────┐
│              GitHub Codespace Environment (Ubuntu 64-bit)              │
│                                                                        │
│   ┌──────────────────────────────────────────────────────────────┐     │
│   │  DevContainer Worker (.devcontainer/start-worker.sh)         │     │
│   │                                                              │     │
│   │  ┌───────────────────────┐       ┌────────────────────────┐  │     │
│   │  │  Tailscale Daemon     │       │  dockur/windows        │  │     │
│   │  │  Node: rdp-win-xxxx   │       │  QEMU + KVM (2 vCPU)   │  │     │
│   │  │  IP: 100.x.x.x        │       │  Windows Server 2022   │  │     │
│   │  └───────────┬───────────┘       └───────────┬────────────┘  │     │
│   │              │                               │               │     │
│   │              │ Secure socat Relays           │               │     │
│   │              ├───────────────────────────────┤               │     │
│   │              │ 100.x.x.x:3389  ──> 127.0.0.1:3389 (RDP)      │     │
│   │              │ 100.x.x.x:8006  ──> 127.0.0.1:8006 (noVNC)    │     │
│   └──────────────┴───────────────────────────────┴───────────────┘     │
└──────────────────────────────────┬─────────────────────────────────────┘
                                   │
                                   ▼ Tailscale WireGuard Mesh
┌────────────────────────────────────────────────────────────────────────┐
│                      Operator Tailscale Client                         │
│                                                                        │
│   • Microsoft Remote Desktop (mstsc / xfreerdp)  --> 100.x.x.x:3389    │
│   • Web Browser (Direct noVNC Console)           --> 100.x.x.x:8006    │
└────────────────────────────────────────────────────────────────────────┘
```

---

## ✨ Features

- **Single Binary Executable**: Built in pure Go with embedded HTML5 dashboard. No PHP, Node.js, or external runtimes required.
- **Zero Exposed Ports**: RDP (3389) and noVNC (8006) are bound strictly to the Tailscale overlay IP. No GitHub public ports or firewall holes.
- **Hardware Virtualization**: Utilizes native `/dev/kvm` hardware acceleration provided by Azure-backed GitHub Codespaces hosts.
- **End-to-End Secret Encryption**: Tailscale authentication keys are encrypted client-side using LibSodium sealed-box encryption (X25519-XSalsa20-Poly1305) before reaching GitHub.
- **Cross-Platform**: Compile for Windows (`.exe`), Linux, or macOS.

---

## 🚀 Quick Start

### 1. Download or Build the Binary

#### Build from source:
```bash
git clone https://github.com/sodikinnaa/rdp-generator.git
cd rdp-generator

# Build for current OS (e.g. Linux)
go build -ldflags="-s -w" -o rdp-generator main.go

# Cross-compile for Windows:
GOOS=windows GOARCH=amd64 go build -ldflags="-s -w" -o rdp-generator.exe main.go
```

### 2. Run the Controller

```bash
./rdp-generator -port 8787
```

On Windows:
```powershell
.\rdp-generator.exe -port 8787
```

Open your browser at **`http://127.0.0.1:8787`**.

### 3. Deploy Windows Guest

1. Provide your **GitHub Personal Access Token** (Classic PAT with `codespace` scope).
2. Provide your **Tailscale Auth Key** (created from [Tailscale Admin](https://login.tailscale.com/admin/settings/keys) with *Reusable*, *Ephemeral*, and *Pre-approved* enabled).
3. Select your desired Windows Edition:
   - **Windows Server 2022** (Evaluation — Recommended)
   - **Windows Server 2025**
   - **Windows Server 2019**
   - **Windows 11 Pro**
4. Select machine type: `standardLinux32gb` (4 Cores, 16 GB RAM).
5. Click **Deploy Windows Lab**.

### 4. Connect via RDP

Once the worker completes booting:
- Open your Remote Desktop client (`mstsc` on Windows or `xfreerdp` on Linux).
- Connect to: `<tailscale-ip>:3389`
- Username: `WinAdmin`
- Password: Shown on the dashboard credentials card.
- Or access the graphical web console via `http://<tailscale-ip>:8006`.

---

## ⚙️ Configuration & Flags

| Flag | Default | Description |
|---|---|---|
| `-port` | `8787` | HTTP port for the local web UI |
| `-host` | `127.0.0.1` | Host address to bind |
| `-open` | `false` | Automatically open default browser on start |
| `-version` | `false` | Display version and architecture |

---

## 🛡️ Security & Lifecycle Notes

1. **Token Hygiene**: Never commit Personal Access Tokens or Tailscale keys to source control. The controller stores keys strictly in memory during active operations.
2. **Tailscale ACL Recommendation**: Restrict access to `tag:winlab` so only authorized devices in your tailnet can communicate with ports `3389` and `8006`.
3. **Codespaces Quotas**: GitHub Free accounts include 120 core-hours and 15 GB-month of storage per month. A 4-core machine utilizes 4 core-hours per wall-clock hour.
4. **Session Limits**: A Codespace automatically suspends after an idle period (default 30–240 minutes) and has a maximum hard runtime limit of 12 hours.
5. **Storage Location**: The guest VM disk is created in `/tmp/windows`. Files in `/tmp` are wiped upon codespace shutdown. Treat test lab instances as ephemeral.

---

## 📄 License

MIT License. See [LICENSE](LICENSE) for details.
