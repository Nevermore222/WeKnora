export interface SkillLike {
  name: string;
  description?: string;
  scripts?: Array<{ path: string; language?: string }>;
}

export interface SkillSlashOption {
  id: string;
  title: string;
  command: string;
  description: string;
  aliases: string[];
  insertText: string;
}

const OFFICE_SKILL_NAME = 'officecli-document-editing';
const OFFICE_BRIDGE_SCRIPT = 'scripts/officecli_bridge.py';

function wordsFromName(name: string): string[] {
  return name.split(/[-_\s]+/).filter(Boolean);
}

function aliasesForSkill(skill: SkillLike): string[] {
  const aliases = new Set([skill.name, ...wordsFromName(skill.name)]);
  if (/\b(xlsx|excel|spreadsheet|csv)\b/i.test(`${skill.name} ${skill.description || ''}`)) {
    ['excel', 'spreadsheet', 'csv', 'sheet'].forEach(alias => aliases.add(alias));
  }
  return Array.from(aliases);
}

function buildGenericSkillOption(skill: SkillLike): SkillSlashOption {
  const scriptHint = skill.scripts?.[0]?.path
    ? `；如需执行脚本，优先使用 script_path="${skill.scripts[0].path}"`
    : '';

  return {
    id: `skill:${skill.name}`,
    title: skill.name,
    command: `/skill ${skill.name}`,
    description: skill.description || '使用该 Skill 的说明、模板、脚本和验证步骤。',
    aliases: aliasesForSkill(skill),
    insertText: `请使用 ${skill.name} 技能处理该任务：先 read_skill("${skill.name}")，理解 SKILL.md 的步骤、模板和脚本约束；需要执行时通过 execute_skill_script 调用该技能${scriptHint}；完成后验证真实结果。\n`,
  };
}

function buildOfficeSkillOption(skill: SkillLike): SkillSlashOption {
  return {
    id: `skill:${OFFICE_SKILL_NAME}`,
    title: OFFICE_SKILL_NAME,
    command: `/skill ${OFFICE_SKILL_NAME}`,
    description: skill.description || '创建或修改 .docx/.xlsx/.pptx 的主入口，避免模型绕开 OfficeCLI bridge。',
    aliases: [OFFICE_SKILL_NAME, 'officecli', 'document', 'office', 'docx', 'xlsx', 'pptx', 'excel', 'word', 'powerpoint'],
    insertText: `请使用 ${OFFICE_SKILL_NAME} 技能处理这个 Office 文件任务：先 read_skill("${OFFICE_SKILL_NAME}")，再用 execute_skill_script，固定 skill_name="${OFFICE_SKILL_NAME}"，script_path="${OFFICE_BRIDGE_SCRIPT}"，args 传 JSON 请求文件名，input 传合法 JSON 请求内容；需要 openpyxl/python-docx/python-pptx 时仅在该 bridge 内用 run_python；完成后验证实际产物文件。\n`,
  };
}

export function buildSkillSlashOptions(skills: SkillLike[] = []): SkillSlashOption[] {
  const dynamicSkillOptions = skills.map(skill =>
    skill.name === OFFICE_SKILL_NAME ? buildOfficeSkillOption(skill) : buildGenericSkillOption(skill),
  );

  return [
    {
      id: 'officecli-sdk',
      title: 'OfficeCLI SDK',
      command: '/office',
      description: '标注 Office 文件必须走 officecli-document-editing，并在 bridge 内使用 SDK 级编辑。',
      aliases: ['office', 'officecli', 'sdk', 'docx', 'xlsx', 'pptx', 'excel', 'word', 'powerpoint'],
      insertText: `请使用 OfficeCLI SDK（${OFFICE_SKILL_NAME} 技能）：Office 文件必须通过 execute_skill_script 调用该技能；简单内容用 write_docx/write_xlsx；原生 OfficeCLI 命令用 action=officecli；需要 openpyxl/python-docx/python-pptx 时仅在该 bridge 内用 run_python；生成 PPT 时必须先整理证据，形成标题、关键发现、数据/文档摘要、详情、风险/缺口、下一步等清晰结构，并验证真实产物。\n`,
    },
    ...dynamicSkillOptions,
    {
      id: 'browser',
      title: 'browser_navigate',
      command: '/browser',
      description: '需要打开、检查或测试网页时，提示模型使用浏览器工具而不是只描述。',
      aliases: ['browser', 'web', 'page', 'navigate', 'test'],
      insertText: '请使用 browser_navigate 打开并检查页面；需要验证交互时请实际操作浏览器并报告结果。\n',
    },
    {
      id: 'workspace-files',
      title: 'workspace file tools',
      command: '/files',
      description: '通用文件写入提示，要求模型真实创建/修改工作区文件并验证。',
      aliases: ['file', 'files', 'workspace', 'write', 'artifact'],
      insertText: '请在当前绑定工作区内真实创建或修改文件；优先选择合适 Skill，通过 execute_skill_script 执行，并在完成后验证文件存在和内容。\n',
    },
  ];
}

export function filterSkillSlashOptions(options: SkillSlashOption[], query: string): SkillSlashOption[] {
  const q = query.trim().toLowerCase();
  if (!q) return options;
  return options.filter(option => {
    const haystack = [
      option.title,
      option.command,
      option.description,
      ...option.aliases,
    ].join(' ').toLowerCase();
    return haystack.includes(q);
  });
}
