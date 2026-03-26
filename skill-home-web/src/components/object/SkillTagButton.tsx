type SkillTagButtonProps = {
  tag: string;
  count?: number;
  onSelect?: (tag: string) => void;
  selected?: boolean;
};

export function SkillTagButton({
  tag,
  count,
  onSelect,
  selected = false,
}: SkillTagButtonProps) {
  const content = (
    <>
      <span>{tag}</span>
      {typeof count === 'number' ? <span className="skill-badge__count">{count}</span> : null}
    </>
  );

  if (!onSelect) {
    return <span className={`skill-badge ${selected ? 'is-selected' : ''}`.trim()}>{content}</span>;
  }

  return (
    <button
      className={`skill-badge skill-badge--interactive ${selected ? 'is-selected' : ''}`.trim()}
      onClick={() => onSelect(tag)}
      type="button"
    >
      {content}
    </button>
  );
}
