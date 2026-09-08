import { render, screen, fireEvent } from '@testing-library/svelte';
import { describe, it, expect, beforeEach } from 'vitest';
import HubEnvironmentTab from './HubEnvironmentTab.svelte';
import { drawerDraft } from '../../../stores/drawerDraft.js';
import { normalizeGame } from '../../../utils/game.js';

describe('HubEnvironmentTab Component', () => {
  const globalCfg = {
    default_runner: 'umu',
    operating_mode: 'loader',
    language: 'auto',
    mangohud: false
  };

  beforeEach(() => {
    drawerDraft.init(null, globalCfg);
  });

  it('renders Windows game settings with UMU runner and hides Wine Prefix by default', () => {
    const game = normalizeGame({
      name: 'Cyberpunk 2077',
      app_id: '1091500',
      store: 'Steam',
      platform: 'win64',
      game_dir: '/games/cyberpunk'
    });
    drawerDraft.init(game, globalCfg);

    render(HubEnvironmentTab, {
      props: {
        game,
        globalCfg
      }
    });

    expect(screen.getByText('Runner & Platform')).toBeInTheDocument();
    expect(screen.getByText('Target Platform')).toBeInTheDocument();
    expect(screen.getByText('Runner Layer')).toBeInTheDocument();
    expect(screen.getByText('DLL Overrides')).toBeInTheDocument();
    expect(screen.queryByText('Proton Path')).not.toBeInTheDocument();
    // Wine Prefix should not be visible when runner is UMU
    expect(screen.queryByText('Wine Prefix')).not.toBeInTheDocument();
    // Steam Emulation should be visible for Steam game
    expect(screen.getByText('Steam Emulation')).toBeInTheDocument();
    expect(screen.getByText('Operating Mode')).toBeInTheDocument();
    expect(screen.getByText('Game Language')).toBeInTheDocument();
  });

  it('reveals Wine Prefix input when runner is set to Wine', async () => {
    const game = normalizeGame({
      name: 'Cyberpunk 2077',
      app_id: '1091500',
      store: 'Steam',
      platform: 'win64',
      game_dir: '/games/cyberpunk',
      runner: 'wine'
    });
    drawerDraft.init(game, globalCfg);

    render(HubEnvironmentTab, {
      props: {
        game,
        globalCfg
      }
    });

    expect(screen.getByText('Wine Prefix')).toBeInTheDocument();
    expect(screen.getByText('DLL Overrides')).toBeInTheDocument();
  });

  it('disables runner selector and hides Wine Prefix & DLL Overrides for Linux native game', () => {
    const game = normalizeGame({
      name: 'Celeste',
      app_id: '504230',
      store: 'Steam',
      platform: 'linux',
      game_dir: '/games/celeste'
    });
    drawerDraft.init(game, globalCfg);

    render(HubEnvironmentTab, {
      props: {
        game,
        globalCfg
      }
    });

    expect(screen.getByText('Linux Native')).toBeInTheDocument();
    expect(screen.queryByText('Wine Prefix')).not.toBeInTheDocument();
    expect(screen.queryByText('DLL Overrides')).not.toBeInTheDocument();
    expect(screen.queryByText('Proton Path')).not.toBeInTheDocument();
  });

  it('hides Steam Emulation section for non-Steam games (e.g. GOG/Standalone)', () => {
    const game = normalizeGame({
      name: 'Witcher 3 GOG',
      app_id: '0',
      store: 'GOG',
      source: 'GOG',
      platform: 'win64',
      game_dir: '/games/witcher3'
    });
    drawerDraft.init(game, globalCfg);

    render(HubEnvironmentTab, {
      props: {
        game,
        globalCfg
      }
    });

    expect(screen.queryByText('Steam Emulation')).not.toBeInTheDocument();
    expect(screen.queryByText('Operating Mode')).not.toBeInTheDocument();
    expect(screen.getByText('DLL Overrides')).toBeInTheDocument();
  });

  it('provides loader and portable operating mode options', () => {
    const game = normalizeGame({
      name: 'Cyberpunk 2077',
      app_id: '1091500',
      store: 'Steam',
      platform: 'win64',
      game_dir: '/games/cyberpunk'
    });
    drawerDraft.init(game, globalCfg);

    render(HubEnvironmentTab, {
      props: {
        game,
        globalCfg
      }
    });

    expect(screen.getByText('Operating Mode')).toBeInTheDocument();
    expect(screen.getAllByText(/Loader Mode/i).length).toBeGreaterThan(0);
  });
});
