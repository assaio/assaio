# What each source's log carries, and what assaio reads

*Part of [Extending assaio](../extending.md). The per-signal summary of this audit is the source-depth matrix in the [generated reference](https://assaio.dev/docs/reference).*

Every parser turns a rich log into one fixed record and drops the rest. This is the inventory
of that drop, source by source: each field a source writes ends in exactly one of two states —
**extracted**, meaning a signal in the catalog is computed from it and a golden covers it, or
**skipped**, with the reason written down. A field whose meaning the vendor does not document
is skipped *with that stated*; it is never guessed at from its name.

**How it was produced.** Key paths were inventoried from real logs on one machine plus each
parser's synthetic fixture — names and counts only, never a value, except for discriminator
keys (`type`, `say`, `role`, `status`, `stop_reason`, …) whose values are the format's own
vocabulary. The corpus is stated per source below, because it is the honest limit of this
table: **a field that does not appear in the corpus is exactly the one most likely to be
missing from it.** Re-run the audit when a vendor ships a major version.

None of these formats is documented as an interface. Every "meaning" below is either stated by
the vendor or inferred from a name that leaves no room — and where it does leave room, the
field is skipped for that reason.

## Claude Code

*Corpus: 5,602 transcripts · 703,320 lines · 2,284 distinct key paths, plus
`testdata/session.jsonl`.*

| Field | State | Notes |
|---|---|---|
| `uuid`, `sessionId`, `timestamp`, `cwd`, `gitBranch`, `entrypoint` | extracted | Record identity, dedupe key and dimensions. |
| `message.model`, `message.usage.{input_tokens,output_tokens,cache_read_input_tokens,cache_creation_input_tokens}` | extracted | Every token signal and the cost estimate. |
| `message.id` | extracted (v0.12) | The API response a line belongs to. Claude writes one line per content block and repeats that response's usage on each, so keying a record on the line `uuid` counted one request once per block — 354,904 lines were 159,175 responses on the audited corpus, inflating output tokens 1.97x and cache-write 2.81x. A record is now keyed on this. |
| `message.content[].type` = `tool_use` / `tool_result` + `.name` / `.is_error` | extracted | Tool-call count, the purpose split, and `ai.tool_errors.count`. |
| `toolUseResult.structuredPatch[].lines` | extracted | `ai.lines.added` / `.removed` and, via the shared helper, `ai.rework.lines`. |
| `toolUseResult.{agentId,agentType,resolvedModel,usage,toolStats.linesAdded,toolStats.linesRemoved}` | extracted | The completed sub-agent's own record. |
| `toolDenialKind` | extracted | `ai.rejected.count`. |
| `isCompactSummary`, `subtype` = `compact_boundary` | extracted | `ai.compactions.count`. |
| `isSidechain`, `attributionSkill`, `attributionAgent` | extracted | Delegation share and the skill / sub-agent split. |
| `message.usage.cache_creation.ephemeral_5m_input_tokens` / `ephemeral_1h_input_tokens` | extracted (v0.12) | The cache TTL tier, on every assistant turn. 59.7% of the audited corpus's cache-write tokens bought the 1-hour lifetime, which bills at 1.6x the 5-minute rate, so reading it raised the cache-write cost component by 35.8%. Signal `ai.tokens.cache_write_1h`. |
| `message.diagnostics.cache_miss_reason.type` | extracted (v0.12) | Six documented-by-vocabulary reasons a prompt cache missed (`messages_changed`, `model_changed`, `previous_message_not_found`, `system_changed`, `tools_changed`, `unavailable`). `cache-hygiene` names the top one, turning a cache rate into a cause. Signal `ai.cache.miss_reason`. |
| `toolUseResult.userModified` | **skipped — worth extracting** | The human edited what the AI wrote. A direct correction signal where `rework` has only a proxy (`B111`). |
| `toolUseResult.toolStats.{readCount,searchCount,bashCount,editFileCount,otherToolCount}` | **skipped — accounting undecided** | The sub-agent's own purpose split, already in the log. Populating it means also setting `ToolCalls`, which would double-count against the parent turn — the open half of `B78`. |
| `attributionPlugin`, `attributionMcpServer`, `attributionMcpTool` | **skipped — worth extracting** | Two more attribution dimensions beside skill and sub-agent, on a third of all turns (`B112`). |
| `version` | **skipped — worth extracting** | The Claude Code build that wrote the line, on every turn. The harness-version cohort input (`B96`) is already on disk (`B112`). |
| `message.stop_reason` | **skipped — worth extracting** | Why generation stopped; `max_tokens` marks a truncated answer (`B113`). |
| `toolUseResult.interrupted` | **skipped — worth extracting** | A command the human cut short (`B113`). |
| `message.content[].thinking` | skipped — no count exists | Thinking blocks are present, but Claude publishes no thinking **token** count, and inferring one from text length would be a fabricated number. What is honestly derivable is a share of turns that carried one — a different signal from `ai.tokens.reasoning`, and not one the catalog declares today. |
| `effort` | skipped — undocumented | Appears on most turns; the vendor documents neither its vocabulary nor whether it is a request or an outcome. Named in `B113` as research, not as an extraction. |
| `message.usage.server_tool_use.{web_search_requests,web_fetch_requests}` | skipped — undocumented billing | Server-side tool calls the vendor bills on its own terms; folding them into any token figure would state a price we cannot compute. |
| `message.usage.{speed,inference_geo,iterations,service_tier}` | skipped — undocumented | No published meaning. `service_tier` reads `standard` on every line here, which is a constant, not a signal. |
| `slug`, `promptId`, `requestId`, `messageId`, `leafUuid`, `parentUuid`, `sourceToolAssistantUUID` | skipped — identity, no measure | Threading and request identifiers. `parentUuid` would let a fork be detected; nothing measures forks yet, so it stays out rather than being stored speculatively. |
| `attachment.*`, `snapshot.*`, `backup.*`, `trackingPath`, `lastPrompt`, `aiTitle`, `content`, `message.content[].text`, `toolUseResult.{file,content,oldString,newString,originalFile,stdout,stderr}` | skipped — content, by construction | Prompts, code, file contents, command output, editor backups and the file-history snapshots. These are what PRIVACY.md promises are never collected; they are read transiently at most (diff markers, a path used only to key rework in memory) and never stored. |
| `attachment.type` = `hook_*`, `skill_listing`, `invoked_skills`, `mcp_instructions_delta`, `queued_command`, `max_turns_reached`, `plan_mode*` | skipped — harness inventory, not usage | Hook outcomes, skill and MCP availability, mode changes. These belong to the harness inventory (`B95`), which stores an artifact's *shape*, not a usage record. |

## Codex CLI

*Corpus: 21 rollouts · 3,599 lines · 273 distinct key paths, plus `testdata/rollout.jsonl`.*

| Field | State | Notes |
|---|---|---|
| `timestamp`, `type`, `payload.type` | extracted | Line routing and every record's own timestamp. |
| `payload` of `session_meta`: `id`, `cwd`, `model`, `timestamp` | extracted | Session identity and dimensions. |
| `turn_context.model` | extracted | Model carried forward across a mid-session switch. |
| `token_count` → `payload.info.total_token_usage.{input_tokens,cached_input_tokens,output_tokens,reasoning_output_tokens}` | extracted | Cumulative totals, differenced into per-turn records. |
| `patch_apply_end` → `success`, `changes.<path>.{type,unified_diff,content}` | extracted | Lines, edits, rework, and the tool error on a failed apply. `changes` is a type-discriminated union: an `update` carries `unified_diff`, a creation carries the whole file as `content`. Reading only the diff dropped every created file's lines — 37% of Codex's added lines on the audited corpus (`B119`, fixed in v0.13). |
| `response_item` → `type` = `function_call` / `custom_tool_call`, `name`, `status` | extracted | Tool-call count, purpose split and call failures. |
| `compacted` | extracted | `ai.compactions.count`. |
| `payload.info.total_token_usage.cache_write_input_tokens` | **skipped — worth extracting** | Codex reports a cache-write count and the parser has no field for it. On the audited corpus it was present on 238 `token_count` events and **zero on every one of them**, so nothing is currently mis-stated here — but a zero nobody read and a zero the vendor reported are different facts, and another plan or model may not share it (`B107`). |
| `payload.info.model_context_window` | **skipped — worth extracting** | The model's own context limit, on every `token_count`. `B16` proposes vendoring a context-window table; for Codex the log states it (`B114`). |
| `payload.rate_limits.{plan_type,primary.used_percent,primary.window_minutes,primary.resets_at,credits.*}` | **skipped — worth extracting** | How close the session ran to the plan's limit, and on which plan. The only place any source states a subscription's real constraint (`B114`). |
| `payload.info.last_token_usage.*` | skipped — redundant, kept as a check | The vendor's own per-turn figure. assaio differences the cumulative totals instead, which survives a missed line; this field is the cross-check that would prove that arithmetic on real data. |
| `payload.changes.<path>.move_path` | **skipped — worth extracting** | A rename is still an undifferentiated edit (`B113`). Its sibling `type` was on this row too, filed as a nice-to-have — and it was the field whose absence dropped a third of Codex's added lines. An audit that asks "is this field read?" cannot see that; that is what the calibration suite (`B137`) is for. |
| `turn_aborted` → `payload.reason` | **skipped — worth extracting** | A turn the human interrupted (`B113`). |
| `payload.thread_settings.reasoning_effort` | skipped — setting, not usage | A configuration value; it belongs to the harness inventory (`B95`) rather than to a usage record. |
| `world_state` → `payload.state.{model,permissions,personality,multi_agent_mode,host_skills,collaboration_mode,git_attribution}` | skipped — harness inventory | The agent's configuration at a point in time. Same reason (`B95`). |
| `payload.{content,input,output,arguments,text,summary,message,encrypted_content}`, `payload.action.{queries,url}`, `base_instructions.text`, `state.agents_md.text` | skipped — content, by construction | Prompts, completions, tool inputs and outputs, search queries, instruction files. |
| `payload.{id,call_id,turn_id,window_id,internal_chat_message_metadata_passthrough.turn_id}` | skipped — identity, no measure | Codex assigns its own turn id; assaio keys on a file fingerprint plus a positional counter, which is stable across a resumed session. Adopting the vendor's id would change a shipped dedupe contract for no new figure. |
| `payload.{approval_policy,approvals_reviewer,sandbox_policy.writable_roots,phase}` | skipped — undocumented | No published meaning, and each reads as a policy setting rather than an observation. |

## GitHub Copilot CLI

*Corpus: 3 sessions · 43 lines · 145 distinct key paths, plus `testdata/session.jsonl`. This is
the thinnest corpus of the five and the table should be read as provisional.*

| Field | State | Notes |
|---|---|---|
| `session.start` → `data.sessionId`, `data.context.cwd`, `timestamp` | extracted | Session identity, project and date. |
| `session.shutdown` → `data.modelMetrics.<model>.tokenDetails.{input,cache_read,cache_write,output}.tokenCount`, `.usage.reasoningTokens` | extracted | Per-model tokens and the cost estimate. |
| `session.shutdown` → `data.codeChanges.{linesAdded,linesRemoved}` | extracted | Session line counts, credited whole to the model with the most requests. |
| `data.toolRequests[].name`, `data.toolName` | **skipped — the depth row understates the source** | Copilot names its tool calls. Its matrix row says it carries no tool-call count, which was true of the parser and not of the log (`B109`). |
| `data.toolTelemetry.metrics.{linesAdded,linesRemoved}` | **skipped — the depth row understates the source** | Per-tool-call line counts. Today only the session total is read, which is why the whole session's changes are credited to one model (`B109`). |
| `data.modelMetrics.<model>.requests.count` | **skipped — worth extracting** | Requests per model — a turn count for a source the matrix says has none (`B109`). |
| `data.context.{branch,gitRoot,baseCommit,headCommit}` | **skipped — deliberate, pending a privacy decision** | The only source that records the commit its session started and ended at. That is attribution evidence of a quality no heuristic reaches (`B85`), and also exactly what the correlation threat model (`B100`) exists to decide about. It is not extracted ahead of that decision. |
| `data.modelCacheState[].{cacheTtlSeconds,cacheExpiresAt}` | **skipped — worth extracting** | A published cache TTL, which `cache-hygiene` states it cannot see (`B109`). |
| `data.parentAgentTaskId` | **skipped — worth extracting** | Sub-agent parentage, the delegation signal Claude Code has and Copilot's matrix row does not claim (`B109`). |
| `data.modelMetrics.<model>.{requests.cost,totalNanoAiu}` | skipped — a different unit | Copilot's own billing units. assaio recomputes cost from tokens for cross-tool consistency (the same decision as Cline's `cost`); this is useful only as an external check on the price table. |
| `data.{currentTokens,conversationTokens,systemTokens,toolDefinitionsTokens}` | skipped — undocumented composition | Context composition at a moment; how these overlap the billed counts is not documented, and adding them to a token total would double-count. |
| `data.{content,arguments,result,attachments,reasoningText,reasoningOpaque}`, `toolRequests[].{arguments,intentionSummary}`, `shellToolInfo.{displayCommand,possiblePaths}`, `codeChanges.filesModified` | skipped — content, by construction | Prompts, completions, tool arguments, command lines and file paths. |
| `data.{allowAllPermissions,previousAllowAllPermissions,remoteSteerable,copilotVersion,reasoningEffort,contextTier}` | skipped — harness inventory | Permission and version state (`B95`). |
| `data.{apiCallId,clientRequestId,requestId,serviceRequestId,interactionId,messageId,toolCallId}` | skipped — identity, no measure | Request correlation identifiers. |

## Gemini CLI

*Corpus: 355 files under `~/.gemini` · 2,045 lines · 27 distinct key paths — of which only **2
files** match the discovery glob, and neither contains a token field.*

| Field | State | Notes |
|---|---|---|
| `sessionId`, `timestamp`, `model`, `tokens.{input,output,cached,thoughts,tool,total}` | extracted | Every token signal, per the shape `testdata/session.jsonl` captures. |
| `type`, `source`, `status`, `step_index`, `content`, `thinking`, `error`, `error_code`, `truncated_fields`, `projectHash`, `startTime`, `workspace`, `$set.*` | **not classified — the corpus does not contain the parsed shape** | The chat files this install writes carry none of the token fields above. Either the recording moved, or these files were never the token source. Until that is settled the fields are not classified, because guessing which of two formats is current is exactly what this audit refuses to do (`B110`). |

The honest reading of that second row is a warning about assaio, not about Gemini: this source
produces **2 discovered files and 0 records** here, and no drift canary fires, because every
canary needs a sample floor (20 files) a two-file source can never reach. `B110` covers both
halves.

## Cline

*Corpus: no Cline install on the audited machine — `testdata/ui_messages.json` and
`testdata/task_metadata.json` only. This table is the weakest of the five and says so.*

| Field | State | Notes |
|---|---|---|
| `ui_messages[].{ts,type,say}` and the `api_req_started` payload's `{tokensIn,tokensOut,cacheReads,cacheWrites}` | extracted | Every token signal, per request. |
| `task_metadata.model_usage[].{ts,model_id}` | extracted | The model in force at each request, carried forward across a mid-task switch. |
| `api_req_started` payload `cost` | skipped — recomputed | Cline's own per-request cost. Record has no cost field by design; cost is computed from tokens for cross-tool consistency. Useful only as an external check on the price table. |
| `api_req_started` payload `request` | skipped — content, by construction | The prompt. |
| `task_metadata.model_usage[].{mode,model_provider_id}` | skipped — unverified against a real install | `mode` looks like Cline's plan/act distinction, which would be a genuine work-kind signal, and `model_provider_id` would separate the same model served by two providers. Neither is confirmed against a real corpus, so both stay skipped rather than being read from a fixture the project wrote itself. |
| `task_metadata.{files_in_context,environment_history}` | skipped — content and paths | File paths and environment snapshots. |
| `ui_messages[]` `say` = `tool` payloads | **skipped — the known activity gap** | The diffs that would give Cline line counts (`B39`). |

## What this audit is not

It does not claim the five formats have no other fields — only that these are the fields the
corpus above contains. Two of the five tables rest on a corpus too thin to be conclusive and
say so in their own heading. Widening that is the same work as widening the golden corpus
(`B20`), and the two should be re-run together.

---
