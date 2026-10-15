-- Batch-scoped content improvements. Original releases and session snapshots stay intact.
CREATE TABLE word_learning_content (
    batch_id UUID NOT NULL,
    word_id UUID NOT NULL,
    translation TEXT NOT NULL CHECK (btrim(translation) <> ''),
    meanings JSONB NOT NULL DEFAULT '[]' CHECK (jsonb_typeof(meanings) = 'array'),
    review_status TEXT NOT NULL DEFAULT 'needs_content_review' CHECK (review_status IN ('needs_content_review', 'approved')),
    revision TEXT NOT NULL,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    PRIMARY KEY (batch_id, word_id),
    FOREIGN KEY (batch_id, word_id) REFERENCES vocabulary_batch_words(batch_id, word_id) ON DELETE CASCADE
);
