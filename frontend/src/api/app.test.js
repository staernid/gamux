import { describe, it, expect } from 'vitest';
import { AppAPI, DEFAULT_MOCKS } from './app.js';
import { BINDINGS } from './bindings.gen.js';

describe('AppAPI bridge', () => {
  it('only mocks methods that exist on the Go gui.App service', () => {
    const stale = Object.keys(DEFAULT_MOCKS).filter((k) => !Object.hasOwn(BINDINGS, k));
    expect(stale).toEqual([]);
  });

  it('exposes unknown methods as undefined so typos fail fast', () => {
    expect(AppAPI.definitelyNotAMethod).toBeUndefined();
    expect('definitelyNotAMethod' in AppAPI).toBe(false);
    expect(typeof AppAPI.getConfig).toBe('function');
  });

  it('rejects calls with the wrong argument count, like Wails v3 does', async () => {
    await expect(AppAPI.inspectGame('/some/dir')).rejects.toThrow(/expects 2 argument\(s\), got 1/);
  });

  it('rejects bound methods that have no browser mock', async () => {
    const unmocked = Object.keys(BINDINGS).find((k) => BINDINGS[k] === 0 && !DEFAULT_MOCKS[k]);
    if (!unmocked) return;
    await expect(AppAPI[unmocked]()).rejects.toThrow(/has no mock implementation/);
  });
});
