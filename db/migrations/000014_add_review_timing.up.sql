ALTER TABLE user_word_progress ADD COLUMN review_streak INTEGER NOT NULL DEFAULT 0 CHECK (review_streak >= 0);
ALTER TABLE user_word_progress ADD COLUMN review_paused BOOLEAN NOT NULL DEFAULT false;
-- Conservative starting point for progress created before explicit review timing.
UPDATE user_word_progress SET review_streak = LEAST(correct_count, strength / 25, 5);
ALTER TABLE learning_events ADD COLUMN review_interval TEXT;
