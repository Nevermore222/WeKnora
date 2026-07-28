<template>
  <t-dialog
    v-model:visible="visibleModel"
    :header="t('capabilityDenied.title')"
    :confirm-btn="t('capabilityDenied.ok')"
    :cancel-btn="null"
    width="420px"
    @confirm="visibleModel = false"
  >
    <p class="capability-denied-msg">{{ t('capabilityDenied.message') }}</p>
  </t-dialog>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import { useI18n } from 'vue-i18n'

const props = defineProps<{ visible: boolean }>()
const emit = defineEmits<{ 'update:visible': [boolean] }>()
const { t } = useI18n()

const visibleModel = computed({
  get: () => props.visible,
  set: (v: boolean) => emit('update:visible', v),
})
</script>

<style scoped>
.capability-denied-msg {
  margin: 0;
  line-height: 1.6;
  color: var(--td-text-color-secondary);
}
</style>
