<template>
  <div v-if="store.hasConnection" class="connection-indicator" :class="indicatorClass">
    <span class="indicator-dot" />
    <span class="indicator-text">{{ label }}</span>
  </div>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import { useI18n } from 'vue-i18n'
import { useEnterpriseStore } from '@/stores/enterprise'

const { t } = useI18n()
const store = useEnterpriseStore()

const indicatorClass = computed(() => {
  const statuses = store.connectedServers.map((s) => s.status)
  if (statuses.every((s) => s === 'connected')) return 'indicator--connected'
  if (statuses.some((s) => s === 'connecting')) return 'indicator--connecting'
  return 'indicator--error'
})

const label = computed(() => {
  const count = store.connectedServers.length
  if (count === 1) {
    return `${store.connectedServers[0].name}`
  }
  return t('enterprise.indicatorMultiple', { count })
})
</script>

<style scoped lang="less">
.connection-indicator {
  display: inline-flex;
  align-items: center;
  gap: 6px;
  padding: 2px 10px;
  border-radius: 12px;
  font-size: 12px;
  line-height: 20px;
  cursor: default;
}
.indicator-dot {
  width: 7px;
  height: 7px;
  border-radius: 50%;
}
.indicator--connected {
  background: var(--td-success-color-light);
  color: var(--td-success-color);
  .indicator-dot { background: var(--td-success-color); }
}
.indicator--connecting {
  background: var(--td-warning-color-light);
  color: var(--td-warning-color);
  .indicator-dot {
    background: var(--td-warning-color);
    animation: pulse 1.2s infinite;
  }
}
.indicator--error {
  background: var(--td-error-color-light);
  color: var(--td-error-color);
  .indicator-dot { background: var(--td-error-color); }
}
@keyframes pulse {
  0%, 100% { opacity: 1; }
  50% { opacity: 0.4; }
}
</style>
