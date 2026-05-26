import { useEffect } from 'react';

type SoulStoreSSOPageProps = {
  locationSearch: string;
  model: {
    authLoading: boolean;
    authError: string | null;
    submitSoulStoreSSO: (ticket: string, nextPath?: string | null) => Promise<boolean>;
  };
};

export function SoulStoreSSOPage({ locationSearch, model }: SoulStoreSSOPageProps) {
  useEffect(() => {
    const params = new URLSearchParams(locationSearch);
    void model.submitSoulStoreSSO(params.get('ticket') || '', params.get('next'));
  }, [locationSearch]);

  return (
    <section className="page-stack">
      <div className="surface-panel">
        <p className="eyebrow">SoulStore SSO</p>
        <h1>正在进入 Skill Home</h1>
        {model.authError ? (
          <p className="empty-panel empty-panel--danger">{model.authError}</p>
        ) : (
          <p>{model.authLoading ? '正在验证登录票据。' : '正在跳转。'}</p>
        )}
      </div>
    </section>
  );
}
