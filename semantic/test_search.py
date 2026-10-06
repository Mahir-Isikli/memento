import os
from pathlib import Path
import sqlite3
import tempfile
import unittest

import search


class SemanticSearchTests(unittest.TestCase):
    @classmethod
    def setUpClass(cls):
        cls.model = search.Embedder(Path.home() / ".memento")

    def setUp(self):
        self.temp = tempfile.TemporaryDirectory()
        self.storage = Path(self.temp.name)
        source = sqlite3.connect(self.storage / "memento.db")
        source.executescript("""
            CREATE TABLE screenshots(id,timestamp,active_app,active_window_title,ocr_text);
            CREATE TABLE typing_sessions(id,start_time,active_app,active_window_title,text);
            CREATE TABLE dictations(source,source_id,timestamp,active_app,raw_text,formatted_text,edited_text,pasted_text);
            INSERT INTO dictations VALUES ('wispr-flow','email','2026-10-05T09:00:00Z','Mail','We sent the customer newsletter and announced the new website.','','','');
            INSERT INTO screenshots VALUES (1,'2026-10-05T10:00:00Z','Terminal','GPU','Configured a graphics driver for a compute server.');
            INSERT INTO typing_sessions VALUES (1,'2026-10-04T10:00:00Z','Notes','Lunch','Bought pasta and mushrooms for dinner.');
        """)
        source.close()
        self.index = search.connect_index(self.storage)

    def tearDown(self):
        self.index.close()
        self.temp.cleanup()

    def test_paraphrase_and_date_filter(self):
        self.assertEqual(search.index_records(self.storage, self.index, lambda: self.model), 3)
        hits = search.search(self.index, self.model, "sending marketing emails", 0, 2_000_000_000, 3)
        self.assertEqual(hits[0]["id"], "email")
        self.assertIn("We sent", hits[0]["excerpt"])
        empty = search.search(self.index, self.model, "newsletter", 0, search.timestamp("2026-10-04T00:00:00Z"), 3)
        self.assertEqual(empty, [])

    def test_cache_invalidation_and_removed_sources(self):
        self.assertEqual(search.index_records(self.storage, self.index, lambda: self.model), 3)
        def unexpected_model_load():
            raise AssertionError("unchanged indexing must not load a model")
        self.assertEqual(search.index_records(self.storage, self.index, unexpected_model_load), 0)
        source = sqlite3.connect(self.storage / "memento.db")
        source.execute("UPDATE dictations SET edited_text='We discussed conversion rates.'")
        source.commit()
        self.assertEqual(search.index_records(self.storage, self.index, lambda: self.model), 1)
        source.execute("DELETE FROM typing_sessions")
        source.commit()
        source.close()
        self.assertEqual(search.index_records(self.storage, self.index, unexpected_model_load), 0)
        self.assertEqual(self.index.execute("SELECT COUNT(*) FROM records").fetchone()[0], 2)

    def test_long_text_keeps_tail_and_original_case(self):
        text = "Boring filler. " * 400 + "Final topic: Qwen embeddings and semantic retrieval."
        chunks = list(self.model.chunks(text))
        self.assertGreater(len(chunks), 1)
        self.assertIn("Qwen embeddings", chunks[-1])
        self.assertTrue(chunks[0].startswith("Boring filler."))
        for chunk in chunks:
            self.model.tokenizer.no_truncation()
            self.assertLessEqual(len(self.model.tokenizer.encode(chunk).ids), 256)


if __name__ == "__main__":
    unittest.main()
