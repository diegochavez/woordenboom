ALTER TABLE learning_sessions DROP COLUMN vocabulary_batch;
DROP TABLE word_dictionary_entries;
-- Leave the general category in place if content already references it.
