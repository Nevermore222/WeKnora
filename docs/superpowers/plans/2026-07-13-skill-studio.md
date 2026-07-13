# Skill Studio Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Build the first Skill Studio release so administrators and developers can inspect, test, bind, and explicitly invoke existing skills without creating a heavy business-domain platform.

**Architecture:** Reuse the existing skill filesystem loader, skill manager, agent skill policy fields, executor gateway, and chat timeline. Add read-mostly skill metadata/detail APIs, a controlled skill test runner that executes through the same gateway as agents, a settings-page Skill Studio UI, and a chat slash picker backed by real skill metadata instead of hard-coded options.

**Tech Stack:** Go 1.26 backend, Gin handlers, existing `internal/agent/skills` loader/manager, existing executor gateway, Vue 3.5 + TypeScript 6 frontend, TDesign components, Docker Compose validation.

## Global Constraints

- Do not create a new business-domain platform in this release.
- Do not add network skill marketplace installation in this release.
- Do not add web source-code editing for skills in this release.
- Keep `SKILL.md` as the source of truth for skill instructions.
- Add database state only if filesystem-derived metadata cannot represent platform state.
- File-producing skill tests must execute through the existing workspace-bound executor path.
- Text-only success is not sufficient for file work; expected artifacts must be visible in the execution result.
- Preserve unrelated dirty worktree changes.

---

## File Structure

Backend:

- Modify `internal/agent/skills/skill.go`: add small public DTO helpers for script metadata and skill detail summaries.
- Modify `internal/application/service/skill_service.go`: add methods for detail loading, file listing, script discovery, and test-run preparation.
- Modify `internal/types/interfaces/skill.go`: extend `SkillService` with detail/test-run methods.
- Modify `internal/handler/skill_handler.go`: add list metadata fields, detail endpoint, file endpoint, and test-run endpoint.
- Modify `internal/router/router.go`: register new skill routes under existing `/api/v1/skills`.
- Modify `docs/api/skill.md`: document the expanded skill APIs.

Frontend:

- Create `frontend/src/api/skill/index.ts`: typed skill API client.
- Create `frontend/src/views/settings/SkillStudioSettings.vue`: settings-page Skill Studio UI.
- Modify `frontend/src/views/settings/Settings.vue`: add Skill Studio nav entry.
- Modify `frontend/src/components/Input-field.vue`: replace hard-coded slash skill options with API-loaded skills plus curated aliases.
- Modify `frontend/src/utils/skillDirectiveDisplay.ts`: keep hidden directive sanitization compatible with real skill IDs.
- Add or modify frontend tests under `frontend/src/utils/`.

Verification:

- Backend Go tests for service/handler behavior.
- Frontend build and focused utility tests.
- Docker rebuild smoke after backend and frontend changes.

---

### Task 1: Expand Skill Metadata And Detail API

**Files:**

- Modify: `internal/agent/skills/skill.go`
- Modify: `internal/application/service/skill_service.go`
- Modify: `internal/types/interfaces/skill.go`
- Modify: `internal/handler/skill_handler.go`
- Modify: `internal/router/router.go`
- Modify: `docs/api/skill.md`
- Test: `internal/application/service/skill_service_test.go`
- Test: `internal/handler/skill_handler_test.go`

**Interfaces:**

- Consumes: existing `skills.Loader.DiscoverSkills()`, `LoadSkillInstructions(name)`, `ListSkillFiles(name)`, and `ReadSkillFile(name, path)` behavior.
- Produces:
  - `SkillService.ListSkillSummaries(ctx context.Context) ([]*SkillSummary, error)`
  - `SkillService.GetSkillDetail(ctx context.Context, name string) (*SkillDetail, error)`
  - `GET /api/v1/skills`
  - `GET /api/v1/skills/:name`
  - `GET /api/v1/skills/:name/files/*path`

- [ ] **Step 1: Add failing service tests**

Create `internal/application/service/skill_service_test.go` with:

```go
package service

import (
	"context"
	"os"
	"path/filepath"
	"testing"
)

func writeSkillForTest(t *testing.T, root, name, body string) {
	t.Helper()
	dir := filepath.Join(root, name)
	if err := os.MkdirAll(filepath.Join(dir, "scripts"), 0o755); err != nil {
		t.Fatal(err)
	}
	content := "---\nname: " + name + "\ndescription: Test skill " + name + "\n---\n" + body + "\n"
	if err := os.WriteFile(filepath.Join(dir, "SKILL.md"), []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "scripts", "run.py"), []byte("print('ok')\n"), 0o644); err != nil {
		t.Fatal(err)
	}
}

func TestSkillServiceDetailIncludesScriptsAndInstructions(t *testing.T) {
	root := t.TempDir()
	writeSkillForTest(t, root, "demo-skill", "Use `scripts/run.py` for execution.")
	t.Setenv("XELORA_SKILLS_DIR", root)

	svc := NewSkillService()
	detail, err := svc.GetSkillDetail(context.Background(), "demo-skill")
	if err != nil {
		t.Fatalf("GetSkillDetail failed: %v", err)
	}
	if detail.Name != "demo-skill" {
		t.Fatalf("name mismatch: %s", detail.Name)
	}
	if detail.Instructions == "" {
		t.Fatal("expected instructions")
	}
	if len(detail.Scripts) != 1 || detail.Scripts[0].Path != "scripts/run.py" {
		t.Fatalf("expected scripts/run.py, got %#v", detail.Scripts)
	}
}
```

- [ ] **Step 2: Run service test and verify failure**

Run:

```powershell
docker run --rm -v "${PWD}:/src" -w /src golang:1.26-bookworm go test ./internal/application/service -run TestSkillServiceDetailIncludesScriptsAndInstructions -v
```

Expected: FAIL because `GetSkillDetail` and detail DTOs do not exist.

- [ ] **Step 3: Add DTOs and service methods**

Add focused DTOs near the skill service implementation:

```go
type SkillScriptSummary struct {
	Path     string `json:"path"`
	Language string `json:"language"`
}

type SkillFileSummary struct {
	Path     string `json:"path"`
	IsScript bool   `json:"is_script"`
}

type SkillSummary struct {
	Name        string               `json:"name"`
	Description string               `json:"description"`
	Source      string               `json:"source"`
	Status      string               `json:"status"`
	Scripts     []SkillScriptSummary `json:"scripts"`
}

type SkillDetail struct {
	Name         string               `json:"name"`
	Description  string               `json:"description"`
	Source       string               `json:"source"`
	Status       string               `json:"status"`
	Instructions string               `json:"instructions"`
	Scripts      []SkillScriptSummary `json:"scripts"`
	Files        []SkillFileSummary   `json:"files"`
}
```

Implement `ListSkillSummaries` and `GetSkillDetail` by deriving files from `loader.ListSkillFiles(name)`, using `skills.IsScript(path)` and `skills.GetScriptLanguage(path)`. Set `Source` to `"preloaded"` and `Status` to `"enabled"` for all currently discovered skills.

- [ ] **Step 4: Extend the interface and handler**

Extend `internal/types/interfaces/skill.go` with the new methods and add handler responses:

```go
func (h *SkillHandler) GetSkill(c *gin.Context) {
	ctx := c.Request.Context()
	name := strings.TrimSpace(c.Param("name"))
	if name == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "skill name is required"})
		return
	}
	detail, err := h.skillService.GetSkillDetail(ctx, name)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "data": detail})
}
```

Register routes:

```go
func RegisterSkillRoutes(r *gin.RouterGroup, handler *handler.SkillHandler, g *rbacGuards) {
	skills := r.Group("/skills")
	{
		skills.GET("", g.Viewer(), handler.ListSkills)
		skills.GET("/:name", g.Viewer(), handler.GetSkill)
		skills.GET("/:name/files/*path", g.Viewer(), handler.GetSkillFile)
	}
}
```

- [ ] **Step 5: Run targeted backend tests**

Run:

```powershell
docker run --rm -v "${PWD}:/src" -w /src golang:1.26-bookworm go test ./internal/application/service ./internal/handler -run 'Skill' -v
```

Expected: PASS for skill service and handler tests.

- [ ] **Step 6: Update API docs**

Update `docs/api/skill.md` to document:

```text
GET /skills
GET /skills/:name
GET /skills/:name/files/*path
```

Show response fields `source`, `status`, `scripts`, `files`, and `instructions`.

- [ ] **Step 7: Commit**

```powershell
git add internal/agent/skills/skill.go internal/application/service/skill_service.go internal/types/interfaces/skill.go internal/handler/skill_handler.go internal/router/router.go internal/application/service/skill_service_test.go internal/handler/skill_handler_test.go docs/api/skill.md
git commit -m "feat: expose skill detail metadata"
```

---

### Task 2: Add Skill Test Runner Backend Endpoint

**Files:**

- Modify: `internal/application/service/skill_service.go`
- Modify: `internal/types/interfaces/skill.go`
- Modify: `internal/handler/skill_handler.go`
- Modify: `internal/router/router.go`
- Modify: `docs/api/skill.md`
- Test: `internal/handler/skill_handler_test.go`

**Interfaces:**

- Consumes: existing `execute_skill_script` input shape and executor gateway behavior.
- Produces:
  - `POST /api/v1/skills/:name/test-run`
  - Request body:

```json
{
  "script_path": "scripts/officecli_bridge.py",
  "args": ["request.json"],
  "input": "{\"action\":\"validate\",\"file\":\"example.pptx\"}",
  "workspace_id": "optional-workspace-id"
}
```

- [ ] **Step 1: Add failing handler test for validation**

Add this test to `internal/handler/skill_handler_test.go`:

```go
func TestSkillTestRunRequiresScriptPath(t *testing.T) {
	handler := NewSkillHandler(&skillServiceStub{})
	recorder, c := testContext(http.MethodPost, "/api/v1/skills/demo/test-run", `{"args":[]}`)
	c.Params = gin.Params{{Key: "name", Value: "demo"}}

	handler.TestRunSkill(c)

	if recorder.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d body=%s", recorder.Code, recorder.Body.String())
	}
	if !strings.Contains(recorder.Body.String(), "script_path") {
		t.Fatalf("expected script_path error, got %s", recorder.Body.String())
	}
}
```

- [ ] **Step 2: Run test and verify failure**

Run:

```powershell
docker run --rm -v "${PWD}:/src" -w /src golang:1.26-bookworm go test ./internal/handler -run TestSkillTestRunRequiresScriptPath -v
```

Expected: FAIL because `TestRunSkill` does not exist.

- [ ] **Step 3: Implement the request parser and validation**

Add:

```go
type SkillTestRunRequest struct {
	ScriptPath  string   `json:"script_path"`
	Args        []string `json:"args"`
	Input       string   `json:"input"`
	WorkspaceID string   `json:"workspace_id,omitempty"`
}

func (h *SkillHandler) TestRunSkill(c *gin.Context) {
	name := strings.TrimSpace(c.Param("name"))
	var req SkillTestRunRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	req.ScriptPath = strings.TrimSpace(req.ScriptPath)
	if name == "" || req.ScriptPath == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "skill name and script_path are required"})
		return
	}
	result, err := h.skillService.TestRunSkill(c.Request.Context(), name, req)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "data": result})
}
```

In the first implementation, `TestRunSkill` may call the same preparation path used by skill execution and, when no workspace is supplied, return `workspace_required` for scripts that need file output. Do not run skill scripts from arbitrary directories.

- [ ] **Step 4: Route test-run as Admin+**

Register:

```go
skills.POST("/:name/test-run", g.Admin(), handler.TestRunSkill)
```

The list/detail endpoints stay Viewer+. Test execution is Admin+ because it can run code in the sandbox and mutate a workspace.

- [ ] **Step 5: Return structured execution details**

The response should include:

```json
{
  "skill_name": "officecli-document-editing",
  "script_path": "scripts/officecli_bridge.py",
  "args": ["request.json"],
  "success": false,
  "exit_code": 1,
  "stdout": "",
  "stderr": "validation error",
  "artifacts": [],
  "artifact_detected": false
}
```

If the service initially reuses `executor.SkillJobExecution`, map its fields directly instead of inventing a second execution result model.

- [ ] **Step 6: Run targeted backend tests**

Run:

```powershell
docker run --rm -v "${PWD}:/src" -w /src golang:1.26-bookworm go test ./internal/handler ./internal/application/service -run 'Skill' -v
```

Expected: PASS.

- [ ] **Step 7: Update API docs**

Document `POST /skills/:name/test-run`, including the Admin+ permission note and the expected `workspace_required` behavior when no workspace is bound for file-producing skills.

- [ ] **Step 8: Commit**

```powershell
git add internal/application/service/skill_service.go internal/types/interfaces/skill.go internal/handler/skill_handler.go internal/router/router.go internal/handler/skill_handler_test.go docs/api/skill.md
git commit -m "feat: add skill test runner endpoint"
```

---

### Task 3: Add Skill Studio Settings UI

**Files:**

- Create: `frontend/src/api/skill/index.ts`
- Create: `frontend/src/views/settings/SkillStudioSettings.vue`
- Modify: `frontend/src/views/settings/Settings.vue`
- Modify: `frontend/src/i18n/locales/en-US.ts`
- Modify: `frontend/src/i18n/locales/zh-CN.ts`

**Interfaces:**

- Consumes:
  - `GET /api/v1/skills`
  - `GET /api/v1/skills/:name`
  - `POST /api/v1/skills/:name/test-run`
- Produces:
  - Settings nav section `skills`.
  - Skill list/detail/test runner UI.

- [ ] **Step 1: Add typed API client**

Create `frontend/src/api/skill/index.ts`:

```ts
import { get, post } from '@/utils/request'

export interface SkillScriptSummary {
  path: string
  language: string
}

export interface SkillSummary {
  name: string
  description: string
  source: 'preloaded' | 'installed' | 'custom'
  status: 'enabled' | 'disabled' | 'invalid' | 'unavailable'
  scripts: SkillScriptSummary[]
}

export interface SkillFileSummary {
  path: string
  is_script: boolean
}

export interface SkillDetail extends SkillSummary {
  instructions: string
  files: SkillFileSummary[]
}

export interface SkillTestRunRequest {
  script_path: string
  args: string[]
  input: string
  workspace_id?: string
}

export interface SkillTestRunResult {
  skill_name: string
  script_path: string
  args: string[]
  success: boolean
  exit_code?: number
  stdout?: string
  stderr?: string
  error?: string
  artifacts?: Array<{ relative_path: string; kind: string; change_type: string; size: number }>
  artifact_detected?: boolean
}

export async function listSkills(): Promise<{ data: SkillSummary[]; skills_available: boolean }> {
  return await get('/api/v1/skills') as { data: SkillSummary[]; skills_available: boolean }
}

export async function getSkill(name: string): Promise<{ data: SkillDetail }> {
  return await get(`/api/v1/skills/${encodeURIComponent(name)}`) as { data: SkillDetail }
}

export async function testRunSkill(name: string, req: SkillTestRunRequest): Promise<{ data: SkillTestRunResult }> {
  return await post(`/api/v1/skills/${encodeURIComponent(name)}/test-run`, req) as { data: SkillTestRunResult }
}
```

- [ ] **Step 2: Create Skill Studio page**

Create `frontend/src/views/settings/SkillStudioSettings.vue` with:

```vue
<template>
  <div class="skill-studio">
    <div class="page-header">
      <h2>{{ $t('skillStudio.title') }}</h2>
      <p>{{ $t('skillStudio.description') }}</p>
    </div>

    <div class="skill-layout">
      <div class="skill-list">
        <t-input v-model="query" :placeholder="$t('skillStudio.searchPlaceholder')" clearable />
        <button
          v-for="skill in filteredSkills"
          :key="skill.name"
          type="button"
          class="skill-card"
          :class="{ active: selectedName === skill.name }"
          @click="selectSkill(skill.name)"
        >
          <span class="skill-name">{{ skill.name }}</span>
          <span class="skill-desc">{{ skill.description }}</span>
          <span class="skill-meta">{{ skill.source }} · {{ skill.status }}</span>
        </button>
      </div>

      <div class="skill-detail" v-if="detail">
        <h3>{{ detail.name }}</h3>
        <p>{{ detail.description }}</p>
        <div class="script-row" v-for="script in detail.scripts" :key="script.path">
          <code>{{ script.path }}</code>
          <span>{{ script.language }}</span>
        </div>

        <t-tabs v-model="activeTab">
          <t-tab-panel value="instructions" :label="$t('skillStudio.instructions')">
            <pre class="skill-md">{{ detail.instructions }}</pre>
          </t-tab-panel>
          <t-tab-panel value="test" :label="$t('skillStudio.testRunner')">
            <t-select v-model="testForm.script_path" :options="scriptOptions" />
            <t-input v-model="argsText" :placeholder="$t('skillStudio.argsPlaceholder')" />
            <t-textarea v-model="testForm.input" :autosize="{ minRows: 6, maxRows: 14 }" />
            <t-button theme="primary" :loading="running" @click="runTest">{{ $t('skillStudio.runTest') }}</t-button>
            <pre v-if="testResult" class="test-result">{{ testResultText }}</pre>
          </t-tab-panel>
        </t-tabs>
      </div>
    </div>
  </div>
</template>
```

Keep styles scoped and modest. This is an admin tool, not a marketing page.

- [ ] **Step 3: Wire the page logic**

Use `listSkills`, `getSkill`, and `testRunSkill`. Parse `argsText` as JSON array if it starts with `[`, otherwise split by spaces. On invalid JSON, show `MessagePlugin.error`.

- [ ] **Step 4: Add settings nav entry**

In `frontend/src/views/settings/Settings.vue`:

- Import `SkillStudioSettings`.
- Add `skills: 'admin'` to `SECTION_MIN_ROLE`.
- Add nav item `{ key: 'skills', icon: 'tools', label: t('skillStudio.title') }`.
- Add render block:

```vue
<div v-if="currentSection === 'skills'" class="section">
  <SkillStudioSettings />
</div>
```

Place it in the existing data/extensions group near MCP.

- [ ] **Step 5: Add i18n keys**

Add keys:

```ts
skillStudio: {
  title: 'Skill Studio',
  description: 'Inspect and test skills used by agents.',
  searchPlaceholder: 'Search skills',
  instructions: 'Instructions',
  testRunner: 'Test runner',
  argsPlaceholder: 'Args, for example ["request.json"]',
  runTest: 'Run test'
}
```

For zh-CN:

```ts
skillStudio: {
  title: 'Skill 管理中心',
  description: '查看、测试和维护智能体可用的 Skills。',
  searchPlaceholder: '搜索 Skill',
  instructions: '说明',
  testRunner: '测试执行',
  argsPlaceholder: '参数，例如 ["request.json"]',
  runTest: '运行测试'
}
```

- [ ] **Step 6: Build frontend**

Run:

```powershell
cd frontend
npm run build
```

Expected: PASS.

- [ ] **Step 7: Commit**

```powershell
git add frontend/src/api/skill/index.ts frontend/src/views/settings/SkillStudioSettings.vue frontend/src/views/settings/Settings.vue frontend/src/i18n/locales/en-US.ts frontend/src/i18n/locales/zh-CN.ts
git commit -m "feat: add skill studio settings page"
```

---

### Task 4: Improve Skill Execution Trace In Chat

**Files:**

- Modify: `internal/agent/tools/skill_execute.go`
- Modify: `frontend/src/views/chat/components/AgentStreamDisplay.vue`
- Test: `internal/agent/tools/skill_execute_test.go`

**Interfaces:**

- Consumes: existing `ToolResult.Data` from `execute_skill_script`.
- Produces: consistently named fields:
  - `skill_name`
  - `script_path`
  - `args`
  - `success`
  - `exit_code`
  - `stdout`
  - `stderr`
  - `artifacts`
  - `artifact_detected`

- [ ] **Step 1: Add backend test for trace fields**

In `internal/agent/tools/skill_execute_test.go`, add:

```go
func TestExecuteSkillScriptResultIncludesTraceFields(t *testing.T) {
	result := buildSkillScriptToolResultForTest("demo", "scripts/run.py", []string{"request.json"})
	for _, key := range []string{"skill_name", "script_path", "args", "success", "artifacts", "artifact_detected"} {
		if _, ok := result.Data[key]; !ok {
			t.Fatalf("missing trace field %s in %#v", key, result.Data)
		}
	}
}
```

If a helper does not exist, create a small fake execution result inside the test and call the formatting function extracted in Step 3.

- [ ] **Step 2: Run test and verify failure**

Run:

```powershell
docker run --rm -v "${PWD}:/src" -w /src golang:1.26-bookworm go test ./internal/agent/tools -run TestExecuteSkillScriptResultIncludesTraceFields -v
```

Expected: FAIL if the formatting function or fields are missing.

- [ ] **Step 3: Extract result formatting**

In `skill_execute.go`, extract a small formatter:

```go
func buildSkillScriptResultData(input ExecuteSkillScriptInput, execution *executor.SkillJobExecution, success bool) map[string]interface{} {
	data := map[string]interface{}{
		"skill_name":        input.SkillName,
		"script_path":       input.ScriptPath,
		"args":              input.Args,
		"success":           success,
		"artifacts":         execution.Artifacts,
		"artifact_detected": execution.ArtifactDetected,
	}
	if execution.Result != nil {
		data["exit_code"] = execution.Result.ExitCode
		data["stdout"] = execution.Result.Stdout
		data["stderr"] = execution.Result.Stderr
	}
	return data
}
```

Use it from `Execute` so test-run and chat trace display share the same field names.

- [ ] **Step 4: Update AgentStreamDisplay rendering**

In `AgentStreamDisplay.vue`, when rendering `execute_skill_script` details, prefer structured fields:

```ts
const skillTrace = computed(() => {
  const data = props.toolCall?.result?.data || {}
  return {
    skillName: data.skill_name,
    scriptPath: data.script_path,
    args: data.args,
    exitCode: data.exit_code,
    artifacts: data.artifacts || [],
    stderr: data.stderr,
  }
})
```

Show script path and artifacts in the expanded raw output area. Keep existing compact timeline labels.

- [ ] **Step 5: Run backend test and frontend build**

Run:

```powershell
docker run --rm -v "${PWD}:/src" -w /src golang:1.26-bookworm go test ./internal/agent/tools -run Skill -v
cd frontend
npm run build
```

Expected: PASS.

- [ ] **Step 6: Commit**

```powershell
git add internal/agent/tools/skill_execute.go internal/agent/tools/skill_execute_test.go frontend/src/views/chat/components/AgentStreamDisplay.vue
git commit -m "feat: surface skill execution trace fields"
```

---

### Task 5: Back Slash Skill Picker With Real Skill Metadata

**Files:**

- Modify: `frontend/src/components/Input-field.vue`
- Modify: `frontend/src/utils/skillDirectiveDisplay.ts`
- Modify: `frontend/src/utils/skillDirectiveDisplay.test.mjs`
- Test: `frontend/src/utils/skillDirectiveDisplay.test.mjs`

**Interfaces:**

- Consumes: `listSkills()` from `frontend/src/api/skill/index.ts`.
- Produces:
  - Slash picker options derived from enabled skills.
  - Curated aliases for important skills such as `officecli-document-editing`.
  - Hidden directive format remains `[Skill Name](xelora-skill://skill-name "...")`.

- [ ] **Step 1: Add display utility test for real skill IDs**

Update `frontend/src/utils/skillDirectiveDisplay.test.mjs` with:

```js
test('keeps real skill label while hiding directive title', () => {
  const input = '使用 [officecli-document-editing](xelora-skill://officecli-document-editing "hidden long instruction")：\n生成 PPT'
  assert.equal(sanitizeSkillDirectiveDisplay(input), '生成 PPT')
})
```

- [ ] **Step 2: Run utility test**

Run:

```powershell
cd frontend
npm test -- src/utils/skillDirectiveDisplay.test.mjs
```

Expected: PASS for existing behavior or FAIL if the regex does not support hyphenated real skill IDs.

- [ ] **Step 3: Replace hard-coded slash options with loaded skill options**

In `Input-field.vue`, import:

```ts
import { listSkills, type SkillSummary } from '@/api/skill'
```

Keep a small curated alias map:

```ts
const SKILL_ALIAS_HINTS: Record<string, { command: string; aliases: string[]; insertText?: string }> = {
  'officecli-document-editing': {
    command: '/office',
    aliases: ['office', 'officecli', 'docx', 'xlsx', 'pptx', 'excel', 'word', 'powerpoint'],
    insertText: '请使用 officecli-document-editing 技能：先 read_skill("officecli-document-editing")，再通过 execute_skill_script 调用，固定 script_path="scripts/officecli_bridge.py"，args 传 JSON 请求文件名，input 传合法 JSON 请求内容，并验证真实产物。'
  },
  xlsx: {
    command: '/xlsx',
    aliases: ['xlsx', 'excel', 'spreadsheet', 'csv']
  }
}
```

Build `SkillSlashOption[]` from API data:

```ts
const skillSlashOptions = ref<SkillSlashOption[]>([])

const loadSkillSlashOptions = async () => {
  const res = await listSkills()
  skillSlashOptions.value = (res.data || [])
    .filter(skill => skill.status !== 'disabled')
    .map((skill: SkillSummary) => {
      const hint = SKILL_ALIAS_HINTS[skill.name]
      return {
        id: skill.name,
        title: skill.name,
        command: hint?.command || `/skill ${skill.name}`,
        description: skill.description,
        aliases: [skill.name, ...(hint?.aliases || [])],
        insertText: hint?.insertText || `请使用 ${skill.name} 技能：先 read_skill("${skill.name}")，再按该技能说明选择脚本和参数执行；完成后验证输出和 artifact。`
      }
    })
}
```

Call `loadSkillSlashOptions()` on mount. If API fails, keep the current minimal fallback for `officecli-document-editing`.

- [ ] **Step 4: Keep hidden directive concise**

Ensure `buildMessageWithSkillDirectives` uses the real skill ID:

```ts
return `[${option.title}](xelora-skill://${option.id} "${hiddenText}")`
```

Do not paste full `SKILL.md` into the directive.

- [ ] **Step 5: Run tests and build**

Run:

```powershell
cd frontend
npm test -- src/utils/skillDirectiveDisplay.test.mjs
npm run build
```

Expected: PASS.

- [ ] **Step 6: Commit**

```powershell
git add frontend/src/components/Input-field.vue frontend/src/utils/skillDirectiveDisplay.ts frontend/src/utils/skillDirectiveDisplay.test.mjs
git commit -m "feat: load slash skills from skill metadata"
```

---

### Task 6: Docker Smoke And Documentation Closeout

**Files:**

- Modify: `docs/customizations/TASKS.md`
- Modify: `docs/customizations/WORKSPACE_OFFICE_PROGRESS.md`
- Modify: `docs/api/skill.md`

**Interfaces:**

- Consumes: Tasks 1-5.
- Produces: verified Docker deployment and documented Skill Studio status.

- [ ] **Step 1: Add task board item**

Add to `docs/customizations/TASKS.md`:

```text
- [~] T-015 Skill Studio first release - expose skill library/detail/test runner, improve execution trace, and back slash skill selection with real skill metadata (@win-main)
```

- [ ] **Step 2: Run backend and frontend validation**

Run:

```powershell
docker run --rm -v "${PWD}:/src" -w /src golang:1.26-bookworm go test ./internal/application/service ./internal/handler ./internal/agent/tools -run 'Skill' -v
cd frontend
npm test -- src/utils/skillDirectiveDisplay.test.mjs
npm run build
```

Expected: all pass.

- [ ] **Step 3: Rebuild app and frontend**

Run:

```powershell
docker compose up -d --build app frontend
docker compose ps app frontend
```

Expected: `Xelora-app` is healthy and `Xelora-frontend` is up.

- [ ] **Step 4: Browser smoke**

Manual browser check:

1. Open `http://localhost/platform/settings?section=skills`.
2. Confirm Skill Studio lists `officecli-document-editing`.
3. Open its detail panel and confirm `scripts/officecli_bridge.py` is visible.
4. Use the test runner with a validation request in a bound test workspace.
5. Open chat input and type `/office`; confirm it inserts a visible chip and the sent message hides the long directive.

- [ ] **Step 5: Mark task complete and update progress**

After browser smoke passes, update:

```text
- [x] T-015 Skill Studio first release - ...
```

Add a short evidence note to `docs/customizations/WORKSPACE_OFFICE_PROGRESS.md`:

```text
2026-07-13: Skill Studio first release verified. Admins can inspect skills,
open skill details, test-run skill scripts through the gateway, see trace
fields, and select real skills through the slash picker.
```

- [ ] **Step 6: Commit**

```powershell
git add docs/customizations/TASKS.md docs/customizations/WORKSPACE_OFFICE_PROGRESS.md docs/api/skill.md
git commit -m "docs: record skill studio verification"
```

---

## Self-Review

Spec coverage:

- Skill Library: Task 1 and Task 3.
- Skill Detail: Task 1 and Task 3.
- Skill Test Runner: Task 2 and Task 3.
- Agent Skill Policy: Task 3 documents current modes in UI context; deeper policy changes are intentionally deferred.
- Slash Skill Picker: Task 5.
- Execution Trace: Task 4.
- Non-goals: no task adds marketplace installation, source editing, versioning, or business-domain modeling.
- Verification: Task 6.

Placeholder scan:

- No unresolved placeholder markers remain in the task steps.
- Every task has concrete files, commands, and expected results.

Type consistency:

- Backend response names use `SkillSummary`, `SkillDetail`, `SkillScriptSummary`, and `SkillTestRunResult`.
- Frontend API mirrors the backend JSON names.
- Slash directive format stays compatible with `sanitizeSkillDirectiveDisplay`.
