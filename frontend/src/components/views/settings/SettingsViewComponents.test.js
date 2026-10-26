import { render, fireEvent, screen } from '@testing-library/svelte';
import { describe, it, expect, vi } from 'vitest';
import SettingsNav from './SettingsNav.svelte';
import SettingsHeader from './SettingsHeader.svelte';

describe('Settings View Subcomponents', () => {
  describe('SettingsNav', () => {
    it('renders all 5 main settings tabs', () => {
      render(SettingsNav, { props: { activeTab: 'general' } });

      expect(screen.getByRole('tab', { name: /General/i })).toBeInTheDocument();
      expect(screen.getByRole('tab', { name: /Storage/i })).toBeInTheDocument();
      expect(screen.getByRole('tab', { name: /Compatibility/i })).toBeInTheDocument();
      expect(screen.getByRole('tab', { name: /Profiles/i })).toBeInTheDocument();
      expect(screen.getByRole('tab', { name: /Advanced/i })).toBeInTheDocument();
    });

    it('triggers onselectTab when clicking a tab button', async () => {
      const selectSpy = vi.fn();
      render(SettingsNav, { props: { activeTab: 'general', onselectTab: selectSpy } });

      const compatTab = screen.getByRole('tab', { name: /Compatibility/i });
      await fireEvent.click(compatTab);

      expect(selectSpy).toHaveBeenCalledWith('compatibility');
    });

    it('triggers ontabKeyDown on keydown', async () => {
      const keySpy = vi.fn();
      render(SettingsNav, { props: { activeTab: 'general', ontabKeyDown: keySpy } });

      const generalTab = screen.getByRole('tab', { name: /General/i });
      await fireEvent.keyDown(generalTab, { key: 'ArrowRight' });

      expect(keySpy).toHaveBeenCalled();
    });
  });

  describe('SettingsHeader', () => {
    it('shows unsaved changes indicator when isDirty', () => {
      render(SettingsHeader, { props: { isDirty: true, isSaving: false } });
      expect(screen.getByText(/Unsaved Changes/i)).toBeInTheDocument();
    });

    it('shows saving indicator when isSaving', () => {
      render(SettingsHeader, { props: { isSaving: true } });
      expect(screen.getByText('Saving...')).toBeInTheDocument();
    });

    it('shows saved indicator when saveSuccess', () => {
      render(SettingsHeader, { props: { saveSuccess: true } });
      expect(screen.getByText('Saved')).toBeInTheDocument();
    });

    it('shows error banner with retry button when settingsLoadError', async () => {
      const retrySpy = vi.fn();
      render(SettingsHeader, { props: { settingsLoadError: 'Failed to read disk', onretry: retrySpy } });

      expect(screen.getByText('Failed to read disk')).toBeInTheDocument();
      const retryBtn = screen.getByRole('button', { name: /Retry/i });
      await fireEvent.click(retryBtn);

      expect(retrySpy).toHaveBeenCalled();
    });
  });
});
