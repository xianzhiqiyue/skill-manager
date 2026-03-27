import {
  SkillObjectPage,
  type SkillObjectPageModel,
} from '../../components/object/SkillObjectPage';
import { formatDateTime, summarizeScanStatus } from '../../lib/format';

type SkillActivityPageProps = {
  model: SkillObjectPageModel;
  navigate: (path: string) => void;
  search?: string;
};

export function SkillActivityPage({ model, navigate, search }: SkillActivityPageProps) {
  return (
    <SkillObjectPage activeTab="activity" model={model} navigate={navigate} search={search}>
      {(skill) => {
        const latestVersion = skill.versions?.[0];
        const latestScan = summarizeScanStatus(latestVersion?.scan_status);
        const events = [
          {
            id: 'updated',
            title: 'Catalog updated',
            body: '当前对象的元信息最近一次被刷新。',
            time: formatDateTime(skill.updated_at),
          },
          latestVersion
            ? {
                id: `version-${latestVersion.version}`,
                title: `Published ${latestVersion.version}`,
                body: `最新版本状态：${latestScan.label}`,
                time: formatDateTime(latestVersion.published_at || latestVersion.created_at),
              }
            : null,
          {
            id: 'visibility',
            title: skill.is_public === false ? 'Private distribution' : 'Public distribution',
            body: skill.is_public === false ? '当前仅限持证用户访问。' : '当前可在公开目录中发现。',
            time: formatDateTime(skill.updated_at),
          },
        ].filter(Boolean) as Array<{ id: string; title: string; body: string; time: string }>;

        return (
          <div className="gh-object-stack">
            <div className="gh-object-section-heading">
              <div>
                <h2>Recent activity</h2>
                <p>用最近的版本、目录更新时间和可见性变化，快速判断这个 skill 是否还在维护。</p>
              </div>
            </div>

            <div className="gh-object-activity-list">
              {events.map((event) => (
                <article className="gh-object-activity-item" key={event.id}>
                  <div className="gh-object-activity-item__dot" />
                  <div className="gh-object-activity-item__body">
                    <div className="gh-object-activity-item__header">
                      <strong>{event.title}</strong>
                      <span>{event.time}</span>
                    </div>
                    <p>{event.body}</p>
                  </div>
                </article>
              ))}
            </div>
          </div>
        );
      }}
    </SkillObjectPage>
  );
}
