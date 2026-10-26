import { render, screen, fireEvent } from '@testing-library/svelte';
import { describe, it, expect, vi } from 'vitest';
import CoopSeatCard from './CoopSeatCard.svelte';
import { multiseatStore } from '../../../../stores/multiseat.js';

describe('CoopSeatCard Component', () => {
  const profileOptions = [
    { value: 'main', label: 'Alice (Host Main)' },
    { value: 'p2', label: 'Bob (Player 2)' },
    { value: 'p3', label: 'Charlie (Player 3)' }
  ];

  const displays = [
    { index: 0, name: 'DP-1', width: 2560, height: 1440 }
  ];

  const seats = [
    { seat_id: 'seat_1', profile_id: 'main', display_index: 0, screen_slot: 0, disabled: false, mode: 'local' },
    { seat_id: 'seat_2', profile_id: 'p2', display_index: 0, screen_slot: 1, disabled: false, mode: 'local' }
  ];

  it('renders player seat cards for P1 and P2', () => {
    render(CoopSeatCard, {
      multiseatDraft: { default_seat1_mode: 'local', default_seat2_mode: 'local' },
      cfgDraft: { multiseat: { default_seat1_mode: 'local', default_seat2_mode: 'local' } },
      seats,
      activeSeats: seats,
      displays,
      profileOptions
    });

    expect(screen.getByText('Player 1 (Host)')).toBeInTheDocument();
    expect(screen.getByText('Seat 2 Profile (Co-op Guest)')).toBeInTheDocument();
    expect(screen.getByRole('button', { name: /Add Player Seat/i })).toBeInTheDocument();
  });

  it('triggers addSeat when clicking Add Player Seat', async () => {
    const addSeatSpy = vi.spyOn(multiseatStore, 'addSeat');

    render(CoopSeatCard, {
      multiseatDraft: { default_seat1_mode: 'local', default_seat2_mode: 'local' },
      cfgDraft: { multiseat: { default_seat1_mode: 'local', default_seat2_mode: 'local' } },
      seats,
      activeSeats: seats,
      displays,
      profileOptions
    });

    const addBtn = screen.getByRole('button', { name: /Add Player Seat/i });
    await fireEvent.click(addBtn);

    expect(addSeatSpy).toHaveBeenCalled();
  });

  it('toggles seat active/benched status', async () => {
    const toggleSpy = vi.spyOn(multiseatStore, 'toggleSeatActive');

    render(CoopSeatCard, {
      multiseatDraft: { default_seat1_mode: 'local', default_seat2_mode: 'local' },
      cfgDraft: { multiseat: { default_seat1_mode: 'local', default_seat2_mode: 'local' } },
      seats,
      activeSeats: seats,
      displays,
      profileOptions
    });

    const activeButtons = screen.getAllByRole('button', { name: /Active/i });
    expect(activeButtons.length).toBeGreaterThan(0);
    await fireEvent.click(activeButtons[1]); // Toggle seat 2

    expect(toggleSpy).toHaveBeenCalledWith('seat_2');
  });

  it('renders remove seat button for player 3 or higher and triggers removeSeat', async () => {
    const removeSpy = vi.spyOn(multiseatStore, 'removeSeat');
    const threeSeats = [
      ...seats,
      { seat_id: 'seat_3', profile_id: 'p3', display_index: 0, screen_slot: 2, disabled: false, mode: 'local' }
    ];

    render(CoopSeatCard, {
      multiseatDraft: { default_seat1_mode: 'local', default_seat2_mode: 'local' },
      cfgDraft: { multiseat: { default_seat1_mode: 'local', default_seat2_mode: 'local' } },
      seats: threeSeats,
      activeSeats: threeSeats,
      displays,
      profileOptions
    });

    const removeBtn = screen.getByRole('button', { name: /Remove Seat/i });
    expect(removeBtn).toBeInTheDocument();
    await fireEvent.click(removeBtn);

    expect(removeSpy).toHaveBeenCalledWith(2);
  });
});
