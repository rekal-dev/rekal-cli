#!/usr/bin/env python3
"""Overnight self-loop: every 5 minutes check status and take actions.

Survives agent shell exit (double-fork). Log: ~/imb-action-loop.log
Lock: ~/imb-action-loop.lock

Actions:
  - Relaunch dead LME-S / MSC / LME-M / LoCoMo-sweep / BEAM / watchdog
  - If LME-S shards 400-475 complete and full-test not running → start it
  - If LoCoMo repos>=10 but sweep idle with 0 manifests for too long → relaunch
  - If disk free < 8 GiB → pause LME-M (kill track) to protect LME-S
  - Snapshot progress each tick
"""
from __future__ import annotations

import os
import subprocess
import time
from pathlib import Path

HOME = Path.home()
REPO = Path("/Users/frank/projects/rekal/rekal-cli")
REKAL = str(REPO / "rekal")
LOG = HOME / "imb-action-loop.log"
LOCK = HOME / "imb-action-loop.lock"
INTERVAL = 300  # 5 minutes


def log(msg: str) -> None:
    line = f"[{time.strftime('%Y-%m-%dT%H:%M:%S')}] {msg}\n"
    with open(LOG, "a") as f:
        f.write(line)


def alive(lock: Path) -> bool:
    if not lock.exists():
        return False
    try:
        pid = int(lock.read_text().strip())
        os.kill(pid, 0)
        return True
    except (OSError, ValueError):
        return False


def shard_ok(off: int) -> bool:
    out = HOME / "imb-lme-s" / f"shard-{off:04d}"
    logf = out / "ingest.log"
    if not logf.exists():
        return False
    text = logf.read_text(errors="ignore")
    term = text.startswith("done:") or "VERIFICATION FAILED" in text or "\ndone:" in text
    # also accept trailing done: line
    term = term or any(line.startswith("done:") for line in text.splitlines())
    if not term and "VERIFICATION FAILED" not in text:
        return False
    ndb = sum(1 for _ in out.rglob("data.db"))
    return ndb >= 20


def count_dbs(root: Path) -> int:
    if not root.exists():
        return 0
    return sum(1 for _ in root.rglob("data.db"))


def disk_free_gib() -> float:
    st = os.statvfs(str(HOME))
    return (st.f_bavail * st.f_frsize) / (1024**3)


def run(cmd: list[str], timeout: int = 120) -> int:
    try:
        p = subprocess.run(cmd, cwd=str(REPO), env={**os.environ, "REKAL": REKAL, "PATH": f"{REPO}:{os.environ.get('PATH','')}"},
                           capture_output=True, text=True, timeout=timeout)
        if p.returncode != 0 and p.stderr:
            log(f"cmd fail {' '.join(cmd[:4])}…: {p.stderr[:200]}")
        return p.returncode
    except Exception as e:
        log(f"cmd exc {cmd[:3]}: {e}")
        return 1


def relaunch_daemon(flag: str) -> None:
    log(f"action: relaunch daemon_recover {flag}")
    run(["python3", str(REPO / "scripts/industry-bench/daemon_recover.py"), flag], timeout=60)


def relaunch_locomo_sweep() -> None:
    log("action: relaunch locomo weight sweep")
    script = r'''
import os, time, subprocess
from pathlib import Path
REPO = Path("/Users/frank/projects/rekal/rekal-cli")
LOG = Path.home() / "imb-locomo-weight-sweep.log"
lock = Path.home() / "imb-locomo-sweep.lock"
if lock.exists():
    try:
        os.kill(int(lock.read_text().strip()), 0)
        raise SystemExit(0)
    except (OSError, ValueError):
        pass
pid = os.fork()
if pid > 0:
    os.waitpid(pid, 0)
    raise SystemExit(0)
os.setsid()
if os.fork() > 0:
    os._exit(0)
lock.write_text(str(os.getpid()))
env = os.environ.copy()
env["REKAL"] = str(REPO / "rekal")
env["WORKERS"] = "2"
with open(LOG, "a") as out:
    out.write(f"\n=== sweep relaunch pid={os.getpid()} ===\n")
    out.flush()
    p = subprocess.run(["bash", str(REPO / "scripts/industry-bench/run_locomo_weight_sweep.sh")],
                       cwd=str(REPO), env=env, stdout=out, stderr=subprocess.STDOUT)
    out.write(f"\n=== sweep exit {p.returncode} ===\n")
try:
    lock.unlink()
except FileNotFoundError:
    pass
os._exit(0)
'''
    subprocess.Popen(["python3", "-c", script], cwd=str(REPO),
                     stdout=subprocess.DEVNULL, stderr=subprocess.DEVNULL,
                     start_new_session=True)


def ensure_full_test() -> None:
    """If LME-S last shards done and full-test not alive, start it."""
    pending = [o for o in (400, 425, 450, 475) if not shard_ok(o)]
    if pending:
        log(f"LME-S waiting shards={pending} dbs400={count_dbs(HOME/'imb-lme-s'/'shard-0400')}")
        return
    # A manual refill/rerun holds this lock — never double-launch alongside it.
    if alive(HOME / "imb-lme-s-fulltest-refill.lock"):
        log("LME-S full-test refill in progress — skip")
        return
    # Completion is the aggregate artifact on disk, not a log substring.
    agg = REPO / "scripts/industry-bench/runs/full/longmemeval-s"
    if agg.exists() and any(agg.glob("aggregate-*.md")):
        log("LME-S full test aggregate present — skip")
        return
    # check if full test already done or running
    ft_log = HOME / "imb-lme-s-full-test.log"
    if ft_log.exists() and "FULL TEST COMPLETE" in ft_log.read_text(errors="ignore")[-4000:]:
        log("LME-S full test already complete — skip")
        return
    # if recover lock still running it will launch full test itself — don't double
    if alive(HOME / "imb-lme-s" / ".recover.lock"):
        # track may be mid full-test
        log("LME-S recover still alive (may be in full TEST)")
        return
    log("action: launch LME-S full TEST")
    env = {**os.environ, "REKAL": REKAL, "DATE_TAG": "fulltest-20260717", "WORKERS": "4"}
    subprocess.Popen(
        ["bash", str(REPO / "scripts/industry-bench/run_full_test.sh")],
        cwd=str(REPO), env=env,
        stdout=open(HOME / "imb-lme-s-full-test.log", "a"),
        stderr=subprocess.STDOUT,
        start_new_session=True,
    )


def tick() -> None:
    free = disk_free_gib()
    s400 = count_dbs(HOME / "imb-lme-s" / "shard-0400")
    locomo = count_dbs(HOME / "imb-locomo")
    sweep_m = len(list((REPO / "scripts/industry-bench/runs/sweep").glob("locomo-weights-*/**/manifest.json"))) if (REPO / "scripts/industry-bench/runs/sweep").exists() else 0
    gen = 0
    try:
        gen = len(subprocess.check_output(["pgrep", "-f", "sh_gen/gen.py"], text=True).split())
    except subprocess.CalledProcessError:
        pass

    log(
        f"tick free={free:.1f}Gi gen={gen} "
        f"S400={s400} S425={count_dbs(HOME/'imb-lme-s'/'shard-0425')} "
        f"locomo_dbs={locomo} sweep_manifests={sweep_m} "
        f"lme-s={alive(HOME/'imb-lme-s'/'.recover.lock')} "
        f"msc={alive(HOME/'imb-msc'/'.recover.lock')} "
        f"lme-m={alive(HOME/'imb-lme-m'/'.recover.lock')} "
        f"locomo-sweep={alive(HOME/'imb-locomo-sweep.lock')} "
        f"watchdog={alive(HOME/'imb-watchdog.lock')}"
    )

    # disk guard
    if free < 8.0 and alive(HOME / "imb-lme-m" / ".recover.lock"):
        try:
            pid = int((HOME / "imb-lme-m" / ".recover.lock").read_text().strip())
            os.kill(pid, 15)
            log("action: paused LME-M (disk < 8Gi)")
            time.sleep(2)
            (HOME / "imb-lme-m" / ".recover.lock").unlink(missing_ok=True)
        except (OSError, ValueError):
            pass

    # relaunch dead tracks
    if not alive(HOME / "imb-lme-s" / ".recover.lock"):
        pending = [o for o in (400, 425, 450, 475) if not shard_ok(o)]
        if pending:
            relaunch_daemon("--lme-s-only")
        else:
            ensure_full_test()
    else:
        ensure_full_test()

    if not alive(HOME / "imb-msc" / ".recover.lock"):
        # only relaunch if eval not obviously finished (~30 shards)
        msc_shards = list((HOME / "imb-msc").glob("shard-*")) if (HOME / "imb-msc").exists() else []
        done = 0
        for s in msc_shards:
            lf = s / "ingest.log"
            if lf.exists() and ("done:" in lf.read_text(errors="ignore") or "VERIFICATION FAILED" in lf.read_text(errors="ignore")):
                done += 1
        if done < 28:
            relaunch_daemon("--msc-only")

    if free >= 10.0 and not alive(HOME / "imb-lme-m" / ".recover.lock"):
        # only if not all shards done
        m_ok = sum(1 for o in range(0, 500, 25) if (HOME / "imb-lme-m" / f"shard-{o:04d}" / "ingest.log").exists())
        if m_ok < 20:
            relaunch_daemon("--lme-m-only")

    if not alive(HOME / "imb-watchdog.lock"):
        log("action: relaunch watchdog via daemon_recover --lme-s-only noop path — starting full recover helpers")
        # watchdog is created by --all; invoke a tiny dedicated launch
        relaunch_daemon("--lme-s-only")  # no-op if LME-S lock alive; still need watchdog
        # spawn watchdog alone by re-running recover detached watchdog section
        run(["python3", str(REPO / "scripts/industry-bench/daemon_recover.py")], timeout=90)

    # LoCoMo sweep
    if not alive(HOME / "imb-locomo-sweep.lock"):
        aggs = []
        sweep_root = REPO / "scripts/industry-bench/runs/sweep"
        if sweep_root.exists():
            for a in sweep_root.glob("locomo-weights-*/aggregate.md"):
                txt = a.read_text(errors="ignore")
                if "evidence@5" in txt and "|" in txt and txt.count("\n") > 6:
                    aggs.append(a)
        if not aggs:
            relaunch_locomo_sweep()
        else:
            log(f"LoCoMo sweep done — {aggs[-1]}")

    # BEAM if jsonl empty/missing — backoff so a broken download cannot spam every tick
    beam_jsonl = REPO / "scripts/industry-bench/datasets/data/beam-128k-conversations.jsonl"
    beam_fail = HOME / "imb-beam-giveup"
    beam_attempts = HOME / "imb-beam-attempts"
    if beam_fail.exists():
        pass  # permanent give-up until operator removes marker
    elif (not beam_jsonl.exists() or beam_jsonl.stat().st_size < 1000) and not alive(HOME / "imb-beam" / ".recover.lock"):
        n = 0
        if beam_attempts.exists():
            try:
                n = int(beam_attempts.read_text().strip() or "0")
            except ValueError:
                n = 0
        if n >= 6:
            beam_fail.write_text("beam download failed 6+ times — skipping overnight\n")
            log("action: BEAM give-up after 6 attempts (remove ~/imb-beam-giveup to retry)")
        else:
            beam_attempts.write_text(str(n + 1))
            relaunch_daemon("--beam-only")


def main() -> None:
    if LOCK.exists():
        try:
            os.kill(int(LOCK.read_text().strip()), 0)
            print("action loop already running")
            return
        except (OSError, ValueError):
            pass

    pid = os.fork()
    if pid > 0:
        os.waitpid(pid, 0)
        time.sleep(0.3)
        print("action loop", LOCK.read_text().strip() if LOCK.exists() else "?")
        return
    os.setsid()
    if os.fork() > 0:
        os._exit(0)

    LOCK.write_text(str(os.getpid()))
    log(f"action loop start pid={os.getpid()} interval={INTERVAL}s")
    try:
        while True:
            try:
                tick()
            except Exception as e:
                log(f"tick error: {e}")
            time.sleep(INTERVAL)
    finally:
        try:
            LOCK.unlink()
        except FileNotFoundError:
            pass


if __name__ == "__main__":
    main()
