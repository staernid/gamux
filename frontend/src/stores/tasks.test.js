import { describe, it, expect, beforeEach } from 'vitest';
import {
  tasksStore,
  extractTaskProgress,
  normalizeTask,
  isLiveStatus,
  taskMatchesGame
} from './tasks.js';

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

    let active = tasksStore.active;
    expect(active.length).toBe(1);
    expect(active[0].id).toBe('task_download_1');
    expect(active[0].status).toBe('running');
    expect(active[0].gameName).toBe('Cyberpunk 2077');

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

    active = tasksStore.active;
    expect(active[0].percent).toBe(50.0);
    expect(active[0].currentBytes).toBe(50000000);
    expect(active[0].speedBps).toBe(25000000);
    expect(tasksStore.totalSpeedBps).toBe(25000000);

    // 3. Pause task
    tasksStore.handleTaskEvent({
      type: 'task:paused',
      task_id: 'task_download_1',
      stage: 'Paused',
      details: 'Paused by user'
    });

    const paused = tasksStore.paused;
    expect(paused.length).toBe(1);
    expect(paused[0].status).toBe('paused');
    expect(paused[0].speedBps).toBe(0);

    // 4. Resume task
    tasksStore.handleTaskEvent({
      type: 'task:resumed',
      task_id: 'task_download_1',
      stage: 'Running'
    });
    expect(tasksStore.active.length).toBe(1);

    // 5. Complete task
    tasksStore.handleTaskEvent({
      type: 'task:completed',
      task_id: 'task_download_1',
      details: 'Download complete'
    });

    const completed = tasksStore.completed;
    expect(completed.length).toBe(1);
    expect(completed[0].status).toBe('completed');
    expect(completed[0].percent).toBe(100);
    expect(tasksStore.active.length).toBe(0);
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

    let active = tasksStore.active;
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

    active = tasksStore.active;
    expect(active[0].percent).toBe(45.5);
    expect(active[0].currentBytes).toBe(45500000);
    expect(tasksStore.totalSpeedBps).toBe(25000000);

    tasksStore.handleTransferEvent({
      type: 'transfer:completed',
      task_id: 'dl_hades',
      stage: 'Completed',
    });

    const completed = tasksStore.completed;
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

    const items = tasksStore.items;
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
    const items = tasksStore.items;
    const match = items.find((t) => t.gameName === 'Portal 2');
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

    let items = tasksStore.items;
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
    const allActive = tasksStore.active;
    expect(allActive.length).toBe(3);

    // Downloads partition should contain only the 2 downloads
    const activeDownloads = tasksStore.activeDownloads;
    expect(activeDownloads.length).toBe(2);
    expect(activeDownloads.map(d => d.id)).toEqual(['dl_escape', 'dl_garbanzo']);

    // Operations partition should contain only the 1 operation
    const activeOps = tasksStore.activeOperations;
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
    let item = tasksStore.items.find(t => t.id === 'action_test_1');
    expect(item.status).toBe('paused');

    await tasksStore.resume('action_test_1');
    item = tasksStore.items.find(t => t.id === 'action_test_1');
    expect(item.status).toBe('downloading');

    await tasksStore.cancel('action_test_1');
    item = tasksStore.items.find(t => t.id === 'action_test_1');
    expect(item.status).toBe('cancelled');

    await tasksStore.remove('action_test_1');
    item = tasksStore.items.find(t => t.id === 'action_test_1');
    expect(item).toBeUndefined();
  });

  it('extractTaskProgress correctly extracts Go wire progress properties', () => {
    const emptyProg = extractTaskProgress(null);
    expect(emptyProg.percent).toBe(0);
    expect(emptyProg.currentBytes).toBe(0);

    const snakeObj = {
      progress: {
        percent: 75.5,
        current_bytes: 7500,
        total_bytes: 10000,
        bytes_per_sec: 2500,
        eta_seconds: 1,
        current_items: 3,
        total_items: 4,
      }
    };
    const extractedSnake = extractTaskProgress(snakeObj);
    expect(extractedSnake.percent).toBe(75.5);
    expect(extractedSnake.currentBytes).toBe(7500);
    expect(extractedSnake.totalBytes).toBe(10000);
    expect(extractedSnake.speedBps).toBe(2500);
    expect(extractedSnake.etaSeconds).toBe(1);
    expect(extractedSnake.currentItems).toBe(3);
    expect(extractedSnake.totalItems).toBe(4);
  });

  it('normalizeTask normalizes raw Go task wire shape to canonical camelCase Task', () => {
    const rawGoTask = {
      id: 'task_wire_1',
      type: 'download',
      title: 'Hades II',
      game_name: 'Hades II',
      game_dir: '/games/Hades2',
      app_id: '1145360',
      status: 'running',
      stage: 'Downloading Depots',
      details: 'chunk_123.bin',
      capabilities: 15,
      progress: {
        percent: 45.5,
        current_bytes: 45500000,
        total_bytes: 100000000,
        bytes_per_sec: 25000000,
        eta_seconds: 4,
        current_items: 45,
        total_items: 100,
      },
      created_at: 1700000000,
      started_at: 1700000010,
    };

    const task = normalizeTask(rawGoTask);
    expect(task.id).toBe('task_wire_1');
    expect(task.type).toBe('download');
    expect(task.title).toBe('Hades II');
    expect(task.gameName).toBe('Hades II');
    expect(task.gameDir).toBe('/games/Hades2');
    expect(task.appId).toBe('1145360');
    expect(task.status).toBe('running');
    expect(task.percent).toBe(45.5);
    expect(task.currentBytes).toBe(45500000);
    expect(task.totalBytes).toBe(100000000);
    expect(task.speedBps).toBe(25000000);
    expect(task.bytesPerSec).toBe(25000000);
    expect(task.etaSeconds).toBe(4);
    expect(task.currentItems).toBe(45);
    expect(task.totalItems).toBe(100);
    expect(task.progress).toEqual({
      percent: 45.5,
      currentBytes: 45500000,
      totalBytes: 100000000,
      currentItems: 45,
      totalItems: 100,
      speedBps: 25000000,
      etaSeconds: 4,
    });
  });

  describe('isLiveStatus', () => {
    it('returns true for active, queued, or paused statuses', () => {
      expect(isLiveStatus('running')).toBe(true);
      expect(isLiveStatus('downloading')).toBe(true);
      expect(isLiveStatus('queued')).toBe(true);
      expect(isLiveStatus('pending')).toBe(true);
      expect(isLiveStatus('paused')).toBe(true);
    });

    it('returns false for terminal or empty statuses', () => {
      expect(isLiveStatus('completed')).toBe(false);
      expect(isLiveStatus('failed')).toBe(false);
      expect(isLiveStatus('cancelled')).toBe(false);
      expect(isLiveStatus('')).toBe(false);
      expect(isLiveStatus(null)).toBe(false);
      expect(isLiveStatus(undefined)).toBe(false);
    });
  });

  describe('taskMatchesGame', () => {
    it('matches by gameDir via isSamePath', () => {
      expect(taskMatchesGame({ gameDir: '/games/Hades2' }, { gameDir: '/games/Hades2/' })).toBe(true);
      expect(taskMatchesGame({ gameDir: '/games/Hades2' }, { gameDir: '/games/Other' })).toBe(false);
    });

    it('matches by appId when both are valid non-zero strings/numbers', () => {
      expect(taskMatchesGame({ appId: '1145360' }, { appId: 1145360 })).toBe(true);
      expect(taskMatchesGame({ appId: '1145360' }, { appId: '1091500' })).toBe(false);
    });

    it('ignores 0, empty, or missing sentinels per tenet 4', () => {
      expect(taskMatchesGame({ appId: '0' }, { appId: '0' })).toBe(false);
      expect(taskMatchesGame({ appId: 0 }, { appId: 0 })).toBe(false);
      expect(taskMatchesGame({ appId: '' }, { appId: '' })).toBe(false);
      expect(taskMatchesGame({ appId: '123' }, { appId: '0' })).toBe(false);
      expect(taskMatchesGame({ appId: '0' }, { appId: '123' })).toBe(false);
    });

    it('safely handles null/undefined arguments', () => {
      expect(taskMatchesGame(null, null)).toBe(false);
      expect(taskMatchesGame({ gameDir: '/a' }, null)).toBe(false);
      expect(taskMatchesGame(null, { gameDir: '/a' })).toBe(false);
    });
  });
});
