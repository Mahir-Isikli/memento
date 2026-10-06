"""On-demand local embeddings with an incremental SQLite index."""

import argparse
import datetime as dt
import hashlib
import json
import os
from pathlib import Path
import re
import resource
import sqlite3
import time
import urllib.request

MODEL = "Xenova/all-MiniLM-L6-v2"
REVISION = "751bff37182d3f1213fa05d7196b954e230abad9"
FILES = ("tokenizer.json", "onnx/model_quantized.onnx")
FINGERPRINT = f"{MODEL}:{REVISION}:254:32:v2"


def timestamp(value):
    return dt.datetime.fromisoformat(value.replace("Z", "+00:00")).timestamp()


def connect_index(storage):
    db = sqlite3.connect(storage / "semantic.sqlite", timeout=30)
    db.executescript("""
        PRAGMA journal_mode=WAL;
        CREATE TABLE IF NOT EXISTS records (
            source TEXT, record_id TEXT, hash TEXT, timestamp REAL, app TEXT,
            title TEXT, PRIMARY KEY(source,record_id));
        CREATE TABLE IF NOT EXISTS chunks (
            source TEXT, record_id TEXT, position INTEGER, text TEXT, vector BLOB,
            PRIMARY KEY(source,record_id,position));
        CREATE INDEX IF NOT EXISTS record_time ON records(timestamp);
    """)
    return db


def source_records(storage):
    source = sqlite3.connect((storage / "memento.db").as_uri() + "?mode=ro", uri=True, timeout=30)
    try:
        for sid, when, app, title, text in source.execute("SELECT id,CAST(timestamp AS TEXT),COALESCE(active_app,''),COALESCE(active_window_title,''),COALESCE(ocr_text,'') FROM screenshots"):
            yield "screenshot", str(sid), timestamp(when), app, title, text
        for sid, when, app, title, text in source.execute("SELECT id,CAST(start_time AS TEXT),COALESCE(active_app,''),COALESCE(active_window_title,''),text FROM typing_sessions"):
            yield "typing", str(sid), timestamp(when), app, title, text
        for src, sid, when, app, raw, formatted, edited, pasted in source.execute("SELECT source,source_id,CAST(timestamp AS TEXT),active_app,raw_text,formatted_text,edited_text,pasted_text FROM dictations"):
            variants = list(dict.fromkeys(v.strip() for v in (raw, formatted, pasted, edited) if v.strip()))
            yield src, sid, timestamp(when), app, "Dictation", "\n".join(variants)
    finally:
        source.close()


class Embedder:
    def __init__(self, storage):
        import numpy as np
        import onnxruntime as ort
        from tokenizers import Tokenizer

        model = storage / "models" / "minilm-l6-int8"
        for name in FILES:
            if not (model / name).is_file():
                raise RuntimeError("Local model is missing. Run memento semantic setup first.")
        self.np = np
        self.tokenizer = Tokenizer.from_file(str(model / "tokenizer.json"))
        options = ort.SessionOptions()
        options.intra_op_num_threads = 1
        options.inter_op_num_threads = 1
        options.execution_mode = ort.ExecutionMode.ORT_SEQUENTIAL
        options.log_severity_level = 3
        self.session = ort.InferenceSession(str(model / "onnx/model_quantized.onnx"), options, providers=["CPUExecutionProvider"])

    def chunks(self, text):
        self.tokenizer.no_truncation()
        self.tokenizer.no_padding()
        encoded = self.tokenizer.encode(text, add_special_tokens=False)
        ids = encoded.ids
        for start in range(0, len(ids), 222):
            end = min(start + 254, len(ids))
            yield text[encoded.offsets[start][0]:encoded.offsets[end - 1][1]]
            if start + 254 >= len(ids):
                break

    def encode(self, texts):
        np = self.np
        self.tokenizer.enable_truncation(max_length=256)
        self.tokenizer.enable_padding(pad_id=0, pad_token="[PAD]")
        encoded = self.tokenizer.encode_batch(texts)
        values = {
            "input_ids": np.asarray([e.ids for e in encoded], dtype=np.int64),
            "attention_mask": np.asarray([e.attention_mask for e in encoded], dtype=np.int64),
            "token_type_ids": np.asarray([e.type_ids for e in encoded], dtype=np.int64),
        }
        inputs = {i.name: values[i.name] for i in self.session.get_inputs()}
        hidden = self.session.run(None, inputs)[0]
        mask = values["attention_mask"][..., None]
        vectors = (hidden * mask).sum(axis=1) / mask.sum(axis=1).clip(min=1)
        vectors /= np.linalg.norm(vectors, axis=1, keepdims=True).clip(min=1e-12)
        return vectors.astype(np.float32)


def index_records(storage, db, get_embedder):
    pending = []
    live = set()
    for source, sid, when, app, title, text in source_records(storage):
        if not text.strip() and not title.strip():
            continue
        live.add((source, sid))
        content = f"{app}\n{title}\n{text}"
        digest = hashlib.sha256((FINGERPRINT + content).encode()).hexdigest()
        old = db.execute("SELECT hash FROM records WHERE source=? AND record_id=?", (source, sid)).fetchone()
        if old is None or old[0] != digest:
            pending.append((source, sid, when, app, title, content, digest))
    for source, sid, when, app, title, content, digest in pending:
        model = get_embedder()
        chunks = list(model.chunks(content))
        with db:
            db.execute("DELETE FROM chunks WHERE source=? AND record_id=?", (source, sid))
            for start in range(0, len(chunks), 8):
                batch = chunks[start:start + 8]
                for offset, (text, vector) in enumerate(zip(batch, model.encode(batch))):
                    db.execute("INSERT INTO chunks VALUES (?,?,?,?,?)", (source, sid, start + offset, text, vector.tobytes()))
            db.execute("INSERT OR REPLACE INTO records VALUES (?,?,?,?,?,?)", (source, sid, digest, when, app, title))
    # Remove derived vectors for records removed from the source archive.
    for source, sid in db.execute("SELECT source,record_id FROM records").fetchall():
        if (source, sid) not in live:
            with db:
                db.execute("DELETE FROM records WHERE source=? AND record_id=?", (source, sid))
                db.execute("DELETE FROM chunks WHERE source=? AND record_id=?", (source, sid))
    return len(pending)


def search(db, model, query, begin, end, limit):
    np = model.np
    query_vector = model.encode([query])[0]
    terms = set(re.findall(r"\w+", query.lower()))
    best = {}
    cursor = db.execute("""SELECT r.source,r.record_id,r.timestamp,r.app,r.title,c.text,c.vector
        FROM chunks c JOIN records r USING(source,record_id)
        WHERE r.timestamp BETWEEN ? AND ?""", (begin, end))
    while rows := cursor.fetchmany(512):
        vectors = np.stack([np.frombuffer(r[-1], dtype=np.float32) for r in rows])
        similarities = vectors @ query_vector
        for row, similarity in zip(rows, similarities):
            src, sid, when, app, title, text, _ = row
            words = set(re.findall(r"\w+", f"{app}\n{title}\n{text}".lower()))
            lexical = len(terms & words) / max(1, len(terms))
            score = 0.85 * float(similarity) + 0.15 * lexical
            key = (src, sid)
            if key not in best or score > best[key]["score"]:
                best[key] = {"source": src, "id": sid, "timestamp": dt.datetime.fromtimestamp(when, dt.timezone.utc).isoformat(),
                             "app": app, "title": title, "excerpt": text, "similarity": round(float(similarity), 4), "score": round(score, 4)}
    return sorted(best.values(), key=lambda r: r["score"], reverse=True)[:limit]


def setup(storage):
    target = storage / "models" / "minilm-l6-int8"
    for name in FILES:
        path = target / name
        if not path.is_file():
            path.parent.mkdir(parents=True, exist_ok=True)
            request = urllib.request.Request(f"https://huggingface.co/{MODEL}/resolve/{REVISION}/{name}", headers={"User-Agent": "memento-local-semantic/0.1"})
            temporary = path.with_suffix(path.suffix + ".part")
            with urllib.request.urlopen(request, timeout=60) as response, temporary.open("wb") as output:
                while block := response.read(1024 * 1024):
                    output.write(block)
            temporary.replace(path)
    return {"model": MODEL, "revision": REVISION, "model_bytes": sum((target / n).stat().st_size for n in FILES)}


def main():
    parser = argparse.ArgumentParser()
    parser.add_argument("mode", choices=["setup", "index", "status", "search"])
    parser.add_argument("--storage", type=Path, required=True)
    parser.add_argument("--query", default="")
    parser.add_argument("--from", dest="begin", default="1970-01-01T00:00:00Z")
    parser.add_argument("--to", dest="end", default="2100-01-01T00:00:00Z")
    parser.add_argument("--limit", type=int, default=10)
    parser.add_argument("--json", action="store_true")
    args = parser.parse_args()
    if args.limit < 1 or args.limit > 1000:
        parser.error("limit must be between 1 and 1,000")
    started = time.perf_counter()
    storage = args.storage.expanduser().resolve()
    os.environ["TOKENIZERS_PARALLELISM"] = "false"
    os.environ["OMP_NUM_THREADS"] = "1"
    os.environ["OPENBLAS_NUM_THREADS"] = "1"
    if args.mode == "setup":
        result = setup(storage)
    else:
        db = connect_index(storage)
        model = None

        def get_embedder():
            nonlocal model
            if model is None:
                model = Embedder(storage)
            return model

        updated = 0
        if args.mode in ("index", "search"):
            updated = index_records(storage, db, get_embedder)
        result = {"model": MODEL, "updated_records": updated,
                  "indexed_records": db.execute("SELECT COUNT(*) FROM records").fetchone()[0],
                  "indexed_chunks": db.execute("SELECT COUNT(*) FROM chunks").fetchone()[0]}
        if args.mode == "search":
            results = search(db, get_embedder(), args.query, timestamp(args.begin), timestamp(args.end), args.limit)
            result.update(query=args.query, count=len(results), results=results)
        db.close()
    result["elapsed_seconds"] = round(time.perf_counter() - started, 3)
    result["peak_memory_bytes"] = resource.getrusage(resource.RUSAGE_SELF).ru_maxrss
    if args.json or args.mode != "search":
        print(json.dumps(result, ensure_ascii=False))
    else:
        for hit in result["results"]:
            print(f"[{hit['timestamp']}] {hit['app']} ({hit['source']}, similarity {hit['similarity']})\n{hit['excerpt']}\n")


if __name__ == "__main__":
    main()
