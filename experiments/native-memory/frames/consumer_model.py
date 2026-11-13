"""Proposed consumer contract; NOT Rekal's search engine or shipped config.

Uses a small lexical fixture scorer and optional injected semantic scores to
test interactions/invariants. It makes no retrieval-quality or latency claim.
"""
from dataclasses import dataclass
from datetime import datetime
import math
import re

from memory_proof import CITATION, canonical, digest, source_key, validated_records


@dataclass(frozen=True)
class Context:
    repo: str
    owner: str
    profile: str
    ancestors: frozenset | None = None
    turn_limits: dict | None = None  # Fixture contract: persisted checkpoint extents required.


DEFAULTS = {"keyword_weight": 1.0, "semantic_weight": 0.0, "recency_boost": 0.0,
            "memory_limit": 5, "snippet_bytes": 96, "evidence_bytes": 16000, "max_slices": 80}


def settings(*layers):
    """Prototype-only machine -> repository -> request configuration merge."""
    resolved = dict(DEFAULTS)
    for layer in layers:
        if not layer:
            continue
        if set(layer) - set(DEFAULTS):
            raise ValueError("unknown setting")
        resolved.update(layer)
    for key in ("keyword_weight", "semantic_weight", "recency_boost"):
        value = resolved[key]
        if not isinstance(value, (int, float)) or isinstance(value, bool) or not math.isfinite(value) or value < 0:
            raise ValueError("invalid weight")
    if resolved["keyword_weight"] + resolved["semantic_weight"] == 0:
        raise ValueError("at least one retrieval weight must be nonzero")
    for key, minimum, maximum in (("memory_limit",0,100),("snippet_bytes",0,4096),("evidence_bytes",128,1048576),("max_slices",1,512)):
        if type(resolved[key]) is not int or not minimum <= resolved[key] <= maximum:
            raise ValueError("invalid budget or limit")
    return resolved


def utf8_window(text, start, end):
    raw = text.encode()
    start, end = max(0, start), min(len(raw), end)
    while start < len(raw) and raw[start] & 0xC0 == 0x80:
        start += 1
    while 0 < end < len(raw) and raw[end] & 0xC0 == 0x80:
        end -= 1
    if end < start:
        return "", start, start
    return raw[start:end].decode(), start, end


def permitted(source, context):
    if source.get("repo") != context.repo:
        return False
    if source.get("scope") == "repo_shared":
        return True
    return (source.get("scope") in {"repo_private", "personal", "account", "agent_private"}
            and source.get("owner") == context.owner and source.get("profile") == context.profile)


def citations(content):
    result = []
    for match in CITATION.finditer(content):
        sid, turn, start, end = match.groups()
        item = {"session_id": sid, "turn": int(turn),
                "start": int(start) if start else None, "end": int(end) if end else None}
        # Mechanical bridge to CURRENT evidence --anchor grammar; no agent rule needed.
        item["cli_anchor"] = f"{sid}:{turn}" + (f"@{start}-{end}" if start else "")
        result.append(item)
    return result


def stamp(value):
    if not value:
        return None
    try:
        parsed = datetime.fromisoformat(value.replace("Z", "+00:00"))
        return parsed.timestamp() if parsed.tzinfo is not None else None
    except (ValueError, TypeError):
        return None


class Consumer:
    def __init__(self, records, sessions=None, semantic=None):
        self.records, self.blobs, _ = validated_records(records)
        self.sessions = sessions or {}
        self.semantic = semantic or {}
        self.generation = digest(canonical(records))

    def handshake(self, supported_memory_schemas=(1,)):
        enabled = 1 in supported_memory_schemas
        return {"contract": "prototype/1", "memory": "available" if enabled else "unsupported",
                "memory_schema": 1, "session_search": True, "evidence": True,
                "memory_writeback": False, "index_generation": self.generation}

    def _record_allowed(self, record, context):
        return (permitted(record["source"], context) and
                (context.ancestors is None or record["observed_at_commit"] in context.ancestors))

    def _row(self, record, member):
        identity = digest(canonical([source_key(record["source"]),member["path"],member["blob"]]))
        return {"ref": "memory:"+identity, "kind": "memory", "source": record["source"],
                "path": member["path"], "blob": member["blob"], "snapshot_id": record["snapshot_id"],
                "content": self.blobs[member["blob"]].decode(),
                "source_modified_at": member["source_modified_at"], "observed_at": record["observed_at"],
                "observed_at_commit": record["observed_at_commit"], "trust": "retrieved_context"}

    def rows(self, context):
        latest = {}
        for record in self.records:
            if self._record_allowed(record, context):
                latest[source_key(record["source"])] = record
        return [self._row(record, member) for _,record in sorted(latest.items()) for member in record["files"]]

    def fetch_memory(self, ref, snapshot_id, context):
        for record in self.records:
            if record["snapshot_id"] != snapshot_id or not self._record_allowed(record, context):
                continue
            for member in record["files"]:
                row = self._row(record,member)
                if row["ref"] == ref:
                    return {"status":"ok",**row,"anchors":citations(row["content"]),"adopted":False}
        # Same response for missing, inaccessible and ineligible-as-of references.
        return {"status":"unavailable"}

    def search(self, query, context, config=None, supported_memory_schemas=(1,)):
        cfg = settings(config)
        tokens = set(re.findall(r"\w+",query.lower()))
        def lexical(text):
            return len(tokens & set(re.findall(r"\w+",text.lower()))) / max(1,len(tokens))
        # Separate result blocks: memory tuning never changes raw-session ranking.
        raw = []
        for sid, session in self.sessions.items():
            if not permitted(session["source"],context):
                continue
            if context.ancestors is not None and session["commit"] not in context.ancestors:
                continue
            turns = session["turns"]
            if context.turn_limits is not None:
                turns = turns[:context.turn_limits.get(sid,0)]
            score = max((lexical(text) for text in turns),default=0)
            if score > 0:
                raw.append({"session_id":sid,"score":score})
        raw.sort(key=lambda item:(-item["score"],item["session_id"]))
        if 1 not in supported_memory_schemas:
            return {"memories":[],"sessions":raw,"memory_status":"unsupported"}
        candidates = []
        for row in self.rows(context):
            kw = lexical(row["content"])
            sem = self.semantic.get(row["blob"])
            if sem is None:
                base = kw  # Fixture fallback: missing embeddings retain keyword search.
            else:
                total = cfg["keyword_weight"]+cfg["semantic_weight"]
                base = (cfg["keyword_weight"]*kw+cfg["semantic_weight"]*sem)/total
            if base <= 0:
                continue
            candidates.append({**row,"base":base,"native_time":stamp(row["source_modified_at"])})
        timestamps = [row["native_time"] for row in candidates if row["native_time"] is not None]
        low, high = (min(timestamps),max(timestamps)) if timestamps else (0,0)
        grouped = {}
        for row in candidates:
            recency = ((row["native_time"]-low)/(high-low)
                       if row["native_time"] is not None and high>low else 0)
            score = row["base"] + cfg["recency_boost"]*recency
            snippet,_,_ = utf8_window(row["content"],0,cfg["snippet_bytes"])
            hit = {k:row[k] for k in ("ref","kind","source","path","snapshot_id","source_modified_at","observed_at","observed_at_commit","trust")}
            hit.update(snippet=snippet,score=score,signals={"relevance":row["base"],"native_recency":recency},
                       score_is_confidence=False,origins=[{"ref":row["ref"],"source":row["source"]}])
            # Copies retain attribution without accumulating votes or score boosts.
            key = row["blob"]
            if key in grouped:
                previous=grouped[key]
                origins=previous["origins"]+hit["origins"]
                grouped[key] = hit if hit["score"]>previous["score"] else previous
                grouped[key]["origins"]=origins
            else:
                grouped[key]=hit
        hits=sorted(grouped.values(),key=lambda hit:(-hit["score"],hit["ref"]))
        return {"memories":hits[:cfg["memory_limit"]],"sessions":raw,"memory_status":"available"}

    def backlinks(self, session_id, turn, context, config=None):
        """One explicit, bounded reverse hop; no automatic graph traversal."""
        cfg=settings(config)
        session=self.sessions.get(session_id)
        if not session or not permitted(session["source"],context):
            return {"memories":[],"total":0,"truncated":False}
        if context.ancestors is not None and session["commit"] not in context.ancestors:
            return {"memories":[],"total":0,"truncated":False}
        if context.turn_limits is not None and turn>=context.turn_limits.get(session_id,0):
            return {"memories":[],"total":0,"truncated":False}
        hits=[]
        for row in self.rows(context):
            if any(a["session_id"]==session_id and a["turn"]==turn for a in citations(row["content"])):
                hits.append({"ref":row["ref"],"snapshot_id":row["snapshot_id"],"source":row["source"],
                             "relationship":"explicit_citation_not_verified_support"})
        hits.sort(key=lambda hit:hit["ref"])
        return {"memories":hits[:cfg["memory_limit"]],"total":len(hits),"truncated":len(hits)>cfg["memory_limit"]}

    def evidence(self, anchors, context, config=None):
        cfg=settings(config)
        groups, warnings = {}, []
        for anchor in anchors:
            sid,turn=anchor["session_id"],anchor["turn"]
            session=self.sessions.get(sid)
            if not session or not permitted(session["source"],context):
                warnings.append("source_unavailable");continue
            if context.ancestors is not None and session["commit"] not in context.ancestors:
                warnings.append("source_unavailable");continue
            if context.turn_limits is not None and turn>=context.turn_limits.get(sid,0):
                warnings.append("past_checkpoint_extent");continue
            if turn<0 or turn>=len(session["turns"]):
                warnings.append("source_unavailable");continue
            text=session["turns"][turn]; raw=text.encode()
            if anchor.get("expected_hash") and digest(raw)!=anchor["expected_hash"]:
                warnings.append("stale_source");continue
            start=anchor.get("start") if anchor.get("start") is not None else 0
            end=anchor.get("end") if anchor.get("end") is not None else len(raw)
            if not 0<=start<end<=len(raw):
                warnings.append("invalid_range");continue
            groups.setdefault((sid,turn),{"text":text,"ranges":[]})["ranges"].append((start,end))
        pending=[]
        for (sid,turn),group in sorted(groups.items()):
            merged=[]
            for start,end in sorted(group["ranges"]):
                if merged and start<=merged[-1][1]:
                    merged[-1]=(merged[-1][0],max(end,merged[-1][1]))
                else: merged.append((start,end))
            for start,end in merged:
                pending.append({"session_id":sid,"turn":turn,"start":start,"end":end,
                                "expected_hash":digest(group["text"].encode()),"text":group["text"]})
        delivered, remaining, used = [], [], 0
        for item in pending:
            capacity=cfg["evidence_bytes"]-used
            if capacity<=0 or len(delivered)>=cfg["max_slices"]:
                remaining.append({k:v for k,v in item.items() if k!="text"});continue
            text,start,end=utf8_window(item["text"],item["start"],min(item["end"],item["start"]+capacity))
            if text:
                delivered.append({"session_id":item["session_id"],"turn":item["turn"],"content":text,
                                  "start":start,"end":end,"citation":f'{item["session_id"]}:t{item["turn"]}:bytes{start}-{end}'})
                used+=len(text.encode())
            if end<item["end"]:
                remaining.append({k:(end if k=="start" else v) for k,v in item.items() if k!="text"})
        return {"slices":delivered,"content_bytes":used,"truncated":bool(remaining),
                "next":remaining,"warnings":sorted(set(warnings))}
