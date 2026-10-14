#!/usr/bin/env bash
# verify-fresh-binary-build.sh — LMV-003 gate: scripts that run a locally built
# binary must not build it only when the file is absent (a stale binary would
# then be scored/used silently). Checks:
#   1. generate-ultimate-opencode.sh, BEHAVIOURAL: with a pre-existing (stale)
#      binary and a fake `go` on PATH, `go build` must still be invoked.
#   2. run_all_providers_challenge.sh / run_comprehensive_challenge.sh, STATIC:
#      no `if [ ! -x|-f "$BINARY" ]` guard in front of a go build.
# Exit 0 pass · 1 finding · 2 could not determine.
# --prove-failure: the same checks run against the pre-fix versions of those
#   scripts (taken from a git revision, default the fix's parent) and MUST fail.
# Scratch is on disk under ~/.cache/agent-scratch, never /tmp.
set -uo pipefail
ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
SCR="${LMV_SCRATCH:-$HOME/.cache/agent-scratch/lmv-fresh-build}"
mkdir -p -m 700 "$SCR" || { echo "COULD NOT DETERMINE: scratch"; exit 2; }
GEN=scripts/generate-ultimate-opencode.sh
STATIC="llm-verifier/challenges/scripts/run_all_providers_challenge.sh llm-verifier/challenges/scripts/run_comprehensive_challenge.sh"

# check_generator <script> -> 0 ok / 1 finding
check_generator() {
  local script="$1" w="$SCR/gen"
  rm -rf "$w"; mkdir -p "$w/scripts" "$w/bin" "$w/llm-verifier" "$w/fakebin"
  printf '#!/bin/sh\nexit 0\n' > "$w/bin/ultimate-challenge"; chmod +x "$w/bin/ultimate-challenge"   # STALE binary
  printf '#!/bin/sh\necho "go $*" >> "%s/go.log"\nexit 0\n' "$w" > "$w/fakebin/go"; chmod +x "$w/fakebin/go"
  ( cd "$w/scripts" && PATH="$w/fakebin:$PATH" bash "$script" >/dev/null 2>&1 )
  if grep -q '^go build' "$w/go.log" 2>/dev/null; then echo "PASS generator rebuilds despite an existing binary"; return 0; fi
  echo "FINDING generator did not rebuild an existing (stale) binary"; return 1
}
# check_static <file>... -> 0 ok / 1 finding
check_static() {
  local f bad=0
  for f in "$@"; do
    if grep -A4 -E 'if \[ ! -[xf] "\$BINARY" \]' "$f" | grep -q 'go build'; then echo "FINDING $f: go build only when the binary is absent"; bad=1; fi
  done
  [ $bad = 0 ] && echo "PASS static: no absent-only build guard"
  return $bad
}

if [ "${1:-}" = "--prove-failure" ]; then
  rev="${2:-$(git -C "$ROOT" log --format=%H -n1 --diff-filter=M -- "$GEN" | head -1)^}"
  fail=0; rm -rf "$SCR/old"; mkdir -p "$SCR/old"
  git -C "$ROOT" show "$rev:$GEN" > "$SCR/old/gen.sh" 2>/dev/null || { echo "COULD NOT DETERMINE: no pre-fix revision $rev"; exit 2; }
  check_generator "$SCR/old/gen.sh" >/dev/null; [ $? -eq 1 ] && echo "M1 CAUGHT pre-fix generator ($rev)" || { echo "M1 MISSED"; fail=1; }
  olds=(); for f in $STATIC; do n="$SCR/old/$(basename "$f")"; git -C "$ROOT" show "$rev:llm-verifier/${f#llm-verifier/}" > "$n" 2>/dev/null || { echo "COULD NOT DETERMINE: $f at $rev"; exit 2; }; olds+=("$n"); done
  check_static "${olds[@]}" >/dev/null; [ $? -eq 1 ] && echo "M2 CAUGHT pre-fix challenge scripts" || { echo "M2 MISSED"; fail=1; }
  exit $fail
fi
rc=0
check_generator "$ROOT/$GEN" || rc=1
check_static $(for f in $STATIC; do echo "$ROOT/$f"; done) || rc=1
exit $rc
