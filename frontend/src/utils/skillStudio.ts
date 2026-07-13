import type { SkillDetail, SkillInfo, SkillTestRunRequest } from '@/api/skill'

export interface SkillStudioStats {
  scriptCount: number
  fileCount: number
  instructionLines: number
}

export function getSkillStudioStats(skill?: SkillInfo | SkillDetail | null): SkillStudioStats {
  const scripts = Array.isArray(skill?.scripts) ? skill.scripts : []
  const files = 'files' in (skill || {}) && Array.isArray((skill as SkillDetail).files) ? (skill as SkillDetail).files : []
  const instructions = 'instructions' in (skill || {}) ? (skill as SkillDetail).instructions || '' : ''

  return {
    scriptCount: scripts.length,
    fileCount: files.length,
    instructionLines: instructions.trim() ? instructions.trim().split(/\r?\n/).length : 0,
  }
}

export function getSkillPrimaryScript(skill?: SkillInfo | SkillDetail | null): string {
  const scripts = Array.isArray(skill?.scripts) ? skill.scripts : []
  return scripts[0]?.path || 'No script'
}

export function buildSkillTestRunPayload(skill?: SkillInfo | SkillDetail | null): SkillTestRunRequest {
  const scriptPath = getSkillPrimaryScript(skill)
  return {
    script_path: scriptPath === 'No script' ? '' : scriptPath,
    args: [],
    input: '',
  }
}
