import { describe, it, expect, vi } from 'vitest';
import { AppAPI, DEFAULT_MOCKS, AppService } from './app.js';

describe('AppAPI bridge', () => {
  it('only mocks methods that exist on the Go gui.App service', () => {
    const stale = Object.keys(DEFAULT_MOCKS).filter((k) => {
      const pascal = k.charAt(0).toUpperCase() + k.slice(1);
      return typeof AppService[pascal] !== 'function';
    });
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
    const unmocked = Object.keys(AppService).find((pascal) => {
      const fn = AppService[pascal];
      if (typeof fn !== 'function') return false;
      const camel = pascal.charAt(0).toLowerCase() + pascal.slice(1);
      return fn.length === 0 && !DEFAULT_MOCKS[camel] && !DEFAULT_MOCKS[pascal];
    });
    if (!unmocked) return;
    const camel = unmocked.charAt(0).toLowerCase() + unmocked.slice(1);
    await expect(AppAPI[camel]()).rejects.toThrow(/has no mock implementation/);
  });

  it('delegates to generated AppService when running in desktop mode', async () => {
    const origWails = window.wails;
    window.wails = { invoke: vi.fn() };
    const spy = vi.spyOn(AppService, 'GetRunningGames').mockResolvedValue(['game1']);
    try {
      const res = await AppAPI.getRunningGames();
      expect(spy).toHaveBeenCalled();
      expect(res).toEqual(['game1']);
    } finally {
      spy.mockRestore();
      if (origWails !== undefined) {
        window.wails = origWails;
      } else {
        delete window.wails;
      }
    }
  });

  it('allows overrides and spies on AppAPI methods in tests', async () => {
    const spy = vi.spyOn(AppAPI, 'getConfig').mockResolvedValue({ mocked: true });
    expect(await AppAPI.getConfig()).toEqual({ mocked: true });
    spy.mockRestore();
    const cfg = await AppAPI.getConfig();
    expect(cfg).not.toEqual({ mocked: true });
  });
});
