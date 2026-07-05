CREATE TABLE IF NOT EXISTS step_up_assertions (
  id text PRIMARY KEY,
  user_id text NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  session_id text NOT NULL REFERENCES user_sessions(id) ON DELETE CASCADE,
  purpose text NOT NULL,
  expires_at timestamptz NOT NULL,
  created_at timestamptz NOT NULL DEFAULT now(),
  updated_at timestamptz NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_step_up_assertions_user_id ON step_up_assertions(user_id);
CREATE INDEX IF NOT EXISTS idx_step_up_assertions_session_id ON step_up_assertions(session_id);
CREATE INDEX IF NOT EXISTS idx_step_up_assertions_purpose ON step_up_assertions(purpose);
CREATE INDEX IF NOT EXISTS idx_step_up_assertions_expires_at ON step_up_assertions(expires_at);
CREATE UNIQUE INDEX IF NOT EXISTS idx_step_up_assertions_session_purpose ON step_up_assertions(session_id, purpose);
