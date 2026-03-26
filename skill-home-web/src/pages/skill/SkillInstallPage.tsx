import { useState } from 'react';

import { CopyActionButton } from '../../components/object/CopyActionButton';
import {
  SkillObjectPage,
  type SkillObjectPageModel,
} from '../../components/object/SkillObjectPage';
import { getInstallRecipes, type InstallIDE } from '../../lib/format';

type SkillInstallPageProps = {
  model: SkillObjectPageModel;
  navigate: (path: string) => void;
  search?: string;
};

export function SkillInstallPage({ model, navigate, search }: SkillInstallPageProps) {
  const [activeIDE, setActiveIDE] = useState<InstallIDE>('codex');

  return (
    <SkillObjectPage activeTab="install" model={model} navigate={navigate} search={search}>
      {(skill) => {
        const recipes = getInstallRecipes(skill);
        const recipe = recipes.find((item) => item.id === activeIDE) || recipes[0];

        return (
          <div className="gh-object-stack">
            <div className="gh-object-section-heading">
              <div>
                <h2>Install command set</h2>
                <p>按目标 AI 选择命令集，先复制主安装命令，再用辅助命令做验证或排查。</p>
              </div>
            </div>

            <div className="install-guide__tabs gh-object-install-tabs">
              {recipes.map((item) => (
                <button
                  className={`segmented-button ${item.id === recipe.id ? 'is-active' : ''}`.trim()}
                  key={item.id}
                  onClick={() => setActiveIDE(item.id)}
                  type="button"
                >
                  {item.label}
                </button>
              ))}
            </div>

            <div className="gh-object-command-card">
              <div className="gh-object-command-card__header">
                <div>
                  <strong>{recipe.label}</strong>
                  <p>{recipe.description}</p>
                </div>
                <div className="gh-object-command-card__actions">
                  <CopyActionButton className="button button--primary" label="复制安装命令" value={recipe.install} />
                  <CopyActionButton label="复制引用" value={recipe.reference} />
                </div>
              </div>
              <code>{recipe.install}</code>
            </div>

            <div className="gh-object-install-grid">
              <article className="gh-object-command-card">
                <div className="gh-object-command-card__header">
                  <div>
                    <strong>环境检查</strong>
                    <p>确认 CLI、registry 和认证状态正常。</p>
                  </div>
                  <CopyActionButton className="button button--ghost" value={recipe.verify} />
                </div>
                <code>{recipe.verify}</code>
              </article>

              <article className="gh-object-command-card">
                <div className="gh-object-command-card__header">
                  <div>
                    <strong>搜索目标</strong>
                    <p>先确认 skill 名称和版本，再决定安装目标。</p>
                  </div>
                  <CopyActionButton className="button button--ghost" value={recipe.search} />
                </div>
                <code>{recipe.search}</code>
              </article>

              <article className="gh-object-command-card">
                <div className="gh-object-command-card__header">
                  <div>
                    <strong>拉取包内容</strong>
                    <p>需要检查缓存或包内容时，再执行 pull。</p>
                  </div>
                  <CopyActionButton className="button button--ghost" value={recipe.pull} />
                </div>
                <code>{recipe.pull}</code>
              </article>

              <article className="gh-object-command-card">
                <div className="gh-object-command-card__header">
                  <div>
                    <strong>直接下载</strong>
                    <p>适合手动检查产物或离线分发。</p>
                  </div>
                  <CopyActionButton className="button button--ghost" label="复制链接" value={recipe.download} />
                </div>
                <code>{recipe.download}</code>
              </article>
            </div>
          </div>
        );
      }}
    </SkillObjectPage>
  );
}
