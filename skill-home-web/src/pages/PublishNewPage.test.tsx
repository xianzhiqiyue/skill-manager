import { fireEvent, render, screen } from '@testing-library/react';
import { useState } from 'react';
import { describe, expect, it, vi } from 'vitest';

import { PublishNewPage } from './PublishNewPage';

function PublishHarness() {
  const [publishForm, setPublishForm] = useState({
    namespace: 'testuser',
    name: 'deploy-buddy',
    description: 'Ship releases safely.',
    category: '',
    version: '1.0.0',
    license: 'MIT',
    tags: [] as string[],
    isPublic: true,
  });

  return (
    <PublishNewPage
      model={{
        token: 'token-1',
        publishError: null,
        publishLoading: false,
        publishSuccess: null,
        publishForm,
        setPublishForm,
        setPublishFile: vi.fn(),
        submitPublish: vi.fn(),
      }}
      navigate={vi.fn()}
    />
  );
}

describe('PublishNewPage', () => {
  it('requires category and at least one official tag before enabling submit', () => {
    render(<PublishHarness />);

    const submit = screen.getByRole('button', { name: 'Upload release' });
    expect(screen.getByRole('combobox', { name: 'Category' })).toBeInTheDocument();
    expect(screen.getAllByText('Official tags')).toHaveLength(2);
    expect(submit).toBeDisabled();

    fireEvent.change(screen.getByRole('combobox', { name: 'Category' }), {
      target: { value: 'ops' },
    });
    fireEvent.click(screen.getByRole('button', { name: 'deployment' }));

    expect(submit).not.toBeDisabled();
  });
});
