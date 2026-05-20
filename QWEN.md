## INHERITED FROM Helix Constitution

This module is a submodule of a Helix-family project (e.g.
HelixCode, HelixAgent, ATMOSphere) that includes the Helix
Constitution submodule at the parent's `constitution/` path. All
rules in `constitution/CLAUDE.md` and the
`constitution/Constitution.md` it references (universal anti-bluff
covenant §11.4, no-guessing mandate §11.4.6, credentials-handling
mandate §11.4.10, host-session safety §12, data safety §9, mutation-
paired gates §1.1) apply unconditionally to every change landed here.
The module-specific rules below extend them — they never weaken any
universal clause.

When this file disagrees with the constitution submodule, the
constitution wins. Locate the constitution submodule from any
arbitrary nested depth using its `find_constitution.sh` helper.

Canonical reference: <https://github.com/HelixDevelopment/HelixConstitution>

---

# QWEN.md — LLMsVerifier AI Agent Manual (Qwen Code)

This `QWEN.md` is the Qwen Code sibling of this submodule's `CLAUDE.md`
and `AGENTS.md`. Per the parent project's CLAUDE.md §1.1, `QWEN.md` and
`CRUSH.md` are sibling agent manuals for other CLI tools — rule changes
MUST cascade to all of them. When this file disagrees with
`CLAUDE.md` / `AGENTS.md` / `CONSTITUTION.md` the constitution wins.

**Your mandate**: Write real, working, tested code. No simulations. No
placeholders. No "for now" implementations. Every feature you implement
MUST actually work when a user invokes it.

---

## Universal Mandatory Rules (Non-Negotiable)

These rules cascade from the HelixCode Constitution. They are permanent
and apply to every task.

- **Rule 1 — No CI/CD Pipelines.** No `.github/workflows/`,
  `.gitlab-ci.yml`, `Jenkinsfile`, `.travis.yml`, `.circleci/`, or any
  automated pipeline. Builds/tests run manually or via Makefile/script.
- **Rule 2 — No Mocks in Production.** Mocks, stubs, fakes, placeholder
  classes, TODO implementations are STRICTLY FORBIDDEN in production
  code. Only unit tests may use mocks.
- **Rule 3 — No HTTPS for Git.** SSH URLs only (`git@github.com:…`).
- **Rule 4 — No Manual Container Commands.** Use the orchestrator
  binary; direct `docker`/`docker-compose` workflows are prohibited.
- **Rule 5 — Real Data for Non-Unit Tests.** All integration, E2E, and
  challenge tests MUST use real infrastructure.
- **Rule 6 — 100% Challenge Coverage.** Every component MUST have
  Challenge scripts validating real-life use cases.
- **Rule 7 — Reproduction-Before-Fix.** Every bug MUST be reproduced by
  a Challenge script BEFORE any fix is attempted.
- **Rule 8 — Definition of Done.** A change is NOT done because code
  compiles. "Done" requires pasted terminal output from a real run.
- **Rule 9 — No Self-Certification.** Words like *verified, tested,
  working, complete, fixed, passing* are forbidden unless accompanied by
  pasted command output from that session.
- **Rule 10 — Zero-Bluff Mandate (CONST-035).** A passing test is a
  claim that the feature **works for the end user**. Every test must
  guarantee Quality + Completion + Full Usability. Any test that doesn't
  certify all three is a bluff and must be tightened.

---

## MANDATORY ANTI-BLUFF COVENANT — END-USER QUALITY GUARANTEE

**Forensic anchor — direct user mandate (verbatim):**

> "We had been in position that all tests do execute with success and
> all Challenges as well, but in reality the most of the features does
> not work and can't be used! This MUST NOT be the case and execution
> of tests and Challenges MUST guarantee the quality, the completion
> and full usability by end users of the product!"

**Operative rule:** the bar for shipping is **not** "tests pass" but
**"users can use the feature."** Every PASS in this codebase MUST carry
positive runtime evidence captured during execution that the feature
works for the end user. Metadata-only PASS, configuration-only PASS,
"absence-of-error" PASS, and grep-based PASS without runtime evidence
are all critical defects regardless of how green the summary line looks.

**Tests AND Challenges (HelixQA) are bound equally** — a Challenge that
scores PASS on a non-functional feature is the same class of defect as
a unit test that does. Both must produce positive end-user evidence.

### Article XI §11.9 — Anti-Bluff Forensic Anchor

> Verbatim user mandate: *"We had been in position that all tests do
> execute with success and all Challenges as well, but in reality the
> most of the features does not work and can't be used! This MUST NOT
> be the case and execution of tests and Challenges MUST guarantee the
> quality, the completion and full usability by end users of the
> product!"*
>
> Operative rule: **The bar for shipping is not "tests pass" but
> "users can use the feature."** Every PASS in this codebase MUST carry
> positive runtime evidence captured during execution. Metadata-only /
> configuration-only / absence-of-error / grep-based PASS without
> runtime evidence are critical defects regardless of how green the
> summary line looks. No false-success results are tolerable.

---

## Constitutional anchors (cascaded from `CONSTITUTION.md`)

- **CONST-035 — Zero-Bluff / End-User Usability Mandate.** Every PASS
  guarantees Quality + Completion + Full Usability or it is a bluff.
- **Article XII §12.1 (CONST-042) — No-Secret-Leak.** No API key,
  token, password, certificate, or other credential may be committed.
  All secrets live in `.env` files (mode 0600) listed in `.gitignore`.
- **Article XII §12.2 (CONST-043) — No-Force-Push.** No force push,
  history rewrite, or branch deletion of `main`/`master` without
  explicit, in-conversation user approval per operation.
- **Article XIII §13.1 (CONST-044) — Continuation Document Maintenance.**
  The meta-repo's `docs/CONTINUATION.md` MUST be kept in sync with
  actual programme state. Out-of-sync CONTINUATION is a CRITICAL DEFECT.
- **CONST-036 through CONST-040 — LLMsVerifier mandates.** LLMsVerifier
  is the single source of truth for model/provider metadata,
  verification status, and capability flags. NO hardcoded model lists.
- **CONST-046 through CONST-074** — see this submodule's `CLAUDE.md`,
  `AGENTS.md`, and `CONSTITUTION.md` for the full set of cascaded
  anchors (no-fakes-beyond-unit-tests, recursive submodule application,
  full-automation coverage, documentation always-sync, fetch-before-edit
  / pre-push merge-first discipline, subagent-driven default, etc.).
  This `QWEN.md` carries the anti-bluff covenant verbatim; the complete
  per-clause text lives in the canonical trio.

---

## §11.4 anti-bluff extensions (summary — full text in CLAUDE.md)

- **§11.4.1** — FAIL-bluffs equally forbidden: a test that crashes for a
  script-internal reason is as misleading as a PASS on a broken feature.
- **§11.4.2** — recorded-evidence requirement: a PASS for a user-visible
  feature MUST carry captured visual/audio evidence.
- **§11.4.5** — audio + video quality analysis comprehensiveness.
- **§11.4.6** — no-guessing mandate: no `likely`/`probably`/`maybe`;
  prove the cause with captured forensic evidence or mark `UNKNOWN:`.
- **§11.4.68 / §11.4.69** — positive sink-side / downstream evidence:
  empty / `<unreachable>` placeholders are NOT positive evidence.
- **§11.4.70** — subagent-driven execution is the default.

Non-compliance with any anchor is a release blocker regardless of
context. When in doubt, consult this submodule's `CLAUDE.md` /
`CONSTITUTION.md` — they carry the complete clause text and the
constitution submodule remains the canonical source of truth.

---

*Remember: Your code will be used by real people. Write code that
actually works.*
