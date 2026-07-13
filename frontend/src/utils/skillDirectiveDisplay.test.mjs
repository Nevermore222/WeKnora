import assert from 'node:assert/strict';
import test from 'node:test';
import { readFileSync } from 'node:fs';

const source = readFileSync(new URL('./skillDirectiveDisplay.ts', import.meta.url), 'utf8');
const functionBody = source
  .replace(/export function sanitizeSkillDirectiveDisplay/, 'function sanitizeSkillDirectiveDisplay')
  .replace(/content\?: string/g, 'content')
  .replace(/\): string/g, ')')
  .replace(/export \{[^}]+\};?/g, '');
const sanitizeSkillDirectiveDisplay = Function(`${functionBody}; return sanitizeSkillDirectiveDisplay;`)();

test('hides xelora skill link title from user-visible message', () => {
  const raw = '使用 [OfficeCLI SDK](xelora-skill://officecli-sdk "请使用 OfficeCLI SDK（officecli-document-editing 技能）：Office 文件必须通过 execute_skill_script 调用该技能")：\n查看数据库并整理为ppt并输出';
  assert.equal(sanitizeSkillDirectiveDisplay(raw), '查看数据库并整理为ppt并输出');
});

test('keeps normal user text unchanged', () => {
  assert.equal(sanitizeSkillDirectiveDisplay('普通问题'), '普通问题');
});

test('shows a compact skill label when no user task text exists', () => {
  const raw = '使用 [OfficeCLI SDK](xelora-skill://officecli-sdk "hidden")：';
  assert.equal(sanitizeSkillDirectiveDisplay(raw), '使用 OfficeCLI SDK');
});
