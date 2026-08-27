import { render, fireEvent, screen } from '@testing-library/svelte';
import { describe, it, expect, vi } from 'vitest';
import GamePropertiesModal from './GamePropertiesModal.svelte';
import { normalizeGame } from '../../utils/game.js';
import { drawerDraft } from '../../stores/drawerDraft.js';

describe('GamePropertiesModal Component', () => {
  const mockGame = normalizeGame({
    name: 'Cyberpunk 2077',
    app_id: '1091500',
    platform: 'Windows PE',
    game_dir: '/home/user/Games/Cyberpunk 2077',
    exe_path: 'bin/x64/Cyberpunk2077.exe',
    exe_args: '-dx12',
    runner: 'UMU',
    is_patched: true,
    is_ready: true,
  });

  it('renders header, tabs, and action buttons', () => {
    render(GamePropertiesModal, { props: { game: mockGame } });

    expect(screen.getByText(/Game Properties & Compatibility/i)).toBeInTheDocument();
    expect(screen.getByRole('button', { name: /^General$/i })).toBeInTheDocument();
    expect(screen.getByRole('button', { name: /^Runner$/i })).toBeInTheDocument();
    expect(screen.getByRole('button', { name: /^Emulator$/i })).toBeInTheDocument();
    expect(screen.getByRole('button', { name: /^Saves$/i })).toBeInTheDocument();
    expect(screen.getByRole('button', { name: /^Logs$/i })).toBeInTheDocument();
    expect(screen.getByText(/Save Changes/i)).toBeInTheDocument();
    expect(screen.getByText(/Cancel/i)).toBeInTheDocument();
    expect(screen.getByText(/Reset Defaults/i)).toBeInTheDocument();
  });

  it('allows switching between tabs', async () => {
    render(GamePropertiesModal, { props: { game: mockGame } });

    const runnerBtn = screen.getByRole('button', { name: /^Runner$/i });
    await fireEvent.click(runnerBtn);
    expect(runnerBtn.className).toContain('text-sky-400');

    const savesBtn = screen.getByRole('button', { name: /^Saves$/i });
    await fireEvent.click(savesBtn);
    expect(savesBtn.className).toContain('text-emerald-400');
  });

  it('invokes drawerDraft save on Save Changes click', async () => {
    const saveSpy = vi.spyOn(drawerDraft, 'save').mockResolvedValue(true);
    render(GamePropertiesModal, { props: { game: mockGame } });

    const saveBtn = screen.getByText(/Save Changes/i);
    await fireEvent.click(saveBtn);

    expect(saveSpy).toHaveBeenCalled();
  });
});
