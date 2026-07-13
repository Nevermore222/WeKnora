import assert from 'node:assert/strict';
import test from 'node:test';
import { readFileSync } from 'node:fs';

const source = readFileSync(new URL('./skillStudio.ts', import.meta.url), 'utf8');
const functionBody = source
  .replace(/import type \{[^}]+\} from '[^']+'\n/g, '')
  .replace(/export interface SkillStudioStats \{[\s\S]*?\}\n/g, '')
  .replace(/export function getSkillStudioStats/, 'function getSkillStudioStats')
  .replace(/export function getSkillPrimaryScript/, 'function getSkillPrimaryScript')
  .replace(/export function buildSkillTestRunPayload/, 'function buildSkillTestRunPayload')
  .replace(/\?: SkillInfo \| SkillDetail \| null/g, '')
  .replace(/: SkillStudioStats/g, '')
  .replace(/: SkillTestRunRequest/g, '')
  .replace(/: string/g, '')
  .replace(/ as SkillDetail/g, '');

const helpers = Function(`${functionBody}; return { getSkillStudioStats, getSkillPrimaryScript, buildSkillTestRunPayload };`)();

test('summarizes skill detail for studio cards', () => {
  const skill = {
    scripts: [{ path: 'scripts/run.py', language: 'python' }],
    files: [{ path: 'SKILL.md', is_script: false }, { path: 'scripts/run.py', is_script: true }],
    instructions: 'Step one\nStep two',
  };

  assert.deepEqual(helpers.getSkillStudioStats(skill), {
    scriptCount: 1,
    fileCount: 2,
    instructionLines: 2,
  });
  assert.equal(helpers.getSkillPrimaryScript(skill), 'scripts/run.py');
});

test('handles skills without resource summaries', () => {
  assert.deepEqual(helpers.getSkillStudioStats({ name: 'plain', description: 'Plain skill' }), {
    scriptCount: 0,
    fileCount: 0,
    instructionLines: 0,
  });
  assert.equal(helpers.getSkillPrimaryScript(null), 'No script');
});

test('builds a default skill test-run payload from the primary script', () => {
  const skill = {
    scripts: [{ path: 'scripts/run.py', language: 'python' }],
  };

  assert.deepEqual(helpers.buildSkillTestRunPayload(skill), {
    script_path: 'scripts/run.py',
    args: [],
    input: '',
  });
});
