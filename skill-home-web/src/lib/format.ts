import { getDownloadUrl, type SkillDetail, type SkillSummary } from '../api';

type SkillLike = SkillSummary | SkillDetail;

export type InstallIDE = 'codex' | 'claude' | 'cursor' | 'copilot';

export type InstallRecipe = {
  id: InstallIDE;
  label: string;
  description: string;
  reference: string;
  search: string;
  pull: string;
  install: string;
  verify: string;
  download: string;
};

const installLabels: Record<InstallIDE, { label: string; description: string }> = {
  codex: {
    label: 'Codex',
    description: '推荐 WSL / Windows 混合环境使用 mirror 模式，安装后可直接被 Codex 发现。',
  },
  claude: {
    label: 'Claude Code',
    description: '适合团队内共享技能库，用统一命令同步到 Claude Code 技能目录。',
  },
  cursor: {
    label: 'Cursor',
    description: '会自动导出为 Cursor 规则格式，适合在编辑器内快速复用。',
  },
  copilot: {
    label: 'GitHub Copilot',
    description: '适合需要统一团队技能分发的 Copilot 工作流。',
  },
};

function getPrimaryVersion(skill: SkillLike) {
  if (skill.latest_version) {
    return skill.latest_version;
  }

  if ('versions' in skill && skill.versions?.length) {
    return skill.versions[0]?.version;
  }

  return undefined;
}

function parseValidDate(value?: string) {
  if (!value) {
    return null;
  }

  const normalized = value.trim();
  if (!normalized || normalized.startsWith('0001-01-01') || normalized.startsWith('0000-00-00')) {
    return null;
  }

  const date = new Date(normalized);
  if (Number.isNaN(date.getTime()) || date.getUTCFullYear() <= 1) {
    return null;
  }

  return date;
}

export function skillKey(skill: Pick<SkillLike, 'namespace' | 'name'>) {
  return `${skill.namespace}/${skill.name}`;
}

export function skillRef(skill: Pick<SkillLike, 'namespace' | 'name'>) {
  return `@${skill.namespace}/${skill.name}`;
}

export function formatDate(value?: string) {
  const date = parseValidDate(value);
  if (!date) {
    return '未记录';
  }

  return new Intl.DateTimeFormat('zh-CN', {
    month: 'short',
    day: 'numeric',
  }).format(date);
}

export function formatDateTime(value?: string) {
  const date = parseValidDate(value);
  if (!date) {
    return '未记录';
  }

  return new Intl.DateTimeFormat('zh-CN', {
    year: 'numeric',
    month: 'short',
    day: 'numeric',
    hour: '2-digit',
    minute: '2-digit',
  }).format(date);
}

export function formatBytes(value?: number) {
  if (!value || value <= 0) {
    return '未记录';
  }

  if (value < 1024) {
    return `${value} B`;
  }

  if (value < 1024 * 1024) {
    return `${(value / 1024).toFixed(1)} KB`;
  }

  return `${(value / (1024 * 1024)).toFixed(1)} MB`;
}

export function parseTags(value: string) {
  return value
    .split(',')
    .map((item) => item.trim())
    .filter(Boolean);
}

export function copyText(value: string) {
  if (typeof navigator !== 'undefined' && navigator.clipboard?.writeText) {
    return navigator.clipboard.writeText(value).catch(() => copyTextFallback(value));
  }

  return copyTextFallback(value);
}

function copyTextFallback(value: string) {
  if (typeof document === 'undefined') {
    return Promise.reject(new Error('clipboard-unavailable'));
  }

  const textarea = document.createElement('textarea');
  textarea.value = value;
  textarea.setAttribute('readonly', 'true');
  textarea.style.position = 'fixed';
  textarea.style.opacity = '0';
  textarea.style.pointerEvents = 'none';

  document.body.appendChild(textarea);
  textarea.select();
  textarea.setSelectionRange(0, textarea.value.length);

  const copied = document.execCommand('copy');
  document.body.removeChild(textarea);

  if (!copied) {
    return Promise.reject(new Error('clipboard-unavailable'));
  }

  return Promise.resolve();
}

export function getInstallRecipes(skill: SkillLike): InstallRecipe[] {
  const version = getPrimaryVersion(skill);
  const ref = skillRef(skill);
  const targetRef = version ? `${ref}@${version}` : ref;

  return (Object.keys(installLabels) as InstallIDE[]).map((id) => ({
    id,
    label: installLabels[id].label,
    description: installLabels[id].description,
    reference: ref,
    search: `skill-home search ${skill.name}`,
    pull: `skill-home pull ${targetRef}`,
    install: `skill-home install ${targetRef} --ide ${id} --global --mode mirror`,
    verify: 'skill-home doctor',
    download: getDownloadUrl(skill),
  }));
}

export function summarizeScanStatus(status?: string) {
  switch ((status || '').toLowerCase()) {
    case 'pass':
      return { label: '扫描通过', tone: 'success' as const };
    case 'fail':
      return { label: '扫描失败', tone: 'danger' as const };
    case 'pending':
      return { label: '扫描中', tone: 'warning' as const };
    default:
      return { label: '状态未知', tone: 'neutral' as const };
  }
}
