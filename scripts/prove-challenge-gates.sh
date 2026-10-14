#!/usr/bin/env bash
# prove-challenge-gates.sh — LMV-002 paired failure proofs for the verify/challenge
# scripts that can be driven hermetically. Each script is run in a hostile or
# broken environment and MUST report failure (non-zero, never 0); a healthy
# control must pass where one is constructible.
#   verify-deployment.sh        stub 500 -> 1, dead port -> 1, stub 200 -> 0 (control)
#   run_challenges.sh           unknown option -> 1, --help -> 0 (control)
#   run_comprehensive_challenge.sh  copied into a scratch tree with no ../.env -> 1
# NOT covered here (still no proof arm, by design — see the LMV-002 report):
#   run_all_providers/actual_binary/provider_binary/messaging (write to
#   ~/Downloads, need keys/brokers), security-audit.sh (informational greps,
#   no failing mode), the two validate_implementation.sh and the two
#   historical verify_* scripts (operator keep/retire decision pending).
# Scratch on disk under ~/.cache/agent-scratch; live services are never touched.
# Exit 0 all proofs held · 1 a proof was MISSED · 2 could not determine.
set -uo pipefail
ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
SCR="${LMV_PROOF_SCRATCH:-$HOME/.cache/agent-scratch/lmv-challenge-gates}"
mkdir -p -m 700 "$SCR" || exit 2
command -v python3 >/dev/null && command -v curl >/dev/null || { echo "COULD NOT DETERMINE: python3/curl"; exit 2; }
fail=0
expect() { # name expected_rc actual_rc
  if [ "$3" = "$2" ]; then echo "PASS $1 (rc=$3)"; else echo "MISSED $1 (rc=$3, wanted $2)"; fail=1; fi
}
free_port() { python3 -c 'import socket;s=socket.socket();s.bind(("127.0.0.1",0));print(s.getsockname()[1])'; }
stub() { # port status
  cat > "$SCR/stub.py" <<PY
import http.server,sys
class H(http.server.BaseHTTPRequestHandler):
    def do_GET(self):
        b=b'{"status":"healthy"}'; self.send_response($2); self.send_header("Content-Length",str(len(b))); self.end_headers(); self.wfile.write(b)
    def log_message(self,*a): pass
http.server.HTTPServer(("127.0.0.1",int(sys.argv[1])),H).serve_forever()
PY
  python3 "$SCR/stub.py" "$1" >/dev/null 2>&1 & echo $!
}
VD="$ROOT/llm-verifier/verify-deployment.sh"
for st in 500 200; do
  p="$(free_port)"; pid="$(stub "$p" $st)"; sleep 1
  API_URL="http://127.0.0.1:$p" TIMEOUT=5 bash "$VD" >/dev/null 2>&1; r=$?
  kill "$pid" 2>/dev/null; wait "$pid" 2>/dev/null
  if [ $st = 500 ]; then expect "verify-deployment vs HTTP 500 server must fail" 1 $r; else expect "verify-deployment control vs HTTP 200 server" 0 $r; fi
done
API_URL="http://127.0.0.1:$(free_port)" TIMEOUT=3 bash "$VD" >/dev/null 2>&1; expect "verify-deployment vs dead port must fail" 1 $?

RC="$ROOT/llm-verifier/challenges/scripts/run_challenges.sh"
bash "$RC" --definitely-not-an-option >/dev/null 2>&1; expect "run_challenges unknown option must fail" 1 $?
bash "$RC" --help >/dev/null 2>&1; expect "run_challenges --help control" 0 $?

rm -rf "$SCR/tree"; mkdir -p "$SCR/tree/proj/challenges/scripts"
cp "$ROOT/llm-verifier/challenges/scripts/run_comprehensive_challenge.sh" "$SCR/tree/proj/challenges/scripts/"
( cd "$SCR/tree/proj/challenges/scripts" && go() { :; }; PATH="$SCR/nogo:$PATH" timeout 60 bash run_comprehensive_challenge.sh >/dev/null 2>&1 ); r=$?
expect "run_comprehensive without ../.env must fail" 1 $r
exit $fail
