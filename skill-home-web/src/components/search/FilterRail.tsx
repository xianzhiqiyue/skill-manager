import { useEffect, useState } from 'react';

import type { SkillSummary } from '../../api';
import type { CatalogFilters, CatalogSort, CatalogView } from '../../lib/catalogState';

type FilterRailProps = {
  activeFilterLabels: string[];
  filters: CatalogFilters;
  licenseOptions: string[];
  namespaceOptions: string[];
  onChange: (
    key: 'namespace' | 'tag' | 'license' | 'sort' | 'view',
    value: string,
  ) => void;
  onQueryChange: (value: string) => void;
  onReset: () => void;
  skills: SkillSummary[];
  tagOptions: string[];
};

const sortOptions: Array<{ label: string; value: CatalogSort }> = [
  { label: '热门', value: 'downloads' },
  { label: '最近更新', value: 'updated' },
  { label: '高评分', value: 'rating' },
  { label: '名称', value: 'name' },
];

const viewOptions: Array<{ label: string; value: CatalogView }> = [
  { label: '列表', value: 'list' },
  { label: '卡片', value: 'cards' },
];

function collectTopTags(skills: SkillSummary[]) {
  const counts = new Map<string, number>();

  skills.forEach((skill) => {
    (skill.tags || []).forEach((tag) => {
      counts.set(tag, (counts.get(tag) || 0) + 1);
    });
  });

  return Array.from(counts.entries())
    .sort((left, right) => right[1] - left[1] || left[0].localeCompare(right[0], 'zh-CN'))
    .slice(0, 8)
    .map(([tag]) => tag);
}

export function FilterRail({
  activeFilterLabels,
  filters,
  licenseOptions,
  namespaceOptions,
  onChange,
  onQueryChange,
  onReset,
  skills,
  tagOptions,
}: FilterRailProps) {
  const [queryDraft, setQueryDraft] = useState(filters.query);
  const topTags = collectTopTags(skills);

  useEffect(() => {
    setQueryDraft(filters.query);
  }, [filters.query]);

  useEffect(() => {
    if (queryDraft === filters.query) {
      return;
    }

    const timer = window.setTimeout(() => {
      onQueryChange(queryDraft);
    }, 250);

    return () => window.clearTimeout(timer);
  }, [filters.query, onQueryChange, queryDraft]);

  return (
    <aside className="search-filter-rail">
      <div className="search-filter-rail__header">
        <div>
          <strong>Filter by</strong>
          <span>保留 URL 可分享的筛选条件。</span>
        </div>
        {activeFilterLabels.length ? (
          <button className="button button--ghost" onClick={onReset} type="button">
            清除
          </button>
        ) : null}
      </div>

      <label className="field">
        <span>关键词</span>
        <input
          onBlur={() => {
            if (queryDraft !== filters.query) {
              onQueryChange(queryDraft);
            }
          }}
          onChange={(event) => setQueryDraft(event.target.value)}
          onKeyDown={(event) => {
            if (event.key === 'Enter' && queryDraft !== filters.query) {
              onQueryChange(queryDraft);
            }
          }}
          placeholder="搜索 skill、能力、场景"
          value={queryDraft}
        />
      </label>

      <label className="field">
        <span>命名空间</span>
        <select
          onChange={(event) => onChange('namespace', event.target.value)}
          value={filters.namespace}
        >
          <option value="all">全部</option>
          {namespaceOptions.map((namespace) => (
            <option key={namespace} value={namespace}>
              @{namespace}
            </option>
          ))}
        </select>
      </label>

      <label className="field">
        <span>标签</span>
        <select onChange={(event) => onChange('tag', event.target.value)} value={filters.tag}>
          <option value="all">全部</option>
          {tagOptions.map((tag) => (
            <option key={tag} value={tag}>
              {tag}
            </option>
          ))}
        </select>
      </label>

      {topTags.length ? (
        <div className="search-filter-rail__group">
          <strong>常用标签</strong>
          <div className="search-filter-rail__chips">
            {topTags.map((tag) => (
              <button
                className={`search-chip ${filters.tag === tag ? 'is-active' : ''}`.trim()}
                key={tag}
                onClick={() => onChange('tag', filters.tag === tag ? 'all' : tag)}
                type="button"
              >
                {tag}
              </button>
            ))}
          </div>
        </div>
      ) : null}

      <label className="field">
        <span>License</span>
        <select
          onChange={(event) => onChange('license', event.target.value)}
          value={filters.license}
        >
          <option value="all">全部</option>
          {licenseOptions.map((license) => (
            <option key={license} value={license}>
              {license}
            </option>
          ))}
        </select>
      </label>

      <div className="search-filter-rail__group">
        <strong>排序</strong>
        <div className="search-filter-rail__segmented">
          {sortOptions.map((option) => (
            <button
              className={`search-segment ${filters.sort === option.value ? 'is-active' : ''}`.trim()}
              key={option.value}
              onClick={() => onChange('sort', option.value)}
              type="button"
            >
              {option.label}
            </button>
          ))}
        </div>
      </div>

      <div className="search-filter-rail__group">
        <strong>视图</strong>
        <div className="search-filter-rail__segmented">
          {viewOptions.map((option) => (
            <button
              className={`search-segment ${filters.view === option.value ? 'is-active' : ''}`.trim()}
              key={option.value}
              onClick={() => onChange('view', option.value)}
              type="button"
            >
              {option.label}
            </button>
          ))}
        </div>
      </div>

      <div className="search-filter-rail__summary">
        <span>{namespaceOptions.length} 个命名空间</span>
        <span>{licenseOptions.length} 个 License</span>
      </div>
    </aside>
  );
}
