import { getDownloadUrl, type SkillDetail } from '../../api';
import { formatDateTime, skillRef, summarizeScanStatus } from '../../lib/format';
import { CopyActionButton } from './CopyActionButton';

type SkillHeaderProps = {
  onBack?: () => void;
  skill: SkillDetail;
};

export function SkillHeader({ onBack, skill }: SkillHeaderProps) {
  const latestVersion = skill.latest_version || skill.versions?.[0]?.version || 'draft';
  const latestScan = summarizeScanStatus(skill.versions?.[0]?.scan_status);
  const visibilityLabel = skill.is_public === false ? '私有' : '公开';

  return (
    <div className="gh-object-header">
      <div className="gh-object-header__main">
        <div className="gh-object-header__breadcrumb">
          {onBack ? (
            <button className="gh-object-header__back" onClick={onBack} type="button">
              Skills
            </button>
          ) : null}
          <span className="gh-object-header__scope">Skill</span>
        </div>

        <div className="gh-object-header__title-row">
          <h1>{skill.namespace} / {skill.name}</h1>
          <div className="chip-row chip-row--start">
            <span className={`status-pill status-pill--${latestScan.tone}`}>{latestScan.label}</span>
            <span className="catalog-chip">{latestVersion}</span>
            <span className="catalog-chip">{visibilityLabel}</span>
            {skill.is_deprecated ? (
              <span className="status-pill status-pill--warning">已弃用</span>
            ) : null}
          </div>
        </div>

        <p className="gh-object-header__description">{skill.description || '暂无描述。'}</p>

        <div className="gh-object-header__meta">
          <code>{skillRef(skill)}</code>
          <span>{skill.versions?.length || 0} 个版本</span>
          <span>{skill.download_count} 下载</span>
          <span>{formatDateTime(skill.updated_at)} 更新</span>
        </div>
      </div>

      <div className="gh-object-header__actions">
        <a
          className="button button--primary"
          href={getDownloadUrl(skill)}
          rel="noreferrer"
          target="_blank"
        >
          下载 ZIP
        </a>
        <CopyActionButton label="复制引用" value={skillRef(skill)} />
        <CopyActionButton
          className="button button--ghost"
          label="复制下载链接"
          value={getDownloadUrl(skill)}
        />
      </div>
    </div>
  );
}
