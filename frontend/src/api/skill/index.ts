import { get } from "../../utils/request";

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
