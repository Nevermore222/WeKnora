<template>
  <div class="runtime-context">
    <t-dropdown trigger="click" placement="bottom-left" :min-column-width="220">
      <button type="button" class="runtime-context__button">
        <t-icon :name="runtimeStore.kind === 'enterprise' ? 'server' : 'user'" size="15px" />
        <span class="runtime-context__label">{{ activeLabel }}</span>
        <t-icon name="chevron-down" size="14px" />
      </button>
      <template #dropdown>
        <t-dropdown-menu>
          <t-dropdown-item v-if="!defaultServerRequired" :active="runtimeStore.kind === 'personal'" @click="switchPersonal">
            <t-icon name="user" size="14px" />
            <span>Personal</span>
          </t-dropdown-item>
          <t-dropdown-item
            v-for="server in store.servers"
            :key="server.id"
            :active="runtimeStore.kind === 'enterprise' && runtimeStore.profileId === server.id"
            :disabled="server.status !== 'connected'"
            @click="switchEnterprise(server.id)"
          >
            <t-icon name="server" size="14px" />
            <span>{{ server.name }}</span>
          </t-dropdown-item>
          <t-dropdown-item v-if="store.servers.length === 0" disabled>
            <t-icon name="add" size="14px" />
            <span>{{ $t('enterprise.addServer') }}</span>
          </t-dropdown-item>
        </t-dropdown-menu>
      </template>
    </t-dropdown>
  </div>
</template>

<script setup lang="ts">
import { computed, onMounted } from 'vue'
import { MessagePlugin } from 'tdesign-vue-next'
import { useI18n } from 'vue-i18n'
import { useEnterpriseStore } from '@/stores/enterprise'
import { useRuntimeContextStore } from '@/stores/runtimeContext'
import { isDefaultEnterpriseRequired } from '@/utils/api-context'

const { t } = useI18n()
const store = useEnterpriseStore()
const runtimeStore = useRuntimeContextStore()
const defaultServerRequired = isDefaultEnterpriseRequired()

const activeLabel = computed(() => {
  if (runtimeStore.kind !== 'enterprise') return 'Personal'
  return store.servers.find((server) => server.id === runtimeStore.profileId)?.name || 'Enterprise'
})

onMounted(() => {
  store.fetchServers()
})

async function switchPersonal() {
  if (defaultServerRequired) return
  await runtimeStore.usePersonal()
}

async function switchEnterprise(id: string) {
  try {
    await store.connect(id)
  } catch {
    MessagePlugin.warning(t('enterprise.connectFailed'))
  }
}
</script>

<style scoped lang="less">
.runtime-context {
  padding: 0 12px 8px;
}

.runtime-context__button {
  width: 100%;
  height: 30px;
  display: flex;
  align-items: center;
  gap: 6px;
  padding: 0 8px;
  border: 1px solid var(--td-border-level-1-color);
  border-radius: 6px;
  background: var(--td-bg-color-container);
  color: var(--td-text-color-primary);
  cursor: pointer;
}

.runtime-context__label {
  flex: 1;
  min-width: 0;
  text-align: left;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
  font-size: 12px;
}
</style>
