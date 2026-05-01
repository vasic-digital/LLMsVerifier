# LLMsVerifier Development Guidelines

## MANDATORY: No CI/CD Pipelines

**NO GitHub Actions, GitLab CI/CD, or any automated pipeline may exist in this repository!**

- No `.github/workflows/` directory
- No `.gitlab-ci.yml` file
- No Jenkinsfile, .travis.yml, .circleci, or any other CI configuration
- All builds and tests are run manually or via Makefile targets
- This rule is permanent and non-negotiable

Auto-generated from all feature plans. Last updated: 2025-12-28

## Active Technologies

- Go 1.21+ + Existing LLM Verifier codebase, HTTP client libraries (001-extend-llm-providers)
- Python 3.8+ for security and export tools
- SQLite with SQL Cipher for encrypted databases

## Project Structure

```text
src/
tests/
scripts/          # Security and export utilities
challenges/       # Challenge framework and results
docs/             # Documentation
data/             # Test data and fixtures
```

## 🔐 Security-Critical Commands

### Export OpenCode Configuration (Secure)

```bash
# Standard export - generates VALID OpenCode configuration with embedded API keys
# File is automatically protected with 600 permissions and gitignore rules
python3 scripts/export_opencode_config_fixed.py

# Custom output location
python3 scripts/export_opencode_config_fixed.py --output /path/to/secure/location/

# Validate gitignore protections (recommended before export)
python3 scripts/export_opencode_config_fixed.py --validate-only
```

**Security Features:**
- ✅ Automatic 600 file permissions (owner read/write only)
- ✅ Gitignore protection verification
- ✅ Comprehensive security warnings displayed
- ✅ All API keys embedded from `.env` file
- ✅ Feature detection (MCP, LSP, ACP, Embeddings)
- ✅ Performance metrics and scoring included

**Output:** `~/Downloads/opencode_{timestamp}.json` (52-60 KB typical)

### Run Model Verification Challenge

```bash
make build
cd llm-verifier
../bin/llm-verifier run_full_verification_fixed
```

**What it does:**
- Discovers all providers with API keys
- Tests 40+ models with real HTTP requests
- Verifies features: Streaming, Tool Calling, Vision, ACP, LSP, MCP
- Generates scores (0-100 scale)
- Saves results to database and JSON exports

**Results Location:**
`challenges/full_verification/{year}/{month}/{day}/{timestamp}/results/`

### Run All Challenges

```bash
cd llm-verifier
./run_challenges
```

## 🛡️ Security Requirements

### Configuration Export Rules

1. **ALWAYS use the official export script:**
   ```bash
   python3 scripts/export_opencode_config_fixed.py
   ```

2. **NEVER commit exported configurations:**
   - Files contain embedded API keys
   - Protected by `.gitignore` (lines 180-220)
   - Manual check: `git check-ignore opencode*.json`

3. **Verify permissions after export:**
   ```bash
   ls -la ~/Downloads/opencode*.json  # Should show -rw-------
   ```

4. **Rotate API keys if exposed:**
   - Immediately regenerate on provider dashboards
   - Update `.env` file
   - Re-run export script

### Gitignore Protection

The `.gitignore` file protects:
- `opencode.json` (root level)
- `opencode_*_with_keys.json` (pattern)
- `Downloads/opencode*.json` (user Downloads)
- `**/*api_key*` (any API key files)
- `**/*secret*` (any secret files)
- `**/*.env` (environment files)

**Validation:**
```bash
python3 scripts/export_opencode_config_fixed.py --validate-only
```

## 📖 Code Style

### Go Code
- Follow standard Go conventions
- Use `gofmt` for formatting
- Minimum Go version: 1.21
- Error handling: explicit, wrapped errors

### Python Scripts
- Follow PEP 8
- Use type hints where applicable
- Security-first: validate inputs, sanitize outputs
- Use `pathlib` for file operations

## 🏗️ Architecture Patterns

### Security Pattern: Secure Exports

When creating export functionality:

1. **Always validate gitignore:**
   ```python
   if not gitignore_has_protection():
       raise SecurityError("Gitignore protections missing")
   ```

2. **Always set restrictive permissions:**
   ```python
   os.chmod(output_file, 0o600)  # Owner read/write only
   ```

3. **Always include security warnings:**
   ```python
   config = {
       "security_warning": "CONTAINS API KEYS - DO NOT COMMIT",
       "safe_to_commit": False
   }
   ```

4. **Never embed keys without warnings:**
   - Display warnings during export
   - Include warnings in generated files
   - Document in generated files

### Feature Detection Pattern

When testing model capabilities:

```python
# Features to test
features = {
    "streaming": test_streaming(...),
    "tool_calling": test_tool_calling(...),
    "embeddings": test_embeddings(...),
    "vision": test_vision(...),
    "mcp": test_mcp_support(...),
    "lsp": test_lsp_support(...),
    "acp": test_acp_support(...)
}

# Validate with real HTTP requests (not just config)
# Score based on actual performance, not claims
```

### Scoring Pattern

Scoring algorithm:
```
Overall Score = 
  (Responsiveness × 0.30) +
  (Code Capability × 0.25) +
  (Feature Richness × 0.25) +
  (Reliability × 0.20)
```

Where:
- Responsiveness: 0-30 points (based on response time)
- Code Capability: 0-25 points (based on features + tests)
- Feature Richness: 0-25 points (count of supported features)
- Reliability: 0-20 points (verification status)

## 📋 Recent Changes

### 2025-12-28: Security Enhancement (v2.0-ultimate)

**Added:**
- `scripts/export_opencode_config_fixed.py` - Secure export tool with:
  - Automatic 600 permissions
  - Gitignore validation
  - Security warnings
  - Comprehensive feature detection
  - Performance metrics
  
- Enhanced `.gitignore` with expanded security patterns
  - Protects all `*api_key*` files
  - Protects all `*secret*` files
  - Protects `opencode_*_with_keys.json` patterns
  - Protects user Downloads directory exports

- Documentation: `SECURITY_CONFIGURATION_EXPORT.md`
  - Complete security guidelines
  - Incident response procedures
  - Pre-flight checklists
  - Usage examples

**What Gets Exported:**
- All 25 providers with API keys
- All 40+ models with verification status
- MCP/LSP/ACP feature detection
- Performance metrics (response time, TTFT)
- Comprehensive scoring (0-100 scale)
- MCP server configurations
- Model groups for easy selection

**Verified Models:**
- OpenRouter GPT-4: Score 80/100 ⭐
- OpenRouter Claude 3.5: Score 80/100 ⭐
- DeepSeek Chat: Score 73/100 ⭐

### 2025-12-24: Challenge Framework v1.0

**Added:**
- Challenge runner system
- Model verification with real HTTP tests
- Database schema for results storage
- Initial export functionality

## 🔧 Development Commands

```bash
# Setup
cd /media/milosvasic/DATA4TB/Projects/LLM/LLMsVerifier

# Run tests
make test

# Format code
make format

# Lint
go vet ./...
golangci-lint run

# Build
make build

# Run verifier
cd llm-verifier
./llm-verifier

# Export configuration
python3 scripts/export_opencode_config_fixed.py
```

## 📚 Documentation

- **Security Guide:** `SECURITY_CONFIGURATION_EXPORT.md`
- **Challenge Framework:** `challenges/docs/CHALLENGE_FRAMEWORK.md`
- **API Documentation:** `docs/API_DOCUMENTATION.md`
- **Database Schema:** `llm-verifier/database/schema.sql`

## 🆘 Security Contacts

If you discover security issues with exports or API key handling:

1. **DO NOT file public GitHub issues**
2. Check `SECURITY.md` for private reporting process
3. Include: Description, severity, reproduction steps
4. Expected response: Within 24 hours

---

**Last Updated:** 2025-12-28  
**Export Version:** 2.0-ultimate  
**Security Level:** Maximum (600 permissions, gitignore protected, embedded warnings)

## Universal Mandatory Constraints

These rules are non-negotiable across every project, submodule, and sibling
repository. They are derived from the HelixAgent root `CLAUDE.md`. Each
project MUST surface them in its own `CLAUDE.md`, `AGENTS.md`, and
`CONSTITUTION.md`. Project-specific addenda are welcome but cannot weaken
or override these.

### Hard Stops (permanent, non-negotiable)

1. **NO CI/CD pipelines.** No `.github/workflows/`, `.gitlab-ci.yml`,
   `Jenkinsfile`, `.travis.yml`, `.circleci/`, or any automated pipeline.
   No Git hooks either. All builds and tests run manually or via Makefile/
   script targets.
2. **NO HTTPS for Git.** SSH URLs only (`git@github.com:…`,
   `git@gitlab.com:…`, etc.) for clones, fetches, pushes, and submodule
   updates. Including for public repos. SSH keys are configured on every
   service.
3. **NO manual container commands.** Container orchestration is owned by
   the project's binary/orchestrator (e.g. `make build` → `./bin/<app>`).
   Direct `docker`/`podman start|stop|rm` and `docker-compose up|down`
   are prohibited as workflows. The orchestrator reads its configured
   `.env` and brings up everything.

### Mandatory Development Standards

1. **100% Test Coverage.** Every component MUST have unit, integration,
   E2E, automation, security/penetration, and benchmark tests. No false
   positives. Mocks/stubs ONLY in unit tests; all other test types use
   real data and live services.
2. **Challenge Coverage.** Every component MUST have Challenge scripts
   (`./challenges/scripts/`) validating real-life use cases. No false
   success — validate actual behavior, not return codes.
3. **Real Data.** Beyond unit tests, all components MUST use actual API
   calls, real databases, live services. No simulated success. Fallback
   chains tested with actual failures.
4. **Health & Observability.** Every service MUST expose health
   endpoints. Circuit breakers for all external dependencies. Prometheus
   / OpenTelemetry integration where applicable.
5. **Documentation & Quality.** Update `CLAUDE.md`, `AGENTS.md`, and
   relevant docs alongside code changes. Pass language-appropriate
   format/lint/security gates. Conventional Commits:
   `<type>(<scope>): <description>`.
6. **Validation Before Release.** Pass the project's full validation
   suite (`make ci-validate-all`-equivalent) plus all challenges
   (`./challenges/scripts/run_all_challenges.sh`).
7. **No Mocks or Stubs in Production.** Mocks, stubs, fakes, placeholder
   classes, TODO implementations are STRICTLY FORBIDDEN in production
   code. All production code is fully functional with real integrations.
   Only unit tests may use mocks/stubs.
8. **Comprehensive Verification.** Every fix MUST be verified from all
   angles: runtime testing (actual HTTP requests / real CLI invocations),
   compile verification, code structure checks, dependency existence
   checks, backward compatibility, and no false positives in tests or
   challenges. Grep-only validation is NEVER sufficient.
9. **Resource Limits for Tests & Challenges (CRITICAL).** ALL test and
   challenge execution MUST be strictly limited to 30-40% of host system
   resources. Use `GOMAXPROCS=2`, `nice -n 19`, `ionice -c 3`, `-p 1`
   for `go test`. Container limits required. The host runs
   mission-critical processes — exceeding limits causes system crashes.
10. **Bugfix Documentation.** All bug fixes MUST be documented in
    `docs/issues/fixed/BUGFIXES.md` (or the project's equivalent) with
    root cause analysis, affected files, fix description, and a link to
    the verification test/challenge.
11. **Real Infrastructure for All Non-Unit Tests.** Mocks/fakes/stubs/
    placeholders MAY be used ONLY in unit tests (files ending `_test.go`
    run under `go test -short`, equivalent for other languages). ALL
    other test types — integration, E2E, functional, security, stress,
    chaos, challenge, benchmark, runtime verification — MUST execute
    against the REAL running system with REAL containers, REAL
    databases, REAL services, and REAL HTTP calls. Non-unit tests that
    cannot connect to real services MUST skip (not fail).
12. **Reproduction-Before-Fix (CONST-032 — MANDATORY).** Every reported
    error, defect, or unexpected behavior MUST be reproduced by a
    Challenge script BEFORE any fix is attempted. Sequence:
    (1) Write the Challenge first. (2) Run it; confirm fail (it
    reproduces the bug). (3) Then write the fix. (4) Re-run; confirm
    pass. (5) Commit Challenge + fix together. The Challenge becomes
    the regression guard for that bug forever.
13. **Concurrent-Safe Containers (Go-specific, where applicable).** Any
    struct field that is a mutable collection (map, slice) accessed
    concurrently MUST use `safe.Store[K,V]` / `safe.Slice[T]` from
    `digital.vasic.concurrency/pkg/safe` (or the project's equivalent
    primitives). Bare `sync.Mutex + map/slice` combinations are
    prohibited for new code.

### Definition of Done (universal)

A change is NOT done because code compiles and tests pass. "Done"
requires pasted terminal output from a real run, produced in the same
session as the change.

- **No self-certification.** Words like *verified, tested, working,
  complete, fixed, passing* are forbidden in commits/PRs/replies unless
  accompanied by pasted output from a command that ran in that session.
- **Demo before code.** Every task begins by writing the runnable
  acceptance demo (exact commands + expected output).
- **Real system, every time.** Demos run against real artifacts.
- **Skips are loud.** `t.Skip` / `@Ignore` / `xit` / `describe.skip`
  without a trailing `SKIP-OK: #<ticket>` comment break validation.
- **Evidence in the PR.** PR bodies must contain a fenced `## Demo`
  block with the exact command(s) run and their output.

<!-- BEGIN host-power-management addendum (CONST-033) -->

## Host Power Management — Hard Ban (CONST-033)

**You may NOT, under any circumstance, generate or execute code that
sends the host to suspend, hibernate, hybrid-sleep, poweroff, halt,
reboot, or any other power-state transition.** This rule applies to:

- Every shell command you run via the Bash tool.
- Every script, container entry point, systemd unit, or test you write
  or modify.
- Every CLI suggestion, snippet, or example you emit.

**Forbidden invocations** (non-exhaustive — see CONST-033 in
`CONSTITUTION.md` for the full list):

- `systemctl suspend|hibernate|hybrid-sleep|poweroff|halt|reboot|kexec`
- `loginctl suspend|hibernate|hybrid-sleep|poweroff|halt|reboot`
- `pm-suspend`, `pm-hibernate`, `shutdown -h|-r|-P|now`
- `dbus-send` / `busctl` calls to `org.freedesktop.login1.Manager.Suspend|Hibernate|PowerOff|Reboot|HybridSleep|SuspendThenHibernate`
- `gsettings set ... sleep-inactive-{ac,battery}-type` to anything but `'nothing'` or `'blank'`

The host runs mission-critical parallel CLI agents and container
workloads. Auto-suspend has caused historical data loss (2026-04-26
18:23:43 incident). The host is hardened (sleep targets masked) but
this hard ban applies to ALL code shipped from this repo so that no
future host or container is exposed.

**Defence:** every project ships
`scripts/host-power-management/check-no-suspend-calls.sh` (static
scanner) and
`challenges/scripts/no_suspend_calls_challenge.sh` (challenge wrapper).
Both MUST be wired into the project's CI / `run_all_challenges.sh`.

**Full background:** `docs/HOST_POWER_MANAGEMENT.md` and `CONSTITUTION.md` (CONST-033).

<!-- END host-power-management addendum (CONST-033) -->



<!-- CONST-035 anti-bluff addendum (cascaded) -->

## CONST-035 — Anti-Bluff Tests & Challenges (mandatory; inherits from root)

Tests and Challenges in this submodule MUST verify the product, not
the LLM's mental model of the product. A test that passes when the
feature is broken is worse than a missing test — it gives false
confidence and lets defects ship to users. Functional probes at the
protocol layer are mandatory:

- TCP-open is the FLOOR, not the ceiling. Postgres → execute
  `SELECT 1`. Redis → `PING` returns `PONG`. ChromaDB → `GET
  /api/v1/heartbeat` returns 200. MCP server → TCP connect + valid
  JSON-RPC handshake. HTTP gateway → real request, real response,
  non-empty body.
- Container `Up` is NOT application healthy. A `docker/podman ps`
  `Up` status only means PID 1 is running; the application may be
  crash-looping internally.
- No mocks/fakes outside unit tests (already CONST-030; CONST-035
  raises the cost of a mock-driven false pass to the same severity
  as a regression).
- Re-verify after every change. Don't assume a previously-passing
  test still verifies the same scope after a refactor.
- Verification of CONST-035 itself: deliberately break the feature
  (e.g. `kill <service>`, swap a password). The test MUST fail. If
  it still passes, the test is non-conformant and MUST be tightened.

## CONST-033 clarification — distinguishing host events from sluggishness

Heavy container builds (BuildKit pulling many GB of layers, parallel
podman/docker compose-up across many services) can make the host
**appear** unresponsive — high load average, slow SSH, watchers
timing out. **This is NOT a CONST-033 violation.** Suspend / hibernate
/ logout are categorically different events. Distinguish via:

- `uptime` — recent boot? if so, the host actually rebooted.
- `loginctl list-sessions` — session(s) still active? if yes, no logout.
- `journalctl ... | grep -i 'will suspend\|hibernate'` — zero broadcasts
  since the CONST-033 fix means no suspend ever happened.
- `dmesg | grep -i 'killed process\|out of memory'` — OOM kills are
  also NOT host-power events; they're memory-pressure-induced and
  require their own separate fix (lower per-container memory limits,
  reduce parallelism).

A sluggish host under build pressure recovers when the build finishes;
a suspended host requires explicit unsuspend (and CONST-033 should
make that impossible by hardening `IdleAction=ignore` +
`HandleSuspendKey=ignore` + masked `sleep.target`,
`suspend.target`, `hibernate.target`, `hybrid-sleep.target`).

If you observe what looks like a suspend during heavy builds, the
correct first action is **not** "edit CONST-033" but `bash
challenges/scripts/host_no_auto_suspend_challenge.sh` to confirm the
hardening is intact. If hardening is intact AND no suspend
broadcast appears in journal, the perceived event was build-pressure
sluggishness, not a power transition.

<!-- BEGIN no-session-termination addendum (CONST-036) -->

## User-Session Termination — Hard Ban (CONST-036)

**You may NOT, under any circumstance, generate or execute code that
ends the currently-logged-in user's desktop session, kills their
`user@<UID>.service` user manager, or indirectly forces them to
manually log out / power off.** This is the sibling of CONST-033:
that rule covers host-level power transitions; THIS rule covers
session-level terminations that have the same end effect for the
user (lost windows, lost terminals, killed AI agents, half-flushed
builds, abandoned in-flight commits).

**Why this rule exists.** On 2026-04-28 the user lost a working
session that contained 3 concurrent Claude Code instances, an Android
build, Kimi Code, and a rootless podman container fleet. The
`user.slice` consumed 60.6 GiB peak / 5.2 GiB swap, the GUI became
unresponsive, the user was forced to log out and then power off via
the GNOME shell. The host could not auto-suspend (CONST-033 was in
place and verified) and the kernel OOM killer never fired — but the
user had to manually end the session anyway, because nothing
prevented overlapping heavy workloads from saturating the slice.
CONST-036 closes that loophole at both the source-code layer and the
operational layer. See
`docs/issues/fixed/SESSION_LOSS_2026-04-28.md` in the HelixAgent
project.

**Forbidden direct invocations** (non-exhaustive):

- `loginctl terminate-user|terminate-session|kill-user|kill-session`
- `systemctl stop user@<UID>` / `systemctl kill user@<UID>`
- `gnome-session-quit`
- `pkill -KILL -u $USER` / `killall -u $USER`
- `dbus-send` / `busctl` calls to `org.gnome.SessionManager.Logout|Shutdown|Reboot`
- `echo X > /sys/power/state`
- `/usr/bin/poweroff`, `/usr/bin/reboot`, `/usr/bin/halt`

**Indirect-pressure clauses:**

1. Do not spawn parallel heavy workloads casually; check `free -h`
   first; keep `user.slice` under 70% of physical RAM.
2. Long-lived background subagents go in `system.slice`. Rootless
   podman containers die with the user manager.
3. Document AI-agent concurrency caps in CLAUDE.md.
4. Never script "log out and back in" recovery flows.

**Defence:** every project ships
`scripts/host-power-management/check-no-session-termination-calls.sh`
(static scanner) and
`challenges/scripts/no_session_termination_calls_challenge.sh`
(challenge wrapper). Both MUST be wired into the project's CI /
`run_all_challenges.sh`.

<!-- END no-session-termination addendum (CONST-036) -->

<!-- BEGIN const035-strengthening-2026-04-29 -->

## CONST-035 — End-User Usability Mandate (2026-04-29 strengthening)

A test or Challenge that PASSES is a CLAIM that the tested behavior
**works for the end user of the product**. The HelixAgent project
has repeatedly hit the failure mode where every test ran green AND
every Challenge reported PASS, yet most product features did not
actually work — buggy challenge wrappers masked failed assertions,
scripts checked file existence without executing the file,
"reachability" tests tolerated timeouts, contracts were honest in
advertising but broken in dispatch. **This MUST NOT recur.**

Every PASS result MUST guarantee:

a. **Quality** — the feature behaves correctly under inputs an end
   user will send, including malformed input, edge cases, and
   concurrency that real workloads produce.
b. **Completion** — the feature is wired end-to-end from public
   API surface down to backing infrastructure, with no stub /
   placeholder / "wired lazily later" gaps that silently 503.
c. **Full usability** — a CLI agent / SDK consumer / direct curl
   client following the documented model IDs, request shapes, and
   endpoints SUCCEEDS without having to know which of N internal
   aliases the dispatcher actually accepts.

A passing test that doesn't certify all three is a **bluff** and
MUST be tightened, or marked `t.Skip("...SKIP-OK: #<ticket>")`
so absence of coverage is loud rather than silent.

### Bluff taxonomy (each pattern observed in HelixAgent and now forbidden)

- **Wrapper bluff** — assertions PASS but the wrapper's exit-code
  logic is buggy, marking the run FAILED (or the inverse: assertions
  FAIL but the wrapper swallows them). Every aggregating wrapper MUST
  use a robust counter (`! grep -qs "|FAILED|" "$LOG"` style) —
  never inline arithmetic on a command that prints AND exits
  non-zero.
- **Contract bluff** — the system advertises a capability but
  rejects it in dispatch. Every advertised capability MUST be
  exercised by a test or Challenge that actually invokes it.
- **Structural bluff** — `check_file_exists "foo_test.go"` passes
  if the file is present but doesn't run the test or assert anything
  about its content. File-existence checks MUST be paired with at
  least one functional assertion.
- **Comment bluff** — a code comment promises a behavior the code
  doesn't actually have. Documentation written before / about code
  MUST be re-verified against the code on every change touching the
  documented function.
- **Skip bluff** — `t.Skip("not running yet")` without a
  `SKIP-OK: #<ticket>` marker silently passes. Every skip needs the
  marker; CI fails on bare skips.

The taxonomy is illustrative, not exhaustive. Every Challenge or
test added going forward MUST pass an honest self-review against
this taxonomy before being committed.

<!-- END const035-strengthening-2026-04-29 -->



## CONST-035 Operative Rule (MANDATORY)

> "We had been in position that all tests do execute with success and
> all Challenges as well, but in reality the most of the features does
> not work and can't be used! This MUST NOT be the case."

**The operative rule:** Execution of tests and Challenges MUST guarantee
the quality, the completion and full usability by end users of the product.

- A green test or challenge for a feature that does not actually work
  is a **BLUFF** and is **FORBIDDEN**.
- Every test must assert concrete user-visible outcomes, not just
  internal state.
- Every challenge must run real code and verify real behavior;
  grep/file-existence checks are **NOT sufficient**.
- Mutation testing is **MANDATORY**: deliberately break the feature →
  the test/challenge **MUST then FAIL**.
- The bar for shipping is **NOT "tests pass"** but **"users can use
  the feature."**
- No false-success results are tolerable.
