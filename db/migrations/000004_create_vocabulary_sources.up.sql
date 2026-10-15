CREATE TABLE vocabulary_sources (
    id TEXT PRIMARY KEY,
    title TEXT NOT NULL,
    creator TEXT NOT NULL,
    source_url TEXT NOT NULL,
    version TEXT NOT NULL,
    commit_sha TEXT NOT NULL,
    license TEXT NOT NULL,
    content_sha256 TEXT NOT NULL,
    word_count INTEGER NOT NULL DEFAULT 0,
    imported_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE word_sources (
    word_id UUID NOT NULL REFERENCES words (id) ON DELETE CASCADE,
    source_id TEXT NOT NULL REFERENCES vocabulary_sources (id) ON DELETE CASCADE,
    source_entry_type TEXT NOT NULL DEFAULT 'complete-wordlist',
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    PRIMARY KEY (word_id, source_id)
);

CREATE INDEX word_sources_source_id_idx ON word_sources (source_id);
