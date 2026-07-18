#!/usr/bin/env python3
"""Double-fork daemon launcher for industry-bench recover tracks.

Survives Cursor/agent shell cleanup (setsid + double fork).
Usage:
  REKAL=./rekal python3 scripts/industry-bench/daemon_recover.py
"""
from __future__ import annotations

import os
import sys
import time
from pathlib import Path

REPO = Path(__file__).resolve().parents[2]
REKAL = os.environ.get("REKAL", str(REPO / "rekal"))
HOME = Path.home()
MASTER = HOME / "imb-recover-detached.log"


def log(msg: str) -> None:
    line = f"[{time.strftime('%Y-%m-%dT%H:%M:%S%z')}] {msg}\n"
    with open(MASTER, "a") as f:
        f.write(line)


def daemonize(work) -> int:
    """Double-fork; child runs work(); returns child pid to parent."""
    pid = os.fork()
    if pid > 0:
        # first parent
        os.waitpid(pid, 0)
        return 0
    os.setsid()
    pid2 = os.fork()
    if pid2 > 0:
        os._exit(0)
    # grandchild
    os.chdir("/")
    os.umask(0)
    devnull = open("/dev/null", "r+")
    os.dup2(devnull.fileno(), 0)
    work()
    os._exit(0)


def run_bash(script: str, logfile: Path, lock: Path) -> None:
    if lock.exists():
        try:
            old = int(lock.read_text().strip())
            os.kill(old, 0)
            log(f"skip {lock.name}: pid {old} alive")
            return
        except (OSError, ValueError):
            pass

    def work() -> None:
        import subprocess

        # write lock with our pid after fork
        lock.parent.mkdir(parents=True, exist_ok=True)
        lock.write_text(str(os.getpid()))
        env = os.environ.copy()
        env["REKAL"] = REKAL
        env["PATH"] = f"{REPO}:{env.get('PATH', '')}"
        with open(logfile, "a") as out:
            out.write(f"\n=== daemon start pid={os.getpid()} ===\n")
            out.flush()
            p = subprocess.run(
                ["bash", "-lc", script],
                stdout=out,
                stderr=subprocess.STDOUT,
                env=env,
                cwd=str(REPO),
            )
            out.write(f"\n=== daemon exit code={p.returncode} ===\n")
        try:
            lock.unlink(missing_ok=True)
        except TypeError:
            if lock.exists():
                lock.unlink()

    # fork a dedicated daemon for this track
    pid = os.fork()
    if pid > 0:
        # wait for intermediate
        os.waitpid(pid, 0)
        # lock written by grandchild after second fork — brief wait
        time.sleep(0.3)
        log(f"launched track → {logfile} lock={lock}")
        return
    os.setsid()
    pid2 = os.fork()
    if pid2 > 0:
        os._exit(0)
    os.chdir("/")
    os.umask(0)
    work()
    os._exit(0)


def lme_s_script() -> str:
    return f"""
set -eu
REPO_ROOT="{REPO}"
REKAL="{REKAL}"
JSONL="$REPO_ROOT/scripts/industry-bench/datasets/data/longmemeval-s-conversations.jsonl"
echo "[$(date -Iseconds)] LME-S track start"
for off in 400 425 450 475; do
  out=$(printf "$HOME/imb-lme-s/shard-%04d" "$off")
  ndb=$(find "$out" -name data.db 2>/dev/null | wc -l | tr -d ' ')
  if [[ -f "$out/ingest.log" ]] && grep -qE '^(done:)|VERIFICATION FAILED' "$out/ingest.log" 2>/dev/null && [[ "${{ndb:-0}}" -ge 20 ]]; then
    echo "[$(date -Iseconds)] skip shard $off dbs=$ndb"
    continue
  fi
  echo "[$(date -Iseconds)] ingest shard $off"
  rm -rf "$out"; mkdir -p "$out"
  PYTHONUNBUFFERED=1 python3 "$REPO_ROOT/scripts/industry-bench/sh_gen/gen.py" \\
    --input "$JSONL" --offset "$off" --limit 25 --out "$out" --rekal "$REKAL" --verify \\
    > "$out/ingest.log" 2>&1 || true
  echo "[$(date -Iseconds)] shard $off exit dbs=$(find "$out" -name data.db | wc -l | tr -d ' ')"
done
echo "[$(date -Iseconds)] launching full TEST"
DATE_TAG=fulltest-20260717 bash "$REPO_ROOT/scripts/industry-bench/run_full_test.sh" \\
  >> "$HOME/imb-lme-s-full-test.log" 2>&1 || true
echo "[$(date -Iseconds)] LME-S track done"
"""


def msc_script() -> str:
    return f"""
set -eu
REPO_ROOT="{REPO}"
REKAL="{REKAL}"
SUBSET="$REPO_ROOT/scripts/industry-bench/datasets/data/msc-eval-conversations.jsonl"
WORKERS=3
echo "[$(date -Iseconds)] MSC track start"
TOTAL=$(wc -l < "$SUBSET" | tr -d ' ')
off=0
while [[ $off -lt $TOTAL ]]; do
  out=$(printf "$HOME/imb-msc/shard-%04d" "$off")
  if [[ -f "$out/ingest.log" ]] && grep -qE '^(done:)|VERIFICATION FAILED' "$out/ingest.log" 2>/dev/null; then
    echo "[$(date -Iseconds)] skip MSC $off"
  else
    echo "[$(date -Iseconds)] MSC ingest $off"
    rm -rf "$out"; mkdir -p "$out"
    (
      PYTHONUNBUFFERED=1 python3 "$REPO_ROOT/scripts/industry-bench/sh_gen/gen.py" \\
        --input "$SUBSET" --offset "$off" --limit 50 --out "$out" --rekal "$REKAL" --verify --fast 3 \\
        > "$out/ingest.log" 2>&1 || true
      echo "[$(date -Iseconds)] MSC $off done"
    ) &
    while [[ "$(jobs -rp | wc -l | tr -d ' ')" -ge "$WORKERS" ]]; do sleep 20; done
  fi
  off=$((off + 50))
done
wait
echo "[$(date -Iseconds)] MSC track done"
"""


def lme_m_script() -> str:
    return f"""
set -eu
REPO_ROOT="{REPO}"
REKAL="{REKAL}"
JSONL="$REPO_ROOT/scripts/industry-bench/datasets/data/longmemeval-m-conversations.jsonl"
WORKERS=2
echo "[$(date -Iseconds)] LME-M track start"
for off in $(seq 0 25 475); do
  out=$(printf "$HOME/imb-lme-m/shard-%04d" "$off")
  ndb=$(find "$out" -name data.db 2>/dev/null | wc -l | tr -d ' ')
  if [[ -f "$out/ingest.log" ]] && grep -qE '^(done:)|VERIFICATION FAILED' "$out/ingest.log" 2>/dev/null && [[ "${{ndb:-0}}" -ge 20 ]]; then
    echo "[$(date -Iseconds)] skip M $off"
    continue
  fi
  echo "[$(date -Iseconds)] M ingest $off"
  rm -rf "$out"; mkdir -p "$out"
  (
    PYTHONUNBUFFERED=1 python3 "$REPO_ROOT/scripts/industry-bench/sh_gen/gen.py" \\
      --input "$JSONL" --offset "$off" --limit 25 --out "$out" --rekal "$REKAL" --verify --fast 5 \\
      > "$out/ingest.log" 2>&1 || true
    echo "[$(date -Iseconds)] M $off done"
  ) &
  while [[ "$(jobs -rp | wc -l | tr -d ' ')" -ge "$WORKERS" ]]; do sleep 45; done
done
wait
echo "[$(date -Iseconds)] LME-M track done"
"""


def beam_script() -> str:
    return f"""
set -eu
REPO_ROOT="{REPO}"
REKAL="{REKAL}"
echo "[$(date -Iseconds)] BEAM track start"
bash "$REPO_ROOT/scripts/industry-bench/datasets/get_beam.sh" 128k >> "$HOME/imb-beam-download.log" 2>&1 || true
python3 "$REPO_ROOT/scripts/industry-bench/datasets/normalize_beam.py" --tier 128k >> "$HOME/imb-beam-download.log" 2>&1 || true
JSONL="$REPO_ROOT/scripts/industry-bench/datasets/data/beam-128k-conversations.jsonl"
if [[ -f "$JSONL" ]]; then
  mkdir -p "$HOME/imb-beam"
  PYTHONUNBUFFERED=1 python3 "$REPO_ROOT/scripts/industry-bench/sh_gen/gen.py" \\
    --input "$JSONL" --limit 5 --out "$HOME/imb-beam" --rekal "$REKAL" --verify --index --fast 10 \\
    > "$HOME/imb-beam/ingest-limit5.log" 2>&1 || true
fi
echo "[$(date -Iseconds)] BEAM track done"
"""


def watchdog_script() -> str:
    return f"""
set -eu
REPO_ROOT="{REPO}"
REKAL="{REKAL}"
MASTER="{MASTER}"
while true; do
  sleep 180
  echo "[$(date -Iseconds)] watchdog tick free=$(df -h /Users/frank | awk 'NR==2{{print $4}}')" >> "$MASTER"
  need=0
  for off in 400 425 450 475; do
    out=$(printf "$HOME/imb-lme-s/shard-%04d" "$off")
    ndb=$(find "$out" -name data.db 2>/dev/null | wc -l | tr -d ' ')
    if ! {{ [[ -f "$out/ingest.log" ]] && grep -qE '^(done:)|VERIFICATION FAILED' "$out/ingest.log" 2>/dev/null && [[ "${{ndb:-0}}" -ge 20 ]]; }}; then
      need=1
    fi
  done
  lock="$HOME/imb-lme-s/.recover.lock"
  alive=0
  if [[ -f "$lock" ]]; then
    kill -0 "$(cat "$lock")" 2>/dev/null && alive=1 || true
  fi
  if [[ "$need" -eq 1 && "$alive" -eq 0 ]]; then
    echo "[$(date -Iseconds)] watchdog relaunch LME-S" >> "$MASTER"
    REKAL="$REKAL" python3 "$REPO_ROOT/scripts/industry-bench/daemon_recover.py" --lme-s-only
  fi
done
"""


def main() -> None:
    only = sys.argv[1] if len(sys.argv) > 1 else ""
    log(f"daemon_recover start only={only or 'all'} rekal={REKAL}")

    tracks = []
    if only in ("", "--all"):
        tracks = [
            (lme_s_script(), HOME / "imb-lme-s-recover-track.log", HOME / "imb-lme-s" / ".recover.lock"),
            (msc_script(), HOME / "imb-msc-recover-track.log", HOME / "imb-msc" / ".recover.lock"),
            (beam_script(), HOME / "imb-beam-recover-track.log", HOME / "imb-beam" / ".recover.lock"),
            (watchdog_script(), HOME / "imb-watchdog.log", HOME / "imb-watchdog.lock"),
        ]
        # delay LME-M
        def delayed_m() -> None:
            time.sleep(90)
            run_bash(lme_m_script(), HOME / "imb-lme-m-recover-track.log", HOME / "imb-lme-m" / ".recover.lock")

        pid = os.fork()
        if pid == 0:
            os.setsid()
            if os.fork() == 0:
                delayed_m()
                os._exit(0)
            os._exit(0)
        os.waitpid(pid, 0)
    elif only == "--lme-s-only":
        tracks = [(lme_s_script(), HOME / "imb-lme-s-recover-track.log", HOME / "imb-lme-s" / ".recover.lock")]
    elif only == "--msc-only":
        tracks = [(msc_script(), HOME / "imb-msc-recover-track.log", HOME / "imb-msc" / ".recover.lock")]
    elif only == "--lme-m-only":
        tracks = [(lme_m_script(), HOME / "imb-lme-m-recover-track.log", HOME / "imb-lme-m" / ".recover.lock")]
    elif only == "--beam-only":
        tracks = [(beam_script(), HOME / "imb-beam-recover-track.log", HOME / "imb-beam" / ".recover.lock")]
    else:
        print("unknown flag", only, file=sys.stderr)
        sys.exit(2)

    for script, logfile, lock in tracks:
        run_bash(script, logfile, lock)

    log("daemon_recover parent exit (children detached)")


if __name__ == "__main__":
    main()
