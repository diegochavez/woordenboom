CREATE TABLE classifier_runs (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    classifier_name TEXT NOT NULL,
    classifier_version TEXT NOT NULL,
    model_name TEXT,
    source_id TEXT REFERENCES vocabulary_sources (id),
    status TEXT NOT NULL DEFAULT 'running' CHECK (status IN ('running', 'completed', 'failed')),
    words_processed INTEGER NOT NULL DEFAULT 0,
    started_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    completed_at TIMESTAMPTZ,
    notes TEXT
);

CREATE TABLE learning_dimensions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    slug TEXT NOT NULL UNIQUE,
    title TEXT NOT NULL,
    sort_order INTEGER NOT NULL DEFAULT 0,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE learning_labels (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    dimension_id UUID NOT NULL REFERENCES learning_dimensions (id) ON DELETE CASCADE,
    slug TEXT NOT NULL,
    title TEXT NOT NULL,
    description TEXT NOT NULL DEFAULT '',
    sort_order INTEGER NOT NULL DEFAULT 0,
    color TEXT,
    is_active BOOLEAN NOT NULL DEFAULT true,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (dimension_id, slug)
);

CREATE TABLE word_learning_metadata (
    word_id UUID PRIMARY KEY REFERENCES words (id) ON DELETE CASCADE,
    entry_type TEXT NOT NULL CHECK (entry_type IN ('base_word', 'inflection', 'compound', 'proper_name', 'abbreviation', 'multiword', 'symbolic', 'unknown')),
    cefr_level TEXT CHECK (cefr_level IN ('A1', 'A2', 'B1', 'B2', 'C1', 'C2')),
    usage_rank INTEGER,
    usage_score DOUBLE PRECISION,
    learner_priority INTEGER NOT NULL DEFAULT 0,
    usefulness_score DOUBLE PRECISION NOT NULL DEFAULT 0,
    complexity_score DOUBLE PRECISION NOT NULL DEFAULT 0,
    is_learning_candidate BOOLEAN NOT NULL DEFAULT false,
    has_translation BOOLEAN NOT NULL DEFAULT false,
    has_example BOOLEAN NOT NULL DEFAULT false,
    classification_status TEXT NOT NULL DEFAULT 'classified' CHECK (classification_status IN ('classified', 'needs_frequency', 'needs_review', 'rejected')),
    classification_confidence DOUBLE PRECISION NOT NULL DEFAULT 0,
    classifier_run_id UUID REFERENCES classifier_runs (id),
    classified_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX word_learning_metadata_cefr_idx ON word_learning_metadata (cefr_level);
CREATE INDEX word_learning_metadata_candidate_priority_idx ON word_learning_metadata (is_learning_candidate, learner_priority DESC);
CREATE INDEX word_learning_metadata_entry_type_idx ON word_learning_metadata (entry_type);

CREATE TABLE word_labels (
    word_id UUID NOT NULL REFERENCES words (id) ON DELETE CASCADE,
    label_id UUID NOT NULL REFERENCES learning_labels (id) ON DELETE CASCADE,
    confidence DOUBLE PRECISION NOT NULL DEFAULT 0,
    priority_score INTEGER NOT NULL DEFAULT 0,
    classifier_run_id UUID REFERENCES classifier_runs (id),
    needs_review BOOLEAN NOT NULL DEFAULT false,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    PRIMARY KEY (word_id, label_id)
);

CREATE INDEX word_labels_label_priority_idx ON word_labels (label_id, priority_score DESC);

INSERT INTO learning_dimensions (slug, title, sort_order) VALUES
    ('topic', 'Topic', 10),
    ('cefr', 'CEFR level', 20),
    ('entry_type', 'Entry type', 30)
ON CONFLICT (slug) DO NOTHING;

INSERT INTO learning_labels (dimension_id, slug, title, description, sort_order)
SELECT d.id, item.slug, item.title, item.description, item.sort_order
FROM learning_dimensions d
JOIN (VALUES
    ('daily-basics', 'Daily Basics', 'Core words for everyday interaction.', 10),
    ('housing-renting', 'Housing & Renting', 'Contracts, repairs, utilities, and moving.', 20),
    ('supermarket-food', 'Supermarket & Food', 'Shopping, ingredients, meals, and labels.', 30),
    ('public-transport', 'Public Transport', 'Trains, trams, buses, tickets, and delays.', 40),
    ('healthcare', 'Healthcare', 'Appointments, symptoms, pharmacy, and insurance.', 50),
    ('government-municipality', 'Government & Municipality', 'Forms, registration, taxes, and official letters.', 60),
    ('immigration-documents', 'Immigration & Documents', 'Permits, identity documents, and applications.', 70),
    ('work-office', 'Work & Office', 'Meetings, contracts, colleagues, and planning.', 80),
    ('banking-money', 'Banking & Money', 'Payments, accounts, invoices, and budgeting.', 90),
    ('social-life', 'Social Life', 'Invitations, relationships, hobbies, and small talk.', 100),
    ('weather-small-talk', 'Weather & Small Talk', 'Everyday conversation starters.', 110),
    ('home-repair', 'Home Repair', 'Tools, leaks, electricity, paint, and furniture.', 120),
    ('emergencies', 'Emergencies', 'Urgent help, safety, and reporting problems.', 130),
    ('school-childcare', 'School & Childcare', 'Children, school communication, and care.', 140),
    ('politics-news', 'Politics & News', 'Government, elections, society, and media.', 150),
    ('engineering-technical', 'Engineering & Technical Work', 'Systems, materials, measurements, and maintenance.', 160),
    ('software-it', 'Software & IT', 'Digital work, support, security, and products.', 170),
    ('legal-contracts', 'Legal & Contracts', 'Agreements, rights, obligations, and disputes.', 180),
    ('utilities-bills', 'Utilities & Bills', 'Energy, water, internet, invoices, and meters.', 190),
    ('appointments-phone', 'Appointments & Phone Calls', 'Planning, rescheduling, calls, and confirmations.', 200)
) AS item(slug, title, description, sort_order) ON d.slug = 'topic'
ON CONFLICT (dimension_id, slug) DO UPDATE SET title = EXCLUDED.title, description = EXCLUDED.description, sort_order = EXCLUDED.sort_order, updated_at = now();

INSERT INTO learning_labels (dimension_id, slug, title, description, sort_order)
SELECT d.id, item.slug, item.title, item.description, item.sort_order
FROM learning_dimensions d
JOIN (VALUES
    ('A1', 'A1', 'First practical words.', 10),
    ('A2', 'A2', 'Everyday survival words.', 20),
    ('B1', 'B1', 'Independent daily-life words.', 30),
    ('B2', 'B2', 'Work and abstract vocabulary.', 40),
    ('C1', 'C1', 'Advanced vocabulary.', 50),
    ('C2', 'C2', 'Rare, formal, or specialized vocabulary.', 60)
) AS item(slug, title, description, sort_order) ON d.slug = 'cefr'
ON CONFLICT (dimension_id, slug) DO UPDATE SET title = EXCLUDED.title, description = EXCLUDED.description, sort_order = EXCLUDED.sort_order, updated_at = now();

INSERT INTO learning_labels (dimension_id, slug, title, description, sort_order)
SELECT d.id, item.slug, item.title, item.description, item.sort_order
FROM learning_dimensions d
JOIN (VALUES
    ('base_word', 'Base word', 'Likely standalone learner word.', 10),
    ('inflection', 'Inflection', 'Likely plural, conjugated, or declined form.', 20),
    ('compound', 'Compound', 'Likely compound word.', 30),
    ('proper_name', 'Proper name', 'Likely name or place.', 40),
    ('abbreviation', 'Abbreviation', 'Likely abbreviation or acronym.', 50),
    ('multiword', 'Multiword expression', 'Entry contains a space.', 60),
    ('symbolic', 'Symbolic entry', 'Entry contains digits or symbols.', 70),
    ('unknown', 'Unknown', 'Needs review.', 80)
) AS item(slug, title, description, sort_order) ON d.slug = 'entry_type'
ON CONFLICT (dimension_id, slug) DO UPDATE SET title = EXCLUDED.title, description = EXCLUDED.description, sort_order = EXCLUDED.sort_order, updated_at = now();
