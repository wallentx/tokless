# Security Fix Plan - 2026-06-26

This note preserves the Codex Security scan results and the implementation plan so the work can continue safely after any context reset.

## Completed Implementation

Status after the completed fix set through `3400c6f`:

- `main` remains untouched at upstream commit `769cd6d` and tracks `upstream/main`.
- `dev` tracks `origin/dev` on the fork `wallentx/tokless`.
- GitHub fork default branch is `dev`.
- All ten reportable findings from the scan have code or process-control fixes committed on `dev`.

Completed commits:

1. `1446b72 docs: preserve security scan fix plan`
2. `c4145bf fix(agents): restore owned permission boundaries`
   - Covers `F-001`, `F-004`, `F-005`, and `F-007`.
3. `0d8e03c fix(exec): trust MCP and hook binaries`
   - Covers `F-002`, `F-003`, and `F-006`.
4. `d6a2d1e fix(npm): require trusted installer sources`
   - Covers `F-008`.
5. `c816326 fix(release): verify update artifacts`
   - Covers `F-009`.
6. `6081336 fix(bootstrap): verify dependency downloads`
   - Covers `F-010`.
7. `3400c6f docs(install): avoid raw pipe examples`
   - Follow-up documentation hardening for installer usage examples.

Verification completed:

- `env TMPDIR=/data/data/com.termux/files/usr/tmp go test ./...` passed.
- `bash -n scripts/install.sh scripts/build-release.sh` passed.
- Focused regression tests passed for agent permission boundaries, trusted binary resolution, npm registry and mutable installer gating, self-update checksum verification, and bootstrap checksum verification.
- Final source audit found remaining raw RTK fallback installers only inside the explicit `TOKLESS_ALLOW_UNVERIFIED_BOOTSTRAP=1` opt-in path.
- PowerShell runtime validation was not available in this Termux environment; `scripts/install.ps1` was reviewed statically.

## Repository State

- Upstream repository: `HoangP8/tokless`
- Fork repository: `wallentx/tokless`
- Local remotes:
  - `upstream`: `git@github.com:HoangP8/tokless.git`
  - `origin`: `git@github.com:wallentx/tokless.git`
- Local working branch for fixes: `dev`
- Fork default branch: `dev`
- `main` should remain untouched and tracking `upstream/main`.

## Scan Artifacts

- Scan directory: `/data/data/com.termux/files/usr/tmp/codex-security-scans/tokless/769cd6d_20260626T140044Z`
- Markdown report: `/data/data/com.termux/files/usr/tmp/codex-security-scans/tokless/769cd6d_20260626T140044Z/report.md`
- HTML report: `/data/data/com.termux/files/usr/tmp/codex-security-scans/tokless/769cd6d_20260626T140044Z/report.html`
- Validation included `go test ./...`, focused Go tests, an RTK hook PoC, a CodeGraph hook PoC, and a context-mode uninstall PoC.

## Findings To Fix

### High

1. `F-001-codex-approval-never`: Codex wiring globally sets `approval_policy = "never"` and uninstall does not restore it.
   - Primary files: `internal/agents/codex.go`, `internal/tools/contextmode.go`, `internal/tools/rtk.go`.
   - Fix invariant: tokless must not silently broaden the global Codex approval policy. If a policy change is required, it must be owned, reversible, and tested.

2. `F-003-rtk-hook-autoallow`: RTK hook executes PATH-selected `rtk`, accepts weak rewrite output, and emits `permissionDecision = "allow"`.
   - Primary files: `internal/commands/rtk_hook.go`, `internal/util/exec.go`, `internal/tools/rtk.go`.
   - Fix invariant: hook-time RTK execution must use a trusted/pinned binary path, and rewrites must not bypass approval unless the command is strongly constrained.

### Medium

3. `F-002-path-mcp-trust`: PATH-selected MCP executables are persisted into trusted or enabled agent configs.
   - Primary files: `internal/util/mcpspawn.go`, `internal/util/exec.go`, `internal/agents/*.go`.
   - Fix invariant: persisted MCP commands should resolve from trusted install locations or package-managed invocations, not arbitrary first PATH hit.

4. `F-004-claude-stale-mcp-allow`: Claude uninstall leaves `mcp__<tool>__.*` allow entries behind.
   - Primary file: `internal/agents/claude.go`.
   - Fix invariant: uninstall removes only tokless-owned allow entries and preserves unrelated user entries.

5. `F-005-antigravity-command-rtk`: Antigravity MCP setup grants persistent `command(rtk)` approval unrelated to selected MCP.
   - Primary file: `internal/agents/antigravity.go`.
   - Fix invariant: `command(rtk)` is owned by RTK hook setup only and is removed with RTK hook teardown.

6. `F-006-codegraph-hook-index`: CodeGraph auto-index hook trusts PATH-selected `codegraph` and hook-supplied `workspacePaths[0]`.
   - Primary file: `internal/commands/index.go`.
   - Fix invariant: hook code should execute a trusted CodeGraph binary and validate the workspace path against a real project boundary.

7. `F-007-contextmode-gemini-delete`: Antigravity context-mode uninstall deletes project-local `GEMINI.md` by marker substring.
   - Primary file: `internal/tools/contextmode.go`.
   - Fix invariant: uninstall removes only tokless-owned fenced content; it must not delete mixed-content project instruction files.

8. `F-008-npm-registry-npx`: npm/npx installer flows can be redirected through registry config or mutable package refs.
   - Primary files: `internal/util/npminstall.go`, `internal/tools/caveman.go`.
   - Fix invariant: known tool installs should default to trusted registry/source and require explicit opt-in for non-default registries or mutable refs.

9. `F-009-installer-release-integrity`: Install and self-update paths rely on mutable unsigned latest assets/actions.
   - Primary files: `scripts/install.sh`, `scripts/install.ps1`, `internal/commands/selfupdate.go`, `.github/workflows/release.yml`.
   - Fix invariant: release artifacts and installer scripts should have verifiable integrity and CI actions should be pinned or otherwise constrained.

10. `F-010-bootstrap-archive-integrity`: Node/Git/RTK bootstrap downloads trusted binaries without independent integrity checks.
    - Primary files: `internal/util/nodewin.go`, `internal/util/nodelinux.go`, `internal/util/gitwin.go`, `internal/tools/rtk.go`.
    - Fix invariant: dependency bootstrap must verify pinned checksums/signatures or require an explicit unsafe opt-in when verification is unavailable.

## Proposed Upstream Grouping

These should not be one large upstream PR. The safest grouping is four PR-sized areas, with at least one commit per finding or tightly coupled finding pair:

1. Agent permission cleanup and reversible uninstall
   - Fix `F-001`, `F-004`, `F-005`, and `F-007`.
   - Rationale: all are owned-config and uninstall-symmetry problems, with focused tests that do not need network.

2. Trusted binary resolution for MCP and hooks
   - Fix `F-002`, `F-003`, and `F-006`.
   - Rationale: these share PATH trust, hook-time execution, and persisted command selection. This PR should introduce one reusable policy for trusted executable resolution.

3. npm and tool installer source hardening
   - Fix `F-008`.
   - Rationale: npm registry behavior and mutable `npx` flows are separate from agent config semantics and need installer UX choices.

4. Release and dependency bootstrap integrity
   - Fix `F-009` and `F-010`.
   - Rationale: installer/self-update/release provenance and Node/Git/RTK bootstrap integrity are supply-chain work and may need release-process decisions.

If upstream prefers smaller patches, split each numbered finding into its own PR. Locally, keep changes separated by commit even if several commits later become one PR.

## Suggested Commit Order

1. `docs: preserve security scan fix plan`
2. `fix(codex): stop forcing global approval policy`
3. `fix(agents): clean owned Claude and Antigravity permissions`
4. `fix(contextmode): preserve project GEMINI instructions on uninstall`
5. `fix(exec): harden MCP and hook binary resolution`
6. `fix(rtk): remove hook auto-allow for untrusted rewrites`
7. `fix(codegraph): validate hook workspace paths`
8. `fix(npm): constrain registry and mutable npx installers`
9. `fix(release): add install and self-update integrity controls`
10. `fix(bootstrap): verify downloaded dependency artifacts`

## Verification Checklist

- `go test ./...`
- Focused tests in `internal/agents`, `internal/commands`, `internal/tools`, and `internal/util`.
- Re-run the original RTK hook PoC and confirm it no longer emits `permissionDecision = "allow"` for fake PATH `rtk`.
- Re-run the CodeGraph hook PoC and confirm fake PATH `codegraph` is not executed and untrusted `workspacePaths[0]` is rejected or ignored.
- Re-run the context-mode uninstall PoC and confirm mixed-content project `GEMINI.md` survives.
- Verify `git status --short --branch` before each commit.
