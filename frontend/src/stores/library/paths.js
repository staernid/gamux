import { writable } from 'svelte/store';
import { isWailsDesktop } from '../../api/app.js';

function getInitialLibraryPaths() {
  if (typeof localStorage !== 'undefined') {
    const stored = localStorage.getItem('gamux_library_paths');
    if (stored) {
      try {
        const parsed = JSON.parse(stored);
        if (Array.isArray(parsed) && parsed.length > 0) {
          return parsed.filter(Boolean);
        }
      } catch {}
    }
    const single = localStorage.getItem('gamux_library_path');
    if (single) return [single];
  }
  if (typeof window !== 'undefined' && !isWailsDesktop()) {
    return ['~/Games'];
  }
  return [];
}

export const libraryPaths = writable(getInitialLibraryPaths());
export const libraryPath = writable(getInitialLibraryPaths()[0] || '');

if (typeof localStorage !== 'undefined') {
  libraryPaths.subscribe(paths => {
    localStorage.setItem('gamux_library_paths', JSON.stringify(paths));
    if (paths.length > 0) {
      libraryPath.set(paths[0]);
      localStorage.setItem('gamux_library_path', paths[0]);
    } else {
      libraryPath.set('');
      localStorage.removeItem('gamux_library_path');
    }
  });
}

// Internal hook to trigger library rescan when paths change
let onPathsChangedHook = null;
export function setOnPathsChangedHook(cb) {
  onPathsChangedHook = cb;
}

export function addLibraryPath(newPath) {
  if (!newPath) return;
  libraryPaths.update(list => {
    if (list.includes(newPath)) return list;
    return [...list, newPath];
  });
  if (typeof onPathsChangedHook === 'function') {
    onPathsChangedHook();
  }
}

export function removeLibraryPath(pathToRemove) {
  libraryPaths.update(list => list.filter(p => p !== pathToRemove));
  if (typeof onPathsChangedHook === 'function') {
    onPathsChangedHook();
  }
}
