ALTER TABLE learning_sessions
    ADD COLUMN topic_slugs TEXT[] NOT NULL DEFAULT '{}',
    ADD COLUMN cefr_levels TEXT[] NOT NULL DEFAULT '{}',
    ADD COLUMN mode TEXT NOT NULL DEFAULT 'balanced'
        CHECK (mode IN ('balanced', 'quick_win', 'topic_focus'));

UPDATE learning_sessions SET topic_slugs = ARRAY[topic_slug] WHERE topic_slug <> '';
