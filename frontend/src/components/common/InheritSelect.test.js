import { render, screen, fireEvent } from '@testing-library/svelte';
import { describe, it, expect } from 'vitest';
import InheritSelect from './InheritSelect.svelte';

describe('InheritSelect Component', () => {
  const options = [
    { value: 'umu', label: 'UMU Launcher' },
    { value: 'proton', label: 'Proton Direct' },
    { value: 'wine', label: 'Wine Direct' }
  ];

  it('renders standard select dropdown without inheritance', () => {
    render(InheritSelect, {
      props: {
        label: 'Default Runner',
        value: 'umu',
        options
      }
    });

    expect(screen.getByText('Default Runner')).toBeInTheDocument();
    expect(screen.queryByText('(Override)')).not.toBeInTheDocument();
    expect(screen.queryByText(/Default \(/)).not.toBeInTheDocument();
    expect(screen.queryByLabelText('Reset to global default')).not.toBeInTheDocument();
  });

  it('renders inherited dropdown showing Default (UMU Launcher) when value is empty', () => {
    render(InheritSelect, {
      props: {
        label: 'Runner Layer',
        value: '',
        globalValue: 'umu',
        options
      }
    });

    expect(screen.getByText('Runner Layer')).toBeInTheDocument();
    expect(screen.getByText('Default (UMU Launcher)')).toBeInTheDocument();
    expect(screen.queryByText('(Override)')).not.toBeInTheDocument();
    expect(screen.queryByLabelText('Reset to global default')).not.toBeInTheDocument();
  });

  it('shows override badge and reset button when custom option is selected in inherit mode', async () => {
    render(InheritSelect, {
      props: {
        label: 'Runner Layer',
        value: 'proton',
        globalValue: 'umu',
        options
      }
    });

    expect(screen.getByText('Runner Layer')).toBeInTheDocument();
    expect(screen.getByText('(Override)')).toBeInTheDocument();
    const resetBtn = screen.getByLabelText('Reset to global default');
    expect(resetBtn).toBeInTheDocument();

    await fireEvent.click(resetBtn);
    expect(screen.queryByText('(Override)')).not.toBeInTheDocument();
  });
});
