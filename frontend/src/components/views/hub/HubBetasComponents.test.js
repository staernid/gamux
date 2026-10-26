import { render, fireEvent, screen } from '@testing-library/svelte';
import { describe, it, expect, vi } from 'vitest';
import HubBranchList from './HubBranchList.svelte';
import HubBetasStatusBanner from './HubBetasStatusBanner.svelte';
import HubPrivateBetasCard from './HubPrivateBetasCard.svelte';

describe('Hub Betas Subcomponents', () => {
  describe('HubBranchList', () => {
    const branches = [
      { name: 'public', description: 'Public release', build_id: '100' },
      { name: 'beta', description: 'Beta branch', build_id: '200' },
      { name: 'staging', description: 'Staging branch', build_id: '300', pwd_required: true }
    ];

    it('renders branch list with active branch marker', () => {
      render(HubBranchList, {
        props: {
          branches,
          currentBranch: 'public'
        }
      });

      expect(screen.getByText('None / Default (public)')).toBeInTheDocument();
      expect(screen.getByText('ACTIVE')).toBeInTheDocument();
      expect(screen.getByText('Selected')).toBeInTheDocument();
      expect(screen.getByText('beta')).toBeInTheDocument();
      expect(screen.getByText('Private')).toBeInTheDocument();
    });

    it('filters branches by query when input changed', async () => {
      render(HubBranchList, {
        props: {
          branches: [
            ...branches,
            { name: 'b4', description: 'extra4' },
            { name: 'b5', description: 'extra5' }
          ],
          currentBranch: 'public'
        }
      });

      const input = screen.getByPlaceholderText('Filter branches...');
      await fireEvent.input(input, { target: { value: 'staging' } });

      expect(screen.getByText('staging')).toBeInTheDocument();
      expect(screen.queryByText('beta')).not.toBeInTheDocument();
    });

    it('calls onselectBranch when Switch to Branch is clicked', async () => {
      const selectSpy = vi.fn();
      render(HubBranchList, {
        props: {
          branches,
          currentBranch: 'public',
          onselectBranch: selectSpy
        }
      });

      const switchBtns = screen.getAllByRole('button', { name: /Switch to Branch/i });
      await fireEvent.click(switchBtns[0]); // beta branch

      expect(selectSpy).toHaveBeenCalledWith(expect.objectContaining({ name: 'beta' }));
    });
  });

  describe('HubBetasStatusBanner', () => {
    it('renders app id and current branch badge', () => {
      render(HubBetasStatusBanner, {
        props: {
          effectiveAppId: '1091500',
          currentBranch: 'beta',
          targetBuildId: '200',
          isInstalled: true
        }
      });

      expect(screen.getByText('#1091500')).toBeInTheDocument();
      expect(screen.getByText('beta')).toBeInTheDocument();
      expect(screen.getByText('#200')).toBeInTheDocument();
    });

    it('renders mismatch banner when isBuildMismatch is true', async () => {
      const triggerSpy = vi.fn();
      render(HubBetasStatusBanner, {
        props: {
          effectiveAppId: '1091500',
          currentBranch: 'beta',
          targetBuildId: '200',
          isBuildMismatch: true,
          cleanInstalledBuild: '100',
          ontriggerUpdate: triggerSpy
        }
      });

      expect(screen.getByText('Files Pending Branch Sync')).toBeInTheDocument();
      const btn = screen.getByRole('button', { name: /Switch Files to Branch/i });
      await fireEvent.click(btn);
      expect(triggerSpy).toHaveBeenCalled();
    });
  });

  describe('HubPrivateBetasCard', () => {
    it('triggers oncheckCode on submit button click and enter key', async () => {
      const checkSpy = vi.fn();
      render(HubPrivateBetasCard, {
        props: {
          code: 'secret123',
          oncheckCode: checkSpy
        }
      });

      const btn = screen.getByRole('button', { name: /Check Code/i });
      await fireEvent.click(btn);
      expect(checkSpy).toHaveBeenCalledTimes(1);

      const input = screen.getByPlaceholderText(/Enter beta access code/i);
      await fireEvent.keyDown(input, { key: 'Enter' });
      expect(checkSpy).toHaveBeenCalledTimes(2);
    });
  });
});
