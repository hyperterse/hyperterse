-- Seed script for SQLite demo.
-- Run: sqlite3 demo.db < demo/seed-sqlite.sql
-- Or from demo/: sqlite3 demo.db < seed-sqlite.sql

CREATE TABLE IF NOT EXISTS notes (
  id INTEGER PRIMARY KEY,
  title TEXT NOT NULL,
  content TEXT,
  created_at TEXT DEFAULT (datetime('now'))
);

INSERT OR IGNORE INTO notes (id, title, content) VALUES
  (1, 'Welcome', 'This is the SQLite connector demo.'),
  (2, 'Getting started', 'Use the list-sqlite-notes tool to query notes.');
