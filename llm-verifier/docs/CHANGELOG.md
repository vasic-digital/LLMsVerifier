## Unreleased - Security: SSH host key verification in pkg/local (MEDIUM)

- `pkg/local`: `ProbeLocalModels*` no longer runs `ssh -o StrictHostKeyChecking=no` (MITM-able: a forged host could return fake model inventories). It now always uses `StrictHostKeyChecking=yes`; hosts must already be in known_hosts. Optional managed file: `ProbeOptions.KnownHostsFile` or env `LLMSVERIFIER_SSH_KNOWN_HOSTS` (absolute path, `[A-Za-z0-9._/-]`, not `/dev/null`) -> `-o UserKnownHostsFile=...`. `accept-new` is deliberately not offered.
- Option injection: the ssh target is now placed after `--`; host (DNS name, IPv4/IPv6 literal) and user (`^[A-Za-z_][A-Za-z0-9._-]{0,31}$`) are validated and values such as `-oProxyCommand=...` are rejected before the runner is invoked (a bad host is skipped like an unreachable one).
- New API: `ProbeLocalModelsWithOptions`, `ProbeOptions`, `ProbeOptionsFromEnv`, `KnownHostsEnv`. Existing signatures unchanged.
- Repo-wide audit: `pkg/local/registry.go` is the only ssh/scp invocation and the only place host/user strings are built into a command.
- Tests: `pkg/local/ssh_hardening_test.go` (exact argv through the real code path, injection negatives); paired mutations: `scripts/mutation_ssh_hardening.sh` (10 mutants).
- Operational note: first-time hosts must be trusted out of band, e.g. `ssh-keyscan -H host >> known_hosts` after verifying the fingerprint.

## Unreleased - Consumer packages moved in from Catalogizer

- Added `pkg/bridge`, `pkg/catalogizer`, `pkg/chat`, `pkg/local`, `pkg/recipe`, `pkg/selector`, `pkg/strategy`, `pkg/vision`. These existed only in the Catalogizer superproject (plain copy `submodules/llms_verifier`, 42 files) and were not previously in this repository.
- `pkg/helixqa`: replaced the 70-line stub with the Catalogizer implementation (`VisionModelRegistry() []strategy.ModelInfo`, recipe, strategy). `VisionModel` is kept as an alias of `strategy.ModelInfo`; the stub's 10 entries are superseded by the 18 in the new registry.
- `.gitignore`: added `!pkg/local/` because the existing `local/` rule ignored `pkg/local`.
- Docs: `docs/consumer_packages_architecture.md`, `docs/PROVIDER_USAGE_CONFIGURATION.md`.

## Latest Changes - Version 2026.01.03

### Security Improvements (P0 Priority)
- **Rate Limiting**: Implemented proper per-client rate limiting in middleware.go with configurable limits
- **LDAP Authentication**: Removed hardcoded credentials, implemented real LDAP authentication with proper bind operations
- **SSO Authentication**: Replaced test token pattern with proper OAuth2/OIDC flow supporting Google, Microsoft, GitHub, and Okta providers
- **LDAP TLS Security**: Fixed InsecureSkipVerify vulnerability, implemented proper certificate verification with configurable CA certificates
- **RBAC Enforcement**: Implemented proper role-based access control with permission checking and client active state validation
- **Usage Tracking**: Implemented comprehensive client usage tracking with request counting, token usage, and cost calculation

### Feature Implementations (P1 Priority)
- **LDAP User Sync**: Implemented user synchronization from LDAP directory with group support
- **Database Schema**: Added missing tables (usage_tracking, token_usage, client_quotas, model_preferences)
- **Multimodal Processing**: Implemented proper image/audio processing with provider integration
- **WebSocket**: Implemented real WebSocket connections with proper upgrade handling and bidirectional messaging
- **GDPR Compliance**: Implemented data export, deletion, and anonymization functionality
- **JSON Schema Validation**: Implemented full JSON Schema validation with Draft-07 support

### Test Suite Updates (P2 Priority)
- Fixed all test compilation errors across 40+ packages
- Updated test assertions to match new implementation signatures
- Added missing test helper functions
- All tests now pass with `go test ./...`

---

## Version 2025.12.18

### Backend
- Fixed API test hanging by adding server.Shutdown() calls
- Replaced context.TODO() with context.Background() in cloud_providers.go
- Fixed all network-dependent test expectations
- All enhanced packages now pass tests without -short flag

### Frontend
- Flutter: Implemented complete profile editing system with validation
- Electron: Added verification controls with HTTP API integration
- Angular: Fixed optional chaining warning and increased SCSS budget

### Platform Support
- All platforms now feature-complete
- No remaining TODO markers in backend code
- Production-ready deployment configuration
