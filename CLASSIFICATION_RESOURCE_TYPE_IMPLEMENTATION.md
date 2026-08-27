# 分类目录按资源类型独立实现

## 概述
实现了知识库和智能体各自拥有独立的分类目录，互不干扰。

## 实施内容

### 1. 数据库迁移 (000067)
**文件**: `migrations/versioned/000067_classification_resource_type.{up,down}.sql`

**新增字段**:
- `classification_modules.resource_type` (VARCHAR(50), NOT NULL, DEFAULT 'knowledge_base')
- `classification_options.resource_type` (VARCHAR(50), NOT NULL, DEFAULT 'knowledge_base')

**索引**:
- `idx_classification_modules_resource_type`
- `idx_classification_options_resource_type`

**唯一约束调整**:
- 原: `classification_modules_tenant_id_name_key (tenant_id, name)`
- 新: `classification_modules_tenant_id_resource_type_name_key (tenant_id, resource_type, name)`

**数据迁移**:
- 现有分类数据自动标记为 `knowledge_base`（通过默认值）

### 2. 后端修改

#### 类型定义 (`internal/types/classification.go`)
```go
type ClassificationModule struct {
    ResourceType string `json:"resource_type" gorm:"type:varchar(50);not null;default:'knowledge_base';index:idx_classification_modules_resource_type"`
    // ... 其他字段
}

type ClassificationOption struct {
    ResourceType string `json:"resource_type" gorm:"type:varchar(50);not null;default:'knowledge_base';index:idx_classification_options_resource_type"`
    // ... 其他字段
}
```

#### 接口定义 (`internal/types/interfaces/classification.go`)
所有方法增加 `resourceType string` 参数：
- `List(ctx, resourceType)`
- `CreateModule(ctx, resourceType, ...)`
- `UpdateModule(ctx, resourceType, ...)`
- `DeleteModule(ctx, resourceType, ...)`
- `CreateOption(ctx, resourceType, ...)`
- `UpdateOption(ctx, resourceType, ...)`
- `DeleteOption(ctx, resourceType, ...)`
- `ValidateSelections(ctx, resourceType, ...)`

#### Repository 层 (`internal/application/repository/classification.go`)
所有查询和操作增加 `resource_type` 过滤条件：
```go
func (r *classificationRepository) ListModules(ctx context.Context, tenantID uint64, resourceType string) ([]*types.ClassificationModule, error) {
    var v []*types.ClassificationModule
    err := r.db.WithContext(ctx).Where("tenant_id = ? AND resource_type = ?", tenantID, resourceType).Order("sort_order,id").Find(&v).Error
    return v, err
}
```

#### Service 层 (`internal/application/service/classification.go`)
- 创建模块/选项时设置 `ResourceType` 字段
- 更新/删除前验证 `resource_type` 匹配
- 树组装逻辑保持不变（递归深拷贝）

#### Handler 层 (`internal/handler/classification.go`)
从查询参数读取 `resource_type`，默认为 `knowledge_base`：
```go
func (h *ClassificationHandler) List(c *gin.Context) {
    resourceType := c.DefaultQuery("resource_type", "knowledge_base")
    v, e := h.service.List(c.Request.Context(), resourceType)
    // ...
}
```

### 3. 前端修改

#### API 层 (`frontend/src/api/classification/index.ts`)
所有分类 API 增加 `resourceType` 参数，默认为 `knowledge_base`：
```typescript
export function listClassifications(resourceType: string = 'knowledge_base'): Promise<{ data: ClassificationModule[] }> {
  return get('/api/v1/classification', { params: { resource_type: resourceType } })
}

export function createClassificationModule(data: {...}, resourceType: string = 'knowledge_base') {
  return post('/api/v1/classification/modules', data, { params: { resource_type: resourceType } })
}
```

#### 知识库页面 (`frontend/src/views/knowledge/KnowledgeBaseList.vue`)
所有分类操作传递 `'knowledge_base'`：
```typescript
const loadClassifications = async () => {
  const res = await listClassifications('knowledge_base');
  // ...
}

const addClassificationModule = async () => {
  await createClassificationModule({ name, sort_order }, 'knowledge_base');
  // ...
}
```

#### 智能体页面 (`frontend/src/views/agent/AgentList.vue`)
所有分类操作传递 `'agent'`：
```typescript
const loadClassifications = async () => {
  const res = await listClassifications('agent');
  // ...
}

const addClassificationOption = async () => {
  await createClassificationOption({ module_id, parent_id, name }, 'agent');
  // ...
}
```

## 数据隔离

### 知识库分类
- `resource_type = 'knowledge_base'`
- 知识库页面只能看到和管理 `knowledge_base` 类型的分类
- 不会看到智能体的分类

### 智能体分类
- `resource_type = 'agent'`
- 智能体页面只能看到和管理 `agent` 类型的分类
- 不会看到知识库的分类

### 跨类型隔离
- 租户 A 的知识库分类不影响租户 A 的智能体分类
- 租户 A 和租户 B 的分类完全隔离（原有租户隔离逻辑保持）

## 部署状态

### 数据库
- 迁移版本: 67
- 现有模块: 1 个（项目，已标记为 `knowledge_base`）
- 现有选项: 15 个（TOIN/FUSHAN 层级树，已标记为 `knowledge_base`）

### 服务
- **后端**: `wechatopenai/xelora-app:knavis-internal-local` (健康)
- **前端**: `wechatopenai/xelora-ui:knavis-internal-local` (运行)
- **数据库**: Xelora-postgres (健康)
- **其他**: Redis, DocReader, OpenSandbox (正常)

### 验证项
- [x] 数据库迁移成功
- [x] 后端编译通过
- [x] 前端构建通过
- [x] 容器部署成功
- [x] 健康检查通过
- [ ] 浏览器功能验证（待用户测试）

## 用户验证步骤

### 1. 知识库分类管理
1. 访问 `http://localhost/platform/knowledge-bases`
2. 点击"自定义筛选结构"按钮
3. 应该看到现有的"项目"模块和完整的层级树
4. 新增模块/选项，验证保存成功
5. 刷新页面，验证新增内容保留

### 2. 智能体分类管理
1. 访问 `http://localhost/platform/agents`
2. 点击"自定义筛选结构"按钮
3. 应该看到空的分类列表（因为还没创建 `agent` 类型分类）
4. 新增模块"智能体类型"
5. 新增选项"客服"、"数据分析"、"文档处理"
6. 验证树形结构正常展示

### 3. 验证隔离
1. 在知识库页面创建分类"A"
2. 切换到智能体页面
3. 应该看不到分类"A"
4. 在智能体页面创建分类"B"
5. 切换回知识库页面
6. 应该看不到分类"B"

### 4. 验证筛选功能
1. 在智能体页面创建完整分类树
2. 给智能体打上分类标签（通过编辑器）
3. 在智能体列表左侧树形筛选区选择分类
4. 验证列表只显示匹配的智能体

## 技术说明

### resource_type 值
- `'knowledge_base'`: 知识库分类
- `'agent'`: 智能体分类

### 扩展性
如果未来需要其他资源类型（如"文档"、"数据集"），只需：
1. 新增 `resource_type` 值（如 `'document'`）
2. 在对应页面传递该值
3. 无需修改数据库结构或后端逻辑

### 兼容性
- 旧数据自动标记为 `knowledge_base`
- API 默认值为 `knowledge_base`，确保未传参数时不会出错
- 前端显式传递 `resource_type`，避免依赖默认值

## 注意事项

1. **数据库默认值**: 新记录默认为 `knowledge_base`，创建 `agent` 分类时必须显式指定
2. **唯一约束**: 同一租户、同一资源类型下，模块名称不能重复
3. **树组装**: 递归深拷贝逻辑确保多级子节点正确返回
4. **前端缓存**: 首次访问智能体页面会看到空列表，需要手动创建分类

## 问题排查

### 问题: 智能体页面看到知识库的分类
- **原因**: 前端未传递 `resource_type='agent'`
- **检查**: 浏览器 Network 查看 `/api/v1/classification` 请求参数

### 问题: 新增分类后刷新消失
- **原因**: `resource_type` 不匹配，创建时用了 `knowledge_base`，查询时用了 `agent`
- **检查**: 数据库中的 `resource_type` 字段值

### 问题: 唯一约束冲突
- **原因**: 同一租户、同一资源类型下，重复的模块名称或同级选项名称
- **解决**: 换一个名称，或删除旧的重名项

## 后续工作

1. 浏览器功能验证（用户完成）
2. 为智能体创建初始分类树
3. 批量为现有智能体打分类标签
4. 移除 Repository 层的调试日志 `fmt.Printf`
5. 补充单元测试覆盖 `resource_type` 参数
