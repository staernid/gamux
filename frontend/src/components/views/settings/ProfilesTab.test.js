import { render, screen, fireEvent } from '@testing-library/svelte';
import { describe, it, expect, beforeEach, vi } from 'vitest';
import { get } from 'svelte/store';
import ProfilesTab from './ProfilesTab.svelte';
import { profilesStore } from '../../../stores/profiles.js';
import { modalStore } from '../../../stores/modal.js';

describe('ProfilesTab Component', () => {
  const languagesList = [
    { value: 'english', label: 'English' },
    { value: 'french', label: 'French' }
  ];

  const countriesList = [
    { value: 'US', label: 'United States' },
    { value: 'FR', label: 'France' }
  ];

  const mainProfile = { id: 'main', index: 1, name: 'Alice', is_main: true, steam_id: '76561197960287930' };
  const p2Profile = { id: 'p2', index: 2, name: 'Bob', is_main: false, steam_id: '76561198998877665' };

  beforeEach(() => {
    modalStore.close();
    profilesStore.setProfiles([mainProfile, p2Profile]);
    profilesStore.setActiveProfile(mainProfile);
  });

  it('renders main profile information and hides delete button', () => {
    render(ProfilesTab, { languagesList, countriesList });

    expect(screen.getByText('Alice')).toBeInTheDocument();
    expect(screen.getByText('Primary Host')).toBeInTheDocument();
    expect(screen.getByText(/Uses host environment and default prefix/)).toBeInTheDocument();
    expect(screen.queryByRole('button', { name: /Delete Profile/i })).not.toBeInTheDocument();
  });

  it('renders secondary profile and displays delete button', () => {
    profilesStore.setActiveProfile(p2Profile);
    render(ProfilesTab, { languagesList, countriesList });

    expect(screen.getByText('Bob')).toBeInTheDocument();
    expect(screen.getByText('Seat 2')).toBeInTheDocument();
    expect(screen.getByText(/Isolated sandbox with profile prefix/)).toBeInTheDocument();
    expect(screen.getByRole('button', { name: /Delete Profile/i })).toBeInTheDocument();
  });

  it('opens standard confirm modal when Delete Profile is clicked', async () => {
    profilesStore.setActiveProfile(p2Profile);
    render(ProfilesTab, { languagesList, countriesList });

    const deleteBtn = screen.getByRole('button', { name: /Delete Profile/i });
    await fireEvent.click(deleteBtn);

    const currentModal = get(modalStore);
    expect(currentModal).toBeTruthy();
    expect(currentModal.type).toBe('confirm');
    expect(currentModal.props.title).toContain('Delete Profile "Bob"?');
    expect(currentModal.props.message).toContain('isolated sandbox, and its Wine prefixes (~/Games/umu)');
    expect(currentModal.props.confirmLabel).toBe('Delete Profile');
    expect(currentModal.props.danger).toBe(true);

    // Test onConfirm execution
    const deleteSpy = vi.spyOn(profilesStore, 'delete').mockResolvedValue(true);
    await currentModal.props.onConfirm();
    expect(deleteSpy).toHaveBeenCalledWith('p2');
  });
});
