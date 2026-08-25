import { describe, it, expect, beforeEach } from 'vitest';
import { GamuxDevBridge } from './debugBridge.js';
import { tasksStore } from '../stores/tasks.js';

describe('GamuxDevBridge diagnostic harness', () => {
  let bridge;

  beforeEach(() => {
    tasksStore.reset();
    bridge = new GamuxDevBridge();
    bridge.registerEventListener('gamux:task-event', (ev) => {
      tasksStore.handleTaskEvent(ev);
    });
    bridge.registerEventListener('gamux:transfer-event', (ev) => {
      tasksStore.handleTransferEvent(ev);
    });
  });

  it('records timeline events correctly', () => {
    bridge.emit('gamux:task-event', {
      type: 'task:started',
      task_id: 'test_123',
      operation: 'apply',
      game_name: 'Test Game'
    });

    const entries = bridge.timeline.getEntries();
    expect(entries.length).toBeGreaterThan(0);
    expect(entries[entries.length - 1].eventName).toBe('gamux:task-event');
    expect(entries[entries.length - 1].payload.task_id).toBe('test_123');
  });

  it('audits tasksStore invariants cleanly on complete lifecycle', async () => {
    await bridge.runScenario('apply-game-lifecycle');
    const report = bridge.audit();
    expect(report.passed).toBe(true);
    expect(report.errorsCount).toBe(0);
  });

  it('audits transfer lifecycle cleanly in tasksStore', async () => {
    await bridge.runScenario('download-transfer-lifecycle');
    const report = bridge.audit();
    expect(report.passed).toBe(true);
    expect(report.errorsCount).toBe(0);
  });

  it('detects invariant violation when activeTask status is inconsistent', () => {
    const invalidSnapshot = {
      ...bridge.getStoreSnapshots(),
      tasks: {
        activeTask: { id: 'broken_job', status: 'completed', title: 'Broken' },
        items: [{ id: 'broken_job', status: 'completed', title: 'Broken' }],
        queued: [],
        activeCount: 0
      }
    };

    const report = bridge.audit(invalidSnapshot);
    expect(report.passed).toBe(false);
    expect(report.errorsCount).toBeGreaterThan(0);
    const issue = report.issues.find(i => i.rule === 'activeTask_must_be_running');
    expect(issue).toBeDefined();
  });

  it('generates structured state dump', () => {
    const dump = bridge.dumpState();
    expect(dump.version).toBe('1.0');
    expect(dump.stores).toBeDefined();
    expect(dump.timeline).toBeDefined();
    expect(dump.audit).toBeDefined();
  });
});
