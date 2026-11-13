#!/usr/bin/env python3
"""Create a synthetic real-codec body and an inspectable consumer journey."""
import json
from pathlib import Path
import subprocess
from consumer_model import Consumer, Context
from memory_proof import MAGIC,canonical,capture,decode,project_records

root=Path(__file__).resolve().parent
example=root/'example';example.mkdir(exist_ok=True)
probe=root/'frameprobe'
def run(*args):
    return subprocess.check_output([str(probe),*[str(a) for a in args]])
body=example/'rekal.body';dictionary=example/'dict.bin'
run('seed',body,dictionary)
source={'adapter':'claude','profile':'synthetic-alice','owner':'alice','repo':'demo','scope':'repo_shared','root':'memory'}
sid='01J'+'0'*23
text=(f'---\nmodified: 2026-04-02T10:00:00Z\n---\nUse idempotency keys for payment retries.\nSource: {sid}:t0\n').encode()
records=decode(capture(MAGIC,source,{'payment.md':text},'2026-11-07T12:00:00Z','1'*40))
payload=example/'memory-record.json';payload.write_bytes(canonical(records[0])+b'\n')
run('append',body,dictionary,body,dictionary,'synthetic-checkpoint-1','1'*40,payload)
saved=json.loads(run('extract',body))
raw=json.loads(run('session-text',body,dictionary))
engine=Consumer(saved,{sid:{'source':source,'commit':'1'*40,'turns':turns} for sid,turns in raw.items()})
ctx=Context('demo','bob','synthetic-bob')
search=engine.search('payment retries',ctx,{'snippet_bytes':64})
hit=search['memories'][0]
opened=engine.fetch_memory(hit['ref'],hit['snapshot_id'],ctx)
evidence=engine.evidence(opened['anchors'],ctx,{'evidence_bytes':128})
reverse=engine.backlinks(sid,0,ctx)
trace={'notice':'Synthetic prototype. Actual Rekal codec/session bytes; modeled consumer interface, no native writeback.',
       'steps':[{'step':'discover support','response':engine.handshake()},
                {'step':'search','query':'payment retries','response':search},
                {'step':'open exact memory revision','response':opened},
                {'step':'follow citation into bounded raw evidence','response':evidence},
                {'step':'look back from session to citing memory','response':reverse},
                {'step':'agent judges whether to adopt','native_memory_changed':False}]}
(example/'consumer-journey.json').write_text(json.dumps(trace,indent=2,ensure_ascii=False)+'\n')
(example/'rebuilt-index.json').write_text(json.dumps(project_records(saved),indent=2)+'\n')
print(json.dumps({'example_body_bytes':body.stat().st_size,'memory_frames':len(saved),'raw_evidence':evidence['slices'][0]['content'],'journey':str(example/'consumer-journey.json')}))
