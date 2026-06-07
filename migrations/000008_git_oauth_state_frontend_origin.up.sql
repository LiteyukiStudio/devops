ALTER TABLE git_oauth_states
  ADD COLUMN IF NOT EXISTS frontend_origin text NOT NULL DEFAULT '';
