import { getSkillDescription, type SkillSummary } from '../../api';
import { formatDate, skillRef } from '../../lib/format';

type SkillResultListProps = {
  activeFilterLabels: string[];
  error: string | null;
  loading: boolean;
  onClearFilters: () => void;
  onOpen: (namespace: string, name: string) => void;
  onRefresh: () => void;
  skills: SkillSummary[];
  sortLabel: string;
  total: number;
  view: 'cards' | 'list';
};

function formatRatingLabel(skill: SkillSummary) {
  if (!skill.rating_count) {
    return {
      score: '暂无评分',
      count: '等待首个评分',
    };
  }

  return {
    score: `${(skill.rating || 0).toFixed(1)} 分`,
    count: `${skill.rating_count} 人评分`,
  };
}

function ResultCard({
  onOpen,
  skill,
}: {
  onOpen: (namespace: string, name: string) => void;
  skill: SkillSummary;
}) {
  const ratingLabel = formatRatingLabel(skill);
  const description = getSkillDescription(skill);

  return (
    <article className="skill-result-card">
      <div className="skill-result-card__header">
        <div>
          <strong>{skill.name}</strong>
          <span>{skillRef(skill)}</span>
        </div>
        <div className="chip-row chip-row--start">
          {skill.is_recommended ? (
            <span className="status-pill status-pill--success">推荐</span>
          ) : null}
          <span className="search-badge">{skill.latest_version || 'draft'}</span>
        </div>
      </div>
      <p>{description || '暂无描述。'}</p>
      <div className="skill-result-card__meta">
        <span>{skill.license || '未填写 License'}</span>
        <span>{skill.download_count} 下载</span>
        <span>{formatDate(skill.updated_at)} 更新</span>
      </div>
      <div className="skill-result-card__signals">
        <span className="search-badge">{ratingLabel.score}</span>
        <span>{ratingLabel.count}</span>
      </div>
      <button className="button button--secondary" onClick={() => onOpen(skill.namespace, skill.name)} type="button">
        打开
      </button>
    </article>
  );
}

function ResultRow({
  onOpen,
  skill,
}: {
  onOpen: (namespace: string, name: string) => void;
  skill: SkillSummary;
}) {
  const ratingLabel = formatRatingLabel(skill);
  const description = getSkillDescription(skill);

  return (
    <article className="skill-result-row">
      <div className="skill-result-row__identity">
        <button className="skill-result-row__title" onClick={() => onOpen(skill.namespace, skill.name)} type="button">
          {skill.name}
        </button>
        <span>{skillRef(skill)}</span>
      </div>
      <p>{description || '暂无描述。'}</p>
      <div className="skill-result-row__meta">
        {skill.is_recommended ? <span>推荐</span> : null}
        <span>{skill.license || '未填写 License'}</span>
        <span>{skill.latest_version || 'draft'}</span>
        <span>{skill.download_count} 下载</span>
        <span>{formatDate(skill.updated_at)} 更新</span>
        <span>{ratingLabel.score}</span>
        <span>{ratingLabel.count}</span>
      </div>
      {(skill.tags || []).length ? (
        <div className="skill-result-row__tags">
          {(skill.tags || []).slice(0, 4).map((tag) => (
            <span className="search-badge" key={tag}>
              {tag}
            </span>
          ))}
        </div>
      ) : null}
    </article>
  );
}

export function SkillResultList({
  activeFilterLabels,
  error,
  loading,
  onClearFilters,
  onOpen,
  onRefresh,
  skills,
  sortLabel,
  total,
  view,
}: SkillResultListProps) {
  const refreshing = loading && skills.length > 0;
  const loadingEmptyState = loading && !skills.length;

  return (
    <section className="skill-results-panel">
      <div className="skill-results-panel__toolbar">
        <div>
          <strong>{total} 结果</strong>
          <span>{sortLabel}</span>
          {refreshing ? <span aria-live="polite">更新结果中...</span> : null}
        </div>
        <button className="button button--ghost" onClick={onRefresh} type="button">
          刷新
        </button>
      </div>

      {activeFilterLabels.length ? (
        <div className="skill-results-panel__filters">
          {activeFilterLabels.map((label) => (
            <span className="search-badge" key={label}>
              {label}
            </span>
          ))}
          <button className="button button--ghost" onClick={onClearFilters} type="button">
            清除筛选
          </button>
        </div>
      ) : null}

      {loadingEmptyState ? (
        <div className="empty-panel">正在读取技能目录...</div>
      ) : error ? (
        <div className="empty-panel empty-panel--danger">读取技能目录失败：{error}</div>
      ) : !skills.length ? (
        <div className="empty-panel">没有找到匹配的 skill，换个关键词或筛选条件试试。</div>
      ) : view === 'cards' ? (
        <div className="skill-result-grid">
          {skills.map((skill) => (
            <ResultCard key={`${skill.namespace}/${skill.name}`} onOpen={onOpen} skill={skill} />
          ))}
        </div>
      ) : (
        <div className="skill-result-list">
          {skills.map((skill) => (
            <ResultRow key={`${skill.namespace}/${skill.name}`} onOpen={onOpen} skill={skill} />
          ))}
        </div>
      )}
    </section>
  );
}
