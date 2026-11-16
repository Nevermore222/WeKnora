<template>
  <div class="mcp-endpoint-settings">
    <div class="section-header">
      <h2>{{ $t('mcpEndpointSettings.title') }}</h2>
      <p class="section-description">{{ $t('mcpEndpointSettings.description') }}</p>
    </div>

    <section class="mcp-endpoint-tutorial" aria-labelledby="mcp-endpoint-tutorial-title">
      <div class="tutorial-header">
        <span class="tutorial-icon"><t-icon name="help-circle" /></span>
        <div>
          <h3 id="mcp-endpoint-tutorial-title">{{ $t('mcpEndpointSettings.tutorial.title') }}</h3>
          <p>{{ $t('mcpEndpointSettings.tutorial.description') }}</p>
        </div>
      </div>
      <div class="tutorial-steps">
        <div v-for="(step, index) in tutorialSteps" :key="step" class="tutorial-step">
          <span class="tutorial-step__number">{{ index + 1 }}</span>
          <div>
            <h4>{{ $t(`mcpEndpointSettings.tutorial.steps.${step}.title`) }}</h4>
            <p>{{ $t(`mcpEndpointSettings.tutorial.steps.${step}.description`) }}</p>
          </div>
        </div>
      </div>
      <div class="tutorial-config">
        <div class="tutorial-config__header">
          <strong>{{ $t('mcpEndpointSettings.tutorial.configTitle') }}</strong>
          <t-button variant="text" size="small" @click="copy(MCP_CONFIG_EXAMPLE)">
            <t-icon name="file-copy" />
            {{ $t('mcpEndpointSettings.tutorial.copyConfig') }}
          </t-button>
        </div>
        <pre><code>{{ MCP_CONFIG_EXAMPLE }}</code></pre>
      </div>
      <p class="tutorial-security">
        <t-icon name="secured" />
        {{ $t('mcpEndpointSettings.tutorial.security') }}
      </p>
    </section>

    <div class="endpoint-toolbar">
      <div>
        <h3>{{ $t('mcpEndpointSettings.publishedEndpoints') }}</h3>
        <p>{{ $t('mcpEndpointSettings.publishedEndpointsDesc') }}</p>
      </div>
      <t-button theme="primary" @click="openCreate">
        <template #icon><t-icon name="add" /></template>
        {{ $t('mcpEndpointSettings.create') }}
      </t-button>
    </div>

    <div v-if="loading" class="loading-container">
      <t-loading :text="$t('common.loading')" />
    </div>

    <t-empty v-else-if="endpoints.length === 0" :description="$t('mcpEndpointSettings.empty')" />

    <div v-else class="endpoint-grid">
      <div v-for="endpoint in endpoints" :key="endpoint.id" class="endpoint-card">
        <div class="endpoint-card__header">
          <div class="endpoint-card__title-wrap">
            <span class="endpoint-card__icon"><t-icon name="share" /></span>
            <div>
              <h3>{{ endpoint.name || $t('mcpEndpointSettings.unnamed') }}</h3>
              <p v-if="endpoint.description">{{ endpoint.description }}</p>
            </div>
          </div>
          <t-tag :theme="endpoint.enabled ? 'success' : 'default'" variant="light">
            {{ endpoint.enabled ? $t('mcpEndpointSettings.enabled') : $t('mcpEndpointSettings.disabled') }}
          </t-tag>
        </div>

        <div class="endpoint-card__url">
          <code>{{ publicUrl(endpoint) }}</code>
          <t-button variant="text" shape="square" size="small" @click="copy(publicUrl(endpoint))">
            <t-icon name="file-copy" />
          </t-button>
        </div>

        <div class="endpoint-card__meta">
          <span>{{ endpoint.knowledge_base_ids.length > 0
            ? $t('mcpEndpointSettings.scopedKnowledgeBases', { count: endpoint.knowledge_base_ids.length })
            : $t('mcpEndpointSettings.allKnowledgeBases') }}</span>
          <span>{{ $t('mcpEndpointSettings.toolsCount', { count: endpoint.tools.length }) }}</span>
          <span>{{ $t('mcpEndpointSettings.tokenHint', { hint: endpoint.token_hint || '—' }) }}</span>
        </div>

        <div class="endpoint-card__actions">
          <t-button size="small" variant="outline" @click="openEdit(endpoint)">{{ $t('common.edit') }}</t-button>
          <t-button size="small" variant="outline" @click="rotateToken(endpoint)">{{ $t('mcpEndpointSettings.rotateToken') }}</t-button>
          <t-button size="small" variant="text" theme="danger" @click="deleteEndpoint(endpoint)">{{ $t('common.delete') }}</t-button>
        </div>
      </div>
    </div>

    <div v-if="oneTimeToken" class="token-banner">
      <div>
        <strong>{{ $t('mcpEndpointSettings.tokenShownOnce') }}</strong>
        <p>{{ $t('mcpEndpointSettings.tokenCopyHint') }}</p>
        <code>{{ oneTimeToken }}</code>
      </div>
      <t-button theme="primary" variant="outline" @click="copy(oneTimeToken)">
        {{ $t('mcpEndpointSettings.copyToken') }}
      </t-button>
    </div>

    <t-dialog
      v-model:visible="dialogVisible"
      :header="editingId ? $t('mcpEndpointSettings.editTitle') : $t('mcpEndpointSettings.createTitle')"
      :confirm-btn="{ content: $t('common.save'), loading: saving }"
      width="760px"
      @confirm="saveEndpoint"
    >
      <div class="endpoint-form">
        <div class="form-item">
          <label>{{ $t('mcpEndpointSettings.name') }}</label>
          <t-input v-model="form.name" :placeholder="$t('mcpEndpointSettings.namePlaceholder')" />
        </div>
        <div class="form-item">
          <label>{{ $t('mcpEndpointSettings.descriptionLabel') }}</label>
          <t-textarea v-model="form.description" :placeholder="$t('mcpEndpointSettings.descriptionPlaceholder')" :autosize="{ minRows: 2, maxRows: 4 }" />
        </div>

        <div class="form-item form-item--inline">
          <div>
            <label>{{ $t('mcpEndpointSettings.enabledLabel') }}</label>
            <p>{{ $t('mcpEndpointSettings.enabledDesc') }}</p>
          </div>
          <t-switch v-model="form.enabled" />
        </div>

        <div class="form-item">
          <label>{{ $t('mcpEndpointSettings.knowledgeBases') }}</label>
          <p>{{ $t('mcpEndpointSettings.knowledgeBasesDesc') }}</p>
          <t-select
            v-model="form.knowledge_base_ids"
            multiple
            filterable
            :disabled="form.allKnowledgeBases"
            :placeholder="$t('mcpEndpointSettings.knowledgeBasesPlaceholder')"
          >
            <t-option v-for="kb in knowledgeBases" :key="kb.id" :value="String(kb.id)" :label="kb.name" />
          </t-select>
          <t-checkbox v-model="form.allKnowledgeBases">{{ $t('mcpEndpointSettings.allKnowledgeBasesOption') }}</t-checkbox>
        </div>

        <div class="form-item">
          <label>{{ $t('mcpEndpointSettings.defaultAgent') }}</label>
          <p>{{ $t('mcpEndpointSettings.defaultAgentDesc') }}</p>
          <t-select v-model="form.default_agent_id" clearable :placeholder="$t('mcpEndpointSettings.defaultAgentPlaceholder')">
            <t-option v-for="agent in agents" :key="agent.id" :value="agent.id" :label="agent.name" />
          </t-select>
        </div>

        <div class="form-item">
          <label>{{ $t('mcpEndpointSettings.tools') }}</label>
          <p>{{ $t('mcpEndpointSettings.toolsDesc') }}</p>
          <div v-for="group in toolGroups" :key="group" class="tool-group">
            <div class="tool-group__title">{{ toolGroupLabel(group) }}</div>
            <t-checkbox-group v-model="form.tools">
              <t-checkbox v-for="tool in toolsByGroup(group)" :key="tool.name" :value="tool.name">
                <span>{{ tool.name }}</span>
                <t-tag v-if="tool.destructive" theme="warning" variant="light" size="small">
                  {{ $t('mcpEndpointSettings.writeTool') }}
                </t-tag>
              </t-checkbox>
            </t-checkbox-group>
          </div>
        </div>

        <div class="form-item">
          <label>{{ $t('mcpEndpointSettings.rateLimit') }}</label>
          <p>{{ $t('mcpEndpointSettings.rateLimitDesc') }}</p>
          <t-input-number v-model="form.rate_limit_per_minute" :min="0" :max="10000" />
        </div>
      </div>
    </t-dialog>
  </div>
</template>

<script setup lang="ts">
import { computed, onMounted, reactive, ref } from 'vue'
import { MessagePlugin } from 'tdesign-vue-next'
import { useI18n } from 'vue-i18n'
import { listKnowledgeBases } from '@/api/knowledge-base'
import { listAgents, type CustomAgent } from '@/api/agent'
import {
  createMcpEndpoint,
  deleteMcpEndpoint,
  getMcpEndpointToolCatalog,
  listMcpEndpoints,
  rotateMcpEndpointToken,
  updateMcpEndpoint,
  type McpEndpoint,
  type McpEndpointPayload,
  type McpEndpointToolDefinition,
  type McpEndpointToolGroup,
} from '@/api/mcp-endpoint'
import { copyText } from '@/utils/clipboard'
import { buildMcpEndpointUrl, resolveMcpPublicOrigin } from '@/utils/mcp-endpoint-view'

interface KnowledgeBaseOption { id: string | number; name: string }

const endpoints = ref<McpEndpoint[]>([])
const { t } = useI18n()
const knowledgeBases = ref<KnowledgeBaseOption[]>([])
const agents = ref<CustomAgent[]>([])
const tools = ref<McpEndpointToolDefinition[]>([])
const toolGroups = ref<McpEndpointToolGroup[]>([])
const loading = ref(false)
const saving = ref(false)
const dialogVisible = ref(false)
const editingId = ref<string | null>(null)
const oneTimeToken = ref('')
const tutorialSteps = ['create', 'copy', 'configure', 'use'] as const
const MCP_CONFIG_EXAMPLE = `{
  "mcpServers": {
    "knavis-knowledge": {
      "url": "https://your-knavis.example.com/mcp/<endpoint-id>",
      "headers": {
        "Authorization": "Bearer <your-mcp-token>"
      }
    }
  }
}`

const form = reactive({
  name: '',
  description: '',
  enabled: true,
  allKnowledgeBases: true,
  knowledge_base_ids: [] as string[],
  default_agent_id: '',
  tools: [] as string[],
  rate_limit_per_minute: 30,
})

const defaultTools = computed(() => tools.value.filter((tool) => !tool.destructive).map((tool) => tool.name))

const toolGroupLabel = (group: McpEndpointToolGroup) => {
  const labels: Record<McpEndpointToolGroup, string> = {
    retrieve: 'mcpEndpointSettings.groups.retrieve',
    chat: 'mcpEndpointSettings.groups.chat',
    wiki: 'mcpEndpointSettings.groups.wiki',
    ingest: 'mcpEndpointSettings.groups.ingest',
  }
  return t(labels[group])
}

const toolsByGroup = (group: McpEndpointToolGroup) => tools.value.filter((tool) => tool.group === group)

const publicUrl = (endpoint: McpEndpoint) => {
  const runtimeOrigin = typeof window === 'undefined'
    ? ''
    : (window as Window & { __RUNTIME_CONFIG__?: { MCP_PUBLIC_ORIGIN?: string } }).__RUNTIME_CONFIG__?.MCP_PUBLIC_ORIGIN || ''
  const buildOrigin = import.meta.env.VITE_MCP_PUBLIC_ORIGIN || ''
  const browserOrigin = typeof window === 'undefined' ? '' : window.location.origin
  const origin = resolveMcpPublicOrigin(browserOrigin, runtimeOrigin || buildOrigin)
  return buildMcpEndpointUrl(origin, endpoint.path)
}

const errorMessage = (error: any, fallback: string) => error?.message || error?.error || fallback

const load = async () => {
  loading.value = true
  try {
    const [endpointRows, catalog, kbResponse, agentResponse] = await Promise.all([
      listMcpEndpoints(),
      getMcpEndpointToolCatalog(),
      listKnowledgeBases(),
      listAgents(),
    ])
    endpoints.value = endpointRows
    tools.value = catalog.tools || []
    toolGroups.value = catalog.groups || []
    knowledgeBases.value = ((kbResponse as any)?.data || []).map((kb: any) => ({ id: kb.id, name: kb.name }))
    agents.value = (agentResponse as any)?.data || []
  } catch (error) {
    MessagePlugin.error(errorMessage(error, t('mcpEndpointSettings.loadFailed')))
  } finally {
    loading.value = false
  }
}

const resetForm = () => {
  form.name = ''
  form.description = ''
  form.enabled = true
  form.allKnowledgeBases = true
  form.knowledge_base_ids = []
  form.default_agent_id = ''
  form.tools = [...defaultTools.value]
  form.rate_limit_per_minute = 30
}

const openCreate = () => {
  editingId.value = null
  oneTimeToken.value = ''
  resetForm()
  dialogVisible.value = true
}

const openEdit = (endpoint: McpEndpoint) => {
  editingId.value = endpoint.id
  oneTimeToken.value = ''
  form.name = endpoint.name
  form.description = endpoint.description || ''
  form.enabled = endpoint.enabled
  form.allKnowledgeBases = endpoint.knowledge_base_ids.length === 0
  form.knowledge_base_ids = [...endpoint.knowledge_base_ids]
  form.default_agent_id = endpoint.default_agent_id || ''
  form.tools = [...endpoint.tools]
  form.rate_limit_per_minute = endpoint.rate_limit_per_minute || 30
  dialogVisible.value = true
}

const payload = (): McpEndpointPayload => ({
  name: form.name.trim(),
  description: form.description.trim(),
  enabled: form.enabled,
  knowledge_base_ids: form.allKnowledgeBases ? [] : [...form.knowledge_base_ids],
  tools: [...form.tools],
  default_agent_id: form.default_agent_id || '',
  rate_limit_per_minute: form.rate_limit_per_minute,
})

const saveEndpoint = async () => {
  if (!form.name.trim()) {
    MessagePlugin.warning(t('mcpEndpointSettings.nameRequired'))
    return
  }
  if (form.tools.length === 0) {
    MessagePlugin.warning(t('mcpEndpointSettings.toolRequired'))
    return
  }
  saving.value = true
  try {
    const data = payload()
    const result = editingId.value
      ? await updateMcpEndpoint(editingId.value, data)
      : await createMcpEndpoint(data)
    if (result.token) oneTimeToken.value = result.token
    MessagePlugin.success(editingId.value ? t('mcpEndpointSettings.updated') : t('mcpEndpointSettings.created'))
    dialogVisible.value = false
    await load()
  } catch (error) {
    MessagePlugin.error(errorMessage(error, t('mcpEndpointSettings.saveFailed')))
  } finally {
    saving.value = false
  }
}

const rotateToken = async (endpoint: McpEndpoint) => {
  if (!window.confirm(t('mcpEndpointSettings.rotateConfirm'))) return
  try {
    const result = await rotateMcpEndpointToken(endpoint.id)
    oneTimeToken.value = result.token || ''
    MessagePlugin.success(t('mcpEndpointSettings.rotated'))
    await load()
  } catch (error) {
    MessagePlugin.error(errorMessage(error, t('mcpEndpointSettings.rotateFailed')))
  }
}

const deleteEndpoint = async (endpoint: McpEndpoint) => {
  if (!window.confirm(t('mcpEndpointSettings.deleteConfirm', { name: endpoint.name }))) return
  try {
    await deleteMcpEndpoint(endpoint.id)
    MessagePlugin.success(t('mcpEndpointSettings.deleted'))
    await load()
  } catch (error) {
    MessagePlugin.error(errorMessage(error, t('mcpEndpointSettings.deleteFailed')))
  }
}

const copy = async (value: string) => {
  try {
    await copyText(value)
    MessagePlugin.success(t('mcpEndpointSettings.copied'))
  } catch {
    MessagePlugin.error(t('mcpEndpointSettings.copyFailed'))
  }
}

onMounted(load)
</script>

<style scoped lang="less">
.mcp-endpoint-settings { width: 100%; }
.section-header { margin-bottom: 24px; }
.section-header h2 { margin: 0 0 8px; font-size: 20px; color: var(--td-text-color-primary); }
.section-description, .endpoint-toolbar p, .form-item p { margin: 0; color: var(--td-text-color-secondary); font-size: 13px; line-height: 1.5; }
.mcp-endpoint-tutorial { margin-bottom: 24px; padding: 18px; border: 1px solid var(--td-brand-color-2); border-radius: 10px; background: var(--td-brand-color-1); }
.tutorial-header { display: flex; align-items: flex-start; gap: 10px; }
.tutorial-icon { display: inline-flex; align-items: center; justify-content: center; flex: 0 0 auto; width: 30px; height: 30px; border-radius: 8px; color: var(--td-brand-color); background: var(--td-bg-color-container); }
.tutorial-header h3 { margin: 0 0 4px; font-size: 16px; color: var(--td-text-color-primary); }
.tutorial-header p, .tutorial-step p, .tutorial-security { margin: 0; color: var(--td-text-color-secondary); font-size: 13px; line-height: 1.5; }
.tutorial-steps { display: grid; grid-template-columns: repeat(4, minmax(0, 1fr)); gap: 12px; margin-top: 16px; }
.tutorial-step { display: flex; align-items: flex-start; gap: 9px; padding: 12px; border-radius: 8px; background: var(--td-bg-color-container); }
.tutorial-step__number { display: inline-flex; align-items: center; justify-content: center; flex: 0 0 auto; width: 22px; height: 22px; border-radius: 50%; color: var(--td-text-color-anti); background: var(--td-brand-color); font-size: 12px; font-weight: 600; }
.tutorial-step h4 { margin: 1px 0 4px; color: var(--td-text-color-primary); font-size: 13px; }
.tutorial-step p { font-size: 12px; }
.tutorial-config { margin-top: 14px; padding: 12px; border-radius: 8px; background: var(--td-bg-color-container); }
.tutorial-config__header { display: flex; align-items: center; justify-content: space-between; gap: 8px; color: var(--td-text-color-primary); font-size: 13px; }
.tutorial-config pre { margin: 8px 0 0; padding: 10px; overflow-x: auto; border-radius: 6px; background: var(--td-bg-color-secondarycontainer); color: var(--td-text-color-primary); font-size: 12px; line-height: 1.5; }
.tutorial-config code { white-space: pre; }
.tutorial-security { display: flex; align-items: flex-start; gap: 6px; margin-top: 12px; font-size: 12px; }
.tutorial-security :deep(.t-icon) { flex: 0 0 auto; margin-top: 2px; color: var(--td-warning-color-6); }
.endpoint-toolbar { display: flex; align-items: center; justify-content: space-between; gap: 16px; margin-bottom: 16px; }
.endpoint-toolbar h3 { margin: 0 0 4px; font-size: 16px; color: var(--td-text-color-primary); }
.loading-container { padding: 56px 0; text-align: center; }
.endpoint-grid { display: grid; grid-template-columns: repeat(auto-fill, minmax(360px, 1fr)); gap: 14px; }
.endpoint-card { padding: 16px; border: 1px solid var(--td-component-stroke); border-radius: 10px; background: var(--td-bg-color-container); }
.endpoint-card__header, .endpoint-card__title-wrap, .endpoint-card__actions, .endpoint-card__url, .endpoint-card__meta { display: flex; align-items: center; }
.endpoint-card__header { justify-content: space-between; gap: 12px; }
.endpoint-card__title-wrap { min-width: 0; gap: 10px; }
.endpoint-card__title-wrap h3 { margin: 0; font-size: 15px; color: var(--td-text-color-primary); }
.endpoint-card__title-wrap p { margin: 3px 0 0; color: var(--td-text-color-secondary); font-size: 12px; overflow: hidden; text-overflow: ellipsis; white-space: nowrap; max-width: 240px; }
.endpoint-card__icon { display: inline-flex; align-items: center; justify-content: center; width: 34px; height: 34px; border-radius: 9px; background: var(--td-brand-color-1); color: var(--td-brand-color); }
.endpoint-card__url { justify-content: space-between; gap: 8px; margin-top: 16px; padding: 8px 10px; background: var(--td-bg-color-secondarycontainer); border-radius: 6px; }
.endpoint-card__url code { min-width: 0; overflow: hidden; text-overflow: ellipsis; white-space: nowrap; font-size: 12px; }
.endpoint-card__meta { flex-wrap: wrap; gap: 8px 14px; margin-top: 12px; color: var(--td-text-color-secondary); font-size: 12px; }
.endpoint-card__actions { gap: 8px; margin-top: 16px; }
.token-banner { display: flex; align-items: flex-start; justify-content: space-between; gap: 16px; margin-top: 20px; padding: 14px 16px; border: 1px solid var(--td-warning-color-3); border-radius: 8px; background: var(--td-warning-color-1); }
.token-banner strong { color: var(--td-warning-color-7); }
.token-banner p { margin: 4px 0 8px; color: var(--td-text-color-secondary); font-size: 12px; }
.token-banner code { display: inline-block; padding: 6px 8px; border-radius: 4px; background: var(--td-bg-color-container); word-break: break-all; }
.endpoint-form { display: flex; flex-direction: column; gap: 18px; }
.form-item { display: flex; flex-direction: column; gap: 7px; }
.form-item > label, .form-item--inline label { font-size: 13px; font-weight: 600; color: var(--td-text-color-primary); }
.form-item--inline { flex-direction: row; align-items: center; justify-content: space-between; }
.tool-group { padding: 10px 12px; border: 1px solid var(--td-component-stroke); border-radius: 7px; }
.tool-group + .tool-group { margin-top: 8px; }
.tool-group__title { margin-bottom: 6px; font-size: 12px; font-weight: 600; color: var(--td-text-color-secondary); }
.tool-group :deep(.t-checkbox-group) { display: flex; flex-wrap: wrap; gap: 7px 16px; }
.tool-group :deep(.t-checkbox) { margin-right: 0; }
@media (max-width: 900px) { .tutorial-steps { grid-template-columns: repeat(2, minmax(0, 1fr)); } }
@media (max-width: 700px) { .endpoint-toolbar, .token-banner { align-items: stretch; flex-direction: column; } .endpoint-grid, .tutorial-steps { grid-template-columns: 1fr; } }
</style>
