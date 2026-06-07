UPDATE git_providers
SET scope = 'global', owner_ref = ''
WHERE scope = 'user' AND COALESCE(owner_ref, '') = '';

UPDATE git_accounts
SET scope = 'global', owner_ref = ''
WHERE COALESCE(scope, 'user') = 'user'
  AND COALESCE(owner_ref, '') = ''
  AND access_scope = 'provider';

UPDATE git_accounts
SET scope = 'user', owner_ref = user_id
WHERE COALESCE(scope, 'user') = 'user'
  AND COALESCE(owner_ref, '') = ''
  AND access_scope = 'personal';
