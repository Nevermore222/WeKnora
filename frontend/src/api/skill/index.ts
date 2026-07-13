import { get, post } from "../../utils/request";

export interface SkillScriptSummary {
  path: string;
  language: string;
}

export interface SkillFileSummary {
  path: string;
  is_script: boolean;
}

export interface SkillInfo {
  name: string;
  description: string;
  source?: string;
  status?: string;
  scripts?: SkillScriptSummary[];
}

export interface SkillDetail extends SkillInfo {
  instructions: string;
  scripts: SkillScriptSummary[];
  files: SkillFileSummary[];
}

export interface SkillFileContent {
  path: string;
  content: string;
  is_script: boolean;
}

export interface SkillTestRunRequest {
  provider?: string;
  script_path: string;
  args?: string[];
  input?: string;
  workspace_id?: string;
}

export interface SkillTestRunArtifact {
  name: string;
  relative_path: string;
  kind: string;
  size: number;
}

export interface SkillTestRunResult {
  skill_name: string;
  script_path: string;
  args?: string[];
  success: boolean;
  exit_code?: number;
  stdout: string;
  stderr: string;
  error?: string;
  artifacts: SkillTestRunArtifact[];
  artifact_detected: boolean;
}

// List preloaded Skills. skills_available=false means the sandbox is disabled.
export function listSkills() {
  return get('/api/v1/skills') as Promise<{ data: SkillInfo[]; skills_available?: boolean }>;
}

export function getSkill(name: string) {
  return get(`/api/v1/skills/${encodeURIComponent(name)}`) as Promise<{ data: SkillDetail }>;
}

export function getSkillFile(name: string, path: string) {
  const encodedPath = path.split('/').map(part => encodeURIComponent(part)).join('/');
  return get(`/api/v1/skills/${encodeURIComponent(name)}/files/${encodedPath}`) as Promise<{ data: SkillFileContent }>;
}

export function testRunSkill(name: string, payload: SkillTestRunRequest) {
  return post(`/api/v1/skills/${encodeURIComponent(name)}/test-run`, payload) as Promise<{ data: SkillTestRunResult }>;
}
