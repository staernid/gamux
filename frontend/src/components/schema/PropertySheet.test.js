import { render, screen, fireEvent } from '@testing-library/svelte';
import { describe, it, expect, vi, beforeEach } from 'vitest';
import PropertySheet from './PropertySheet.svelte';
import { AppAPI } from '../../api/app.js';

describe('PropertySheet Component (Server-Driven Dynamic Configuration Engine)', () => {
  beforeEach(() => {
    vi.restoreAllMocks();
  });

  const sampleSchema = [
    {
      key: 'enable_feature_x',
      label: 'Feature X Toggle',
      description: 'Enables experimental system feature X.',
      type: 'boolean',
      category: 'Experimental',
      scope: 'global',
      default: false
    },
    {
      key: 'audio_backend',
      label: 'Audio Backend',
      description: 'Selected audio subsystem.',
      type: 'select',
      category: 'Audio',
      scope: 'global',
      default: 'pipewire',
      options: [
        { value: 'pipewire', label: 'PipeWire (Default)' },
        { value: 'pulseaudio', label: 'PulseAudio' },
        { value: 'alsa', label: 'ALSA Direct' }
      ]
    },
    {
      key: 'install_dir',
      label: 'Installation Directory',
      description: 'Target installation path on host.',
      type: 'path',
      category: 'Storage',
      scope: 'global',
      default: '/opt/games'
    },
    {
      key: 'custom_library_folders',
      label: 'Custom Library Folders',
      description: 'Multiple library storage locations.',
      type: 'path_list',
      category: 'Storage',
      scope: 'global',
      default: []
    },
    {
      key: 'custom_env_vars',
      label: 'Custom Environment Variables',
      description: 'Custom environment key-value pairs.',
      type: 'env',
      category: 'System',
      scope: 'global',
      default: {}
    },
    {
      key: 'max_threads',
      label: 'Worker Thread Pool',
      description: 'Number of parallel CPU worker threads.',
      type: 'number',
      category: 'Performance',
      scope: 'global',
      default: 4
    },
    {
      key: 'multiseat.player2_account_name',
      label: 'Player 2 Name',
      description: 'Display persona for player 2.',
      type: 'string',
      category: 'Multi-Seat',
      scope: 'global',
      default: 'PlayerTwo'
    }
  ];

  it('renders arbitrary schemas dynamically with polymorphic controls without bespoke HTML', () => {
    render(PropertySheet, {
      props: {
        schema: sampleSchema,
        values: {
          enable_feature_x: false,
          audio_backend: 'pipewire',
          install_dir: '/opt/games',
          custom_library_folders: ['/mnt/games1', '/mnt/games2'],
          custom_env_vars: { WINE_DEBUG: 'warn' },
          max_threads: 8,
          multiseat: { player2_account_name: 'Gamer2' }
        }
      }
    });

    // 1. SwitchControl rendered dynamically
    expect(screen.getByTestId('control-switch-enable_feature_x')).toBeInTheDocument();
    expect(screen.getByText('Feature X Toggle')).toBeInTheDocument();
    expect(screen.getByText('Enables experimental system feature X.')).toBeInTheDocument();

    // 2. SelectControl rendered dynamically with its options
    expect(screen.getByTestId('control-select-audio_backend')).toBeInTheDocument();
    expect(screen.getByText('Audio Backend')).toBeInTheDocument();
    expect(screen.getByRole('combobox', { name: /Audio Backend/i })).toBeInTheDocument();
    expect(screen.getByText('PipeWire (Default)')).toBeInTheDocument();

    // 3. PathPickerControl rendered dynamically (single path and path list)
    expect(screen.getByTestId('control-path-install_dir')).toBeInTheDocument();
    expect(screen.getByText('Installation Directory')).toBeInTheDocument();
    expect(screen.getByTestId('control-path-custom_library_folders')).toBeInTheDocument();
    expect(screen.getByText('/mnt/games1')).toBeInTheDocument();
    expect(screen.getByText('/mnt/games2')).toBeInTheDocument();

    // 4. EnvEditorControl rendered dynamically
    expect(screen.getByTestId('control-env-custom_env_vars')).toBeInTheDocument();
    expect(screen.getByText('WINE_DEBUG')).toBeInTheDocument();
    expect(screen.getByText('warn')).toBeInTheDocument();

    // 5. Number and string controls
    expect(screen.getByTestId('control-number-max_threads')).toBeInTheDocument();
    expect(screen.getByTestId('control-text-multiseat.player2_account_name')).toBeInTheDocument();
  });

  it('handles SwitchControl interaction and emits change event', async () => {
    let changeEvent = null;
    render(PropertySheet, {
      props: {
        schema: sampleSchema,
        values: { enable_feature_x: false },
        onChange: (detail) => {
          changeEvent = detail;
        }
      }
    });

    const toggle = screen.getByRole('switch', { name: /Feature X Toggle/i });
    expect(toggle.getAttribute('aria-checked')).toBe('false');

    await fireEvent.click(toggle);

    expect(toggle.getAttribute('aria-checked')).toBe('true');
    expect(changeEvent).not.toBeNull();
    expect(changeEvent.key).toBe('enable_feature_x');
    expect(changeEvent.value).toBe(true);
    expect(changeEvent.values.enable_feature_x).toBe(true);
  });

  it('handles SelectControl interaction and emits change event', async () => {
    let changeEvent = null;
    render(PropertySheet, {
      props: {
        schema: sampleSchema,
        values: { audio_backend: 'pipewire' },
        onChange: (detail) => {
          changeEvent = detail;
        }
      }
    });

    const select = screen.getByRole('combobox', { name: /Audio Backend/i });
    await fireEvent.change(select, { target: { value: 'pulseaudio' } });

    expect(changeEvent).not.toBeNull();
    expect(changeEvent.key).toBe('audio_backend');
    expect(changeEvent.value).toBe('pulseaudio');
    expect(changeEvent.values.audio_backend).toBe('pulseaudio');
  });

  it('handles PathPickerControl native directory selection and list modifications', async () => {
    vi.spyOn(AppAPI, 'selectDirectory').mockResolvedValue('/selected/folder');

    let changeEvent = null;
    render(PropertySheet, {
      props: {
        schema: sampleSchema,
        values: {
          install_dir: '/default/dir',
          custom_library_folders: ['/existing/folder']
        },
        onChange: (detail) => {
          changeEvent = detail;
        }
      }
    });

    // Single path browse button
    const singlePathSection = screen.getByTestId('control-path-install_dir');
    const browseBtn = singlePathSection.querySelector('button');
    await fireEvent.click(browseBtn);

    expect(AppAPI.selectDirectory).toHaveBeenCalledWith('Installation Directory', '/default/dir');
    expect(changeEvent.key).toBe('install_dir');
    expect(changeEvent.value).toBe('/selected/folder');

    // List add folder
    const listSection = screen.getByTestId('control-path-custom_library_folders');
    const addFolderBtn = listSection.querySelector('button[aria-label="Add Folder"]');
    await fireEvent.click(addFolderBtn);

    expect(changeEvent.key).toBe('custom_library_folders');
    expect(changeEvent.value).toEqual(['/existing/folder', '/selected/folder']);

    // List remove folder
    const removeBtn = screen.getByLabelText('Remove path /existing/folder');
    await fireEvent.click(removeBtn);

    expect(changeEvent.key).toBe('custom_library_folders');
    expect(changeEvent.value).toEqual(['/selected/folder']);
  });

  it('handles EnvEditorControl adding and removing key-value pairs', async () => {
    let changeEvent = null;
    render(PropertySheet, {
      props: {
        schema: sampleSchema,
        values: {
          custom_env_vars: { INITIAL_VAR: '123' }
        },
        onChange: (detail) => {
          changeEvent = detail;
        }
      }
    });

    const envSection = screen.getByTestId('control-env-custom_env_vars');
    const keyInput = envSection.querySelector('input[aria-label="Variable Name"]');
    const valInput = envSection.querySelector('input[aria-label="Variable Value"]');
    const addBtn = envSection.querySelector('button[aria-label="Add Variable"]');

    await fireEvent.input(keyInput, { target: { value: 'NEW_VAR' } });
    await fireEvent.input(valInput, { target: { value: 'abc' } });
    await fireEvent.click(addBtn);

    expect(changeEvent.key).toBe('custom_env_vars');
    expect(changeEvent.value).toEqual({ INITIAL_VAR: '123', NEW_VAR: 'abc' });
    expect(screen.getByText('NEW_VAR')).toBeInTheDocument();
    expect(screen.getByText('abc')).toBeInTheDocument();

    // Remove pair
    const removeBtn = screen.getByLabelText('Remove variable INITIAL_VAR');
    await fireEvent.click(removeBtn);

    expect(changeEvent.key).toBe('custom_env_vars');
    expect(changeEvent.value).toEqual({ NEW_VAR: 'abc' });
  });

  it('updates nested dot properties like multiseat.player2_account_name', async () => {
    let changeEvent = null;
    render(PropertySheet, {
      props: {
        schema: sampleSchema,
        values: {
          multiseat: { player2_account_name: 'PlayerTwo' }
        },
        onChange: (detail) => {
          changeEvent = detail;
        }
      }
    });

    const textInput = screen.getByLabelText(/Player 2 Name/i);
    await fireEvent.input(textInput, { target: { value: 'CoopPartner' } });

    expect(changeEvent.key).toBe('multiseat.player2_account_name');
    expect(changeEvent.value).toBe('CoopPartner');
    expect(changeEvent.values.multiseat.player2_account_name).toBe('CoopPartner');
  });

  it('filters by category when category prop is specified', () => {
    render(PropertySheet, {
      props: {
        schema: sampleSchema,
        values: {},
        category: 'Audio'
      }
    });

    expect(screen.getByText('Audio Backend')).toBeInTheDocument();
    expect(screen.queryByText('Feature X Toggle')).not.toBeInTheDocument();
    expect(screen.queryByText('Installation Directory')).not.toBeInTheDocument();
  });

  it('fetches schema from AppAPI.getConfigSchema() when schema prop is not provided', async () => {
    const mockApiSchema = [
      {
        key: 'auto_schema_prop',
        label: 'Auto Schema Prop',
        description: 'Loaded automatically from backend.',
        type: 'boolean',
        category: 'Auto',
        scope: 'global',
        default: true
      }
    ];
    vi.spyOn(AppAPI, 'getConfigSchema').mockResolvedValue(mockApiSchema);

    render(PropertySheet, {
      props: {
        values: {}
      }
    });

    expect(await screen.findByText('Auto Schema Prop')).toBeInTheDocument();
    expect(AppAPI.getConfigSchema).toHaveBeenCalled();
  });

  it('fetches game schema from AppAPI.getGameConfigSchema() when scope="game" and schema prop is not provided', async () => {
    const mockGameSchema = [
      {
        key: 'runner',
        label: 'Game Runner',
        description: 'Selected runner.',
        type: 'select',
        category: 'Runtime',
        scope: 'game',
        default: 'umu',
        options: [{ value: 'umu', label: 'UMU' }]
      }
    ];
    vi.spyOn(AppAPI, 'getGameConfigSchema').mockResolvedValue(mockGameSchema);

    render(PropertySheet, {
      props: {
        scope: 'game',
        values: {}
      }
    });

    expect(await screen.findByText('Game Runner')).toBeInTheDocument();
    expect(AppAPI.getGameConfigSchema).toHaveBeenCalled();
  });

  it('filters schema by scope prop when provided', () => {
    const mixedSchema = [
      { key: 'prop_global', label: 'Global Prop', type: 'boolean', category: 'General', scope: 'global' },
      { key: 'prop_game', label: 'Game Prop', type: 'boolean', category: 'General', scope: 'game' }
    ];

    render(PropertySheet, {
      props: {
        schema: mixedSchema,
        scope: 'game',
        values: {}
      }
    });

    expect(screen.getByText('Game Prop')).toBeInTheDocument();
    expect(screen.queryByText('Global Prop')).not.toBeInTheDocument();
  });

  it('renders inherited global value when not overridden in game scope', () => {
    const schema = [
      {
        key: 'runner',
        label: 'Execution Runner',
        type: 'select',
        category: 'Runtime',
        scope: 'game',
        inherits_from: 'default_runner',
        default: 'umu',
        options: [
          { value: 'umu', label: 'UMU' },
          { value: 'wine', label: 'Wine' }
        ]
      }
    ];

    render(PropertySheet, {
      props: {
        schema,
        scope: 'game',
        values: {},
        inheritedValues: { default_runner: 'wine' }
      }
    });

    expect(screen.getByTestId('property-pill-inherited-runner')).toBeInTheDocument();
    expect(screen.getByText(/Inherited: Wine/i)).toBeInTheDocument();
    expect(screen.queryByTestId('property-pill-overridden-runner')).not.toBeInTheDocument();
  });

  it('renders overridden pill and triggers reset button in game scope', async () => {
    let resetDetail = null;
    const schema = [
      {
        key: 'mangohud',
        label: 'MangoHud Overlay',
        type: 'boolean',
        category: 'Graphics',
        scope: 'game',
        default: false
      }
    ];

    render(PropertySheet, {
      props: {
        schema,
        scope: 'game',
        values: { mangohud: true },
        inheritedValues: { mangohud: false },
        onReset: (detail) => {
          resetDetail = detail;
        }
      }
    });

    expect(screen.getByTestId('property-pill-overridden-mangohud')).toBeInTheDocument();
    const resetBtn = screen.getByTestId('property-reset-btn-mangohud');
    expect(resetBtn).toBeInTheDocument();

    await fireEvent.click(resetBtn);
    expect(resetDetail).not.toBeNull();
    expect(resetDetail.key).toBe('mangohud');
  });

  it('supports camelCase object keys like drawerDraft for game scope properties', async () => {
    let changedDetail = null;
    const schema = [
      {
        key: 'wine_prefix',
        label: 'Wine Prefix',
        type: 'path',
        category: 'Runtime',
        scope: 'game',
        default: ''
      }
    ];

    render(PropertySheet, {
      props: {
        schema,
        scope: 'game',
        values: { winePrefix: '/pfx/custom' },
        inheritedValues: { default_runner: 'umu' },
        onChange: (detail) => {
          changedDetail = detail;
        }
      }
    });

    expect(screen.getByTestId('property-pill-overridden-wine_prefix')).toBeInTheDocument();
    const container = screen.getByTestId('control-path-wine_prefix');
    const input = container.querySelector('input');
    expect(input.value).toBe('/pfx/custom');
  });
});


