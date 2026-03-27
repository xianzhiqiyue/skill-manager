import type { ReactNode } from 'react';

type PageHeaderProps = {
  eyebrow?: string;
  title: string;
  description?: string;
  actions?: ReactNode;
  meta?: ReactNode;
};

export function PageHeader({ eyebrow, title, description, actions, meta }: PageHeaderProps) {
  return (
    <div className="gh-page-header">
      <div className="gh-page-header__copy">
        {eyebrow ? <span className="gh-page-header__eyebrow">{eyebrow}</span> : null}
        <h1>{title}</h1>
        {description ? <p>{description}</p> : null}
        {meta ? <div className="gh-page-header__meta">{meta}</div> : null}
      </div>
      {actions ? <div className="gh-page-header__actions">{actions}</div> : null}
    </div>
  );
}
