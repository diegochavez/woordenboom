DROP INDEX IF EXISTS word_examples_source_idx;
DROP INDEX IF EXISTS word_translations_source_idx;

ALTER TABLE word_translations
    DROP COLUMN IF EXISTS updated_at,
    DROP COLUMN IF EXISTS source;
