import assert from 'node:assert/strict';
import test from 'node:test';
import { readFileSync } from 'node:fs';

const source = readFileSync(new URL('./skillSlashOptions.ts', import.meta.url), 'utf8');
const functionBody = source
  .replace(/export interface [\s\S]*?\n\}/g, '')
  .replace(/export function buildSkillSlashOptions/, 'function buildSkillSlashOptions')
  .replace(/export function filterSkillSlashOptions/, 'function filterSkillSlashOptions')
  .replace(/\?: string/g, '')
  .replace(/: string\[\]/g, '')
  .replace(/: SkillSlashOption\[\]/g, '')
  .replace(/: SkillSlashOption/g, '')
  .replace(/: SkillLike\[\]/g, '')
  .replace(/: SkillLike/g, '')
  .replace(/: string/g, '');

const helpers = Function(`${functionBody}; return { buildSkillSlashOptions, filterSkillSlashOptions };`)();

test('builds slash options from backend skills', () => {
  const options = helpers.buildSkillSlashOptions([
    { name: 'officecli-document-editing', description: 'Office bridge', scripts: [{ path: 'scripts/officecli_bridge.py' }] },
    { name: 'xlsx', description: 'Spreadsheet workflow' },
  ]);

  assert.ok(options.some(option => option.id === 'officecli-sdk'));
  assert.ok(options.some(option => option.id === 'skill:officecli-document-editing'));
  assert.ok(options.some(option => option.id === 'skill:xlsx'));
  assert.match(options.find(option => option.id === 'skill:officecli-document-editing').insertText, /script_path="scripts\/officecli_bridge.py"/);
});

test('filters slash options by title command description and aliases', () => {
  const options = helpers.buildSkillSlashOptions([
    { name: 'xlsx', description: 'Spreadsheet workflow' },
  ]);

  assert.deepEqual(
    helpers.filterSkillSlashOptions(options, 'excel').map(option => option.id),
    ['officecli-sdk', 'skill:xlsx'],
  );
});
