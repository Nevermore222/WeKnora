# 分类资源类型独立功能部署总结

## 部署时间
2026-08-27 15:42

## 实现内容

### 功能概述
知识库和智能体现在拥有完全独立的分类目录，互不干扰：
- 知识库使用 `resource_type = 'knowledge_base'`
- 智能体使用 `resource_type = 'agent'`
- 两个页面各自管理自己的分类模块和层级选项

### 部署组件

#### 1. 数据库迁移 ✅
- **版本**: 000067
- **状态**: 已执行
- **内容**:
  - `classification_modules.resource_type` 字段
  - `classification_options.resource_type` 字段
  - 更新唯一约束包含 `resource_type`
  - 添加 `resource_type` 索引
  - 现有数据默认为 `knowledge_base`

#### 2. 后端更新 ✅
- **修改内容**:
  - Repository: 所有查询增加 `resource_type` 过滤
  - Service: 所有方法增加 `resource_type` 参数和验证
  - Handler: 从查询参数读取 `resource_type`，默认 `knowledge_base`
  - 类型定义: `ClassificationModule` 和 `ClassificationOption` 增加 `ResourceType` 字段

- **编译方式**: 使用 `xelora-builder:org-test` 镜像编译（Debian 源故障导致无法重新构建完整镜像）
- **二进制文件**: `/app/service` (317MB)
- **部署方式**: 直接替换容器内二进制文件并重启

#### 3. 前端更新 ✅
- **修改内容**:
  - API 层: 所有分类函数增加 `resourceType` 参数（默认 `knowledge_base`）
  - 知识库页面: 传递 `'knowledge_base'`
  - 智能体页面: 传递 `'agent'`
  - 修复 `request.ts` 中 `put()` 和 `del()` 函数，支持 `config` 参数

- **镜像**: `wechatopenai/xelora-ui:knavis-internal-local`
- **构建时间**: 2026-08-27 15:29

### 关键修复

#### 问题 1: .env DB_HOST 配置错误
- **现象**: 后端容器启动失败，尝试连接 `localhost:5432`
- **根因**: `.env` 文件中 `DB_HOST=localhost` 覆盖了 `.env.knavis-internal` 的正确配置
- **修复**: 将 `.env` 中 `DB_HOST` 改为 `postgres`

#### 问题 2: request.ts 函数签名不完整
- **现象**: 分类 API 返回 404，控制台显示 `/api/v1/classificati-source_type=agent` 格式错误
- **根因**: `put()` 和 `del()` 函数不接受 `config` 参数，导致 `{ params: { resource_type } }` 被忽略
- **修复**:
  ```typescript
  // 修复前
  export function put(url: string, data = {}) {
    return instance.put(url, data);
  }
  export function del(url: string, data?: any) {
    return instance.delete(url, { data });
  }
  
  // 修复后
  export function put(url: string, data = {}, config?: any) {
    return instance.put(url, data, config);
  }
  export function del(url: string, config?: any) {
    return instance.delete(url, config);
  }
  ```

#### 问题 3: 后端镜像重新构建失败
- **现象**: Debian 官方源持续返回 500/502 错误
- **根因**: `deb.debian.org` 网络不稳定
- **解决方案**: 使用现有 `xelora-builder:org-test` 镜像编译新二进制文件，直接替换容器内 `/app/service`

### 当前服务状态

```
Xelora-app           Up 2 minutes (healthy)  - 新二进制，支持 resource_type
Xelora-frontend      Up 20 minutes           - 新镜像，传递 resource_type
Xelora-postgres      Up 2 weeks (healthy)    - 迁移至版本 67
Xelora-redis         Up 2 weeks
Xelora-opensandbox   Up 2 weeks (healthy)
Xelora-docreader     Up 2 weeks (healthy)
```

### API 验证

```bash
# 健康检查
curl http://localhost:8080/health
# 返回: {"status":"ok"}

# 分类 API 路由检查
curl -I "http://localhost:8080/api/v1/classification?resource_type=knowledge_base"
# 返回: 401 Unauthorized (说明路由已注册，需要认证)
```

### 用户验证步骤

#### 1. 知识库分类
1. 访问 `http://localhost/platform/knowledge-bases`
2. 点击"自定义筛选结构"按钮
3. 应该看到现有的"项目"模块（resource_type=knowledge_base）
4. 新增模块/选项，验证保存和读取

#### 2. 智能体分类
1. 访问 `http://localhost/platform/agents`
2. 点击"自定义筛选结构"按钮
3. 应该看到空的分类列表（智能体还没有创建分类）
4. 新增模块"智能体类型"
5. 新增选项"客服"、"数据分析"等
6. 验证树形结构正常展示

#### 3. 验证隔离
1. 在知识库页面创建分类"测试A"
2. 切换到智能体页面，应该看不到"测试A"
3. 在智能体页面创建分类"测试B"
4. 切换回知识库页面，应该看不到"测试B"

### 数据状态

#### 现有分类数据
```sql
-- 知识库分类（resource_type='knowledge_base'）
SELECT COUNT(*) FROM classification_modules WHERE resource_type='knowledge_base';
-- 结果: 1 个模块（项目）

SELECT COUNT(*) FROM classification_options WHERE resource_type='knowledge_base';
-- 结果: 15 个选项（TOIN/FUSHAN 完整层级树）

-- 智能体分类（resource_type='agent'）
SELECT COUNT(*) FROM classification_modules WHERE resource_type='agent';
-- 结果: 0 个模块（待用户创建）
```

### 技术说明

#### resource_type 值
- `'knowledge_base'` - 知识库分类
- `'agent'` - 智能体分类
- 未来可扩展其他资源类型（如 `'document'`, `'dataset'`）

#### API 参数传递
```typescript
// 前端调用示例
// 知识库页面
listClassifications('knowledge_base')
createClassificationModule({ name: '机型' }, 'knowledge_base')

// 智能体页面
listClassifications('agent')
createClassificationModule({ name: '智能体类型' }, 'agent')
```

#### 后端 SQL 查询
```sql
-- Repository 自动添加 resource_type 过滤
SELECT * FROM classification_modules 
WHERE tenant_id = ? AND resource_type = ?
ORDER BY sort_order, id
```

### 注意事项

1. **默认值行为**: API 的 `resource_type` 参数默认为 `knowledge_base`，确保向后兼容
2. **唯一约束**: 同一租户、同一资源类型下，模块名称必须唯一
3. **数据迁移**: 现有分类数据已自动标记为 `knowledge_base`
4. **前端缓存**: 首次访问智能体页面会看到空列表，需要手动创建分类
5. **树组装**: 使用递归深拷贝确保多级子节点正确返回

### 后续工作

1. ✅ 数据库迁移
2. ✅ 后端代码修改和编译
3. ✅ 前端代码修改和构建
4. ✅ 部署和服务验证
5. ⏳ 浏览器功能验证（待用户完成）
6. ⏳ 为智能体创建初始分类树
7. ⏳ 移除 Repository 层的调试日志
8. ⏳ 补充单元测试覆盖 `resource_type` 参数
9. ⏳ 提交代码到 `knavis-internal` 分支

### 故障排查

#### 问题: 分类 API 返回 404
- **检查**: `docker logs Xelora-app | grep classification`
- **可能原因**: 后端容器未使用最新二进制文件
- **解决**: 重新复制 `service-new` 并重启容器

#### 问题: 智能体页面看到知识库的分类
- **检查**: 浏览器 Network 面板，查看 `/api/v1/classification` 请求参数
- **可能原因**: 前端未传递 `resource_type='agent'`
- **解决**: 确认前端镜像为 `knavis-internal-local`

#### 问题: 新增分类后刷新消失
- **检查**: 数据库中的 `resource_type` 字段值
  ```sql
  SELECT id, name, resource_type FROM classification_modules 
  WHERE tenant_id = 10001 ORDER BY created_at DESC LIMIT 5;
  ```
- **可能原因**: 创建时使用了错误的 `resource_type`
- **解决**: 确认前端传递的参数与查询参数一致

### 相关文档
- 详细实现文档: [CLASSIFICATION_RESOURCE_TYPE_IMPLEMENTATION.md](./CLASSIFICATION_RESOURCE_TYPE_IMPLEMENTATION.md)
- 数据库迁移: [migrations/versioned/000067_classification_resource_type.up.sql](./migrations/versioned/000067_classification_resource_type.up.sql)

### 部署签名
- 执行者: Claude Code (Kiro)
- 部署方式: 热更新（容器内二进制替换）
- 验证状态: 后端健康检查通过，API 路由已注册
- 下一步: 等待用户浏览器功能验证
