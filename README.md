# GTM MCP Server

[![License](https://img.shields.io/badge/License-BSD_3--Clause-blue.svg)](https://opensource.org/licenses/BSD-3-Clause)
[![Go](https://img.shields.io/badge/Go-1.24-00ADD8?logo=go&logoColor=white)](https://go.dev/)
[![MCP](https://img.shields.io/badge/MCP-Model_Context_Protocol-8A2BE2)](https://modelcontextprotocol.io)
[![Claude](https://img.shields.io/badge/Claude-Compatible-D97757?logo=anthropic&logoColor=white)](https://claude.ai)
[![ChatGPT](https://img.shields.io/badge/ChatGPT-Compatible-74aa9c?logo=openai&logoColor=white)](https://chatgpt.com)
[![Gemini](https://img.shields.io/badge/Gemini_CLI-Compatible-4285F4?logo=google&logoColor=white)](https://geminicli.com)
[![Docker](https://img.shields.io/badge/Docker-Ready-2496ED?logo=docker&logoColor=white)](https://github.com/notset-es/gtm-mcp-server)

**Let AI manage your Google Tag Manager containers.**

Create tags, audit configurations, generate tracking plans, and publish changes — all through natural conversation with Claude, ChatGPT, or Gemini.

**URL:** `https://gtm-mcp.notset.es`

```
https://gtm-mcp.notset.es
```

---

## Table of Contents

- [What Can You Do?](#what-can-you-do)
- [Quick Start](#quick-start)
- [Features](#features)
- [Complete API Reference](#complete-api-reference)
- [Use Cases](#use-cases)
- [How It Works](#how-it-works)
- [Architecture](#architecture)
- [Safety & Security](#safety--security)
- [Self-Hosting](#self-hosting)
- [Better AI Context](#better-ai-context)
- [Known Issues](#known-issues)
- [Acknowledgments](#acknowledgments)
- [Author](#author)
- [License](#license)

---

## What Can You Do?

Ask your AI assistant to:

- *"List all my GTM containers"*
- *"Create a GA4 event tag for form submissions"*
- *"Audit this container for issues and duplicates"*
- *"Generate a tracking plan document for the marketing team"*
- *"Set up ecommerce tracking for purchases"*
- *"Publish the changes we just made"*
- *"Compare version 5 with the latest published version"*
- *"Export this container as JSON for backup"*
- *"Import this template from the Community Gallery"*
- *"Manage user permissions for my team"*

No more clicking through the GTM interface. No more copy-pasting configurations. Just describe what you need.

---

## Quick Start

### Claude (Web & Desktop)

**Claude.ai:**
1. Go to **Settings** → **Connectors** → **Add Custom Connector**
2. Enter: `https://gtm-mcp.notset.es`
3. Click **Add** and sign in with Google

**Claude Code (CLI):**
```bash
claude mcp add -t http gtm https://gtm-mcp.notset.es
```

### ChatGPT

1. Go to [OpenAI Apps Platform](https://platform.openai.com/apps)
2. Add an MCP integration with URL: `https://gtm-mcp.notset.es`
3. Authorize with your Google account

### Gemini CLI

```bash
gemini mcp add --transport http --url https://gtm-mcp.notset.es gtm
```

### Local (stdio mode)

For direct integration without authentication:
```bash
./gtm-mcp-server --stdio
```

---

## Features

### 🏷️ Complete Tag Management
Create and modify any GTM tag type:
- **GA4 Configuration & Events** — measurement IDs, event parameters, e-commerce
- **Custom HTML** — scripts, pixels, and custom code injection
- **Custom Image** — tracking pixels with cache busting
- **Any tag type** — full parameter support via the GTM API

### ⚡ Trigger Management
Build triggers for any scenario:
- Page views (all pages or specific URL patterns)
- Custom dataLayer events
- Click tracking (all clicks or specific elements)
- Form submissions
- Timer-based triggers
- Trigger groups for complex conditions

### 📦 Container Operations
- Browse accounts, containers, and workspaces
- Create and delete containers
- Check workspace status for pending changes
- Organize entities with folders (including move and audit)

### 🔄 Version Management
Full version lifecycle control:
- **Create** versions from workspace changes
- **Publish** versions to go live
- **Compare** two versions side-by-side
- **Find by date** — locate the version that was live at a specific time
- **Set latest** — mark a version as the current one
- **Export** containers as importable JSON
- **Import** containers from JSON

### 🌐 Server-Side Containers
Full support for server-side GTM:
- **Clients** — create, update, and delete server-side clients (e.g. GA4 client)
- **Transformations** — control event parameters with allow, exclude, and augment rules

### 🏗️ Advanced Resources
Resources not commonly found in other GTM tools:
- **Environments** — manage preview, staging, and live environments
- **User Permissions** — grant, update, and revoke access at account/container level
- **Zones** — configure content security zones
- **GTag Configs** — manage Google Tag configurations
- **Destinations** — list, inspect, and link tag destinations
- **Built-in Variables** — enable/disable GTM's built-in variable types

### 🧩 Community Template Gallery
Import templates from Google's Community Template Gallery:
- *"Import the iubenda cookie consent template"*
- *"Add Cookiebot to my container"*
- *"Set up Facebook Pixel using the gallery template"*

The AI will search for the template, find the GitHub repository, and import it automatically.

### 🤖 AI-Powered Workflows

**Container Audit**
*"Audit my container for issues"* — deep analysis of:
- Naming inconsistencies and convention violations
- Duplicate tags and orphaned triggers
- Security concerns and best practice violations
- Data layer gaps and ecommerce coverage

**Tracking Plan Generation**
*"Generate a tracking plan"* — complete markdown documentation of:
- All events and their triggers
- Data layer requirements and variable definitions
- Implementation notes and dependencies

**Debug & Troubleshooting**
Built-in debugging prompts to diagnose tag firing issues, consent conflicts, and data layer problems.

---

## Complete API Reference

All operations go through a single unified `gtm` tool using the gateway pattern:

```json
{"resource": "<resource>", "action": "<action>", "args": {<params>}}
```

### Resources & Actions

| Resource | Actions |
|---|---|
| `account` | `list` |
| `container` | `list` · `create` · `delete` |
| `workspace` | `list` · `create` · `status` |
| `tag` | `list` · `get` · `create` · `update` · `delete` · `revert` |
| `trigger` | `list` · `get` · `create` · `update` · `delete` · `revert` |
| `variable` | `list` · `get` · `create` · `update` · `delete` · `revert` |
| `folder` | `list` · `get` · `create` · `update` · `delete` · `move` · `audit` · `revert` |
| `template` | `list` · `get` · `create` · `update` · `delete` · `import` · `revert` |
| `built_in_variable` | `list` · `enable` · `disable` · `revert` |
| `client` | `list` · `get` · `create` · `update` · `delete` · `revert` |
| `transformation` | `list` · `get` · `create` · `update` · `delete` · `revert` |
| `environment` | `list` · `get` · `create` · `update` · `delete` |
| `user_permission` | `list` · `get` · `create` · `update` · `delete` |
| `version` | `list` · `get` · `create` · `publish` · `compare` · `find_by_date` · `set_latest` · `export` · `import` |
| `destination` | `list` · `get` · `link` |
| `zone` | `list` · `get` · `create` · `update` · `delete` · `revert` |
| `gtag_config` | `list` · `get` · `create` · `update` · `delete` |
| `templates_ref` | `tag_templates` · `trigger_templates` |
| `ping` | *(no action needed)* |
| `auth_status` | *(no action needed)* |

> **20 resources · 100+ operations** — the most complete GTM MCP implementation available.

### MCP Resources (URI-based access)

```
gtm://accounts
gtm://accounts/{id}/containers
gtm://accounts/{id}/containers/{id}/workspaces
gtm://accounts/.../workspaces/{id}/tags
gtm://accounts/.../workspaces/{id}/triggers
gtm://accounts/.../workspaces/{id}/variables
```

### MCP Prompts (Workflow templates)

| Prompt | Description |
|--------|-------------|
| `audit_container` | Comprehensive container analysis with issue detection |
| `generate_tracking_plan` | Complete markdown documentation generator |
| `suggest_ga4_setup` | GA4 implementation recommendations |
| `find_gallery_template` | Guide to find and import Community Gallery templates |
| `debug_container` | Troubleshoot tag firing, consent, and data layer issues |

### Backward Compatibility

Legacy tool names (e.g. `list_accounts`, `create_tag`) are automatically routed to the gateway via built-in middleware. Existing integrations continue to work without changes.

---

## Use Cases

### Build Complete Tracking Setups
Ask AI to create a full GA4 ecommerce implementation from scratch:
- Creates 12+ tags (configuration + all ecommerce events)
- Creates matching triggers for each dataLayer event
- Creates data layer variables for items, currency, value, transaction_id
- Follows Google's recommended event naming and parameters

### Implement Consent Management
Integrate privacy tools like OneTrust with your tracking:
- Creates consent-checking variables
- Sets up conditional triggers
- Updates existing tags to respect user choices

### Version Control & Rollback
Compare versions, find what was live at a specific date, and roll back:
- *"Compare the current version with what was live last week"*
- *"Find the version that was published on March 1st"*
- *"Export a backup of the container before making changes"*

### Team Management
Control who has access to what:
- *"Give maria@company.com edit access to the production container"*
- *"List all users with publish permissions"*
- *"Revoke access for the former contractor"*

### Bulk Operations & Renaming
Manage containers at scale:
- *"Add 'ecom -' prefix to all ecommerce triggers"*
- *"Update all tags to use a measurement ID variable"*

### For Agencies
- Manage multiple client containers across accounts
- Standardize implementations across clients
- Rapid setup for new projects
- Version and publish changes safely

---

## How It Works

```
┌─────────────┐     ┌──────────────────┐     ┌─────────────┐
│  Claude /    │     │   GTM MCP Server │     │  Google Tag │
│  ChatGPT /  │◄───►│                  │◄───►│  Manager    │
│  Gemini     │ MCP │  OAuth 2.1+PKCE  │     │  API        │
└─────────────┘     │  Rate Limiting   │     └─────────────┘
                    │  Encrypted Store │
                    └──────────────────┘
```

1. **Connect** — Add the server URL to your AI client
2. **Authenticate** — Sign in with your Google account (OAuth 2.1 with PKCE)
3. **Operate** — Ask your AI to manage GTM in natural language
4. **Stay safe** — Destructive operations require confirmation before executing

Your Google credentials are stored encrypted (AES-GCM) in a local SQLite database. Tokens are refreshed automatically and can be revoked from your Google account at any time.

---

## Architecture

### Technical Stack

| Component | Technology |
|---|---|
| **Language** | Go 1.24 |
| **Protocol** | MCP (Model Context Protocol) over HTTP + stdio |
| **Auth** | OAuth 2.1 with PKCE (RFC 8414, RFC 7591, RFC 9728) |
| **Token Storage** | SQLite with AES-GCM encryption |
| **API** | Google Tag Manager API v2 |
| **Containerization** | Docker multi-stage (Alpine, non-root) |

### Design Decisions

- **Single Gateway Tool** — All 100+ operations route through one `gtm` tool, keeping the MCP tool surface minimal and within AI client limits
- **Backward Compatibility Middleware** — Legacy tool names (`list_accounts`, `create_tag`, etc.) are transparently mapped to gateway calls
- **Audit Middleware** — Every request is logged with structured metadata for observability
- **Rate Limiting** — Separate limiters for OAuth (10 req/s), registration (2 req/s), and MCP (30 req/s) endpoints
- **Encrypted Token Store** — Google OAuth tokens are encrypted at rest using AES-GCM derived from the JWT secret
- **Graceful Shutdown** — Signal handling with 10-second timeout for in-flight requests

---

## Safety & Security

### Operational Safety
- **Workspace isolation** — all changes happen in workspaces, nothing goes live until you publish
- **Confirmation required** — deletions and publishing require explicit approval
- **Version control** — changes create a new version automatically
- **Audit logging** — structured logs for every operation

### Security Features
- **OAuth 2.1 with PKCE** — no client secrets exposed to the browser
- **Encrypted token storage** — AES-GCM encryption at rest
- **Rate limiting** — protection against abuse on all endpoints
- **Non-root Docker** — runs as `appuser` with minimal privileges
- **Request size limits** — body size caps on all endpoints
- **Dynamic Client Registration (DCR)** — with domain allowlist support
- **No hardcoded secrets** — all credentials via environment variables

---

## Self-Hosting

### Docker (recommended)

```bash
git clone https://github.com/notset-es/gtm-mcp-server.git
cd gtm-mcp-server

# Create .env file
cat > .env << 'EOF'
GOOGLE_CLIENT_ID=your-client-id.apps.googleusercontent.com
GOOGLE_CLIENT_SECRET=your-client-secret
JWT_SECRET=$(openssl rand -hex 32)
BASE_URL=https://your-domain.com
PORT=8080
EOF

# Start the server
docker build -t gtm-mcp-server .
docker run -p 8080:8080 --env-file .env gtm-mcp-server
```

### Binary (no Docker needed)

Download a pre-built binary from [Releases](https://github.com/notset-es/gtm-mcp-server/releases) or build from source:

```bash
# Build for your platform
go build -ldflags="-w -s" -o gtm-mcp-server .

# Or cross-compile for Linux ARM64 (e.g. Oracle Cloud)
CGO_ENABLED=0 GOOS=linux GOARCH=arm64 go build -ldflags="-w -s" -o gtm-mcp-server .
```

### Stdio Mode (local, no auth)

For direct integration with tools like Claude Code without running an HTTP server:
```bash
./gtm-mcp-server --stdio
```

### Google Cloud Setup

1. Go to [Google Cloud Console](https://console.cloud.google.com/)
2. Enable the **Tag Manager API**
3. Create **OAuth 2.0 credentials** (Web application)
4. Add redirect URIs:
   ```
   https://claude.ai/api/mcp/auth_callback
   https://claude.com/api/mcp/auth_callback
   https://chatgpt.com/connector_platform_oauth_redirect
   https://gtm-mcp.notset.es/oauth/callback
   ```

### Environment Variables

| Variable | Required | Default | Description |
|---|---|---|---|
| `GOOGLE_CLIENT_ID` | Yes* | — | Google OAuth 2.0 Client ID |
| `GOOGLE_CLIENT_SECRET` | Yes* | — | Google OAuth 2.0 Client Secret |
| `JWT_SECRET` | Yes* | — | Secret for JWT signing (min. 32 chars) |
| `BASE_URL` | No | `http://localhost:8080` | Public URL of the server |
| `PORT` | No | `8080` | HTTP port |
| `LOG_LEVEL` | No | `info` | `info` or `debug` |
| `ACCESS_TOKEN_TTL` | No | `8h` | Access token lifetime |
| `MAX_RETRIES` | No | `3` | API retry attempts |
| `GOOGLE_SCOPES` | No | All GTM scopes | Comma-separated scope list |
| `ALLOWED_HOSTS` | No | — | Trusted hostnames for Docker-to-Docker |
| `TRUSTED_PROXIES` | No | — | Trusted reverse proxy IPs/CIDRs |
| `ALLOWED_DCR_DOMAINS` | No | — | Restrict DCR to specific domains |

*Required for HTTP mode with auth. Not needed in stdio mode.

### Docker-to-Docker

If another container needs to reach the MCP server via an internal Docker network alias:

```bash
ALLOWED_HOSTS=gtm-mcp:8080
```

This enables dynamic URL resolution for trusted internal hostnames while keeping the server secure against host header injection.

---

## Better AI Context

For best results, install the **GTM API skill** so your AI assistant understands GTM's API structure, parameter formats, and validation rules.

### Claude Code

```bash
# One-liner install
curl -sL https://github.com/notset-es/gtm-api-for-llms/archive/main.tar.gz | tar xz && \
  mkdir -p ~/.claude/skills && \
  cp -r gtm-api-for-llms-main/skills/gtm-api ~/.claude/skills/ && \
  rm -rf gtm-api-for-llms-main
```

### OpenAI Codex

```bash
curl -sL https://github.com/notset-es/gtm-api-for-llms/archive/main.tar.gz | tar xz && \
  mkdir -p ~/.codex/skills && \
  cp -r gtm-api-for-llms-main/skills/gtm-api ~/.codex/skills/ && \
  rm -rf gtm-api-for-llms-main
```

The [GTM API for LLMs](https://github.com/notset-es/gtm-api-for-llms) repository provides LLM-optimized documentation: request templates, validation rules, workflow algorithms, and complete schemas for all GTM entity types including server-side containers.

---

## Known Issues

### 🐛 `autoEventFilter` silently dropped by Google Tag Manager API

When creating or updating `linkClick`, `click`, or `formSubmission` triggers via the API, the `autoEventFilter` field is silently dropped by the Google Tag Manager API. The API returns `200 OK` but does not persist the field.

**Workaround:** Configure `autoEventFilter` conditions through the [GTM web interface](https://tagmanager.google.com). The MCP server can read triggers that have `autoEventFilter` set via the UI.

**Status:** [#33](https://github.com/notset-es/gtm-mcp-server/issues/33)

---

## Acknowledgments

This project was inspired by [Paolo Bietolini's gtm-mcp-server](https://github.com/paolobietolini/gtm-mcp-server), which demonstrated the potential of connecting GTM with AI through MCP. This implementation is a ground-up rewrite with a different architecture, expanded resource coverage, encrypted token storage, and additional security hardening.

---

## Links

- [GitHub Repository](https://github.com/notset-es/gtm-mcp-server)
- [GTM API Reference](https://github.com/notset-es/gtm-api-for-llms)
- [MCP Specification](https://modelcontextprotocol.io)

---

## Author

**Raúl Fernández Molina** — [notset.es](https://notset.es)

raul.fernandez@notset.es

---

## License

[BSD-3-Clause](LICENSE)
