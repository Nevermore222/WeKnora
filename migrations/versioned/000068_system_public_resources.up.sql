-- Promote administrator-managed classification trees to a system-wide catalog
-- and add explicit resource visibility scopes.

ALTER TABLE knowledge_bases
    ADD COLUMN IF NOT EXISTS visibility VARCHAR(20) NOT NULL DEFAULT 'workspace';
ALTER TABLE custom_agents
    ADD COLUMN IF NOT EXISTS visibility VARCHAR(20) NOT NULL DEFAULT 'workspace';

ALTER TABLE knowledge_bases
    ADD CONSTRAINT knowledge_bases_visibility_check
    CHECK (visibility IN ('workspace', 'public', 'private'));
ALTER TABLE custom_agents
    ADD CONSTRAINT custom_agents_visibility_check
    CHECK (visibility IN ('workspace', 'public', 'private'));

CREATE INDEX IF NOT EXISTS idx_knowledge_bases_visibility
    ON knowledge_bases (visibility) WHERE deleted_at IS NULL;
CREATE INDEX IF NOT EXISTS idx_custom_agents_visibility
    ON custom_agents (visibility) WHERE deleted_at IS NULL;

-- System-administrator-created resources are public by default after upgrade.
UPDATE knowledge_bases kb
   SET visibility = 'public'
  FROM users u
 WHERE kb.creator_id = u.id
   AND u.is_system_admin = TRUE
   AND kb.visibility = 'workspace';
UPDATE custom_agents agent
   SET visibility = 'public'
  FROM users u
 WHERE agent.created_by = u.id
   AND u.is_system_admin = TRUE
   AND agent.visibility = 'workspace';

-- The classification catalog is global. Preserve every existing ID so resource
-- classification_tags remain valid; the current administrator tenant owns the
-- only populated catalog in existing deployments.
UPDATE classification_modules m
   SET tenant_id = 0
 WHERE EXISTS (
    SELECT 1 FROM users u
     WHERE u.is_system_admin = TRUE
       AND u.tenant_id = m.tenant_id
 );
UPDATE classification_options o
   SET tenant_id = 0
 WHERE EXISTS (
    SELECT 1 FROM classification_modules m
     WHERE m.id = o.module_id
       AND m.tenant_id = 0
 );
