CREATE TABLE typesafe_word_judgments (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    word_id UUID NOT NULL REFERENCES words (id) ON DELETE CASCADE,
    classifier_run_id UUID NOT NULL REFERENCES classifier_runs (id) ON DELETE CASCADE,
    model_name TEXT NOT NULL,
    request_state JSONB NOT NULL,
    raw_answers JSONB NOT NULL,
    input_tokens INTEGER NOT NULL DEFAULT 0,
    output_tokens INTEGER NOT NULL DEFAULT 0,
    applied BOOLEAN NOT NULL DEFAULT false,
    needs_review BOOLEAN NOT NULL DEFAULT false,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX typesafe_word_judgments_word_created_idx ON typesafe_word_judgments (word_id, created_at DESC);
CREATE INDEX typesafe_word_judgments_run_idx ON typesafe_word_judgments (classifier_run_id);
