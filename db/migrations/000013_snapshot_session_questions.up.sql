-- NULL identifies legacy sessions, which used the first ten batch words.
ALTER TABLE learning_sessions ADD COLUMN question_snapshot JSONB;
ALTER TABLE learning_sessions ADD CONSTRAINT learning_sessions_snapshot_shape
    CHECK (question_snapshot IS NULL OR
        (jsonb_typeof(question_snapshot) = 'array' AND jsonb_array_length(question_snapshot) = question_count));
