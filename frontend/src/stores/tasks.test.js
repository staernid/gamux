import { describe, it, expect, beforeEach } from 'vitest';
import { get } from 'svelte/store';
import { tasksStore } from './tasks.js';

describe('tasksStore', () => {
  beforeEach(() => {
    tasksStore.reset();
  });

  it('tracks both task and transfer lifecycle events seamlessly', () => {
    // 1. Task started
    tasksStore.handleTaskEvent({
      type: 'task:started',
      task_id: 'task_download_1',
      operation: 'download',
      game_name: 'Cyberpunk 2077',
      stage: 'Connecting',
      details: 'Connecting to CDN...',
      timestamp: Date.now()
    });

    let active = get(tasksStore.active);
    expect(active.length).toBe(1);
    expect(active[0].id).toBe('task_download_1');
    expect(active[0].status).toBe('running');
    expect(active[0].game_name).toBe('Cyberpunk 2077');

    // 2. Progress update with speed and eta
    tasksStore.handleTaskEvent({
      type: 'task:progress',
      task_id: 'task_download_1',
      stage: 'Downloading Chunks',
      details: 'chunk_123.bin',
      progress: {
        percent: 50.0,
        current_bytes: 50000000,
        total_bytes: 100000000,
        bytes_per_sec: 25000000,
        eta_seconds: 2
      }
    });

    active = get(tasksStore.active);
    expect(active[0].percent).toBe(50.0);
    expect(active[0].bytes_downloaded).toBe(50000000);
    expect(active[0].speed_bps).toBe(25000000);
    expect(get(tasksStore.totalSpeedBps)).toBe(25000000);

    // 3. Pause task
    tasksStore.handleTaskEvent({
      type: 'task:paused',
      task_id: 'task_download_1',
      stage: 'Paused',
      details: 'Paused by user'
    });

    const paused = get(tasksStore.paused);
    expect(paused.length).toBe(1);
    expect(paused[0].status).toBe('paused');
    expect(paused[0].speed_bps).toBe(0);

    // 4. Resume task
    tasksStore.handleTaskEvent({
      type: 'task:resumed',
      task_id: 'task_download_1',
      stage: 'Running'
    });
    expect(get(tasksStore.active).length).toBe(1);

    // 5. Complete task
    tasksStore.handleTaskEvent({
      type: 'task:completed',
      task_id: 'task_download_1',
      details: 'Download complete'
    });

    const completed = get(tasksStore.completed);
    expect(completed.length).toBe(1);
    expect(completed[0].status).toBe('completed');
    expect(completed[0].percent).toBe(100);
    expect(get(tasksStore.active).length).toBe(0);
  });

  it('tracks transfer:started, transfer:progress, and transfer:completed events', () => {
    tasksStore.handleTransferEvent({
      type: 'transfer:started',
      task_id: 'dl_hades',
      game_name: 'Hades II',
      app_id: '1145350',
      stage: 'Connecting',
      details: 'Connecting to Steam CDN...',
    });

    let active = get(tasksStore.active);
    expect(active.length).toBe(1);
    expect(active[0].id).toBe('dl_hades');
    expect(active[0].status).toBe('downloading');

    tasksStore.handleTransferEvent({
      type: 'transfer:progress',
      task_id: 'dl_hades',
      stage: 'Downloading Depots',
      progress: {
        percent: 45.5,
        current_bytes: 45500000,
        total_bytes: 100000000,
        bytes_per_sec: 25000000,
        eta_seconds: 4,
      },
    });

    active = get(tasksStore.active);
    expect(active[0].percent).toBe(45.5);
    expect(active[0].bytes_downloaded).toBe(45500000);
    expect(get(tasksStore.totalSpeedBps)).toBe(25000000);

    tasksStore.handleTransferEvent({
      type: 'transfer:completed',
      task_id: 'dl_hades',
      stage: 'Completed',
    });

    const completed = get(tasksStore.completed);
    expect(completed.length).toBe(1);
    expect(completed[0].status).toBe('completed');
    expect(completed[0].percent).toBe(100);
  });

  it('handles task:failed event correctly', () => {
    tasksStore.handleTaskEvent({
      type: 'task:started',
      task_id: 'task_fail',
      operation: 'apply',
      game_name: 'Test Game',
    });

    tasksStore.handleTaskEvent({
      type: 'task:failed',
      task_id: 'task_fail',
      error: 'Disk full error',
    });

    const items = get(tasksStore.items);
    const failedTask = items.find((j) => j.id === 'task_fail');
    expect(failedTask).toBeDefined();
    expect(failedTask.status).toBe('failed');
    expect(failedTask.error).toBe('Disk full error');
  });

  it('executes client-enqueued asynchronous tasks with callbacks', async () => {
    let successResult = null;
    tasksStore.enqueue({
      type: 'apply',
      title: 'Apply GBE',
      gameName: 'Portal 2',
      gameDir: '/games/portal2',
      run: async () => {
        return { applied: true };
      },
      onSuccess: (res) => {
        successResult = res;
      },
    });

    await new Promise((r) => setTimeout(r, 20));

    expect(successResult).toEqual({ applied: true });
    const items = get(tasksStore.items);
    const match = items.find((t) => t.game_name === 'Portal 2');
    expect(match).toBeDefined();
    expect(match.status).toBe('completed');
  });

  it('updates task progress on running operation without creating phantom tasks', () => {
    // 1. Enqueue operation
    tasksStore.handleTaskEvent({
      type: 'task:started',
      task_id: 'op_setup_1',
      operation: 'apply',
      game_name: 'Elden Ring',
      stage: 'Initializing'
    });

    // 2. Step progress update
    tasksStore.updateStepProgress({
      step: 3,
      total: 6,
      title: 'Applying Steamless DRM Patch',
      details: 'Unpacking executable',
      percent: 50
    });

    let items = get(tasksStore.items);
    expect(items.length).toBe(1);
    expect(items[0].id).toBe('op_setup_1');
    expect(items[0].stage).toBe('Applying Steamless DRM Patch');
    expect(items[0].percent).toBe(50);
  });

  it('correctly partitions multiple concurrent downloads and operations', () => {
    // Enqueue 2 downloads
    tasksStore.handleTransferEvent({
      type: 'transfer:started',
      task_id: 'dl_garbanzo',
      game_name: 'Garbanzo Quest',
      app_id: '1001',
      stage: 'Downloading Depots'
    });
    tasksStore.handleTransferEvent({
      type: 'transfer:started',
      task_id: 'dl_escape',
      game_name: 'Escape Academy',
      app_id: '1002',
      stage: 'Downloading Depots'
    });

    // Enqueue 1 operation
    tasksStore.handleTaskEvent({
      type: 'task:started',
      task_id: 'op_patch',
      operation: 'apply',
      game_name: 'Cyberpunk 2077',
      stage: 'Patching GBE'
    });

    // Global active should contain all 3
    const allActive = get(tasksStore.active);
    expect(allActive.length).toBe(3);

    // Downloads partition should contain only the 2 downloads
    const activeDownloads = get(tasksStore.activeDownloads);
    expect(activeDownloads.length).toBe(2);
    expect(activeDownloads.map(d => d.id)).toEqual(['dl_escape', 'dl_garbanzo']);

    // Operations partition should contain only the 1 operation
    const activeOps = get(tasksStore.activeOperations);
    expect(activeOps.length).toBe(1);
    expect(activeOps[0].id).toBe('op_patch');
  });

  it('executes pause, resume, cancel, and remove actions correctly', async () => {
    tasksStore.handleTaskEvent({
      type: 'task:started',
      task_id: 'action_test_1',
      operation: 'download',
      game_name: 'Test Title'
    });

    await tasksStore.pause('action_test_1');
    let item = get(tasksStore.items).find(t => t.id === 'action_test_1');
    expect(item.status).toBe('paused');

    await tasksStore.resume('action_test_1');
    item = get(tasksStore.items).find(t => t.id === 'action_test_1');
    expect(item.status).toBe('downloading');

    await tasksStore.cancel('action_test_1');
    item = get(tasksStore.items).find(t => t.id === 'action_test_1');
    expect(item.status).toBe('cancelled');

    await tasksStore.remove('action_test_1');
    item = get(tasksStore.items).find(t => t.id === 'action_test_1');
    expect(item).toBeUndefined();
  });
});
