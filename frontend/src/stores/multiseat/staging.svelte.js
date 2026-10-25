import { defaultSeats, syncStaging, hydrateStagingFromConfig } from './assignments.js';

export class StagingState {
  isCoopEnabled = $state(false);
  staging = $state({
    seats: [
      { ...defaultSeats[0] },
      { ...defaultSeats[1] },
    ],
    seat1: { ...defaultSeats[0] },
    seat2: { ...defaultSeats[1] },
    layout: 'dual_monitor',
    sunshine_port_p1: 47989,
    sunshine_port_p2: 48010,
  });

  syncFromConfig(cfg) {
    if (!cfg?.multiseat) return;
    const hydrated = hydrateStagingFromConfig(cfg.multiseat);
    if (hydrated) {
      this.staging = syncStaging({
        ...this.staging,
        ...hydrated,
      });
    }
  }

  toggleCoop(forceVal) {
    this.isCoopEnabled = typeof forceVal === 'boolean' ? forceVal : !this.isCoopEnabled;
  }

  addSeat() {
    const currentSeats = this.staging.seats || [this.staging.seat1, this.staging.seat2];
    if (currentSeats.length >= 4) return;
    const nextIdx = currentSeats.length + 1;
    const newSeat = {
      seat_id: `p${nextIdx}`,
      mode: 'local',
      display_index: currentSeats.length % 2,
      disabled: false,
      gamepad_node: '',
      keyboard_node: '',
      mouse_node: '',
      physical_sink: '',
      profile_id: `p${nextIdx}`,
      screen_slot: currentSeats.length,
      force_grab_cursor: false,
      grab_keyboard: false,
      exclusive_grab: false,
      compositor: '',
      nest_cage: false,
      show_cursor: true,
    };
    const nextSeats = [...currentSeats, newSeat];
    this.staging = syncStaging({
      ...this.staging,
      seats: nextSeats,
    });
  }

  toggleSeatActive(seatId) {
    const currentSeats = [...(this.staging.seats || [this.staging.seat1, this.staging.seat2])];
    const idx = currentSeats.findIndex(s => s.seat_id === seatId || (seatId === 'seat1' && s.seat_id === 'p1') || (seatId === 'seat2' && s.seat_id === 'p2'));
    if (idx < 0) return;

    const nextDisabled = !currentSeats[idx].disabled;
    currentSeats[idx] = { ...currentSeats[idx], disabled: nextDisabled };

    this.staging = syncStaging({
      ...this.staging,
      seats: currentSeats,
    });
  }

  setSeatDisplay(seatId, displayIndex) {
    const currentSeats = [...(this.staging.seats || [this.staging.seat1, this.staging.seat2])];
    const idx = currentSeats.findIndex(s => s.seat_id === seatId || (seatId === 'seat1' && s.seat_id === 'p1') || (seatId === 'seat2' && s.seat_id === 'p2'));
    if (idx < 0) return;

    currentSeats[idx] = { ...currentSeats[idx], display_index: Number(displayIndex) };

    this.staging = syncStaging({
      ...this.staging,
      seats: currentSeats,
    });
  }

  setSeatSlot(seatId, targetSlot) {
    const currentSeats = [...(this.staging.seats || [this.staging.seat1, this.staging.seat2])];
    const seatIdx = currentSeats.findIndex(s => s.seat_id === seatId);
    if (seatIdx < 0) return;

    const oldSlot = currentSeats[seatIdx].screen_slot ?? seatIdx;
    const otherIdx = currentSeats.findIndex((s, idx) => idx !== seatIdx && (s.screen_slot ?? idx) === targetSlot);

    currentSeats[seatIdx] = { ...currentSeats[seatIdx], screen_slot: targetSlot };
    if (otherIdx >= 0) {
      let fallbackSlot = oldSlot;
      if (fallbackSlot === targetSlot) {
        fallbackSlot = targetSlot === 0 ? 1 : 0;
      }
      currentSeats[otherIdx] = { ...currentSeats[otherIdx], screen_slot: fallbackSlot };
    }

    this.staging = syncStaging({
      ...this.staging,
      seats: currentSeats,
    });
  }

  swapSeatsSlots(seatId1, seatId2) {
    const currentSeats = [...(this.staging.seats || [this.staging.seat1, this.staging.seat2])];
    const idx1 = currentSeats.findIndex(s => s.seat_id === seatId1);
    const idx2 = currentSeats.findIndex(s => s.seat_id === seatId2);
    if (idx1 < 0 || idx2 < 0) return;

    let slot1 = currentSeats[idx1].screen_slot ?? idx1;
    let slot2 = currentSeats[idx2].screen_slot ?? idx2;
    if (slot1 === slot2) {
      slot1 = 0;
      slot2 = 1;
    } else {
      const temp = slot1;
      slot1 = slot2;
      slot2 = temp;
    }

    currentSeats[idx1] = { ...currentSeats[idx1], screen_slot: slot1 };
    currentSeats[idx2] = { ...currentSeats[idx2], screen_slot: slot2 };

    this.staging = syncStaging({
      ...this.staging,
      seats: currentSeats,
    });
  }

  removeSeat(index) {
    const currentSeats = this.staging.seats || [this.staging.seat1, this.staging.seat2];
    if (currentSeats.length <= 2) return;
    const nextSeats = currentSeats.filter((_, idx) => idx !== index).map((s, idx) => ({
      ...s,
      seat_id: `p${idx + 1}`,
    }));
    this.staging = syncStaging({
      ...this.staging,
      seats: nextSeats,
    });
  }

  updateSeat(indexOrId, partial) {
    const currentSeats = [...(this.staging.seats || [this.staging.seat1, this.staging.seat2])];
    const idx = typeof indexOrId === 'number'
      ? indexOrId
      : currentSeats.findIndex(s => s.seat_id === indexOrId || (indexOrId === 'seat1' && s.seat_id === 'p1') || (indexOrId === 'seat2' && s.seat_id === 'p2'));

    if (idx < 0 || idx >= currentSeats.length) return;
    currentSeats[idx] = { ...currentSeats[idx], ...partial };

    const nextStaging = {
      ...this.staging,
      seats: currentSeats,
    };
    if (idx === 0) nextStaging.seat1 = currentSeats[0];
    if (idx === 1) nextStaging.seat2 = currentSeats[1];

    this.staging = syncStaging(nextStaging);
  }

  setSeatMode(seatId, mode) {
    const key = seatId === 'seat1' || seatId === 'p1' ? 'seat1' : (seatId === 'seat2' || seatId === 'p2' ? 'seat2' : seatId);
    const idx = seatId === 'seat1' || seatId === 'p1' ? 0 : (seatId === 'seat2' || seatId === 'p2' ? 1 : parseInt(seatId.replace(/\D/g, ''), 10) - 1);
    const currentSeats = [...(this.staging.seats || [this.staging.seat1, this.staging.seat2])];
    if (idx >= 0 && idx < currentSeats.length) {
      currentSeats[idx] = { ...currentSeats[idx], mode };
    }
    this.staging = syncStaging({
      ...this.staging,
      [key]: { ...(this.staging[key] || {}), mode },
      seats: currentSeats,
    });
  }

  setLayout(layout) {
    this.staging = {
      ...this.staging,
      layout,
    };
  }

  assignGamepad(seatId, devNode) {
    const key = seatId === 'seat1' || seatId === 'p1' ? 'seat1' : (seatId === 'seat2' || seatId === 'p2' ? 'seat2' : seatId);
    const idx = seatId === 'seat1' || seatId === 'p1' ? 0 : (seatId === 'seat2' || seatId === 'p2' ? 1 : parseInt(seatId.replace(/\D/g, ''), 10) - 1);
    const currentSeats = [...(this.staging.seats || [this.staging.seat1, this.staging.seat2])];
    if (idx >= 0 && idx < currentSeats.length) {
      currentSeats[idx] = { ...currentSeats[idx], gamepad_node: devNode };
    }
    this.staging = syncStaging({
      ...this.staging,
      [key]: { ...(this.staging[key] || {}), gamepad_node: devNode },
      seats: currentSeats,
    });
  }

  assignKeyboard(seatId, devNode) {
    const key = seatId === 'seat1' || seatId === 'p1' ? 'seat1' : (seatId === 'seat2' || seatId === 'p2' ? 'seat2' : seatId);
    const idx = seatId === 'seat1' || seatId === 'p1' ? 0 : (seatId === 'seat2' || seatId === 'p2' ? 1 : parseInt(seatId.replace(/\D/g, ''), 10) - 1);
    const currentSeats = [...(this.staging.seats || [this.staging.seat1, this.staging.seat2])];
    if (idx >= 0 && idx < currentSeats.length) {
      currentSeats[idx] = { ...currentSeats[idx], keyboard_node: devNode };
    }
    this.staging = syncStaging({
      ...this.staging,
      [key]: { ...(this.staging[key] || {}), keyboard_node: devNode },
      seats: currentSeats,
    });
  }

  assignMouse(seatId, devNode) {
    const key = seatId === 'seat1' || seatId === 'p1' ? 'seat1' : (seatId === 'seat2' || seatId === 'p2' ? 'seat2' : seatId);
    const idx = seatId === 'seat1' || seatId === 'p1' ? 0 : (seatId === 'seat2' || seatId === 'p2' ? 1 : parseInt(seatId.replace(/\D/g, ''), 10) - 1);
    const currentSeats = [...(this.staging.seats || [this.staging.seat1, this.staging.seat2])];
    if (idx >= 0 && idx < currentSeats.length) {
      currentSeats[idx] = { ...currentSeats[idx], mouse_node: devNode };
    }
    this.staging = syncStaging({
      ...this.staging,
      [key]: { ...(this.staging[key] || {}), mouse_node: devNode },
      seats: currentSeats,
    });
  }

  updateStaging(partial) {
    let stagingSeats = this.staging.seats ? this.staging.seats.map(s => ({ ...s })) : [];
    if (partial.seats) {
      stagingSeats = partial.seats.map(s => ({ ...s }));
    }
    if (partial.seat1 && stagingSeats.length > 0) {
      stagingSeats[0] = { ...stagingSeats[0], ...partial.seat1 };
    }
    if (partial.seat2 && stagingSeats.length > 1) {
      stagingSeats[1] = { ...stagingSeats[1], ...partial.seat2 };
    }
    this.staging = syncStaging({
      ...this.staging,
      ...partial,
      seats: stagingSeats,
    });
  }
}
