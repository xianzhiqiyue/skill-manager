import type { ReactNode } from 'react';

type DangerZoneProps = {
  actionLabel: string;
  description: string;
  disabled?: boolean;
  pendingLabel?: string;
  children?: ReactNode;
  onAction: () => void;
  title: string;
};

export function DangerZone({
  actionLabel,
  children,
  description,
  disabled = false,
  onAction,
  pendingLabel,
  title,
}: DangerZoneProps) {
  return (
    <section className="gh-danger-zone">
      <div className="gh-danger-zone__copy">
        <div>
          <h2>{title}</h2>
          <p>{description}</p>
        </div>
        {children ? <div className="gh-danger-zone__details">{children}</div> : null}
      </div>
      <button className="button button--danger" disabled={disabled} onClick={onAction} type="button">
        {disabled && pendingLabel ? pendingLabel : actionLabel}
      </button>
    </section>
  );
}
