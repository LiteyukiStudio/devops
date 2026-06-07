DO $$
BEGIN
  IF to_regclass('public.o_id_c_auth_states') IS NOT NULL THEN
    EXECUTE '
      INSERT INTO oidc_auth_states (id, state_hash, nonce, provider_id, user_id, mode, redirect_path, expires_at, created_at, updated_at)
      SELECT id, state_hash, nonce, provider_id, user_id, mode, redirect_path, expires_at, created_at, updated_at
      FROM o_id_c_auth_states
      ON CONFLICT (id) DO NOTHING
    ';
  END IF;

  IF to_regclass('public.git_o_auth_states') IS NOT NULL THEN
    EXECUTE '
      INSERT INTO git_oauth_states (id, state_hash, provider_id, user_id, redirect_path, frontend_origin, expires_at, created_at, updated_at)
      SELECT id, state_hash, provider_id, user_id, redirect_path, frontend_origin, expires_at, created_at, updated_at
      FROM git_o_auth_states
      ON CONFLICT (id) DO NOTHING
    ';
  END IF;
END $$;
