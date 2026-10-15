CREATE TABLE user_profiles (
    user_id UUID PRIMARY KEY REFERENCES users (id) ON DELETE CASCADE,
    display_name TEXT NOT NULL,
    age_range TEXT NOT NULL,
    daily_goal INTEGER NOT NULL DEFAULT 10 CHECK (daily_goal BETWEEN 5 AND 100),
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE learning_sessions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL REFERENCES users (id) ON DELETE CASCADE,
    topic_slug TEXT NOT NULL,
    question_count INTEGER NOT NULL DEFAULT 10,
    correct_count INTEGER NOT NULL DEFAULT 0,
    started_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    completed_at TIMESTAMPTZ
);

CREATE INDEX learning_sessions_user_started_idx ON learning_sessions (user_id, started_at DESC);

CREATE TABLE user_word_progress (
    user_id UUID NOT NULL REFERENCES users (id) ON DELETE CASCADE,
    word_key TEXT NOT NULL,
    topic_slug TEXT NOT NULL,
    status TEXT NOT NULL DEFAULT 'learning' CHECK (status IN ('new', 'learning', 'review', 'mastered')),
    strength INTEGER NOT NULL DEFAULT 0 CHECK (strength BETWEEN 0 AND 100),
    correct_count INTEGER NOT NULL DEFAULT 0,
    wrong_count INTEGER NOT NULL DEFAULT 0,
    interval_days INTEGER NOT NULL DEFAULT 0,
    lapses INTEGER NOT NULL DEFAULT 0,
    last_seen_at TIMESTAMPTZ,
    next_review_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    PRIMARY KEY (user_id, word_key)
);

CREATE INDEX user_word_progress_due_idx ON user_word_progress (user_id, next_review_at);
CREATE INDEX user_word_progress_topic_idx ON user_word_progress (user_id, topic_slug);

CREATE TABLE learning_events (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    session_id UUID NOT NULL REFERENCES learning_sessions (id) ON DELETE CASCADE,
    user_id UUID NOT NULL REFERENCES users (id) ON DELETE CASCADE,
    question_id TEXT NOT NULL,
    word_key TEXT NOT NULL,
    exercise_type TEXT NOT NULL,
    answer TEXT NOT NULL,
    correct_answer TEXT NOT NULL,
    was_correct BOOLEAN NOT NULL,
    response_ms INTEGER,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (session_id, question_id)
);

CREATE INDEX learning_events_user_created_idx ON learning_events (user_id, created_at DESC);
