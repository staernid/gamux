import { render, screen, fireEvent } from '@testing-library/svelte';
import { describe, it, expect, vi } from 'vitest';
import TaskCard from './TaskCard.svelte';

describe('TaskCard Component', () => {
  const downloadTask = {
    id: 'task-dl-1',
    appId: 1086940,
    gameName: "Baldur's Gate 3",
    type: 'download',
    store: 'steam',
    stage: 'Downloading depots',
    details: 'depot 1086941 (chunks 150/500)',
    percent: 45.5,
    currentBytes: 45500000000,
    totalBytes: 100000000000,
    speedBps: 15728640,
    etaSeconds: 3460,
    depots: [
      { id: 1086941, name: 'BG3 Content', downloadedBytes: 45500000000, totalBytes: 100000000000, percent: 45.5 }
    ]
  };

  const operationTask = {
    id: 'task-op-1',
    appId: 1086940,
    gameName: "Baldur's Gate 3",
    type: 'apply',
    store: 'steam',
    stage: 'Applying Steamless & DRM fixes',
    details: 'Patching executable',
    percent: 0
  };

  it('renders download task with progress bar, telemetry and action buttons', () => {
    render(TaskCard, {
      props: {
        task: downloadTask,
        expandedDepots: {}
      }
    });

    expect(screen.getByText("Baldur's Gate 3")).toBeInTheDocument();
    expect(screen.getByText('DOWNLOAD')).toBeInTheDocument();
    expect(screen.getByText(/steam/i)).toBeInTheDocument();
    expect(screen.getByRole('button', { name: /Pause Baldur's Gate 3/i })).toBeInTheDocument();
    expect(screen.getByRole('button', { name: /Cancel Baldur's Gate 3/i })).toBeInTheDocument();

    const progressBar = screen.getByRole('progressbar');
    expect(progressBar).toBeInTheDocument();
    expect(progressBar).toHaveAttribute('aria-valuenow', '45.5');
  });

  it('renders indeterminate state for active non-download tasks at 0%', () => {
    render(TaskCard, {
      props: {
        task: operationTask,
        expandedDepots: {}
      }
    });

    expect(screen.getByText('SETUP & DRM')).toBeInTheDocument();
    expect(screen.getByText('Processing...')).toBeInTheDocument();
    expect(screen.queryByRole('button', { name: /Pause/i })).not.toBeInTheDocument();

    const progressBar = screen.getByRole('progressbar');
    expect(progressBar).toBeInTheDocument();
    expect(progressBar).not.toHaveAttribute('aria-valuenow');
  });

  it('safely handles cancellation with two-step inline confirmation', async () => {
    const oncancel = vi.fn();
    render(TaskCard, {
      props: {
        task: downloadTask,
        expandedDepots: {},
        oncancel
      }
    });

    const cancelBtn = screen.getByRole('button', { name: /Cancel Baldur's Gate 3/i });
    expect(cancelBtn.textContent.trim()).toBe('Cancel');

    // First click enters confirmation state, does NOT trigger oncancel immediately
    await fireEvent.click(cancelBtn);
    expect(cancelBtn.textContent.trim()).toBe('Confirm Cancel?');
    expect(oncancel).not.toHaveBeenCalled();

    // Second click confirms and calls oncancel
    await fireEvent.click(cancelBtn);
    expect(oncancel).toHaveBeenCalledTimes(1);
    expect(oncancel).toHaveBeenCalledWith('task-dl-1');
  });

  it('dispatches pause event on pause button click', async () => {
    const onpause = vi.fn();
    render(TaskCard, {
      props: {
        task: downloadTask,
        expandedDepots: {},
        onpause
      }
    });

    const pauseBtn = screen.getByRole('button', { name: /Pause Baldur's Gate 3/i });
    await fireEvent.click(pauseBtn);

    expect(onpause).toHaveBeenCalledTimes(1);
    expect(onpause).toHaveBeenCalledWith('task-dl-1');
  });

  it('dispatches toggleDepots when expanding depot breakdown', async () => {
    const ontoggleDepots = vi.fn();
    render(TaskCard, {
      props: {
        task: downloadTask,
        expandedDepots: {},
        ontoggleDepots
      }
    });

    const toggleBtn = screen.getByRole('button', { name: /Depots & Chunk Breakdown/i });
    expect(toggleBtn).toHaveAttribute('aria-expanded', 'false');

    await fireEvent.click(toggleBtn);
    expect(ontoggleDepots).toHaveBeenCalledTimes(1);
    expect(ontoggleDepots).toHaveBeenCalledWith('task-dl-1');
  });
});
