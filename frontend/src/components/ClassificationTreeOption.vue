<template>
  <div
    class="classification-tree-node"
    role="treeitem"
    :data-option-id="option.id"
    :aria-selected="isSelected"
    :aria-expanded="hasChildren ? isExpanded : undefined"
  >
    <div
      class="classification-tree-row classification-option-row"
      :class="{ selected: isSelected, 'has-children': hasChildren }"
    >
      <span class="tree-indent" aria-hidden="true" />
      <button
        v-if="hasChildren"
        type="button"
        class="tree-toggle"
        :aria-label="`${isExpanded ? t('common.collapse') : t('common.expand')} ${option.name}`"
        :aria-expanded="isExpanded"
        @click="emit('toggle', option.id)"
      >
        <t-icon :name="isExpanded ? 'chevron-down' : 'chevron-right'" size="13px" />
      </button>
      <span v-else class="tree-toggle tree-toggle-placeholder" aria-hidden="true" />
      <button
        type="button"
        class="tree-node-select"
        :title="option.name"
        @click="emit('select', moduleId, option.id)"
      >
        <t-icon :name="hasChildren ? (isExpanded ? 'folder-open' : 'folder') : 'file'" size="15px" />
        <span class="tree-node-label">{{ option.name }}</span>
        <t-icon v-if="isSelected" class="tree-selected-icon" name="check" size="14px" aria-hidden="true" />
      </button>
    </div>

    <div v-if="hasChildren && isExpanded" class="classification-tree-children nested" role="group">
      <ClassificationTreeOption
        v-for="child in option.children || []"
        :key="child.id"
        :option="child"
        :module-id="moduleId"
        :selected-id="selectedId"
        :expanded-ids="expandedIds"
        @select="(nextModuleId, id) => emit('select', nextModuleId, id)"
        @toggle="id => emit('toggle', id)"
      />
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import { useI18n } from 'vue-i18n'
import type { ClassificationOption } from '@/api/classification'

defineOptions({ name: 'ClassificationTreeOption' })

const props = defineProps<{
  option: ClassificationOption
  moduleId: string
  selectedId: string
  expandedIds: Set<string>
}>()

const emit = defineEmits<{
  select: [moduleId: string, optionId: string]
  toggle: [optionId: string]
}>()

const { t } = useI18n()
const hasChildren = computed(() => Boolean(props.option.children?.length))
const isSelected = computed(() => props.selectedId === props.option.id)
const isExpanded = computed(() => props.expandedIds.has(props.option.id))
</script>

<style scoped lang="less">
.classification-tree-node {
  min-width: 0;
}

.classification-option-row {
  cursor: default;
}

.tree-node-select {
  display: flex;
  min-width: 0;
  min-height: 28px;
  align-items: center;
  gap: 7px;
  flex: 1;
  padding: 0 3px;
  border: 0;
  border-radius: 4px;
  outline: none;
  background: transparent;
  color: inherit;
  font: inherit;
  text-align: left;
  cursor: pointer;
}

.tree-node-select:focus-visible,
.tree-toggle:focus-visible {
  box-shadow: 0 0 0 2px var(--td-brand-color-focus);
}

.tree-node-label {
  min-width: 0;
  overflow: hidden;
  flex: 1;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.tree-toggle {
  display: inline-flex;
  width: 24px;
  height: 28px;
  align-items: center;
  justify-content: center;
  flex: 0 0 24px;
  padding: 0;
  border: 0;
  border-radius: 4px;
  outline: none;
  background: transparent;
  color: var(--td-text-color-placeholder);
  cursor: pointer;
  transition: background-color 160ms ease, color 160ms ease;
}

.tree-toggle:hover {
  background: var(--td-bg-color-component-hover);
  color: var(--td-text-color-primary);
}

.tree-toggle-placeholder {
  pointer-events: none;
}

.tree-selected-icon {
  flex: 0 0 auto;
  color: var(--td-brand-color);
}

.classification-tree-children.nested {
  margin: 2px 0 0 17px;
  padding: 2px 0 2px 9px;
  border-left: 1px solid var(--td-component-stroke);
}

@media (prefers-reduced-motion: reduce) {
  .tree-toggle {
    transition: none;
  }
}
</style>
