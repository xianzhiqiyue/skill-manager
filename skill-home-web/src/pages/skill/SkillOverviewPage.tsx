import { useState } from 'react';

import { getDownloadUrl } from '../../api';
import {
  SkillObjectPage,
  type SkillObjectPageModel,
} from '../../components/object/SkillObjectPage';
import { SkillTagButton } from '../../components/object/SkillTagButton';
import { formatBytes, formatDateTime, skillRef, summarizeScanStatus } from '../../lib/format';

export type SkillOverviewPageModel = SkillObjectPageModel & {
  currentUser?: {
    id: string;
    username: string;
    email: string;
  } | null;
  communityTagSaving?: boolean;
  communityTagRemoving?: string | null;
  communityTagError?: string | null;
  communityTagSuccess?: string | null;
  submitCommunityTag?: (tag: string) => Promise<void>;
  removeCommunityTag?: (tag: string) => Promise<void>;
};

type SkillOverviewPageProps = {
  model: SkillOverviewPageModel;
  navigate: (path: string) => void;
  search?: string;
};

function OverviewStat({ label, value }: { label: string; value: string }) {
  return (
    <div className="detail-fact">
      <span>{label}</span>
      <strong>{value}</strong>
    </div>
  );
}

function OverviewCard({
  eyebrow,
  title,
  children,
}: {
  eyebrow: string;
  title: string;
  children: React.ReactNode;
}) {
  return (
    <article className="gh-object-overview__card">
      <span className="gh-object-overview__eyebrow">{eyebrow}</span>
      <h2>{title}</h2>
      {children}
    </article>
  );
}

export function SkillOverviewPage({ model, navigate, search }: SkillOverviewPageProps) {
  const [communityTagInput, setCommunityTagInput] = useState('');
  const handleTagSelect = (tag: string) => {
    navigate(`/skills?tag=${encodeURIComponent(tag)}`);
  };

  return (
    <SkillObjectPage activeTab="overview" model={model} navigate={navigate} search={search}>
      {(skill) => {
        const latestVersion = skill.latest_version || skill.versions?.[0]?.version || 'draft';
        const latestVersionRecord = skill.versions?.[0];
        const latestScan = summarizeScanStatus(latestVersionRecord?.scan_status);
        const tags = skill.tags || [];
        const communityTags = skill.community_tags || [];
        const viewerTags = skill.viewer_tags || [];
        const primaryInstall = `skill-home install ${skillRef(skill)}${latestVersion === 'draft' ? '' : `@${latestVersion}`}`;
        const downloadUrl = getDownloadUrl(skill);

        return (
            <div className="gh-object-overview">
              <OverviewCard eyebrow="Overview" title="What this skill does">
                <p>{skill.description || '暂无描述。'}</p>
                <div className="detail-fact-list">
                  <OverviewStat label="Latest version" value={latestVersion} />
                  <OverviewStat label="Scan status" value={latestScan.label} />
                  <OverviewStat label="Downloads" value={String(skill.download_count)} />
                  <OverviewStat label="License" value={skill.license || '未填写'} />
                </div>
                {tags.length ? (
                  <div className="skill-tag-row skill-tag-row--dense">
                    {tags.map((tag) => (
                      <SkillTagButton key={tag} onSelect={handleTagSelect} tag={tag} />
                    ))}
                  </div>
                ) : null}
              </OverviewCard>

              <OverviewCard eyebrow="Install" title="Primary install target">
                <p>Use the current release and verify the package before broad rollout.</p>
                <code>{primaryInstall}</code>
                <div className="gh-object-overview__meta">
                  <span>{formatDateTime(skill.updated_at)} 更新</span>
                  <span>{formatBytes(latestVersionRecord?.size_bytes)} 包大小</span>
                  <span>{skillRef(skill)}</span>
                </div>
              </OverviewCard>

              {(communityTags.length || model.currentUser) ? (
                <OverviewCard eyebrow="Community" title="Community tags">
                  {model.communityTagError ? (
                    <div className="status-banner status-banner--danger">{model.communityTagError}</div>
                  ) : null}
                  {model.communityTagSuccess ? (
                    <div className="status-banner status-banner--success">{model.communityTagSuccess}</div>
                  ) : null}

                  {communityTags.length ? (
                    <div className="skill-tag-row skill-tag-row--dense">
                      {communityTags.map((item) => (
                        <SkillTagButton
                          count={item.count}
                          key={item.tag}
                          selected={viewerTags.includes(item.tag)}
                          tag={item.tag}
                        />
                      ))}
                    </div>
                  ) : (
                    <p className="gh-community-tag-copy">还没有社区 tags。</p>
                  )}

                  {model.currentUser ? (
                    <>
                      <form
                        className="gh-community-tag-form"
                        onSubmit={(event) => {
                          event.preventDefault();
                          const nextTag = communityTagInput.trim();
                          if (!nextTag) {
                            return;
                          }
                          void model.submitCommunityTag?.(nextTag);
                          setCommunityTagInput('');
                        }}
                      >
                        <label className="field gh-community-tag-form__field">
                          <span>Add tag</span>
                          <input
                            placeholder="deployment, ci, docs"
                            value={communityTagInput}
                            onChange={(event) => setCommunityTagInput(event.target.value)}
                          />
                        </label>
                        <button
                          className="button button--secondary"
                          disabled={model.communityTagSaving}
                          type="submit"
                        >
                          {model.communityTagSaving ? 'Adding…' : 'Add tag'}
                        </button>
                      </form>

                      {viewerTags.length ? (
                        <div className="gh-community-tag-user-row">
                          <span>Your tags</span>
                          <div className="skill-tag-row skill-tag-row--dense">
                            {viewerTags.map((tag) => (
                              <SkillTagButton
                                key={tag}
                                onSelect={() => {
                                  void model.removeCommunityTag?.(tag);
                                }}
                                selected={model.communityTagRemoving === tag}
                                tag={tag}
                              />
                            ))}
                          </div>
                        </div>
                      ) : null}
                    </>
                  ) : (
                    <p className="gh-community-tag-copy">登录后可补充你自己的社区 tags。</p>
                  )}
                </OverviewCard>
              ) : null}

              <OverviewCard eyebrow="Activity" title="Recent signal">
                <p>
                  {skill.is_public === false ? 'Private skill.' : 'Public skill.'}
                  {skill.is_deprecated ? ' Marked as deprecated.' : ' Active in the catalog.'}
                </p>
                <a className="button button--secondary" href={downloadUrl} rel="noreferrer" target="_blank">
                  Open download
                </a>
              </OverviewCard>
            </div>
        );
      }}
    </SkillObjectPage>
  );
}
