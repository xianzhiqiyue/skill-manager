import { CopyActionButton } from '../../components/object/CopyActionButton';
import {
  SkillObjectPage,
  type SkillObjectPageModel,
} from '../../components/object/SkillObjectPage';
import { formatBytes, formatDateTime, skillRef, summarizeScanStatus } from '../../lib/format';

type SkillVersionsPageProps = {
  model: SkillObjectPageModel;
  navigate: (path: string) => void;
  search?: string;
};

export function SkillVersionsPage({ model, navigate, search }: SkillVersionsPageProps) {
  return (
    <SkillObjectPage activeTab="versions" model={model} navigate={navigate} search={search}>
      {(skill) => (
        <div className="gh-object-stack">
          <div className="gh-object-section-heading">
            <div>
              <h2>Published versions</h2>
              <p>确认可安装版本、包大小和扫描状态，再决定 pull 或 install 的目标版本。</p>
            </div>
          </div>

          {skill.versions?.length ? (
            <div className="version-table">
              <div className="version-table__header">
                <span>版本</span>
                <span>时间</span>
                <span>大小</span>
                <span>状态</span>
                <span>操作</span>
              </div>
              {skill.versions.map((version) => {
                const state = summarizeScanStatus(version.scan_status);

                return (
                  <div className="version-table__row" key={version.id}>
                    <strong>{version.version}</strong>
                    <span>{formatDateTime(version.published_at || version.created_at)}</span>
                    <span>{formatBytes(version.size_bytes)}</span>
                    <span className={`status-pill status-pill--${state.tone}`}>{state.label}</span>
                    <CopyActionButton
                      className="button button--ghost"
                      label="复制 pull"
                      value={`skill-home pull ${skillRef(skill)}@${version.version}`}
                    />
                  </div>
                );
              })}
            </div>
          ) : (
            <div className="empty-panel">这个 skill 还没有公开的版本记录。</div>
          )}
        </div>
      )}
    </SkillObjectPage>
  );
}
