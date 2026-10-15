-- Add categories without rewriting release membership, content or learner state.
CREATE TABLE vocabulary_word_topics (
    batch_id UUID NOT NULL,
    word_id UUID NOT NULL,
    topic_slug TEXT NOT NULL REFERENCES topics(slug),
    translation TEXT NOT NULL CHECK (btrim(translation) <> ''),
    meanings JSONB NOT NULL CHECK (jsonb_typeof(meanings)='array' AND jsonb_array_length(meanings)>0),
    evidence TEXT NOT NULL,
    source TEXT NOT NULL,
    review_status TEXT NOT NULL CHECK (review_status IN ('needs_content_review','approved')),
    revision TEXT NOT NULL,
    content_revision TEXT NOT NULL,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    PRIMARY KEY(batch_id, word_id, topic_slug),
    FOREIGN KEY(batch_id, word_id) REFERENCES vocabulary_batch_words(batch_id, word_id) ON DELETE CASCADE,
    CHECK(topic_slug <> 'general-vocabulary')
);
CREATE INDEX vocabulary_word_topics_by_topic ON vocabulary_word_topics(batch_id,topic_slug,word_id);
-- A stale definition revision must be reclassified before it enters a category.
CREATE VIEW vocabulary_topic_memberships AS
SELECT bw.batch_id,bw.word_id,'general-vocabulary'::text AS topic_slug,NULL::text AS translation,NULL::jsonb AS meanings
FROM vocabulary_batch_words bw
UNION ALL
SELECT bw.batch_id,bw.word_id,bw.topic_slug,NULL::text,NULL::jsonb
FROM vocabulary_batch_words bw WHERE bw.topic_slug IS NOT NULL AND bw.topic_slug<>'general-vocabulary'
UNION ALL
SELECT m.batch_id,m.word_id,m.topic_slug,m.translation,m.meanings
FROM vocabulary_word_topics m
JOIN word_learning_content lc ON lc.batch_id=m.batch_id AND lc.word_id=m.word_id AND lc.revision=m.content_revision
JOIN vocabulary_batch_words bw ON bw.batch_id=m.batch_id AND bw.word_id=m.word_id
WHERE m.topic_slug IS DISTINCT FROM bw.topic_slug;
