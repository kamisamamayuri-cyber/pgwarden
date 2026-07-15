-- name: AuthServiceCreateSession :one
INSERT INTO sessions (
  user_id, token, ip, user_agent, groups, full_access
) VALUES (
  @user_id,
  pgp_sym_encrypt(@token::TEXT, @encryption_key::TEXT),
  @ip,
  @user_agent,
  @groups,
  @full_access
) RETURNING
  id,
  user_id,
  token,
  ip,
  user_agent,
  groups,
  full_access,
  created_at,
  pgp_sym_decrypt(token, @encryption_key::TEXT) AS decrypted_token;
