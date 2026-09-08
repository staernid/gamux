import { render, screen, fireEvent } from '@testing-library/svelte';
import { describe, it, expect, vi } from 'vitest';
import SwitchControl from './SwitchControl.svelte';
import SelectControl from './SelectControl.svelte';
import PathPickerControl from './PathPickerControl.svelte';
import EnvEditorControl from './EnvEditorControl.svelte';
import { AppAPI } from '../../../api/app.js';

describe('Polymorphic Controls', () => {
  describe('SwitchControl', () => {
    it('renders label, description, and accessibility attributes', () => {
      render(SwitchControl, {
        props: {
          property: {
            key: 'mangohud',
            label: 'MangoHud Overlay',
            description: 'Show FPS in game'
          },
          value: false
        }
      });

      expect(screen.getByText('MangoHud Overlay')).toBeInTheDocument();
      expect(screen.getByText('Show FPS in game')).toBeInTheDocument();
      const toggle = screen.getByRole('switch');
      expect(toggle.getAttribute('aria-checked')).toBe('false');
    });

    it('toggles value on click and keyboard events', async () => {
      let changedVal = null;
      render(SwitchControl, {
        props: {
          property: { key: 'steamless', label: 'Unpack DRM' },
          value: true,
          onChange: (val) => {
            changedVal = val;
          }
        }
      });

      const toggle = screen.getByRole('switch');
      expect(toggle.getAttribute('aria-checked')).toBe('true');

      await fireEvent.click(toggle);
      expect(changedVal).toBe(false);

      await fireEvent.keyDown(toggle, { key: ' ' });
      expect(changedVal).toBe(true);
    });
  });

  describe('SelectControl', () => {
    it('renders dropdown with options and handles selection change', async () => {
      let selectedVal = null;
      render(SelectControl, {
        props: {
          property: {
            key: 'operating_mode',
            label: 'Operating Mode',
            description: 'Choose mode',
            options: [
              { value: 'loader', label: 'Loader Mode' },
              { value: 'portable', label: 'Portable Mode' }
            ]
          },
          value: 'loader',
          onChange: (val) => {
            selectedVal = val;
          }
        }
      });

      const select = screen.getByRole('combobox');
      expect(select.value).toBe('loader');

      await fireEvent.change(select, { target: { value: 'portable' } });
      expect(selectedVal).toBe('portable');
    });
  });

  describe('PathPickerControl', () => {
    it('renders single path input and triggers selectDirectory dialog', async () => {
      vi.spyOn(AppAPI, 'selectDirectory').mockResolvedValue('/custom/wine/prefix');

      let pickedPath = null;
      render(PathPickerControl, {
        props: {
          property: {
            key: 'wine_prefix',
            label: 'Wine Prefix Path',
            type: 'path'
          },
          value: '/default/prefix',
          onChange: (val) => {
            pickedPath = val;
          }
        }
      });

      const browseBtn = screen.getByRole('button', { name: /Browse/i });
      await fireEvent.click(browseBtn);

      expect(AppAPI.selectDirectory).toHaveBeenCalledWith('Wine Prefix Path', '/default/prefix');
      expect(pickedPath).toBe('/custom/wine/prefix');
    });

    it('renders path list and handles adding and removing paths', async () => {
      vi.spyOn(AppAPI, 'selectDirectory').mockResolvedValue('/new/library');

      let currentList = null;
      render(PathPickerControl, {
        props: {
          property: {
            key: 'library_paths',
            label: 'Library Folders',
            type: 'path_list'
          },
          value: ['/lib1', '/lib2'],
          onChange: (val) => {
            currentList = val;
          }
        }
      });

      expect(screen.getByText('/lib1')).toBeInTheDocument();
      expect(screen.getByText('/lib2')).toBeInTheDocument();

      const addBtn = screen.getByRole('button', { name: /Add Folder/i });
      await fireEvent.click(addBtn);

      expect(currentList).toEqual(['/lib1', '/lib2', '/new/library']);

      const removeBtn = screen.getByLabelText('Remove path /lib1');
      await fireEvent.click(removeBtn);

      expect(currentList).toEqual(['/lib2', '/new/library']);
    });
  });

  describe('EnvEditorControl', () => {
    it('manages key-value environment variables', async () => {
      let updatedEnv = null;
      render(EnvEditorControl, {
        props: {
          property: {
            key: 'global_env',
            label: 'Global Env'
          },
          value: { PROTON_ENABLE_NVAPI: '1' },
          onChange: (val) => {
            updatedEnv = val;
          }
        }
      });

      expect(screen.getByText('PROTON_ENABLE_NVAPI')).toBeInTheDocument();

      const keyInput = screen.getByPlaceholderText('VARIABLE_NAME');
      const valInput = screen.getByPlaceholderText('value');
      const addBtn = screen.getByRole('button', { name: /Add Variable/i });

      await fireEvent.input(keyInput, { target: { value: 'DXVK_ASYNC' } });
      await fireEvent.input(valInput, { target: { value: '1' } });
      await fireEvent.click(addBtn);

      expect(updatedEnv).toEqual({
        PROTON_ENABLE_NVAPI: '1',
        DXVK_ASYNC: '1'
      });
    });
  });
});
