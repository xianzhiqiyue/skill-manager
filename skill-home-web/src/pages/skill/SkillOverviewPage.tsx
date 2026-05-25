import { useEffect, useState } from 'react';

import { getDownloadUrl, getSkillDescription } from '../../api';
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
  skillRatingSaving?: boolean;
  skillRatingError?: string | null;
  skillRatingSuccess?: string | null;
  submitSkillRating?: (rating: number, comment?: string) => Promise<void>;
  skillLikeSaving?: boolean;
  skillLikeError?: string | null;
  skillShareStatus?: string | null;
  toggleSkillLike?: () => Promise<void>;
  shareDetailSkill?: () => Promise<void>;
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
  const [selectedRating, setSelectedRating] = useState(0);
  const [ratingComment, setRatingComment] = useState('');

  useEffect(() => {
    setSelectedRating(model.detailSkill?.user_rating?.rating || 0);
    setRatingComment(model.detailSkill?.user_rating?.comment || '');
  }, [
    model.detailSkill?.id,
    model.detailSkill?.user_rating?.comment,
    model.detailSkill?.user_rating?.rating,
  ]);

  const handleTagSelect = (tag: string) => {
    navigate(`/skills?tag=${encodeURIComponent(tag)}`);
  };

  return (
    <SkillObjectPage activeTab="overview" model={model} navigate={navigate} search={search}>
      {(skill) => {
        const latestVersion = skill.latest_version || skill.versions?.[0]?.version || 'draft';
        const latestVersionRecord = skill.versions?.[0];
        const latestScan = summarizeScanStatus(latestVersionRecord?.scan_status);
        const description = getSkillDescription(skill);
        const communityTags = skill.community_tags || [];
        const viewerTags = skill.viewer_tags || [];
        const officialTags = skill.tags || [];
        const primaryInstall = `skill-home install ${skillRef(skill)}${latestVersion === 'draft' ? '' : `@${latestVersion}`}`;
        // Resolve the contract URL once so this page explicitly follows download_url when present.
        const downloadUrl = getDownloadUrl(skill);
        const hasRatings = (skill.rating_count || 0) > 0;
        const ownerLabel =
          skill.owner?.display_name_zh ||
          skill.owner_display_name_zh ||
          skill.owner?.username ||
          skill.owner_username ||
          skill.namespace;

        return (
          <div className="gh-object-overview">
            <OverviewCard eyebrow="Social" title="Community signal">
              {model.skillLikeError ? (
                <div className="status-banner status-banner--danger">{model.skillLikeError}</div>
              ) : null}
              {model.skillShareStatus ? (
                <div className="status-banner status-banner--success">{model.skillShareStatus}</div>
              ) : null}
              <div className="detail-fact-list">
                <OverviewStat label="Author" value={ownerLabel} />
                <OverviewStat label="Likes" value={String(skill.like_count || 0)} />
                <OverviewStat label="Installs" value={String(skill.install_count || 0)} />
                <OverviewStat label="Downloads" value={String(skill.download_count)} />
              </div>
              <div className="gh-rating-actions">
                <button
                  className="button button--secondary"
                  disabled={model.skillLikeSaving}
                  onClick={() => {
                    void model.toggleSkillLike?.();
                  }}
                  type="button"
                >
                  {model.skillLikeSaving ? '处理中…' : skill.viewer_liked ? '取消点赞' : '点赞'}
                </button>
                <button
                  className="button button--quiet"
                  onClick={() => {
                    void model.shareDetailSkill?.();
                  }}
                  type="button"
                >
                  分享
                </button>
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
                    <p className="gh-community-tag-copy">登录后可补充你自己的社区 tags 和评分。</p>
                  )}
                </OverviewCard>
              ) : null}

              <OverviewCard eyebrow="Rating" title="Community rating">
                {model.skillRatingError ? (
                  <div className="status-banner status-banner--danger">{model.skillRatingError}</div>
                ) : null}
                {model.skillRatingSuccess ? (
                  <div className="status-banner status-banner--success">{model.skillRatingSuccess}</div>
                ) : null}

                <div className="gh-rating-summary">
                  <strong>{hasRatings && skill.rating != null ? `${skill.rating.toFixed(1)} 分` : '暂无评分'}</strong>
                  <span>{skill.rating_count || 0} 人评分</span>
                  {skill.user_rating ? <span>你的评分：{skill.user_rating.rating}/5</span> : null}
                </div>

                <div className="gh-rating-stars" role="group" aria-label="Choose rating">
                  {[1, 2, 3, 4, 5].map((value) => (
                    <button
                      aria-label={`${value} 星`}
                      aria-pressed={selectedRating === value}
                      className={`gh-rating-star ${selectedRating === value ? 'is-active' : ''}`.trim()}
                      key={value}
                      onClick={() => setSelectedRating(value)}
                      type="button"
                    >
                      {value}
                    </button>
                  ))}
                </div>

                <label className="field gh-rating-comment">
                  <span>Add comment</span>
                  <input
                    placeholder="Optional note about fit, trust, or setup"
                    value={ratingComment}
                    onChange={(event) => setRatingComment(event.target.value)}
                  />
                </label>

                <div className="gh-rating-actions">
                  <button
                    className="button button--secondary"
                    disabled={model.skillRatingSaving || selectedRating === 0}
                    onClick={() => {
                      void model.submitSkillRating?.(selectedRating, ratingComment);
                    }}
                    type="button"
                  >
                    {model.skillRatingSaving ? '保存中…' : model.currentUser ? '保存评分' : '登录后评分'}
                  </button>
                  <span className="gh-rating-copy">
                    {model.currentUser ? '看过详情后也可以先给出主观评价。' : '登录后即可提交评分。'}
                  </span>
                </div>
              </OverviewCard>

              <OverviewCard eyebrow="Overview" title="What this skill does">
                <p>{description || '暂无描述。'}</p>
                <div className="detail-fact-list">
                  <OverviewStat label="Latest version" value={latestVersion} />
                  <OverviewStat label="Scan status" value={latestScan.label} />
                  <OverviewStat label="Downloads" value={String(skill.download_count)} />
                  <OverviewStat label="Installs" value={String(skill.install_count || 0)} />
                  <OverviewStat label="License" value={skill.license || '未填写'} />
                </div>
                {officialTags.length ? (
                  <div className="gh-official-tag-block">
                    <span className="gh-community-tag-copy">Official tags</span>
                    <div className="skill-tag-row skill-tag-row--dense">
                      {officialTags.map((tag) => (
                        <SkillTagButton key={tag} onSelect={handleTagSelect} tag={tag} />
                      ))}
                    </div>
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
