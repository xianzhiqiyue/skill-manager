import { buildAuthPath } from '../../lib/routes';

type SettingsAuthCalloutProps = {
  description: string;
  navigate: (path: string) => void;
  redirectTo?: string;
  title: string;
};

export function SettingsAuthCallout({
  description,
  navigate,
  redirectTo,
  title,
}: SettingsAuthCalloutProps) {
  return (
    <div className="page-stack">
      <section className="surface-panel gh-settings-auth">
        <div className="gh-settings-auth__copy">
          <h1>{title}</h1>
          <p>{description}</p>
        </div>
        <div className="gh-settings-auth__actions">
          <button
            className="button button--primary"
            onClick={() => navigate(buildAuthPath('login', redirectTo))}
            type="button"
          >
            登录
          </button>
          <button
            className="button button--secondary"
            onClick={() => navigate(buildAuthPath('register', redirectTo))}
            type="button"
          >
            注册
          </button>
        </div>
      </section>
    </div>
  );
}
