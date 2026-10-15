CREATE TABLE vocabulary_batches (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    slug TEXT NOT NULL UNIQUE,
    title TEXT NOT NULL,
    target_size INTEGER NOT NULL,
    status TEXT NOT NULL DEFAULT 'draft' CHECK (status IN ('draft', 'translation_pending', 'reviewing', 'approved', 'archived')),
    notes TEXT NOT NULL DEFAULT '',
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE vocabulary_batch_words (
    batch_id UUID NOT NULL REFERENCES vocabulary_batches (id) ON DELETE CASCADE,
    word_id UUID NOT NULL REFERENCES words (id) ON DELETE CASCADE,
    batch_rank INTEGER NOT NULL,
    topic_slug TEXT,
    topic_rank INTEGER,
    candidate_score DOUBLE PRECISION NOT NULL,
    frequency_rank INTEGER,
    zipf_score DOUBLE PRECISION,
    cefr_level TEXT,
    entry_type TEXT NOT NULL,
    review_status TEXT NOT NULL DEFAULT 'needs_translation' CHECK (review_status IN ('needs_translation', 'needs_topic_review', 'needs_content_review', 'approved', 'rejected')),
    needs_topic_review BOOLEAN NOT NULL DEFAULT false,
    notes TEXT NOT NULL DEFAULT '',
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    PRIMARY KEY (batch_id, word_id),
    UNIQUE (batch_id, batch_rank)
);

CREATE INDEX vocabulary_batch_words_topic_rank_idx ON vocabulary_batch_words (batch_id, topic_slug, topic_rank);
CREATE INDEX vocabulary_batch_words_review_status_idx ON vocabulary_batch_words (batch_id, review_status);
