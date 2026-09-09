DROP INDEX IF EXISTS idx_custom_agents_visibility;
DROP INDEX IF EXISTS idx_knowledge_bases_visibility;

ALTER TABLE custom_agents
    DROP CONSTRAINT IF EXISTS custom_agents_visibility_check;
ALTER TABLE knowledge_bases
    DROP CONSTRAINT IF EXISTS knowledge_bases_visibility_check;

ALTER TABLE custom_agents DROP COLUMN IF EXISTS visibility;
ALTER TABLE knowledge_bases DROP COLUMN IF EXISTS visibility;

-- Classification rows stay in the system scope on rollback. Reassigning them
-- to an arbitrary tenant would be lossy in multi-administrator deployments.
