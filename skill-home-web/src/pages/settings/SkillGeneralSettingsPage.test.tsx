import { fireEvent, render, screen } from '@testing-library/react';
import { useState } from 'react';
import { describe, expect, it, vi } from 'vitest';

import { SkillGeneralSettingsPage } from './SkillGeneralSettingsPage';

function SettingsHarness() {
  const [manageForm, setManageForm] = useState({
    description: 'Keep release workflows tidy.',
    category: '',
    license: 'MIT',
    tags: [] as string[],
    isPublic: true,
    isDeprecated: false,
  });

  return (
    <SkillGeneralSettingsPage
      model={{
        token: 'token-1',
        managedSkillKey: 'testuser/deploy-buddy',
        managedSkill: {
          id: 'skill-1',
          namespace: 'testuser',
          name: 'deploy-buddy',
          description: 'Keep release workflows tidy.',
          category: 'ops',
          tags: ['deployment'],
          download_count: 12,
          rating_count: 0,
          latest_version: '1.0.0',
          versions: [
            {
              id: 'v1',
              version: '1.0.0',
              size_bytes: 1024,
              scan_status: 'pass',
            },
          ],
        },
        manageLoading: false,
        manageSaving: false,
        manageError: null,
        manageSuccess: null,
        manageForm,
        setManageForm,
        submitManage: vi.fn(),
      } as never}
      namespace="testuser"
      navigate={vi.fn()}
      skillName="deploy-buddy"
    />
  );
}

describe('SkillGeneralSettingsPage', () => {
  it('requires category and official tags before enabling save', () => {
    render(<SettingsHarness />);

    const submit = screen.getByRole('button', { name: 'Save changes' });
    expect(screen.getByRole('combobox', { name: 'Category' })).toBeInTheDocument();
    expect(screen.getByText('Official tags')).toBeInTheDocument();
    expect(submit).toBeDisabled();

    fireEvent.change(screen.getByRole('combobox', { name: 'Category' }), {
      target: { value: 'ops' },
    });
    fireEvent.click(screen.getByRole('button', { name: 'deployment' }));

    expect(submit).not.toBeDisabled();
  });
});
