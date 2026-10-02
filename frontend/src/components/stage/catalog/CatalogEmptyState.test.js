import { render, fireEvent, screen } from '@testing-library/svelte';
import { describe, it, expect, vi } from 'vitest';
import CatalogEmptyState from './CatalogEmptyState.svelte';

describe('CatalogEmptyState Component', () => {
  it('renders discovery headline, description, and keyboard shortcuts', () => {
    render(CatalogEmptyState);

    expect(screen.getByText('Online Store Catalog Discovery')).toBeInTheDocument();
    expect(screen.getByText(/Search Steam, GOG, and RAWG for uninstalled titles/i)).toBeInTheDocument();
    expect(screen.getByText('Quick Discovery Seeds')).toBeInTheDocument();
    expect(screen.getByText('Navigate')).toBeInTheDocument();
    expect(screen.getByText('Select')).toBeInTheDocument();
    expect(screen.getByText('Close')).toBeInTheDocument();
  });

  it('renders quick suggestion chips and invokes onsuggest callback on click', async () => {
    const onsuggest = vi.fn();
    render(CatalogEmptyState, { props: { onsuggest } });

    const seedBtn = screen.getByRole('button', { name: 'Balatro' });
    expect(seedBtn).toBeInTheDocument();

    await fireEvent.click(seedBtn);
    expect(onsuggest).toHaveBeenCalledWith('Balatro');
  });
});
