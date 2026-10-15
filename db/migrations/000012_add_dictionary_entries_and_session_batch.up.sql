-- Keep dictionary senses and source links available for editorial review.
CREATE TABLE word_dictionary_entries (
    word_id UUID NOT NULL REFERENCES words(id) ON DELETE CASCADE,
    source_id TEXT NOT NULL REFERENCES lexical_sources(id),
    source_url TEXT NOT NULL,
    part_of_speech TEXT NOT NULL DEFAULT '',
    senses JSONB NOT NULL CHECK (jsonb_typeof(senses) = 'array'),
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    PRIMARY KEY (word_id, source_id)
);

-- An answer must use the same vocabulary release as its session.
ALTER TABLE learning_sessions ADD COLUMN vocabulary_batch TEXT NOT NULL DEFAULT 'release-2000';

INSERT INTO topics (slug,title,description,sort_order)
VALUES ('general-vocabulary','General vocabulary','A wider Dutch vocabulary, ordered by frequency. Topic review is ongoing.',100)
ON CONFLICT (slug) DO NOTHING;
