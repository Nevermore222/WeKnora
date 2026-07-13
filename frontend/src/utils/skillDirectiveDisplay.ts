const SKILL_DIRECTIVE_LINK_RE = /\[([^\]]+)\]\(xelora-skill:\/\/[^\s)]+(?:\s+"(?:\\.|[^"\\])*")?\)/g;

export function sanitizeSkillDirectiveDisplay(content?: string): string {
  const text = (content || '').trim();
  if (!text.includes('xelora-skill://')) return content || '';

  const withLabels = text.replace(SKILL_DIRECTIVE_LINK_RE, '$1');
  const prefixMatch = withLabels.match(/^使用\s+(.+?)：\s*([\s\S]*)$/);
  if (!prefixMatch) return withLabels;

  const userText = prefixMatch[2].trim();
  return userText || `使用 ${prefixMatch[1].trim()}`;
}
