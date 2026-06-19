# 🚀 NotSet GTM MCP Server (Community Edition)

[![Go Reference](https://pkg.go.dev/badge/github.com/notset-es/gtm-mcp-community.svg)](https://pkg.go.dev/github.com/notset-es/gtm-mcp-community)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](https://opensource.org/licenses/MIT)

The **NotSet GTM MCP Server** is a Model Context Protocol (MCP) gateway that gives your AI Agents (Cursor, Claude Desktop, Windsurf) superpower-level access to **Google Tag Manager**. 

No more clicking through 50 menus to audit a dataLayer or configure tags. Let your AI agent read, write, audit, and orchestrate tracking deployments directly via code.

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

## 🔑 Authentication
The NotSet GTM MCP Server automatically handles authentication. 
- **Option A (NPX Wrapper):** It will launch a secure Google OAuth flow in your browser on first run.
- **Option B (Source/Self-Hosted):** It natively respects the `GOOGLE_APPLICATION_CREDENTIALS` environment variable if you prefer to use a strict Service Account for your agents.

---

## 🛡️ Community vs Enterprise Pro (Which do you need?)

The Community Edition is permanently free and open source. It contains **full GTM CRUD capabilities** and uses Ephemeral/Volatile memory. When your AI daemon restarts, your session drops to keep your workstation secure.

If you are a serious Data Agency, an Enterprise, or just someone who hates re-authenticating and needs advanced auditing, you might need the **Enterprise Pro** version with **Zero-Trust Encryption at Rest**, **Persistent Sessions**, and **Cross-Auditing**.

| Feature | Community Tier | ✨ Enterprise Pro |
| --- | --- | --- |
| **GTM Capabilities** | Full Access | Full Access |
| **GA4 Capabilities** | ❌ None | ✅ Full Access + Cross-Audits |
| **Connection Method** | Local STDIO & Cloud SSE | Local, Cloud, & M2M Proxy |
| **Secure Token Storage** | ❌ Volatile ephemeral | ✅ Zero Trust at Rest (AES-256-GCM) |
| **Agent Autonomy** | Good for punctual tasks | Uninterrupted 24/7 background CRONs |
| **Multi-layer Governance**| ❌ None | ✅ Centralized Dashboard (Team Management) |
| **Advanced Audits** | ❌ Basic CRUD | ✅ Journey Simulation, Consent validation, PII detection |

**👉 Interested in the Enterprise Pro version?** 
Contact me directly at [raul@measuremesh.io](mailto:raul@measuremesh.io) or [raul.fernandez@notset.es](mailto:raul.fernandez@notset.es) to discuss your needs.

---

## 🎯 Example Prompts for your AI
Once connected, try asking your agent:
- *"Audit my GTM workspace `WS_ID` and tell me if any tags are missing a trigger."*
- *"Create a new Custom HTML tag that fires on all pages, containing a simple console.log."*
- *"Extract all RegEx patterns from the lookup variable `VAR_ID` and format them as a table."*
- *"Delete the workspace `WS_ID`. Yes, I confirm this action."*

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
| `workspace` | list, create, status, delete |
| `tag` | list, get, create, update, delete, revert, append_list_entry, remove_list_entry, list_entries |
| `variable` | list, get, create, update, delete, revert, add_lookup_entry, remove_lookup_entry, list_lookup_entries, append_list_entry, remove_list_entry, list_entries |
| `trigger` | list, get, create, update, delete, revert |
| `folder` | list, get, create, update, delete, move, audit, revert |
| `template` | list, get, create, update, delete, import, revert |
| `version` | list, get, create, publish, compare, export, import |
| `gtag_config` | list, get, create, update, delete |

> **Note:** Community template tags use specific numeric IDs. Since advanced metadata discovery (`templates_ref`) is a Pro feature, you may need to provide the exact template IDs manually to your agent when creating custom tags.

---

## 🤝 Contributing
Found a bug? Want to wrap another Google API? PRs are welcome. Just try not to break the `auth` middleware; we like our secrets kept secret.

## 📄 License
MIT License - See the [LICENSE](LICENSE) file for details. No warranty provided.
