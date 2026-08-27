<template>
  <div class="hierarchy-page">
    <aside class="tree-pane">
      <div class="pane-head"><strong>组织架构</strong><t-button v-if="canManage" variant="text" shape="square" @click="startCreate(null)"><t-icon name="add" /></t-button></div>
      <t-loading :loading="loading">
        <div v-if="!tree.length" class="empty-tree">尚未建立公司架构</div>
        <template v-for="root in tree" :key="root.id">
          <NodeRow :node="root" :selected-id="selectedId" :can-manage="canManage" @select="selectNode" @add="startCreate" @edit="startEdit" @remove="removeNode" />
        </template>
      </t-loading>
      <div class="space-link" @click="$router.push('/platform/organizations')"><t-icon name="usergroup" /><span>共享空间</span><t-icon name="chevron-right" /></div>
    </aside>

    <main class="resource-pane">
      <header class="resource-head">
        <div><div class="eyebrow">公司内部资源归属</div><h2>{{ selected?.name || '组织架构' }}</h2><p>展示当前节点及全部下辖节点的知识库和智能体</p></div>
        <t-button v-if="canManage && selected" theme="primary" @click="startCreate(selected)"><template #icon><t-icon name="add" /></template>新增下级节点</t-button>
      </header>
      <t-loading :loading="resourceLoading">
        <div v-if="!selected" class="resource-empty">请从左侧选择一个架构节点</div>
        <div v-else-if="!sections.length" class="resource-empty">该节点下暂无已归属资源</div>
        <section v-for="section in sections" :key="section.node.id" class="node-section">
          <div class="section-head"><div><span>{{ nodeTypeLabel(section.node.type) }}</span><h3>{{ section.node.name }}</h3></div><small>{{ section.kbs.length }} 个知识库 · {{ section.agents.length }} 个智能体</small></div>
          <div v-if="section.kbs.length" class="resource-group"><div class="group-title"><t-icon name="folder" />知识库</div><div class="cards"><ResourceCard v-for="item in section.kbs" :key="item.id" :item="item" kind="knowledge_base" @open="openResource" /></div></div>
          <div v-if="section.agents.length" class="resource-group"><div class="group-title"><t-icon name="chat-bubble-help" />智能体</div><div class="cards"><ResourceCard v-for="item in section.agents" :key="item.id" :item="item" kind="agent" @open="openResource" /></div></div>
        </section>
        <section v-if="selected && ((resources?.unassigned_knowledge_bases?.length || 0) + (resources?.unassigned_agents?.length || 0) > 0)" class="node-section pending-section">
          <div class="section-head"><div><span>迁移兼容</span><h3>历史资源（待归属）</h3></div><small>挂接只更新归属，不重新嵌入</small></div>
          <div v-if="resources?.unassigned_knowledge_bases?.length" class="resource-group"><div class="group-title"><t-icon name="folder" />知识库</div><div class="cards"><article v-for="item in resources.unassigned_knowledge_bases" :key="item.id" class="resource-card" role="button" tabindex="0" @click="openResource(item, 'knowledge_base')" @keydown.enter="openResource(item, 'knowledge_base')"><div class="kind-badge">历史知识库</div><h4>{{ item.name }}</h4><p>{{ item.description || '已有内容和向量保持不变' }}</p><t-button v-if="canManage" size="small" variant="outline" @click.stop="attachResource('knowledge_base', item.id)">挂接到当前节点</t-button></article></div></div>
          <div v-if="resources?.unassigned_agents?.length" class="resource-group"><div class="group-title"><t-icon name="chat-bubble-help" />智能体</div><div class="cards"><article v-for="item in resources.unassigned_agents" :key="item.id" class="resource-card" role="button" tabindex="0" @click="openResource(item, 'agent')" @keydown.enter="openResource(item, 'agent')"><div class="kind-badge">待归属智能体</div><h4>{{ item.name }}</h4><p>{{ item.description || '配置和知识库引用保持不变' }}</p><t-button v-if="canManage" size="small" variant="outline" @click.stop="attachResource('agent', item.id)">挂接到当前节点</t-button></article></div></div>
        </section>
      </t-loading>
    </main>

    <t-dialog v-model:visible="editorVisible" :header="editing ? '编辑架构节点' : '新增架构节点'" :confirm-btn="{ content: '保存', loading: saving }" @confirm="saveNode">
      <t-form label-align="top">
        <t-form-item label="节点名称"><t-input v-model="form.name" maxlength="80" /></t-form-item>
        <t-form-item v-if="!editing" label="节点类型"><t-select v-model="form.type" :options="typeOptions" /></t-form-item>
      </t-form>
    </t-dialog>
  </div>
</template>

<script setup lang="ts">
import { computed, defineComponent, h, onMounted, reactive, ref } from 'vue'
import { DialogPlugin, MessagePlugin } from 'tdesign-vue-next'
import { useAuthStore } from '@/stores/auth'
import {
  createOrganizationNode, deleteOrganizationNode, getOrganizationNodeResources,
  listOrganizationNodeTree, updateOrganizationNode, assignOrganizationNodeResource,
  type OrganizationNode, type OrganizationNodeResources, type OrganizationNodeType,
} from '@/api/organization'

const NodeRow: any = defineComponent({
  name: 'NodeRow', props: { node: { type: Object as () => OrganizationNode, required: true }, selectedId: String, canManage: Boolean }, emits: ['select','add','edit','remove'],
  setup(props, { emit }) { return () => h('div', [h('div', { class: ['node-row', { selected: props.selectedId === props.node.id }], style: { paddingLeft: `${10 + props.node.depth * 20}px` }, onClick: () => emit('select', props.node) }, [h('span', { class: 'node-icon' }, props.node.type === 'company' ? '▣' : props.node.type === 'department' ? '▤' : '▪'), h('span', { class: 'node-name' }, props.node.name), props.canManage ? h('span', { class: 'node-actions' }, [props.node.type !== 'group' ? h('button', { class: 'node-action node-action--add', title: '新增下级', 'aria-label': '新增下级', onClick: (e: Event) => { e.stopPropagation(); emit('add', props.node) } }, '+') : null, h('button', { class: 'node-action node-action--edit', title: '编辑', 'aria-label': '编辑', onClick: (e: Event) => { e.stopPropagation(); emit('edit', props.node) } }, '✎'), h('button', { class: 'node-action node-action--remove', title: '删除', 'aria-label': '删除', onClick: (e: Event) => { e.stopPropagation(); emit('remove', props.node) } }, '×')]) : null]), ...(props.node.children || []).map(child => h(NodeRow, { node: child, selectedId: props.selectedId, canManage: props.canManage, onSelect: (n: OrganizationNode) => emit('select', n), onAdd: (n: OrganizationNode) => emit('add', n), onEdit: (n: OrganizationNode) => emit('edit', n), onRemove: (n: OrganizationNode) => emit('remove', n) }))]) }
})
const ResourceCard = defineComponent({ name: 'ResourceCard', props: { item: { type: Object, required: true }, kind: { type: String, required: true } }, emits: ['open'], setup(props, { emit }) { return () => h('article', { class: 'resource-card', role: 'link', tabindex: 0, onClick: () => emit('open', props.item, props.kind), onKeydown: (event: KeyboardEvent) => { if (event.key === 'Enter' || event.key === ' ') { event.preventDefault(); emit('open', props.item, props.kind) } } }, [h('div', { class: 'kind-badge' }, props.kind === 'agent' ? '智能体' : '知识库'), h('h4', String((props.item as any).name)), h('p', String((props.item as any).description || '暂无描述')), h('span', { class: 'resource-open-hint' }, '打开详情 →')]) } })

const auth = useAuthStore(); const canManage = computed(() => auth.hasRole('admin') || auth.canAccessAllTenants)
const loading = ref(false), resourceLoading = ref(false), saving = ref(false), editorVisible = ref(false)
const tree = ref<OrganizationNode[]>([]), selected = ref<OrganizationNode | null>(null), resources = ref<OrganizationNodeResources | null>(null)
const selectedId = computed(() => selected.value?.id || '')
const editing = ref<OrganizationNode | null>(null), parent = ref<OrganizationNode | null>(null)
const form = reactive<{name:string;type:OrganizationNodeType}>({ name: '', type: 'company' })
const flatNodes = computed(() => { const out: OrganizationNode[]=[]; const visit=(n:OrganizationNode)=>{out.push(n);(n.children||[]).forEach(visit)};tree.value.forEach(visit);return out })
const sections = computed(() => flatNodes.value.filter(n => n.id === selected.value?.id || isDescendant(n, selected.value?.id)).map(node => ({ node, kbs: (resources.value?.knowledge_bases || []).filter(x => x.organization_node_id === node.id), agents: (resources.value?.agents || []).filter(x => x.organization_node_id === node.id) })).filter(x => x.kbs.length || x.agents.length))
function isDescendant(node: OrganizationNode, ancestor?: string) { let p=node.parent_id; while(p){if(p===ancestor)return true;p=flatNodes.value.find(n=>n.id===p)?.parent_id}return false }
function nodeTypeLabel(type: OrganizationNodeType){ return type==='company'?'公司':type==='department'?'部门':'组' }
const typeOptions = computed(() => parent.value ? [{ label: parent.value.type === 'company' ? '部门' : '组', value: parent.value.type === 'company' ? 'department' : 'group' }] : [{ label:'公司', value:'company' }])
function openResource(item: { id: string }, kind: 'knowledge_base' | 'agent') {
  if (kind === 'knowledge_base') {
    window.location.href = `/platform/knowledge-bases/${encodeURIComponent(item.id)}`
    return
  }
  window.location.href = `/platform/agents?edit=${encodeURIComponent(item.id)}`
}
async function loadTree(){loading.value=true;const res=await listOrganizationNodeTree();loading.value=false;if(res.success){tree.value=res.data||[];if(selected.value)selected.value=flatNodes.value.find(n=>n.id===selected.value?.id)||null}else MessagePlugin.error(res.message||'组织架构加载失败')}
async function selectNode(node:OrganizationNode){selected.value=node;resourceLoading.value=true;const res=await getOrganizationNodeResources(node.id);resourceLoading.value=false;if(res.success)resources.value=res.data||null;else MessagePlugin.error(res.message||'资源加载失败')}
function startCreate(node:OrganizationNode|null){editing.value=null;parent.value=node;form.name='';form.type=node?(node.type==='company'?'department':'group'):'company';editorVisible.value=true}
function startEdit(node:OrganizationNode){editing.value=node;parent.value=null;form.name=node.name;form.type=node.type;editorVisible.value=true}
async function saveNode(){if(!form.name.trim())return MessagePlugin.warning('请输入节点名称');saving.value=true;const res=editing.value?await updateOrganizationNode(editing.value.id,form.name.trim()):await createOrganizationNode({name:form.name.trim(),type:form.type,parent_id:parent.value?.id||null});saving.value=false;if(res.success){editorVisible.value=false;await loadTree();MessagePlugin.success('架构节点已保存')}else MessagePlugin.error(res.message||'保存失败')}
async function attachResource(resourceType:'knowledge_base'|'agent', resourceId:string){if(!selected.value)return;const res=await assignOrganizationNodeResource(selected.value.id,{resource_type:resourceType,resource_id:resourceId});if(res.success){MessagePlugin.success('资源已挂接，原有内容和向量保持不变');await selectNode(selected.value)}else MessagePlugin.error(res.message||'挂接失败')}
function removeNode(node:OrganizationNode){const dialog=DialogPlugin.confirm({header:'删除架构节点',body:`确认删除“${node.name}”？仅允许删除没有下级节点的节点。`,onConfirm:async()=>{const res=await deleteOrganizationNode(node.id);if(res.success){if(selected.value?.id===node.id){selected.value=null;resources.value=null}await loadTree();MessagePlugin.success('节点已删除');dialog.hide()}else MessagePlugin.error(res.message||'删除失败')}})}
onMounted(loadTree)
</script>

<style lang="less">
:global(:root){--hierarchy-ink:#17332f;--hierarchy-muted:#71827e;--hierarchy-line:#dce8e3;--hierarchy-mint:#e7f5ee;--hierarchy-green:#087c59;--hierarchy-orange:#d97b3f}
.hierarchy-page{display:grid;grid-template-columns:292px minmax(0,1fr);height:100%;min-height:680px;background:#f4f7f5;color:var(--hierarchy-ink);font-family:inherit}
.tree-pane{background:#fbfdfc;border-right:1px solid var(--hierarchy-line);padding:22px 14px 14px;overflow:auto}
.pane-head{display:flex;align-items:center;justify-content:space-between;padding:0 8px 17px;border-bottom:1px solid #edf2ef;margin-bottom:12px}
.pane-head strong{font-size:15px;letter-spacing:.02em}.pane-head:before{content:'';position:absolute;width:3px;height:18px;margin-left:-8px;background:var(--hierarchy-green);border-radius:2px}
.empty-tree,.resource-empty{color:var(--hierarchy-muted);text-align:center;padding:60px 20px;font-size:13px}
.node-row{height:42px;border:1px solid transparent;border-radius:8px;display:flex;align-items:center;gap:9px;cursor:pointer;color:#526661;transition:background .18s,border-color .18s,color .18s,transform .18s}
.node-row:hover{background:#f0f7f3;border-color:#dcece4;color:var(--hierarchy-green)}.node-row.selected{background:var(--hierarchy-mint);border-color:#b8dfcc;color:var(--hierarchy-green);font-weight:600;box-shadow:inset 3px 0 0 var(--hierarchy-green)}
.node-icon{width:20px;text-align:center;color:var(--hierarchy-green);font-size:13px}.node-name{overflow:hidden;text-overflow:ellipsis;white-space:nowrap;flex:1;font-size:13px}.node-actions{margin-left:auto;display:none;gap:3px;padding-right:5px}.node-row:hover>.node-actions,.node-row.selected>.node-actions{display:flex}
:deep(.node-actions .node-action){width:25px;height:25px;border:1px solid #c9dcd3;background:#fff;border-radius:5px;color:#55766b;display:grid;place-items:center;padding:0;font-size:13px;line-height:1;cursor:pointer;transition:all .15s}
:deep(.node-actions .node-action:hover){background:var(--hierarchy-green);border-color:var(--hierarchy-green);color:#fff}.node-action--remove:hover{background:#b94b43!important;border-color:#b94b43!important}
.space-link{margin-top:18px;padding:15px 10px;border-top:1px solid var(--hierarchy-line);display:flex;align-items:center;gap:9px;color:#607570;cursor:pointer;font-size:13px;transition:color .18s}.space-link:hover{color:var(--hierarchy-green)}.space-link :last-child{margin-left:auto}
.resource-pane{padding:34px clamp(24px,4vw,56px) 56px;overflow:auto}.resource-head{display:flex;justify-content:space-between;align-items:flex-start;gap:24px;margin-bottom:28px;padding-bottom:25px;border-bottom:1px solid var(--hierarchy-line)}
.eyebrow{display:flex;align-items:center;gap:8px;font-size:11px;font-weight:700;letter-spacing:.11em;color:var(--hierarchy-green);text-transform:uppercase}.eyebrow:before{content:'';width:24px;height:2px;background:var(--hierarchy-orange)}
.resource-head h2{font-size:30px;line-height:1.1;letter-spacing:-.02em;margin:9px 0 8px;color:var(--hierarchy-ink)}.resource-head p{margin:0;color:var(--hierarchy-muted);font-size:13px}
:deep(.resource-head .t-button){box-shadow:0 5px 14px rgba(8,124,89,.16);border-radius:6px;white-space:nowrap}
.node-section{background:#fff;border:1px solid var(--hierarchy-line);border-radius:10px;padding:22px 22px 24px;margin-bottom:18px;box-shadow:0 7px 20px rgba(37,69,58,.035);animation:section-in .35s ease both}.node-section:nth-of-type(3){animation-delay:.06s}.node-section:nth-of-type(4){animation-delay:.12s}
.section-head{display:flex;align-items:flex-end;justify-content:space-between;gap:16px;border-bottom:1px solid #edf2ef;padding-bottom:15px}.section-head span,.section-head small{font-size:11px;color:var(--hierarchy-muted)}.section-head span{letter-spacing:.08em;text-transform:uppercase}.section-head h3{margin:5px 0 0;font-size:19px;letter-spacing:-.01em}.section-head small{white-space:nowrap}
.resource-group{margin-top:20px}.group-title{font-size:12px;font-weight:700;color:#526b63;display:flex;gap:8px;align-items:center;margin-bottom:11px}.group-title :first-child{color:var(--hierarchy-green)}
.cards{display:grid;grid-template-columns:repeat(3,minmax(0,1fr));gap:12px}.resource-card{position:relative;border:1px solid #dfeae5;border-radius:8px;padding:17px 17px 16px;min-height:138px;background:#fbfdfc;cursor:pointer;transition:transform .18s,box-shadow .18s,border-color .18s;overflow:hidden}.resource-card:focus-visible{outline:2px solid var(--hierarchy-green);outline-offset:2px}.resource-card:before{content:'';position:absolute;left:0;top:0;bottom:0;width:3px;background:#b9dccc}.resource-card:hover{transform:translateY(-2px);border-color:#a9d4c0;box-shadow:0 10px 22px rgba(35,86,67,.09)}.resource-card h4{margin:11px 0 8px;font-size:15px;line-height:1.3}.resource-card p{font-size:12px;color:var(--hierarchy-muted);margin:0 0 13px;line-height:1.6}.resource-open-hint{position:absolute;right:15px;bottom:14px;font-size:11px;color:var(--hierarchy-green);opacity:0;transform:translateX(-3px);transition:opacity .18s,transform .18s}.resource-card:hover .resource-open-hint,.resource-card:focus-visible .resource-open-hint{opacity:1;transform:none}.kind-badge{display:inline-flex;font-size:10px;font-weight:700;letter-spacing:.04em;color:var(--hierarchy-green);background:var(--hierarchy-mint);padding:4px 7px;border-radius:4px}
.pending-section{border-style:dashed;background:#fffdfa}.pending-section .resource-card:before{background:#e3a16d}.pending-section .kind-badge{color:#a35b2d;background:#fff0e4}
@keyframes section-in{from{opacity:0;transform:translateY(5px)}to{opacity:1;transform:none}}
@media(max-width:1100px){.hierarchy-page{grid-template-columns:250px 1fr}.resource-pane{padding-left:28px;padding-right:28px}.cards{grid-template-columns:repeat(2,minmax(0,1fr))}}
@media(max-width:720px){.hierarchy-page{display:block}.tree-pane{border-right:0;border-bottom:1px solid var(--hierarchy-line);max-height:310px}.resource-pane{padding:24px 16px 40px}.resource-head{display:block}.resource-head h2{font-size:25px}.resource-head .t-button{margin-top:18px}.section-head{display:block}.section-head small{display:block;margin-top:7px}.cards{grid-template-columns:1fr}}
</style>
