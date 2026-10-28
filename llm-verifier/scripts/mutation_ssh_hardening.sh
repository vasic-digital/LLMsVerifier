#!/usr/bin/env bash
# Paired mutation check for pkg/local SSH hardening. Applies each mutation
# to registry.go in turn and requires `go test ./pkg/local` to FAIL; the
# original file is always restored. Exit 0 only if every mutant is killed.
set -u
cd "$(dirname "$0")/.." || exit 2
f=pkg/local/registry.go
bak=$(mktemp)
cp "$f" "$bak"
trap 'cp "$bak" "$f"; rm -f "$bak"' EXIT
survived=0
run() { # name, python-regex-from, to
  cp "$bak" "$f"
  if ! python3 - "$f" "$2" "$3" <<'PY'
import sys,re
p,a,b=sys.argv[1:4]; s=open(p).read(); n=s.replace(a,b,1)
if n==s: sys.exit(1)
open(p,'w').write(n)
PY
  then echo "MUTATION-NOT-APPLIED $1"; survived=1; return; fi
  if go test -count=1 ./pkg/local >/dev/null 2>&1; then echo "SURVIVED $1"; survived=1; else echo "killed   $1"; fi
}
run M1-strict-no          '"StrictHostKeyChecking=yes"' '"StrictHostKeyChecking=no"'
run M2-no-double-dash     '"--", target,' 'target,'
run M3-skip-host-validate 'if err := validateHost(host); err != nil {' 'if err := error(nil); err != nil {'
run M4-skip-user-validate 'if err := validateUser(user); err != nil {' 'if err := error(nil); err != nil {'
run M5-skip-kh-validate   'if err := validateKnownHostsFile(opts.KnownHostsFile); err != nil {' 'if err := error(nil); err != nil {'
run M6-drop-kh-arg        'args = append(args, "-o", "UserKnownHostsFile="+opts.KnownHostsFile)' '_ = args'
run M7-host-leading-dash  '`^[A-Za-z0-9]([A-Za-z0-9._-]{0,251}[A-Za-z0-9])?$`' '`^[A-Za-z0-9-]([A-Za-z0-9._-]{0,251}[A-Za-z0-9])?$`'
run M8-user-leading-dash  '`^[A-Za-z_][A-Za-z0-9._-]{0,31}$`' '`^[A-Za-z_-][A-Za-z0-9._-]{0,31}$`'
run M9-allow-trailing-slash '|| strings.HasSuffix(p, "/") ||' '||'
run M10-env-ignored       'KnownHostsFile: os.Getenv(KnownHostsEnv)' 'KnownHostsFile: ""'
run M11-entry-ignores-env  'ProbeLocalModelsWithOptions(hosts, user, ProbeOptionsFromEnv(), runner)' 'ProbeLocalModelsWithOptions(hosts, user, ProbeOptions{}, runner)'
run M12-allow-dotdot-host  ' && !strings.Contains(host, "..")' ''
run M13-allow-dev-path     'strings.HasPrefix(filepath.Clean(p), "/dev/") ||' 'false ||'
exit $survived
