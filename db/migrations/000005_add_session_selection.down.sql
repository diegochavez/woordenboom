ALTER TABLE learning_sessions
    DROP COLUMN IF EXISTS mode,
    DROP COLUMN IF EXISTS cefr_levels,
    DROP COLUMN IF EXISTS topic_slugs;
