CREATE TABLE lexical_sources (
    id TEXT PRIMARY KEY,
    title TEXT NOT NULL,
    creator TEXT NOT NULL DEFAULT '',
    source_url TEXT NOT NULL DEFAULT '',
    version TEXT NOT NULL DEFAULT '',
    license TEXT NOT NULL DEFAULT '',
    attribution TEXT NOT NULL DEFAULT '',
    safe_for_commercial_app BOOLEAN NOT NULL DEFAULT false,
    requires_attribution BOOLEAN NOT NULL DEFAULT true,
    share_alike_risk BOOLEAN NOT NULL DEFAULT false,
    notes TEXT NOT NULL DEFAULT '',
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

INSERT INTO lexical_sources (id, title, creator, source_url, version, license, attribution, safe_for_commercial_app, requires_attribution, share_alike_risk, notes)
VALUES
    ('opentaal-wordlist', 'Dutch word list', 'OpenTaal', 'https://github.com/OpenTaal/opentaal-wordlist', '2.20.23', 'BSD-3-Clause', 'Dutch word list by OpenTaal, https://opentaal.org', true, true, false, 'Official-spelling word universe already imported into words.'),
    ('wordfreq-nl', 'wordfreq Dutch frequency estimates', 'Robyn Speer and source contributors', 'https://github.com/rspeer/wordfreq', '3.0.2 data snapshot through about 2021', 'Apache-2.0 code; bundled data includes CC BY-SA and other attributed sources', 'wordfreq by Robyn Speer; see https://github.com/rspeer/wordfreq and bundled notices.', true, true, true, 'Use for ranking/frequency evidence. Preserve attribution and do not redistribute stripped data without notices.')
ON CONFLICT (id) DO UPDATE SET
    title = EXCLUDED.title,
    creator = EXCLUDED.creator,
    source_url = EXCLUDED.source_url,
    version = EXCLUDED.version,
    license = EXCLUDED.license,
    attribution = EXCLUDED.attribution,
    safe_for_commercial_app = EXCLUDED.safe_for_commercial_app,
    requires_attribution = EXCLUDED.requires_attribution,
    share_alike_risk = EXCLUDED.share_alike_risk,
    notes = EXCLUDED.notes,
    updated_at = now();

CREATE TABLE word_frequencies (
    word_id UUID NOT NULL REFERENCES words (id) ON DELETE CASCADE,
    source_id TEXT NOT NULL REFERENCES lexical_sources (id) ON DELETE CASCADE,
    frequency_rank INTEGER,
    zipf_score DOUBLE PRECISION NOT NULL,
    raw_score DOUBLE PRECISION,
    imported_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    PRIMARY KEY (word_id, source_id)
);

CREATE INDEX word_frequencies_source_rank_idx ON word_frequencies (source_id, frequency_rank);
CREATE INDEX word_frequencies_source_zipf_idx ON word_frequencies (source_id, zipf_score DESC);

CREATE TABLE topic_seed_words (
    topic_slug TEXT NOT NULL,
    lemma TEXT NOT NULL,
    source_id TEXT NOT NULL REFERENCES lexical_sources (id),
    confidence DOUBLE PRECISION NOT NULL DEFAULT 1,
    notes TEXT NOT NULL DEFAULT '',
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    PRIMARY KEY (topic_slug, lemma, source_id)
);

CREATE INDEX topic_seed_words_lemma_idx ON topic_seed_words (lemma);

CREATE OR REPLACE VIEW candidate_word_scores AS
SELECT
    w.id AS word_id,
    w.lemma,
    COALESCE(wlm.entry_type, 'unknown') AS entry_type,
    wlm.cefr_level,
    wf.frequency_rank,
    wf.zipf_score,
    COALESCE(wlm.usefulness_score, 0) AS heuristic_usefulness,
    COALESCE(wlm.complexity_score, 0) AS complexity_score,
    COALESCE(wlm.has_translation, false) AS has_translation,
    COALESCE(wlm.has_example, false) AS has_example,
    COALESCE(wlm.classification_confidence, 0) AS classification_confidence,
    COALESCE(wlm.is_learning_candidate, false) AS is_learning_candidate,
    COALESCE(array_remove(array_agg(DISTINCT ll.slug) FILTER (WHERE ld.slug = 'topic'), NULL), '{}') AS topic_labels,
    round((
        COALESCE(wf.zipf_score, 0) * 120
        + COALESCE(wlm.usefulness_score, 0) * 35
        - COALESCE(wlm.complexity_score, 0) * 18
        + CASE WHEN COALESCE(wlm.has_translation, false) THEN 80 ELSE 0 END
        + CASE WHEN COALESCE(wlm.has_example, false) THEN 80 ELSE 0 END
        + CASE COALESCE(wlm.entry_type, 'unknown')
            WHEN 'base_word' THEN 90
            WHEN 'compound' THEN 45
            WHEN 'multiword' THEN 35
            WHEN 'inflection' THEN -120
            WHEN 'proper_name' THEN -220
            WHEN 'abbreviation' THEN -180
            WHEN 'symbolic' THEN -260
            ELSE -80
          END
        + CASE WHEN COALESCE(wlm.cefr_level, '') IN ('A1', 'A2', 'B1', 'B2') THEN 50 ELSE 0 END
    )::numeric, 2) AS candidate_score
FROM words w
LEFT JOIN word_learning_metadata wlm ON wlm.word_id = w.id
LEFT JOIN word_frequencies wf ON wf.word_id = w.id AND wf.source_id = 'wordfreq-nl'
LEFT JOIN word_labels wl ON wl.word_id = w.id
LEFT JOIN learning_labels ll ON ll.id = wl.label_id
LEFT JOIN learning_dimensions ld ON ld.id = ll.dimension_id
GROUP BY w.id, w.lemma, wlm.entry_type, wlm.cefr_level, wf.frequency_rank, wf.zipf_score,
    wlm.usefulness_score, wlm.complexity_score, wlm.has_translation, wlm.has_example,
    wlm.classification_confidence, wlm.is_learning_candidate;
