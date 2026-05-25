import type { SkillDetail } from '../../api';
import { formatBytes, formatDateTime, skillRef, summarizeScanStatus } from '../../lib/format';

type SkillObjectMetadataProps = {
  skill: SkillDetail;
};

function MetadataRow({ label, value }: { label: string; value: string }) {
  return (
    <div className="gh-object-metadata__row">
      <dt>{label}</dt>
      <dd>{value}</dd>
    </div>
  );
}

export function SkillObjectMetadata({ skill }: SkillObjectMetadataProps) {
  const latestVersion = skill.latest_version || skill.versions?.[0]?.version || 'draft';
  const latestScan = summarizeScanStatus(skill.versions?.[0]?.scan_status);

  return (
    <section className="gh-object-metadata" aria-label="Metadata">
      <div className="gh-object-metadata__card">
        <span className="gh-object-metadata__eyebrow">Metadata</span>
        <h2>Object details</h2>
        <dl className="gh-object-metadata__list">
          <MetadataRow label="Namespace" value={skill.namespace} />
          <MetadataRow label="Skill" value={skill.name} />
          <MetadataRow label="Reference" value={skillRef(skill)} />
          <MetadataRow label="Latest version" value={latestVersion} />
          <MetadataRow label="Versions" value={String(skill.versions?.length || 0)} />
          <MetadataRow label="License" value={skill.license || '未填写'} />
          <MetadataRow label="Likes" value={String(skill.like_count || 0)} />
          <MetadataRow label="Installs" value={String(skill.install_count || 0)} />
          <MetadataRow label="Downloads" value={String(skill.download_count)} />
          <MetadataRow label="Updated" value={formatDateTime(skill.updated_at)} />
          <MetadataRow
            label="Visibility"
            value={skill.is_public === false ? 'Private' : 'Public'}
          />
          <MetadataRow
            label="State"
            value={latestScan.label}
          />
          <MetadataRow label="Latest size" value={formatBytes(skill.versions?.[0]?.size_bytes)} />
          {skill.owner?.username || skill.owner_username ? (
            <MetadataRow
              label="Owner"
              value={
                skill.owner?.display_name_zh ||
                skill.owner_display_name_zh ||
                skill.owner?.username ||
                skill.owner_username ||
                ''
              }
            />
          ) : null}
        </dl>
      </div>
    </section>
  );
}
