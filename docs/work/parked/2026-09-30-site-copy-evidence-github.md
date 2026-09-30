# Site copy for `evidence --github`, held for the release that ships it

The site describes the latest release and publishes on every push to `main`
(`.claude/rules/surfaces.md`), so these hand-written sentences wait for the release PR that ships
`evidence --github` (ADR 0022). They went through the GPT door with the rest of B92's text. Apply
them in that PR, then delete this file.

## `site/index.html`

Card "Local analysis runs offline", its `<p>` body:

> Local analysis needs no account and sends no telemetry or uploads. assaio itself uses the network only in <code>sync</code> and <code>serve</code>, and in <code>evidence --github</code>, which asks GitHub for a repository's pull requests through your own <code>gh</code>; a plugin you configure is your own program, and its network use depends on its code.

FAQ "Does assaio send my data anywhere?", its answer `<p>`:

> Local analysis runs offline with no account, telemetry, or upload. Only <code>sync</code> and <code>serve</code> use the network, for your self-hosted team server, and <code>evidence --github</code>, which asks GitHub for a repository's pull requests through your own <code>gh</code> and sends no commit or session. A plugin you configure is your own program; its network use depends on its code.

## `site/llms.txt`

Append to the `evidence` paragraph (after "never ranks people."):

> With `--github` it also reads the repository's pull requests through the user's own `gh` and names the pull requests each candidate commit belongs to; a pull request's state is never a session's outcome.

Replace "`evidence` uses local git and the local store only." with:

> `evidence` uses local git and the local store only, unless run with `--github`: then the user's own `gh` asks GitHub for the repository's pull requests.

Replace "assaio uses the network only for the self-hosted team server (`serve`/`sync`), which
connects only to operator-run infrastructure." with:

> assaio uses the network only for the self-hosted team server (`serve`/`sync`), which connects only to operator-run infrastructure, and for `evidence --github`, which reaches GitHub through the user's own `gh`.

Also re-read the sentence saying session-to-PR correlation remains on the roadmap: after the
release, the pull-request join has shipped; review, CI and merge correlation have not (`B229`).

## Handoff

- Done: text drafted and reviewed; the sentences are not on the site.
- Next: apply in the release PR for the version that ships `evidence --github`; delete this file.
