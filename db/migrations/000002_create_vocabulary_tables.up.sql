CREATE TABLE topics (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    slug TEXT NOT NULL UNIQUE,
    title TEXT NOT NULL,
    description TEXT NOT NULL DEFAULT '',
    sort_order INTEGER NOT NULL DEFAULT 0,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE words (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    lemma TEXT NOT NULL UNIQUE,
    display_word TEXT NOT NULL,
    language_code TEXT NOT NULL DEFAULT 'nl',
    part_of_speech TEXT,
    frequency_rank INTEGER,
    frequency_score DOUBLE PRECISION,
    difficulty_level INTEGER,
    source TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX words_frequency_rank_idx ON words (frequency_rank);

CREATE TABLE word_translations (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    word_id UUID NOT NULL REFERENCES words (id) ON DELETE CASCADE,
    language_code TEXT NOT NULL DEFAULT 'en',
    translation TEXT NOT NULL,
    sort_order INTEGER NOT NULL DEFAULT 0,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX word_translations_word_id_idx ON word_translations (word_id);

CREATE TABLE word_examples (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    word_id UUID NOT NULL REFERENCES words (id) ON DELETE CASCADE,
    dutch_sentence TEXT NOT NULL,
    english_sentence TEXT,
    source TEXT,
    sort_order INTEGER NOT NULL DEFAULT 0,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX word_examples_word_id_idx ON word_examples (word_id);

CREATE TABLE word_topics (
    word_id UUID NOT NULL REFERENCES words (id) ON DELETE CASCADE,
    topic_id UUID NOT NULL REFERENCES topics (id) ON DELETE CASCADE,
    confidence DOUBLE PRECISION,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    PRIMARY KEY (word_id, topic_id)
);
