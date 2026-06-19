# 🚀 NotSet GTM MCP Server (Community Edition)

[![Go Reference](https://pkg.go.dev/badge/github.com/notset-es/gtm-mcp-community.svg)](https://pkg.go.dev/github.com/notset-es/gtm-mcp-community)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](https://opensource.org/licenses/MIT)

The **NotSet GTM MCP Server** is a Model Context Protocol (MCP) gateway that gives your AI Agents (Cursor, Claude Desktop, Windsurf) superpower-level access to **Google Tag Manager** and **Google Analytics 4**. 

No more clicking through 50 menus to audit a dataLayer. Let your AI agent read, write, audit, and orchestrate tracking deployments directly.

---

## ⚡ Quick Start (Zero Config)

Instead of forcing you to navigate Google Cloud's IAM maze to download a Service Account JSON, we've done the heavy lifting. You can connect your favorite AI agent instantly:

### Option A: Local STDIO (Cursor, Windsurf, Claude CLI)
Add a new MCP server in your IDE using our Node wrapper. We handle the OAuth routing securely via STDIO.
```json
{
  "mcpServers": {
    "gtm-community": {
      "command": "npx",
      "args": ["-y", "@notset/gtm-mcp-community"]
    }
  }
}
```

### Option B: Remote SSE (Cloud AI, Glama, ChatGPT)
If your AI operates in the browser or via cloud SSE, just point it to our endpoint.
```text
https://gtm-mcp.notset.es/sse
```

---

## 🛡️ Community vs SaaS Pro (Which do you need?)

The Community Edition is permanently free and open source. It contains the **full** GTM/GA4 capabilities and uses Ephemeral/Volatile memory. When your AI daemon restarts, your session drops to keep your workstation secure.

If you are a serious Data Agency, an Enterprise, or just someone who hates re-authenticating, you need **Zero-Trust Encryption at Rest** and **Persistent Sessions**. 

| Feature | Community Tier | ✨ SaaS Pro / Enterprise |
| --- | --- | --- |
| **GTM/GA4 Capabilities** | Full Access | Full Access |
| **Connection Method** | Local STDIO & Cloud SSE | Local, Cloud, & M2M Proxy |
| **Secure Token Storage** | ❌ Volatile ephemeral | ✅ Zero Trust at Rest (AES-256-GCM) |
| **Agent Autonomy** | Good for punctual tasks | Uninterrupted 24/7 background CRONs |
| **Multi-layer Governance** | ❌ None | ✅ Centralized Dashboard (Team Management) |
| **Advanced Audits** | ❌ Basic | ✅ PII detection, naming conventions, regex limits |

**[👉 Upgrade to NotSet SaaS Pro to stabilize your Agents](https://mcp.notset.es/pricing)**

---

## 🏎️ Building from Source (For Gophers)

Are you paranoid? Good. You should be. Here is how you build the server yourself instead of trusting our NPX wrapper:

```bash
git clone https://github.com/notset-es/gtm-mcp-community
cd gtm-mcp-community
go mod tidy
go build -o gtm-mcp main.go

# Run locally in stdio mode (Bring Your Own Service Account allowed if you want!)
./gtm-mcp -stdio
```

## 🧠 Capabilities (Tools Ref)
This server acts as a single Gateway (`gtm` tool) to avoid blasting the MCP token limits. Operations are partial updates.

| Resource | Supported Actions |
| --- | --- |
| `workspace` | list, create, status |
| `tag` | list, get, create, update, delete, revert |
| `variable` | list, get, create, update, delete, revert |
| `trigger` | list, get, create, update, delete, revert |
| `folder` | list, get, create, update, delete, move |
| `gtag_config` | list, get, create, update, delete |

> **Note:** Community template tags use esoteric IDs like `cvt_CONTAINERID_NNN`. Do not let Claude guess the tag type. Ensure it runs the `templates_ref` action to discover the appropriate types.

---

## 🤝 Contributing
Found a bug? Want to wrap another Google API? PRs are welcome. Just try not to break the `auth` middleware; we like our secrets kept secret.

## 📄 License
MIT License - See the [LICENSE](LICENSE) file for details. No warranty provided.
