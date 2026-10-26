import { render, screen, fireEvent } from '@testing-library/svelte';
import { describe, it, expect, vi } from 'vitest';
import HubDLCConfiguredList from './HubDLCConfiguredList.svelte';
import HubDepotSummaryBanner from './HubDepotSummaryBanner.svelte';
import HubDepotFilters from './HubDepotFilters.svelte';
import HubDepotRow from './HubDepotRow.svelte';

describe('Hub Depot & DLC Child Components', () => {
  describe('HubDLCConfiguredList', () => {
    it('renders configured DLC items and triggers onsync', async () => {
      const onsync = vi.fn();
      render(HubDLCConfiguredList, {
        props: {
          dlcs: [{ appId: '123', name: 'Expansion Pack' }],
          appId: '100',
          isSyncing: false,
          onsync,
        },
      });

      expect(screen.getByText('Expansion Pack')).toBeInTheDocument();
      expect(screen.getByText('AppID #123')).toBeInTheDocument();

      const syncBtn = screen.getByText('Sync Steam DLCs');
      await fireEvent.click(syncBtn);
      expect(onsync).toHaveBeenCalled();
    });
  });

  describe('HubDepotSummaryBanner', () => {
    it('renders installed summary and triggers refresh', async () => {
      const onrefresh = vi.fn();
      render(HubDepotSummaryBanner, {
        props: {
          isInstalled: true,
          installedCount: 2,
          totalDepotsCount: 5,
          totalInstalledSize: 1024 * 1024 * 500,
          onrefresh,
        },
      });

      expect(screen.getByText(/2 of 5 depots installed/i)).toBeInTheDocument();
      const refreshBtn = screen.getByText('Refresh');
      await fireEvent.click(refreshBtn);
      expect(onrefresh).toHaveBeenCalled();
    });

    it('renders pre-install download selected button', async () => {
      const ondownloadSelected = vi.fn();
      render(HubDepotSummaryBanner, {
        props: {
          isInstalled: false,
          totalDepotsCount: 3,
          selectedCount: 2,
          selectedDepotsSize: 1024 * 1024,
          appId: '100',
          ondownloadSelected,
        },
      });

      const dlBtn = screen.getByText(/Download Selected \(2\)/i);
      await fireEvent.click(dlBtn);
      expect(ondownloadSelected).toHaveBeenCalled();
    });
  });

  describe('HubDepotFilters', () => {
    it('renders filter pills and calls onselectFilter and onchangeLanguage', async () => {
      const onselectFilter = vi.fn();
      const onchangeLanguage = vi.fn();

      render(HubDepotFilters, {
        props: {
          totalDepotsCount: 10,
          languageDepotsCount: 3,
          optionalDepotsCount: 2,
          soundtrackDepotsCount: 1,
          dlcDepotsCount: 2,
          coreDepotsCount: 2,
          selectedCategoryFilter: 'all',
          availableLanguages: ['english', 'german'],
          configuredLanguage: 'english',
          onselectFilter,
          onchangeLanguage,
        },
      });

      const langFilterBtn = screen.getByText(/Languages/i);
      await fireEvent.click(langFilterBtn);
      expect(onselectFilter).toHaveBeenCalledWith('languages');

      const select = screen.getByRole('combobox');
      await fireEvent.change(select, { target: { value: 'german' } });
      expect(onchangeLanguage).toHaveBeenCalledWith('german');
    });
  });

  describe('HubDepotRow', () => {
    const depot = {
      depot_id: 200,
      name: 'High Res Textures',
      category: 'common',
      content_category: 'optional_hd',
      file_count: 10,
      size: 5000000,
      installed: false,
    };

    it('renders uninstalled depot row and triggers toggle and download', async () => {
      const ontoggle = vi.fn();
      const ondownload = vi.fn();

      render(HubDepotRow, {
        props: {
          depot,
          isInstalled: false,
          isSelected: true,
          ontoggle,
          ondownload,
        },
      });

      expect(screen.getByText('High Res Textures')).toBeInTheDocument();
      expect(screen.getByText('High-Res / Optional Assets')).toBeInTheDocument();

      const checkbox = screen.getByRole('checkbox');
      expect(checkbox).toBeChecked();
      await fireEvent.click(checkbox);
      expect(ontoggle).toHaveBeenCalled();

      const installBtn = screen.getByText('Install');
      await fireEvent.click(installBtn);
      expect(ondownload).toHaveBeenCalled();
    });

    it('renders installed depot row and triggers uninstall', async () => {
      const onuninstall = vi.fn();
      const installedDepot = { ...depot, installed: true };

      render(HubDepotRow, {
        props: {
          depot: installedDepot,
          isInstalled: true,
          canUninstall: true,
          onuninstall,
        },
      });

      expect(screen.getByText('Installed')).toBeInTheDocument();
      const uninstallBtn = screen.getByText('Uninstall');
      await fireEvent.click(uninstallBtn);
      expect(onuninstall).toHaveBeenCalled();
    });
  });
});
