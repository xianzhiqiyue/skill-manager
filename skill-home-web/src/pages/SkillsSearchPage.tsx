import type { SkillSummary } from '../api';
import { PageHeader } from '../components/layout/PageHeader';
import { SidebarLayout } from '../components/layout/SidebarLayout';
import { FilterRail } from '../components/search/FilterRail';
import { SkillResultList } from '../components/search/SkillResultList';
import { type CatalogFilters } from '../lib/catalogState';

export type SkillsSearchPageModel = {
  catalogFilters: CatalogFilters;
  licenseOptions: string[];
  namespaceOptions: string[];
  openSkill: (namespace: string, name: string) => void;
  refreshCatalog: () => void;
  resetCatalogFilters: () => void;
  setCatalogQuery: (value: string) => void;
  skills: SkillSummary[];
  skillsError: string | null;
  skillsLoading: boolean;
  skillsTotal: number;
  tagOptions: string[];
  updateCatalogFilter: (
    key: 'namespace' | 'tag' | 'license' | 'sort' | 'view',
    value: string,
  ) => void;
};

const sortLabels = {
  downloads: '热门优先',
  updated: '最近更新',
  rating: '高评分',
  name: '按名称',
} as const;

function buildActiveFilterLabels(filters: CatalogFilters) {
  const labels: string[] = [];

  if (filters.query.trim()) {
    labels.push(`关键词：${filters.query.trim()}`);
  }
  if (filters.namespace !== 'all') {
    labels.push(`命名空间：@${filters.namespace}`);
  }
  if (filters.tag !== 'all') {
    labels.push(`标签：${filters.tag}`);
  }
  if (filters.license !== 'all') {
    labels.push(`License：${filters.license}`);
  }

  return labels;
}

export function SkillsSearchPage({ model }: { model: SkillsSearchPageModel }) {
  const activeFilterLabels = buildActiveFilterLabels(model.catalogFilters);

  return (
    <div className="page-stack">
      <section className="surface-panel surface-panel--search">
        <PageHeader
          eyebrow="Search"
          title="技能搜索工作台"
          description="用 GitHub 式搜索工作台浏览 skills。筛选条件写进 URL，结果区只保留适合快速扫描的核心信息。"
        />

        <SidebarLayout
          className="gh-sidebar-layout--search"
          content={(
            <SkillResultList
              activeFilterLabels={activeFilterLabels}
              error={model.skillsError}
              loading={model.skillsLoading}
              onClearFilters={model.resetCatalogFilters}
              onOpen={model.openSkill}
              onRefresh={model.refreshCatalog}
              skills={model.skills}
              sortLabel={sortLabels[model.catalogFilters.sort]}
              total={model.skillsTotal}
              view={model.catalogFilters.view}
            />
          )}
          sidebar={(
            <FilterRail
              activeFilterLabels={activeFilterLabels}
              filters={model.catalogFilters}
              licenseOptions={model.licenseOptions}
              namespaceOptions={model.namespaceOptions}
              onChange={model.updateCatalogFilter}
              onQueryChange={model.setCatalogQuery}
              onReset={model.resetCatalogFilters}
              skills={model.skills}
              tagOptions={model.tagOptions}
            />
          )}
        />
      </section>
    </div>
  );
}
