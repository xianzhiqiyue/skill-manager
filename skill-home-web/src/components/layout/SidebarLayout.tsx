import type { ReactNode } from 'react';

type SidebarLayoutProps = {
  aside?: ReactNode;
  className?: string;
  content: ReactNode;
  sidebar?: ReactNode;
};

export function SidebarLayout({ aside, className, content, sidebar }: SidebarLayoutProps) {
  const classes = ['gh-sidebar-layout', className].filter(Boolean).join(' ');

  return (
    <div className={classes}>
      {sidebar ? <aside className="gh-sidebar-layout__sidebar">{sidebar}</aside> : null}
      <div className="gh-sidebar-layout__content">{content}</div>
      {aside ? <aside className="gh-sidebar-layout__aside">{aside}</aside> : null}
    </div>
  );
}
