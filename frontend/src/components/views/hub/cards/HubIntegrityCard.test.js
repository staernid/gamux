import { render, screen, fireEvent, waitFor } from '@testing-library/svelte';
import { describe, it, expect, vi } from 'vitest';
import HubIntegrityCard from './HubIntegrityCard.svelte';
import { AppAPI } from '../../../../api/app.js';

describe('HubIntegrityCard Component', () => {
  it('renders verify button and clean state without report', () => {
    render(HubIntegrityCard, {
      props: {
        isCheckingIntegrity: false,
        integrityReport: null,
      },
    });

    expect(screen.getByText('Verify Files')).toBeInTheDocument();
    expect(screen.getByText(/Verify local game files against official SteamPipe depot manifests/i)).toBeInTheDocument();
  });

  it('renders intact report with verified status badge', () => {
    render(HubIntegrityCard, {
      props: {
        integrityReport: {
          official_count: 50,
          intact_count: 50,
          modified_files: [],
          missing_files: [],
          untracked_files: [],
        },
      },
    });

    expect(screen.getByText(/All 50 official game files are intact and verified/i)).toBeInTheDocument();
    expect(screen.getByText('Verified')).toBeInTheDocument();
  });

  it('renders damaged report with repair button and allows expanding details', async () => {
    const onUpdateGame = vi.fn();

    render(HubIntegrityCard, {
      props: {
        integrityReport: {
          official_count: 50,
          intact_count: 48,
          modified_files: ['bin/engine.dll'],
          missing_files: ['assets/pakchunk0.pak'],
          untracked_files: ['reshade.ini'],
        },
        onUpdateGame,
      },
    });

    expect(screen.getByText(/2 files need repair/i)).toBeInTheDocument();
    const repairBtn = screen.getByText('Repair Game Files');
    expect(repairBtn).toBeInTheDocument();
    await fireEvent.click(repairBtn);
    expect(onUpdateGame).toHaveBeenCalled();

    // Toggle details list
    const showDetailsBtn = screen.getByText(/Show 2 damaged files/i);
    await fireEvent.click(showDetailsBtn);

    expect(screen.getByText('bin/engine.dll')).toBeInTheDocument();
    expect(screen.getByText('assets/pakchunk0.pak')).toBeInTheDocument();
  });

  it('displays live telemetry banner when verifyTask is running', async () => {
    const cancelSpy = vi.spyOn(AppAPI, 'cancelTask').mockResolvedValue();

    const mockVerifyTask = {
      id: 'task_verify_123',
      type: 'verify',
      status: 'running',
      stage: 'Verifying Integrity',
      details: 'Data/Textures/environment.pak',
      progress: {
        percent: 42.5,
        current_bytes: 4250000000,
        total_bytes: 10000000000,
        bytes_per_sec: 150000000,
        eta_seconds: 38,
        current_items: 42,
        total_items: 100,
      },
    };

    render(HubIntegrityCard, {
      props: {
        activeVerifyTask: mockVerifyTask,
      },
    });

    expect(screen.getByText(/Verifying File Integrity.../i)).toBeInTheDocument();
    expect(screen.getByText('42.5%')).toBeInTheDocument();
    expect(screen.getByText(/Data\/Textures\/environment.pak/i)).toBeInTheDocument();

    const cancelBtn = screen.getByText('Cancel');
    await fireEvent.click(cancelBtn);
    expect(cancelSpy).toHaveBeenCalledWith('task_verify_123');
  });
});
