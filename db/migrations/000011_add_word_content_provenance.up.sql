ALTER TABLE word_translations
    ADD COLUMN source TEXT,
    ADD COLUMN updated_at TIMESTAMPTZ NOT NULL DEFAULT now();

CREATE INDEX word_translations_source_idx ON word_translations (source);
CREATE INDEX word_examples_source_idx ON word_examples (source);
