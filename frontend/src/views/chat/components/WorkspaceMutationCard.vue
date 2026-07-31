<template>
  <div class="mutation-card" :class="{ resolved, approved }">
    <div class="mutation-strip">
      <t-icon :name="resolved ? (approved ? 'check-circle-filled' : 'close-circle-filled') : 'error-circle-filled'" />
      <span>
        <template v-if="!resolved">{{ $t('agentStream.workspaceMutation.banner') }}</template>
        <template v-else-if="approved">{{ $t('agentStream.workspaceMutation.approved') }}</template>
        <template v-else>{{ $t('agentStream.workspaceMutation.rejected') }}</template>
      </span>
    </div>
    <div class="mutation-body">
      <div class="mutation-op">{{ opLabel }}</div>
      <div class="mutation-path">{{ fromPath }}</div>
      <div v-if="toPath" class="mutation-path mutation-path--to">→ {{ toPath }}</div>
    </div>
    <div v-if="!resolved" class="mutation-footer">
      <t-button size="small" variant="outline" :loading="submitting && decision === 'reject'" :disabled="submitting" @click="submit(false)">
        {{ $t('agentStream.workspaceMutation.reject') }}
      </t-button>
      <t-button size="small" theme="primary" :loading="submitting && decision === 'approve'" :disabled="submitting" @click="submit(true)">
        {{ $t('agentStream.workspaceMutation.approve') }}
      </t-button>
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed, ref } from 'vue'
import { MessagePlugin } from 'tdesign-vue-next'
import { useI18n } from 'vue-i18n'
import { applyWorkspaceMutation, rejectWorkspaceMutation } from '@/api/workspace'

const props = defineProps<{
  proposalId: string
  token: string
  op: string
  fromPath: string
  toPath?: string
  resolved?: boolean
  approved?: boolean
}>()

const emit = defineEmits<{
  (e: 'resolved', payload: { approved: boolean }): void
}>()

const { t } = useI18n()
const submitting = ref(false)
const decision = ref<'approve' | 'reject' | ''>('')

const opLabel = computed(() => {
  switch (props.op) {
    case 'delete':
      return t('agentStream.workspaceMutation.opDelete')
    case 'rename':
      return t('agentStream.workspaceMutation.opRename')
    case 'move':
      return t('agentStream.workspaceMutation.opMove')
    default:
      return props.op
  }
})

async function submit(approved: boolean) {
  if (submitting.value || props.resolved) return
  submitting.value = true
  decision.value = approved ? 'approve' : 'reject'
  try {
    if (approved) {
      await applyWorkspaceMutation(props.proposalId, props.token)
    } else {
      await rejectWorkspaceMutation(props.proposalId, props.token)
    }
    emit('resolved', { approved })
  } catch (e: any) {
    MessagePlugin.error(e?.message || t('agentStream.workspaceMutation.failed'))
  } finally {
    submitting.value = false
    decision.value = ''
  }
}
</script>

<style scoped lang="less">
.mutation-card {
  border: 1px solid var(--td-component-border);
  border-radius: 8px;
  padding: 12px;
  background: var(--td-bg-color-container);
}
.mutation-strip {
  display: flex;
  align-items: center;
  gap: 8px;
  font-weight: 600;
  margin-bottom: 8px;
}
.mutation-op {
  font-size: 13px;
  margin-bottom: 4px;
}
.mutation-path {
  font-size: 12px;
  color: var(--td-text-color-secondary);
  word-break: break-all;
}
.mutation-footer {
  display: flex;
  justify-content: flex-end;
  gap: 8px;
  margin-top: 12px;
}
</style>
