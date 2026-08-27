// Frontend GUI Diagnostic & Invariant Auditing Bridge
import { get } from 'svelte/store';
import { tasksStore } from '../stores/tasks.js';
import { notificationStore } from '../stores/notifications.js';
import { configStore } from '../stores/config.js';
import {
  games,
  selectedGame,
  libraryPaths,
  libraryPath,
  activeView,
  showInspector,
  runningGames,
  stats,
  isScanning,
  searchQuery,
  activeFilter,
  sortBy
} from '../stores/library.js';
import { isSamePath } from '../utils/game.js';

class DebugTimeline {
  constructor(maxSize = 100) {
    this.maxSize = maxSize;
    this.events = [];
  }

  record(event) {
    const entry = {
      index: this.events.length,
      timestamp: Date.now(),
      timeIso: new Date().toISOString(),
      ...event
    };
    this.events.push(entry);
    if (this.events.length > this.maxSize) {
      this.events.shift();
    }
    return entry;
  }

  getEntries() {
    return [...this.events];
  }

  clear() {
    this.events = [];
  }
}

export class GamuxDevBridge {
  constructor() {
    this.timeline = new DebugTimeline(100);
    this.eventListeners = new Map();
    this.initialized = false;
  }

  init() {
    if (typeof window === 'undefined') return;

    window.__GAMUX_DEV__ = this;
    this.initialized = true;
  }

  // Register a listener hook for timeline recording
  registerEventListener(eventName, callback) {
    if (!this.eventListeners.has(eventName)) {
      this.eventListeners.set(eventName, new Set());
    }
    this.eventListeners.get(eventName).add(callback);
  }

  // Dispatch an event as if it came from Wails runtime
  emit(eventName, payload) {
    this.timeline.record({
      type: 'event:dispatched',
      eventName,
      payload: JSON.parse(JSON.stringify(payload || {}))
    });

    const listeners = this.eventListeners.get(eventName);
    if (listeners) {
      listeners.forEach(cb => {
        try {
          cb(payload);
        } catch (err) {
          console.error(`[GAMUX_DEV] Error in listener for ${eventName}:`, err);
        }
      });
    }
  }

  // Record an incoming event received by runtime bridge
  recordIncoming(source, eventName, payload) {
    return this.timeline.record({
      source,
      eventName,
      payload: JSON.parse(JSON.stringify(payload || {}))
    });
  }

  // Introspect current Svelte store values
  getStoreSnapshots() {
    return {
      activeView: get(activeView),
      showInspector: get(showInspector),
      libraryPath: get(libraryPath),
      libraryPaths: get(libraryPaths),
      isScanning: get(isScanning),
      searchQuery: get(searchQuery),
      activeFilter: get(activeFilter),
      sortBy: get(sortBy),
      gamesCount: get(games).length,
      games: get(games),
      selectedGame: get(selectedGame),
      runningGames: Array.from(get(runningGames) || []),
      stats: get(stats),
      tasks: {
        items: get(tasksStore.items),
        active: get(tasksStore.active),
        queued: get(tasksStore.queued),
        paused: get(tasksStore.paused),
        completed: get(tasksStore.completed),
        activeTask: get(tasksStore.activeTask),
        totalSpeedBps: get(tasksStore.totalSpeedBps),
        activeCount: get(tasksStore.activeCount),
        speedLimit: get(tasksStore.speedLimit),
      },
      taskQueue: {
        activeTask: get(tasksStore.activeTask),
        queue: get(tasksStore.queued),
        allJobs: get(tasksStore.items),
        totalCount: get(tasksStore.activeCount)
      },
      transfers: {
        items: get(tasksStore.items).filter(t => t.type === 'download' || t.operation === 'download'),
        active: get(tasksStore.activeDownloads),
        queued: get(tasksStore.queuedDownloads),
        paused: get(tasksStore.pausedDownloads),
        completed: get(tasksStore.completedDownloads),
        totalSpeedBps: get(tasksStore.totalSpeedBps),
        activeCount: get(tasksStore.activeTransfersCount),
        speedLimit: get(tasksStore.speedLimit)
      },
      notifications: get(notificationStore.items),
      config: get(configStore)
    };
  }

  // Automated Invariant Auditor
  audit(customState = null) {
    const issues = [];
    const state = customState || this.getStoreSnapshots();

    // 1. Task Queue Invariants
    const taskState = state.tasks || state.taskQueue || {};
    const activeTask = taskState.activeTask;
    const allJobs = taskState.items || taskState.allJobs || [];

    if (activeTask) {
      if (activeTask.status !== 'running' && activeTask.status !== 'downloading') {
        issues.push({
          severity: 'error',
          component: 'tasks',
          rule: 'activeTask_must_be_running',
          message: `activeTask has status '${activeTask.status}', expected 'running' or 'downloading'`,
          context: { activeTaskId: activeTask.id, status: activeTask.status }
        });
      }

      const match = allJobs.find(j => j.id === activeTask.id);
      if (!match) {
        issues.push({
          severity: 'error',
          component: 'tasks',
          rule: 'activeTask_in_jobs_list',
          message: `activeTask (ID: ${activeTask.id}) does not exist in tasks list`,
          context: { activeTaskId: activeTask.id }
        });
      }
    }

    const runningJobs = allJobs.filter(j => j.status === 'running' || j.status === 'downloading');
    if (runningJobs.length > 0 && !activeTask) {
      issues.push({
        severity: 'warn',
        component: 'tasks',
        rule: 'running_jobs_activeTask_sync',
        message: `Found ${runningJobs.length} running tasks, but activeTask is null`,
        context: { runningJobIds: runningJobs.map(j => j.id) }
      });
    }

    // Check for ID mismatches or missing fields in jobs
    allJobs.forEach(job => {
      if (!job.id) {
        issues.push({
          severity: 'error',
          component: 'tasks',
          rule: 'job_missing_id',
          message: `Job found with missing id: ${job.title || 'Untitled'}`,
          context: job
        });
      }
    });

    // 2. Transfers Store Invariants
    const transferItems = state.transfers.items || [];
    transferItems.forEach(tr => {
      if (!tr.id) {
        issues.push({
          severity: 'error',
          component: 'transfers',
          rule: 'transfer_missing_id',
          message: `Transfer found without an ID for game: ${tr.game_name || 'unknown'}`,
          context: tr
        });
      }
      if (tr.status === 'completed' && tr.percent !== 100) {
        issues.push({
          severity: 'warn',
          component: 'transfers',
          rule: 'completed_transfer_percent',
          message: `Transfer ${tr.id} is marked completed but percent is ${tr.percent}%`,
          context: { id: tr.id, percent: tr.percent }
        });
      }
    });

    // 3. Library Invariants
    const selected = state.selectedGame;
    const gameList = state.games || [];
    if (selected && gameList.length > 0) {
      const selectedDir = selected.gameDir;
      const foundInList = gameList.some(g => isSamePath(g.gameDir, selectedDir));
      if (!foundInList) {
        issues.push({
          severity: 'warn',
          component: 'library',
          rule: 'selectedGame_in_games_list',
          message: `selectedGame ('${selected.name}') is not present in the games store`,
          context: { selectedGameDir: selectedDir }
        });
      }
    }

    // 4. Timeline Event Invariants (Check for unhandled or malformed event types)
    const timelineEntries = this.timeline.getEntries();
    timelineEntries.forEach(entry => {
      if (entry.eventName === 'gamux:task-event') {
        const p = entry.payload;
        if (p && !p.task_id && !p.id) {
          issues.push({
            severity: 'warn',
            component: 'timeline',
            rule: 'task_event_missing_id',
            message: `Event '${p.type || 'unknown'}' in gamux:task-event missing both task_id and id`,
            context: entry
          });
        }
      }
    });

    return {
      passed: issues.filter(i => i.severity === 'error').length === 0,
      timestamp: Date.now(),
      issuesCount: issues.length,
      errorsCount: issues.filter(i => i.severity === 'error').length,
      warningsCount: issues.filter(i => i.severity === 'warn').length,
      issues
    };
  }

  // Run lifecycle scenarios
  async runScenario(scenarioName) {
    this.timeline.record({ type: 'scenario:start', scenarioName });

    switch (scenarioName) {
      case 'apply-game-lifecycle': {
        const taskId = 'diag_task_' + Date.now();
        const gameName = 'Cyberpunk 2077';
        const gameDir = '/games/cyberpunk';

        this.emit('gamux:task-event', {
          type: 'task:started',
          task_id: taskId,
          operation: 'apply',
          game_name: gameName,
          game_dir: gameDir,
          stage: 'Initializing Goldberg Emulator...',
          details: 'Applying DRM hooks'
        });

        await new Promise(r => setTimeout(r, 50));

        this.emit('gamux:task-event', {
          type: 'task:progress',
          task_id: taskId,
          stage: 'Patching steam_api64.dll',
          progress: { percent: 50, current_items: 1, total_items: 2 }
        });

        await new Promise(r => setTimeout(r, 50));

        this.emit('gamux:task-event', {
          type: 'task:completed',
          task_id: taskId,
          details: 'Game applied successfully with GBE configuration.'
        });

        return { scenario: scenarioName, taskId, status: 'completed' };
      }

      case 'download-transfer-lifecycle': {
        const transferId = 'diag_tr_' + Date.now();
        const gameName = 'Hollow Knight';

        this.emit('gamux:transfer-event', {
          type: 'transfer:started',
          task_id: transferId,
          game_name: gameName,
          app_id: '367520',
          stage: 'Connecting to Steam CDN...',
          progress: { percent: 0, current_bytes: 0, total_bytes: 1048576000 }
        });

        await new Promise(r => setTimeout(r, 50));

        this.emit('gamux:transfer-event', {
          type: 'transfer:progress',
          task_id: transferId,
          stage: 'Downloading Depot Chunks',
          progress: { percent: 55, current_bytes: 576716800, total_bytes: 1048576000, bytes_per_sec: 15000000, eta_seconds: 31 }
        });

        await new Promise(r => setTimeout(r, 50));

        this.emit('gamux:transfer-event', {
          type: 'transfer:completed',
          task_id: transferId,
          stage: 'Complete',
          details: 'Download and verification complete.'
        });

        return { scenario: scenarioName, transferId, status: 'completed' };
      }

      case 'task-failure-recovery': {
        const taskId = 'diag_fail_' + Date.now();
        this.emit('gamux:task-event', {
          type: 'task:started',
          task_id: taskId,
          operation: 'update',
          game_name: 'Stardew Valley',
          stage: 'Validating checksums...'
        });

        await new Promise(r => setTimeout(r, 50));

        this.emit('gamux:task-event', {
          type: 'task:failed',
          task_id: taskId,
          error: 'Manifest key retrieval failed: 404 Not Found'
        });

        return { scenario: scenarioName, taskId, status: 'failed_as_expected' };
      }

      default:
        throw new Error(`Unknown scenario: ${scenarioName}`);
    }
  }

  // Full state dump
  dumpState() {
    return {
      version: '1.0',
      timestamp: Date.now(),
      audit: this.audit(),
      stores: this.getStoreSnapshots(),
      timeline: this.timeline.getEntries()
    };
  }
}

export const gamuxDev = new GamuxDevBridge();
if (typeof window !== 'undefined') {
  gamuxDev.init();
}
