import { describe, it, expect, vi } from 'vitest';
import { render, fireEvent } from '@testing-library/svelte';
import HubActionToolbar from './HubActionToolbar.svelte';

describe('HubActionToolbar Component', () => {
  it('renders title and default inheriting badge when overrideCount is 0', () => {
    const { getByText, queryByText } = render(HubActionToolbar, {
      props: {
        title: 'Runtime & Environment',
        overrideCount: 0,
        isSaving: false,
        saveSuccess: false
      }
    });

    expect(getByText('Runtime & Environment')).toBeInTheDocument();
    expect(getByText('Inheriting Defaults')).toBeInTheDocument();
    expect(queryByText(/Override/)).toBeNull();
  });

  it('renders override count badge when overrideCount > 0', () => {
    const { getByText, queryByText } = render(HubActionToolbar, {
      props: {
        title: 'General Configuration',
        overrideCount: 3,
        isSaving: false,
        saveSuccess: false
      }
    });

    expect(getByText('3 Overrides')).toBeInTheDocument();
    expect(queryByText('Inheriting Defaults')).toBeNull();
  });

  it('shows saved indicator when saveSuccess is true', () => {
    const { getByText } = render(HubActionToolbar, {
      props: {
        title: 'General Configuration',
        overrideCount: 1,
        isSaving: false,
        saveSuccess: true
      }
    });

    expect(getByText('1 Override')).toBeInTheDocument();
    expect(getByText('Saved')).toBeInTheDocument();
  });

  it('dispatches save and reset events on button click', async () => {
    const onSave = vi.fn();
    const onReset = vi.fn();

    const { getByText, getByTitle } = render(HubActionToolbar, {
      props: {
        title: 'General Configuration',
        overrideCount: 1,
        isSaving: false,
        saveSuccess: false,
        onsave: onSave,
        onreset: onReset
      }
    });

    const saveBtn = getByText('Save');
    await fireEvent.click(saveBtn);
    expect(onSave).toHaveBeenCalledTimes(1);

    const resetBtn = getByTitle('Reset all settings to system defaults');
    await fireEvent.click(resetBtn);
    expect(onReset).toHaveBeenCalledTimes(1);
  });
});
