#!/usr/bin/env bash
# check-no-suspend-calls.sh — CONST-033 static scanner.
#
# Walks the project tree and fails if ANY file invokes a host-level
# power-state transition (suspend, hibernate, hybrid-sleep, poweroff,
# halt, reboot, kexec, suspend-then-hibernate) via systemctl, loginctl,
# pm-*, shutdown, DBus (login1 / UPower), or gsettings sleep-inactive-*
# set to anything other than 'nothing'.
#
# Usage:
#   bash check-no-suspend-calls.sh [project_root]
#   bash check-no-suspend-calls.sh --prove-failure
#
# Exit (three-valued; 2 is never a pass):
#   0 = clean — every readable text file under the root was scanned
#   1 = one or more violations found (printed). A violation outranks an
#       unreadable path: a finding is reported even when part of the tree
#       could not be read.
#   2 = could not determine — invocation error (unknown flag, more than one
#       root), a root that is missing / not a directory / unreadable, or a
#       path under the root that grep could not read (listed on stderr).
#
# --prove-failure (§1.1 paired proof, VSC-368): builds throw-away trees
# under $TMPDIR, plants forbidden calls and runs THIS file against them —
# a planted call must exit 1, the clean control 0, and a missing root, an
# unreadable root, an unreadable file and an ambiguous invocation 2. Exits
# 0 only when every case holds, 1 when any does not, 2 when the proof
# itself cannot run (e.g. as root, where chmod 000 does not deny a read).
#
# Scan scope, stated so its limits are visible:
#   * the scan runs from INSIDE the root (`cd root; grep .`), so allowlist
#     substrings are matched against root-relative paths only — a checkout
#     that happens to live under a directory named like an allowlist entry
#     cannot allowlist the whole tree;
#   * allowlist substrings are matched against the reported PATH only, never
#     against the matched line's content;
#   * binary files are not scanned line by line (`grep -I`, deterministic
#     across grep versions); a non-allowlisted binary file that matches when
#     read as text (`grep -a`) makes the verdict undetermined (rc 2), since a
#     shell will still execute a script that carries a NUL byte;
#   * a root may start with '-' only after `--`; an empty-string root is rc 2;
#   * a dangling symlink is not counted as unreadable: it has no content.

set -uo pipefail

usage() { sed -n '2,21p' "${BASH_SOURCE[0]}" | sed 's/^# \{0,1\}//'; }

PROVE=0
ROOT=""
NPOS=0
while [[ $# -gt 0 ]]; do
  case "$1" in
    --prove-failure) PROVE=1; shift ;;
    -h|--help) usage; exit 0 ;;
    --) shift; for a in "$@"; do NPOS=$((NPOS + 1)); ROOT="$a"; done; break ;;
    -*) echo "ERROR: unknown option: $1" >&2; exit 2 ;;
    *) NPOS=$((NPOS + 1)); ROOT="$1"; shift ;;
  esac
done
if [[ "$NPOS" -gt 1 ]]; then
  echo "ERROR: ambiguous target — $NPOS roots given, expected at most one" >&2
  exit 2
fi
if [[ "$NPOS" -eq 1 && -z "$ROOT" ]]; then
  echo "ERROR: empty root given — refusing to fall back to the current directory" >&2
  exit 2
fi
if [[ "$PROVE" -eq 1 && "$NPOS" -gt 0 ]]; then
  echo "ERROR: --prove-failure takes no root" >&2
  exit 2
fi
ROOT="${ROOT:-.}"

# Directories never scanned (third-party / generated / large binary).
EXCLUDE_DIRS=(
  ".git" ".svn" ".hg"
  "node_modules" "vendor" "third_party" "Upstreams" "upstreams"
  "cli_agents" "MCP" "MCP_Module/submodules"
  ".cache" ".gradle" ".idea" ".vscode" ".venv" "venv" "__pycache__"
  "build" "dist" "target" "out" "bin" "obj"
  "releases"
  "Example_Projects"
  "submodules"
)

# File-path substrings allowlisted (the canonical artifacts and
# governance docs ARE allowed to mention these patterns).
#
# Governance docs are allowlisted as .md AND as their mandated .html/.pdf
# export siblings (§11.4.65 requires every governance .md to ship in-sync
# .html/.pdf exports containing the same documentation text). Allowlisting
# only the .md while the exports carry identical text would make the gate
# fail purely because the docs-export mandate was honoured.
EXCLUDE_PATHS=(
  "host-power-management/"
  "host_no_auto_suspend_challenge.sh"
  "no_suspend_calls_challenge.sh"
  "HOST_POWER_MANAGEMENT.md"
  "HOST_POWER_MANAGEMENT.html"
  "HOST_POWER_MANAGEMENT.pdf"
  "CONSTITUTION.md"
  "CONSTITUTION.html"
  "CONSTITUTION.pdf"
  "CONSTITUTION.json"
  "AGENTS.md"
  "AGENTS.html"
  "AGENTS.pdf"
  "CLAUDE.md"
  "CLAUDE.html"
  "CLAUDE.pdf"
  "QWEN.md"
  "QWEN.html"
  "QWEN.pdf"
  "GEMINI.md"
  "GEMINI.html"
  "GEMINI.pdf"
  # Sandboxed-shell challenge spec documents forbidden power-management
  # commands as example block-list INPUTS, not as invocations.
  "p1-f14-sandboxed-shell/CHALLENGE.md"
  "/docs/issues/fixed/BUGFIXES.md"
  "/CHANGELOG.md"
  "/docs/superpowers/plans/"
  "anthropic-quickstarts/"
  # Test fixtures listing forbidden commands as block-list inputs (not invocations):
  "/tests/security/"
  "/test/security/"
  "/internal/workflow/autonomy/autonomy_test.go"
  "/internal/workflow/executor_test.go"
  "/internal/security/tools_security_test.go"
  # User notes / knowledge-base markdown documenting legacy NAS commands:
  "/Knowledge Base/Tech/NAS/"
  "/Knowledge Base/Tech/Linux/"
  # Documentation-only references in third-party security models:
  "/COMMAND_SECURITY_MODEL.md"
  # Third-party AI agent configs that DENY suspend-related commands:
  "vtcode.toml"
  "vtcode.toml.example"
  "/zed-extension/vtcode.toml"
  "/vtcode-config/src/core/commands.rs"
)

# Forbidden grep -E patterns. Real, tight regexes — not bare words.
FORBIDDEN=(
  '\bsystemctl[[:space:]]+(suspend|hibernate|hybrid-sleep|suspend-then-hibernate|poweroff|halt|reboot|kexec)\b'
  '\bloginctl[[:space:]]+(suspend|hibernate|hybrid-sleep|suspend-then-hibernate|poweroff|halt|reboot)\b'
  '\bpm-(suspend|hibernate|suspend-hybrid)\b'
  '\bshutdown[[:space:]]+(-h|-r|-P|-H|now|--halt|--poweroff|--reboot)\b'
  'org\.freedesktop\.login1\.Manager\.(Suspend|Hibernate|HybridSleep|SuspendThenHibernate|PowerOff|Reboot)'
  'org\.freedesktop\.UPower\.(Suspend|Hibernate|HybridSleep)'
  '(sleep-inactive-(ac|battery)-type)[[:space:]]+["'\'']?(suspend|hibernate|shutdown|hybrid-sleep|interactive)["'\'']?'
  '\bdbus-send\b.*\b(Suspend|Hibernate|PowerOff|Reboot|HybridSleep)\b'
  '\bbusctl\b.*\bcall\b.*\b(Suspend|Hibernate|PowerOff|Reboot|HybridSleep)\b'
)

# ── paired proof ─────────────────────────────────────────────────────────────
if [[ "$PROVE" -eq 1 ]]; then
  SELF="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)/$(basename "${BASH_SOURCE[0]}")"
  T=$(mktemp -d) || { echo "prove-failure: cannot determine — mktemp -d failed"; exit 2; }
  trap 'chmod -R u+rwx "$T" 2>/dev/null; rm -rf "$T"' EXIT
  PASS=0; FAILN=0; UNDET=0
  # The scan matches allowlist substrings against root-relative paths, so the
  # scratch location cannot mask a plant; still refuse a scratch path that
  # carries an allowlist substring, so the proof never depends on that.
  for ex in "${EXCLUDE_PATHS[@]}"; do
    case "$T/" in *"$ex"*) echo "prove-failure: cannot determine — scratch path $T carries allowlist substring '$ex'"; exit 2 ;; esac
  done
  mk() { local d="$T/$1"; mkdir -p "$d/src"; printf 'harmless: systemctl status; xset -dpms; sleep 1\n' > "$d/src/ok.sh"; echo "$d"; }
  expect() { # name want root... -> runs THIS file, compares rc
    local name=$1 want=$2; shift 2
    local out rc
    out=$(bash "$SELF" "$@" 2>&1); rc=$?
    if [[ "$rc" -eq "$want" ]]; then PASS=$((PASS + 1)); echo "  PASS $name (rc $rc)"
    else FAILN=$((FAILN + 1)); echo "  FAIL $name: want rc $want, got rc $rc"; printf '%s\n' "$out" | sed 's/^/       | /' | head -6; fi
  }
  echo "prove-failure: CONST-033 scanner $SELF"
  # C0 clean control
  c0=$(mk c0); expect "C0 clean control" 0 "$c0"
  # M1..M5 planted forbidden calls, one per class
  m=$(mk m1); printf '#!/bin/sh\nsystemctl suspend\n' > "$m/src/plant.sh"; expect "M1 planted systemctl suspend" 1 "$m"
  out=$(bash "$SELF" "$m" 2>&1); case "$out" in *"src/plant.sh:2:"*) PASS=$((PASS + 1)); echo "  PASS M1b finding names src/plant.sh:2" ;; *) FAILN=$((FAILN + 1)); echo "  FAIL M1b finding does not name the planted file:line" ;; esac
  m=$(mk m2); printf 'loginctl hibernate\n' > "$m/src/plant.py"; expect "M2 planted loginctl hibernate" 1 "$m"
  m=$(mk m3); printf "gsettings set org.gnome.settings-daemon.plugins.power sleep-inactive-ac-type 'suspend'\n" > "$m/src/plant.sh"; expect "M3 planted gsettings sleep-inactive suspend" 1 "$m"
  m=$(mk m4); printf 'dbus-send --system --dest=org.freedesktop.login1 /org/freedesktop/login1 org.freedesktop.login1.Manager.PowerOff boolean:true\n' > "$m/src/plant.sh"; expect "M4 planted login1 PowerOff over dbus-send" 1 "$m"
  m=$(mk m5); printf 'pm-hibernate\n' > "$m/src/plant.go"; expect "M5 planted pm-hibernate" 1 "$m"
  # A1/A2 scope controls: an allowlisted path and an excluded dir do not fire
  m=$(mk a1); mkdir -p "$m/scripts/host-power-management"; printf 'systemctl suspend\n' > "$m/scripts/host-power-management/x.sh"; expect "A1 allowlisted path is not a finding" 0 "$m"
  m=$(mk a2); mkdir -p "$m/node_modules/p"; printf 'systemctl suspend\n' > "$m/node_modules/p/x.sh"; expect "A2 excluded dir is not scanned" 0 "$m"
  # A3 the root's own location cannot allowlist the tree (scratch root named like an allowlist entry)
  m="$T/host-power-management"; mkdir -p "$m/src"; printf 'systemctl poweroff\n' > "$m/src/plant.sh"; expect "A3 root under an allowlist-named dir still fires" 1 "$m"
  # A4 a dangling symlink has no content and is not unreadable
  m=$(mk a4); ln -s "$T/does-not-exist" "$m/src/dangling"; expect "A4 dangling symlink stays clean" 0 "$m"
  # U1..U5 could-not-determine
  expect "U1 missing root" 2 "$T/no-such-root"
  printf 'x\n' > "$T/plainfile"; expect "U2 root is a file" 2 "$T/plainfile"
  expect "U3 two roots (ambiguous target)" 2 "$c0" "$c0"
  expect "U4 unknown option" 2 --no-such-flag
  m=$(mk u5); chmod 000 "$m"
  if [[ -r "$m" ]]; then UNDET=$((UNDET + 1)); echo "  UNDET U5/U6/U7: chmod 000 does not deny reads here (running as root?)"
  else
    expect "U5 unreadable root" 2 "$m"
    chmod 755 "$m"
    m=$(mk u6); printf 'systemctl suspend\n' > "$m/src/secret.sh"; chmod 000 "$m/src/secret.sh"; expect "U6 unreadable file in a clean tree is not a pass" 2 "$m"
    m=$(mk u7); printf 'x\n' > "$m/src/secret.sh"; chmod 000 "$m/src/secret.sh"; printf 'systemctl reboot\n' > "$m/src/plant.sh"; expect "U7 finding outranks an unreadable file" 1 "$m"
  fi
  # A5 an allowlist token in the matched LINE (not the path) must not suppress the finding
  m=$(mk a5); printf 'systemctl suspend  # documented in CLAUDE.md\n' > "$m/src/deploy.sh"; expect "A5 allowlist token in line content does not suppress" 1 "$m"
  # U8 a NUL byte must not hide a plant: a file grep treats as binary is not scanned as text
  m=$(mk u8); printf 'echo ran; systemctl suspend\n\000\n' > "$m/src/x.sh"; expect "U8 plant in a NUL-bearing file is not a pass" 2 "$m"
  # U9 an empty-string root is not the current directory
  expect "U9 empty-string root" 2 ""
  # U10 a root beginning with '-' is reachable after --
  m="$T/-dash"; mkdir -p "$m/src"; printf 'systemctl halt\n' > "$m/src/plant.sh"
  ( cd "$T" && bash "$SELF" -- -dash >/dev/null 2>&1 ); rc=$?
  if [[ "$rc" -eq 1 ]]; then PASS=$((PASS + 1)); echo "  PASS U10 root '-dash' after -- is scanned (rc 1)"
  else FAILN=$((FAILN + 1)); echo "  FAIL U10 root '-dash' after --: want rc 1, got rc $rc"; fi
  echo "prove-failure: $PASS passed, $FAILN failed, $UNDET undetermined"
  [[ "$FAILN" -gt 0 ]] && exit 1
  [[ "$UNDET" -gt 0 ]] && exit 2
  exit 0
fi

# ── scan ─────────────────────────────────────────────────────────────────────
if [[ ! -e "$ROOT" ]]; then
  echo "ERROR: $ROOT does not exist" >&2
  exit 2
fi
if [[ ! -d "$ROOT" ]]; then
  echo "ERROR: $ROOT is not a directory" >&2
  exit 2
fi
if [[ ! -r "$ROOT" || ! -x "$ROOT" ]]; then
  echo "ERROR: $ROOT is not readable — cannot determine" >&2
  exit 2
fi

# Build grep arguments
EXCL_ARGS=()
for d in "${EXCLUDE_DIRS[@]}"; do EXCL_ARGS+=( --exclude-dir="$d" ); done
PATTERN=$(IFS='|'; echo "${FORBIDDEN[*]}")

TMP=$(mktemp) || { echo "ERROR: mktemp failed — cannot determine" >&2; exit 2; }
ERR=$(mktemp) || { rm -f "$TMP"; echo "ERROR: mktemp failed — cannot determine" >&2; exit 2; }
trap 'rm -f "$TMP" "$ERR"' EXIT

# Scan from inside the root so every reported path is root-relative.
( cd -- "$ROOT" && LC_ALL=C grep -RInIE "$PATTERN" "${EXCL_ARGS[@]}" -- . ) > "$TMP" 2> "$ERR"
GREP_RC=$?

# Read errors other than a dangling symlink (ENOENT) mean part of the tree
# was not scanned. grep exits 2 on any error, so the stderr text decides.
UNREAD=$(grep -v -e ': No such file or directory$' "$ERR" || true)
if [[ "$GREP_RC" -gt 1 && -z "$UNREAD" && ! -s "$ERR" ]]; then
  UNREAD="grep exited $GREP_RC with no diagnostic"
fi

# Filter allowlist substrings
VIOLATIONS=$(awk -v EXCLUDE_PATHS_PIPED="$(IFS='|'; echo "${EXCLUDE_PATHS[*]}")" '
  BEGIN {
    n = split(EXCLUDE_PATHS_PIPED, arr, "|")
    for (i=1;i<=n;i++) ex[i] = arr[i]
    excount = n
  }
  {
    # Match the allowlist against the PATH only: grep -n output is
    # path:lineno:content, so cut at the first ":<digits>:".
    path = $0
    if (match(path, /:[0-9]+:/)) path = substr(path, 1, RSTART - 1)
    skip = 0
    for (i=1;i<=excount;i++) {
      if (ex[i] != "" && index(path, ex[i]) > 0) { skip = 1; break }
    }
    if (!skip) print
  }
' "$TMP")

# Files grep treats as binary (e.g. a NUL byte) are skipped by -I above, but a
# shell still runs such a file. List every file that matches when read as
# text (-a); one that the -I pass did not see, outside the allowlist, was not
# scanned line by line and makes the verdict undetermined.
BINHITS=$( ( cd -- "$ROOT" && LC_ALL=C grep -RalE "$PATTERN" "${EXCL_ARGS[@]}" -- . ) 2>/dev/null | awk -v EXCLUDE_PATHS_PIPED="$(IFS='|'; echo "${EXCLUDE_PATHS[*]}")" '
  BEGIN { n = split(EXCLUDE_PATHS_PIPED, ex, "|") }
  FILENAME != "-" { p = $0; if (match(p, /:[0-9]+:/)) p = substr(p, 1, RSTART - 1); seen[p] = 1; next }
  {
    if ($0 in seen) next
    for (i=1;i<=n;i++) if (ex[i] != "" && index($0, ex[i]) > 0) next
    print
  }
' "$TMP" - )
if [[ -n "$BINHITS" ]]; then
  UNREAD="${UNREAD:+$UNREAD$'\n'}$(printf '%s\n' "$BINHITS" | sed 's/^/binary (not scanned as text) but matches when read as text: /')"
fi

if [[ -n "$VIOLATIONS" ]]; then
  echo "FAIL: forbidden host-power-management invocations (CONST-033) in $ROOT:"
  echo "$VIOLATIONS"
  echo
  echo "If a hit is a legitimate non-host context (e.g. a container's"
  echo "internal init, a documentation example), add the file path to"
  echo "EXCLUDE_PATHS at the top of this script."
  if [[ -n "$UNREAD" ]]; then
    echo "NOTE: part of the tree could not be read as well:" >&2
    printf '%s\n' "$UNREAD" >&2
  fi
  exit 1
fi

if [[ -n "$UNREAD" ]]; then
  echo "UNDETERMINED: no violation in what was read, but part of $ROOT could not be read:" >&2
  printf '%s\n' "$UNREAD" >&2
  exit 2
fi

echo "OK: no forbidden host-power-management calls in $ROOT"
exit 0
