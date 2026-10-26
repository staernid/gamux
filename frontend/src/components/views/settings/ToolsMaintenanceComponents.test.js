import { render, screen, fireEvent } from '@testing-library/svelte';
import { describe, it, expect, vi } from 'vitest';
import ToolCard from './ToolCard.svelte';
import BatchOperationCard from './BatchOperationCard.svelte';
import { normalizeTool } from '../../../utils/tools.js';

describe('ToolsMaintenance child components', () => {
  it('ToolCard renders uninstalled tool and handles update action', async () => {
    const onupdate = vi.fn();
    const tool = normalizeTool({
      name: 'Goldberg Steam Emulator',
      key: 'gbe',
      is_installed: false,
      latest_tag: 'v0.2.7',
    });

    render(ToolCard, { props: { tool, onupdate } });

    expect(screen.getByText('Goldberg Steam Emulator')).toBeInTheDocument();
    expect(screen.getByText('Missing')).toBeInTheDocument();
    expect(screen.getByText('Not Installed')).toBeInTheDocument();

    const installBtn = screen.getByRole('button', { name: /^Install$/i });
    await fireEvent.click(installBtn);
    expect(onupdate).toHaveBeenCalledWith({ key: 'gbe', name: 'Goldberg Steam Emulator' });
  });

  it('BatchOperationCard renders idle and running states with callbacks', async () => {
    const onrun = vi.fn();
    const onviewTasks = vi.fn();

    const { rerender } = render(BatchOperationCard, {
      props: {
        title: 'Achievements',
        description: 'Download missing schemas',
        actionLabel: 'Download Missing',
        onrun,
        onviewTasks,
      },
    });

    expect(screen.getByText('Achievements')).toBeInTheDocument();
    const runBtn = screen.getByText('Download Missing');
    await fireEvent.click(runBtn);
    expect(onrun).toHaveBeenCalled();

    rerender({
      title: 'Achievements',
      description: 'Download missing schemas',
      actionLabel: 'Download Missing',
      task: { stage: 'Fetching...', percent: 45 },
      onrun,
      onviewTasks,
    });

    expect(screen.getByText(/Running \(45%\)\.\.\./i)).toBeInTheDocument();
    const viewBtn = screen.getByText('View in Tasks');
    await fireEvent.click(viewBtn);
    expect(onviewTasks).toHaveBeenCalled();
  });
});
