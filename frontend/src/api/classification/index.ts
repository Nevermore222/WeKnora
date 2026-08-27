import { get, post, put, del } from '@/utils/request'

export interface ClassificationOption {
  id: string
  module_id: string
  parent_id?: string
  name: string
  sort_order: number
  enabled: boolean
  children?: ClassificationOption[]
  path?: string[]
}
export interface ClassificationModule {
  id: string
  name: string
  sort_order: number
  enabled: boolean
  options?: ClassificationOption[]
}

export function listClassifications(resourceType: string = 'knowledge_base'): Promise<{ data: ClassificationModule[] }> {
  return get('/api/v1/classification', { params: { resource_type: resourceType } })
}
export function createClassificationModule(data: { name: string; sort_order?: number }, resourceType: string = 'knowledge_base') {
  return post('/api/v1/classification/modules', data, { params: { resource_type: resourceType } })
}
export function updateClassificationModule(id: string, data: Partial<{ name: string; sort_order: number; enabled: boolean }>, resourceType: string = 'knowledge_base') {
  return put(`/api/v1/classification/modules/${id}`, data, { params: { resource_type: resourceType } })
}
export function deleteClassificationModule(id: string, resourceType: string = 'knowledge_base') {
  return del(`/api/v1/classification/modules/${id}`, { params: { resource_type: resourceType } })
}
export function createClassificationOption(data: { module_id: string; parent_id?: string; name: string; sort_order?: number }, resourceType: string = 'knowledge_base') {
  return post('/api/v1/classification/options', data, { params: { resource_type: resourceType } })
}
export function updateClassificationOption(id: string, data: Partial<{ name: string; sort_order: number; enabled: boolean }>, resourceType: string = 'knowledge_base') {
  return put(`/api/v1/classification/options/${id}`, data, { params: { resource_type: resourceType } })
}
export function deleteClassificationOption(id: string, resourceType: string = 'knowledge_base') {
  return del(`/api/v1/classification/options/${id}`, { params: { resource_type: resourceType } })
}
export function updateKnowledgeBaseClassificationTags(id: string, classification_tags: Record<string, string[]>) {
  return put(`/api/v1/knowledge-bases/${id}/classification-tags`, { classification_tags })
}
