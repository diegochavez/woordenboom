ALTER TABLE typesafe_word_judgments
ADD COLUMN final_judgment JSONB NOT NULL DEFAULT '{}'::jsonb;
