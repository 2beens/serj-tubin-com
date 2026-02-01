# Gymstats MCP Server

MCP server for **Cursor IDE** (and other MCP clients) when developing the **gymstats** app. It exposes three tools: DB schema, exercises for a time range, and exercise types. Once configured in Cursor, AI agents can use these tools in your prompts to get live gymstats context.

---

## Setting up in Cursor IDE

### Option A: HTTP (recommended — use your deployed backend)

Use the MCP server that runs on your main backend at `/mcp`. No local process needed.

1. **Open Cursor settings**  
   **Cursor Settings → Features → MCP** (or search for “MCP” in settings).

2. **Add a new MCP server**  
   Click **Add new MCP server**.

3. **Configure the server**  
   - **Name**: e.g. `gymstats-context`
   - **Type**: **Streamable HTTP** (or **SSE** if your Cursor version uses that label).
   - **URL**: your backend base URL + `/mcp`, e.g.  
     `https://h.serj-tubin.com/mcp`

4. **Auth** (required for protected `/mcp`):
   - If you use an **MCP secret** (`MCP_SECRET` / `mcp_secret`):  
     - Add a header:  
       `Authorization: Bearer <your-mcp-secret>`  
       or  
       `X-MCP-Secret: <your-mcp-secret>`
   - If you use **session auth** (no MCP secret):  
     - Add header:  
       `X-SERJ-TOKEN: <your-session-token>`

5. **Save** and ensure the server shows as connected (green / available).

You can now mention this MCP server in prompts (e.g. “use the gymstats MCP tools”) so the AI can call `get_gymstats_context`, `get_exercises_for_time_range`, and `get_exercise_types` when answering.

---

### Option B: Stdio (local only, no auth)

Run the MCP server locally over stdio. Useful when you don’t want to hit the deployed backend or use auth.

1. **Open Cursor settings**  
   **Cursor Settings → Features → MCP → Add new MCP server**.

2. **Configure the server**  
   - **Name**: e.g. `gymstats-context-local`
   - **Type**: **stdio**
   - **Command**:  
     `go run ./cmd/gymstats_mcp -config ./config.toml -env development`  
     (run from the repo root; adjust `-config` and `-env` if needed.)

3. **Save**. Cursor will start this command when it needs to talk to the server.

Your local config must point at a reachable DB (e.g. dev or tunnel). No auth is used for stdio.

---

## Using the MCP server in prompts with AI agents

After the server is configured and connected in Cursor:

- **Ask for schema or data explicitly**, e.g.  
  - “Use the gymstats MCP to show me the DB schema for gymstats tables.”  
  - “Call get_exercises_for_time_range for last week and summarize what was logged.”  
  - “List exercise types from the gymstats MCP.”

- **Refer to it by name** if you gave the server a name (e.g. `gymstats-context`):  
  “Use the gymstats-context MCP to get the current exercise types.”

- **Combine with other context**:  
  “Using the gymstats MCP schema and the code in `internal/gymstats/`, suggest a change to the exercises API.”

The AI will invoke the MCP tools when they’re needed to answer; you don’t need to call the tools yourself.

---

## Security: protecting `/mcp`

The `/mcp` route is **not** in the public allowlist and is always protected by the auth middleware.

- **MCP secret (recommended for HTTP)**  
  Set `mcp_secret` in config or `MCP_SECRET` in the environment. Then only requests with that secret can use `/mcp`:
  - Header: `Authorization: Bearer <your-mcp-secret>` or `X-MCP-Secret: <your-mcp-secret>`
  - Use a long, random secret and keep it out of version control.

- **Session auth**  
  If `mcp_secret` / `MCP_SECRET` is empty, `/mcp` uses the same auth as the rest of the API: **X-SERJ-TOKEN** and login session.

---

## Tools

| Tool | Description |
|------|-------------|
| **get_gymstats_context** | DB schema for gymstats tables: exercise, exercise_type, exercise_image, gymstats_event. |
| **get_exercises_for_time_range** | Exercises (sets) in a date range. Args: `from_date`, `to_date` (YYYY-MM-DD); optional: `muscle_group`, `exercise_id`. |
| **get_exercise_types** | All exercise types. Optional filters: `muscle_group`, `exercise_id`. |

---

## Where it runs

- **Integrated (recommended)**  
  The same MCP server is mounted on the main backend at **`/mcp`** (Streamable HTTP). Deploy the main service as usual; no separate MCP deploy.

- **Standalone stdio**  
  This cmd (`cmd/gymstats_mcp`) runs the same `internal/gymstats/mcp.NewServer` over stdio for local Cursor use.
