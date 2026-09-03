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

*Corpus: 2,625 rollouts · 70,522 lines, taken 2026-09-03, plus `testdata/rollout.jsonl`.
Re-audited in v0.26 against a current install; the first pass read 21 rollouts and predates three
of the rows below.*

| Field | State | Notes |
|---|---|---|
| `timestamp`, `type`, `payload.type` | extracted | Line routing and every record's own timestamp. |
| `payload` of `session_meta`: `id`, `cwd`, `model`, `timestamp` | extracted | Session identity and dimensions. |
| `session_meta.source` | extracted | How the run was started: `exec` for a scripted `codex exec` (2,604 rollouts), `cli` for the terminal UI (13), `vscode` for the desktop app (1). It is what the scope vocabulary reads a sequence under, and without it every Codex sequence is `unstated` and excluded from every behaviour detector. **The field is a union**: a bare string on 2,618 rollouts and an object — `{"subagent":"review"}` — on 7, where Codex ran the session as a sub-agent of another. Typing it as a string made those 7 `session_meta` lines fail to unmarshal outright, costing the whole rollout its session id, project, model and cwd. A field read too confidently is a worse outcome than a field not read at all. |
| `response_item` → `type` = `reasoning` / `message` (+`role`) | extracted | Where a turn's model output begins, which is where its assistant step goes. `role` is what separates it from the turn's input: the harness's `developer` messages and the person's `user` messages outnumber `assistant` roughly three to one on this corpus, and opening a turn on one would place the model's response before the work it describes. |
| `response_item` → `call_id` | extracted | A tool call's identity in its own sequence, stable across a re-read where a positional counter is only stable while the file ahead of it does not change. |
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
| `event_msg/item_completed` → `payload.item.{type,status,id}`, `started_at_ms`, `completed_at_ms` | **skipped — measured, and deliberately not read** | A second view of the same turn, added by Codex around 0.148 and present in **1,614 of 1,614 September rollouts** on this corpus, 833 of 1,001 August ones and none of the 10 July ones (first seen 2026-08-20). It carries a per-step wall-clock duration no other source publishes, and a `CommandExecution` status with a real `exit_code` — 57 of 840 commands exited non-zero under a call whose own `status` says `completed`. It is not read because it cannot be joined: its `exec-<uuid>` ids match no `call_id`, and it covers only a subset of calls (28 against 74 in one 10-file sample), so pairing it positionally would be a guess. Reading it as a second step source instead would count one call twice. This row is why `assaio` now names **additive drift** as its third silent failure mode ([format-resilience.md](../format-resilience.md)). |
| `event_msg/context_compacted` | **skipped — the same event under a second name** | Newer builds write it beside the `compacted` line the compaction count already reads: present in 14 of the 18 rollouts that compacted at all, one for one, and in none that lack `compacted`. Reading both would report one overflow as two. |
| `payload.info.total_token_usage.cache_write_input_tokens` (re-measured) | **skipped — worth extracting** | Still worth reading, and the corpus is now larger: of the 5,809 `token_count` events carrying an `info` block, 518 omit the field and 5,291 report it as **zero — with no event anywhere reporting anything else**. The `B107` reasoning is unchanged — a zero nobody read and a zero the vendor reported are different facts — and the field's constancy across 2,625 rollouts is now evidence rather than a small sample. |
| `payload.thread_settings.reasoning_effort` | skipped — setting, not usage | A configuration value; it belongs to the harness inventory (`B95`) rather than to a usage record. |
| `world_state` → `payload.state.{model,permissions,personality,multi_agent_mode,host_skills,collaboration_mode,git_attribution}` | skipped — harness inventory | The agent's configuration at a point in time. Same reason (`B95`). |
| `payload.{content,input,output,arguments,text,summary,message,encrypted_content}`, `payload.action.{queries,url}`, `base_instructions.text`, `state.agents_md.text` | skipped — content, by construction | Prompts, completions, tool inputs and outputs, search queries, instruction files. |
| `payload.{id,turn_id,window_id,internal_chat_message_metadata_passthrough.turn_id}` | skipped — identity, no measure | Codex assigns its own turn id; assaio keys a record on a file fingerprint plus a positional counter, which is stable across a resumed session. Adopting the vendor's id would change a shipped dedupe contract for no new figure. `call_id` is the exception and is now read, above: a step's identity is not a record's. |
| `session_meta.{originator,thread_source,parent_thread_id,cli_version}` | skipped — restates or names what is not modelled | `originator` names the same three starting points `source` does, in display words (`codex_exec`, `codex-tui`, `Codex Desktop`); reading both would be two spellings of one fact. `thread_source: "subagent"` and `parent_thread_id` say a rollout ran as another's sub-agent — real, and 7 of 2,507 here — but a sub-agent needs a timeline of its own before it can be told apart from its parent, which this reading does not give it. |
| `session_meta.git.{branch,commit_hash,repository_url}` | **skipped — deliberate, pending a privacy decision** | The commit a session started at. Same standing as Copilot's `data.context.baseCommit` below and blocked on the same decision (`B85`, `B100`), noted here because the first audit did not record that Codex carries it too. |
| `payload.{approval_policy,approvals_reviewer,sandbox_policy.writable_roots,phase}` | skipped — undocumented | No published meaning, and each reads as a policy setting rather than an observation. |

## GitHub Copilot CLI

*Corpus: 3 sessions · 43 lines · 145 distinct key paths, plus `testdata/session.jsonl`. This is
the thinnest corpus of the six and the table should be read as provisional.*

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
`testdata/task_metadata.json` only. This table is the weakest of the six and says so.*

| Field | State | Notes |
|---|---|---|
| `ui_messages[].{ts,type,say}` and the `api_req_started` payload's `{tokensIn,tokensOut,cacheReads,cacheWrites}` | extracted | Every token signal, per request. |
| `task_metadata.model_usage[].{ts,model_id}` | extracted | The model in force at each request, carried forward across a mid-task switch. |
| `api_req_started` payload `cost` | skipped — recomputed | Cline's own per-request cost. Record has no cost field by design; cost is computed from tokens for cross-tool consistency. Useful only as an external check on the price table. |
| `api_req_started` payload `request` | skipped — content, by construction | The prompt. |
| `task_metadata.model_usage[].{mode,model_provider_id}` | skipped — unverified against a real install | `mode` looks like Cline's plan/act distinction, which would be a genuine work-kind signal, and `model_provider_id` would separate the same model served by two providers. Neither is confirmed against a real corpus, so both stay skipped rather than being read from a fixture the project wrote itself. |
| `task_metadata.{files_in_context,environment_history}` | skipped — content and paths | File paths and environment snapshots. |
| `ui_messages[]` `say` = `tool` payloads | **skipped — the known activity gap** | The diffs that would give Cline line counts (`B39`). |

## Antigravity CLI (`agy`)

*Corpus: 500 conversations · 1,537 transcript lines · 9 distinct key paths, plus 500
`conversations/<uuid>.db` SQLite databases holding 1,975 steps and 638 `gen_metadata` blobs.
Antigravity CLI 1.1.23, captured 2026-09-02.*

The one source with a rich local corpus and no token counter in it. The audit below is why the
depth matrix answers no token signal for it rather than reporting zeros: the search for a
counter was exhaustive and is recorded here so nobody repeats it.

| Field | State | Notes |
|---|---|---|
| `transcript.jsonl` → `step_index`, `source`, `created_at` | extracted | Record identity, the dedupe key, and the timestamp every window is bounded by. `source` = `MODEL` is what makes an entry a model *entry*; `USER_EXPLICIT` and `SYSTEM` are filtered. A **turn** is every consecutive `MODEL` entry sharing one `created_at`, because the vendor writes one response as a `GENERIC` half and a `PLANNER_RESPONSE` half — 8 of 653 entries in the reference capture. The grouping is on the timestamp, not on `type`: there is no response id, so two genuinely distinct responses inside one second read as one turn, which under-counts turns rather than inflating every per-turn figure. |
| `transcript.jsonl` → `tool_calls[].name` | extracted | `ai.tool_calls.count`, `ai.edits.count` and the purpose split, through the shared tool-class allowlist. 26 calls across the corpus: `run_command` ×14, `write_to_file` ×9, `finish` ×3. |
| the conversation directory name | extracted | The session id. Nothing inside the transcript identifies the conversation, which is why this parser takes a directory rather than a reader. |
| `conversations/<uuid>.db` → `gen_metadata.data` field path `1.9.10.1` | **skipped — a real figure, and not a billable one** | Context tokens occupied at that step. Monotonic across steps in 198/198 conversations with more than one row, and `used ≤ window` in 638/638 observations, so the reading is settled. It is not input, output or cache, which is what a cost model needs, and `store.SessionRow.PeakContextTokens` is *derived* on every other source (`cache_read + input`) where this would be *observed* — two different measurements in one column. `B194` carries the evidence. |
| `conversations/<uuid>.db` → `gen_metadata.data` field path `1.9.10.4` | skipped — a model property, not usage | The model's context window in tokens: exactly {80000, 128000, 160000, 256000}, pairing with the model name in the same blob 218/218 without exception. It is the ceiling that proved the field above is denominated in tokens; it says nothing about what a session did. |
| `conversations/<uuid>.db` → `gen_metadata.data` field paths `3.28` / `1.19` | **skipped — present, and not attributable** | The model name (`gemini-3.7-flash-high` ×208, `gemini-3.7-flash` ×45, `claude-opus-4-6-thinking` ×52, `gpt-oss-120b-medium`, `claude-sonnet-4-6`, `gemini-pro-agent`). Reading it means opening another tool's live WAL-mode SQLite database and walking an unnamed protobuf field, for a name that covers 218 of 500 conversations and is frequently several per conversation with no request count to choose between them. `B194`. |
| `conversations/<uuid>.db` → `steps.has_subtrajectory` | skipped — unverifiable, not absent | False on all 1,975 steps of the corpus. A column uniformly false across a whole capture cannot separate "no sub-agent ran" from "this build never sets it", so the matrix claims delegation neither way. |
| `conversations/<uuid>.db` → `steps.{step_type,status}` | skipped — undocumented vocabulary | Integers with no published meaning: `step_type` ∈ {17 ×822, 15 ×642, 14 ×500, 132 ×26}, `status` = 3 on 1,975 of 1,990 steps. Naming them would be guessing from a number. |
| `transcript.jsonl` → `type` = `ERROR_MESSAGE` (384 lines) | **skipped — a different question from the one the signal asks** | A stream or platform failure, not a tool call returning an error. Counting it under `ai.tool_errors.count` would answer "did a tool fail" with "did the connection drop". A session-level failure signal does not exist in the catalog; `B194` names it. |
| `transcript.jsonl` → `type`, `status` | skipped — no variation to read | `status` is `DONE` on all 1,537 lines and `type` takes four values, three of which `source` already separates. A constant is not a signal. |
| `transcript.jsonl` → `content`, `thinking`, `tool_calls[].args` | skipped — content, by construction | The prompt, the model's own reasoning, and every tool argument including the file a write targets. This source is the first whose accounting sits inside the same lines as its content, so the omission is structural: the decoded struct has no field for any of them, and a test asserts a sentinel planted in each never reaches a record, an error or a log. |
| `transcript.jsonl` → `truncated_fields` | skipped — names a content field | Lists which content fields the vendor shortened. Useful only to a reader of the content. |
| `transcript_full.jsonl`, `logs/chunks/**` | skipped — the same entries, less redacted | The untruncated copy of the same 1,537 lines and its chunked form: strictly more prompt text for no additional accounting. |
| `annotations/<uuid>.pbtxt` | skipped — content | One field, `title`, holding a model-written summary of the conversation. |
| `conversation_summaries.db` | **skipped — stale on this machine** | Would answer the project question: `workspace_uris`, `step_count`, `parent_conversation_id`, `nesting_depth`, `agent_name`. It holds 8 rows dated 2025-11-20 to 2026-01-02, from the IDE era, against 500 CLI conversations from 2026-08-26 onward. The CLI does not appear to write it, so no project can be resolved and every `agy` session lands unattributed. |
| `implicit/*.pb`, `presence/*.lock`, `jetski_state.pbtxt`, `settings.json`, `installation_id`, `updater/**` | skipped — harness inventory | Runtime locks, install identity and configuration (`B95`). |

## What this audit is not

It does not claim the six formats have no other fields — only that these are the fields the
corpus above contains. Two of the six tables rest on a corpus too thin to be conclusive and
say so in their own heading. Widening that is the same work as widening the golden corpus
(`B20`), and the two should be re-run together.

---
