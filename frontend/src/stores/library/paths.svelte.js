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

export class PathsState {
  paths = $state([]);
  currentPath = $state('');

  #pathsSubscribers = new Set();
  #currentPathSubscribers = new Set();
  #onPathsChangedHook = null;

  constructor() {
    const initial = getInitialLibraryPaths();
    this.paths = initial;
    this.currentPath = initial[0] || '';
  }

  setOnPathsChangedHook(cb) {
    this.#onPathsChangedHook = cb;
  }

  #persist() {
    if (typeof localStorage !== 'undefined') {
      try {
        localStorage.setItem('gamux_library_paths', JSON.stringify(this.paths));
        if (this.paths.length > 0) {
          localStorage.setItem('gamux_library_path', this.paths[0]);
        } else {
          localStorage.removeItem('gamux_library_path');
        }
      } catch {}
    }
  }

  #notifyPaths() {
    for (const fn of this.#pathsSubscribers) {
      try { fn(this.paths); } catch {}
    }
  }

  #notifyCurrentPath() {
    for (const fn of this.#currentPathSubscribers) {
      try { fn(this.currentPath); } catch {}
    }
  }

  setPaths(newPaths) {
    this.paths = Array.isArray(newPaths) ? newPaths.filter(Boolean) : [];
    this.currentPath = this.paths[0] || '';
    this.#persist();
    this.#notifyPaths();
    this.#notifyCurrentPath();
  }

  addLibraryPath(newPath) {
    if (!newPath) return;
    if (this.paths.includes(newPath)) return;
    this.paths = [...this.paths, newPath];
    if (!this.currentPath) {
      this.currentPath = newPath;
      this.#notifyCurrentPath();
    }
    this.#persist();
    this.#notifyPaths();
    if (typeof this.#onPathsChangedHook === 'function') {
      this.#onPathsChangedHook();
    }
  }

  removeLibraryPath(pathToRemove) {
    this.paths = this.paths.filter(p => p !== pathToRemove);
    this.currentPath = this.paths[0] || '';
    this.#persist();
    this.#notifyPaths();
    this.#notifyCurrentPath();
    if (typeof this.#onPathsChangedHook === 'function') {
      this.#onPathsChangedHook();
    }
  }

  get _libraryPathsBridge() {
    return {
      subscribe: (fn) => {
        this.#pathsSubscribers.add(fn);
        fn(this.paths);
        return () => this.#pathsSubscribers.delete(fn);
      },
      set: (val) => {
        this.setPaths(val);
      },
      update: (fn) => {
        this.setPaths(fn(this.paths));
      }
    };
  }

  get _libraryPathBridge() {
    return {
      subscribe: (fn) => {
        this.#currentPathSubscribers.add(fn);
        fn(this.currentPath);
        return () => this.#currentPathSubscribers.delete(fn);
      },
      set: (val) => {
        this.currentPath = val || '';
        if (typeof localStorage !== 'undefined') {
          if (this.currentPath) {
            localStorage.setItem('gamux_library_path', this.currentPath);
          } else {
            localStorage.removeItem('gamux_library_path');
          }
        }
        this.#notifyCurrentPath();
      },
      update: (fn) => {
        this.currentPath = fn(this.currentPath) || '';
        this.#notifyCurrentPath();
      }
    };
  }
}

export const pathsStore = new PathsState();
export const libraryPaths = pathsStore._libraryPathsBridge;
export const libraryPath = pathsStore._libraryPathBridge;

export function setOnPathsChangedHook(cb) {
  pathsStore.setOnPathsChangedHook(cb);
}

export function addLibraryPath(newPath) {
  pathsStore.addLibraryPath(newPath);
}

export function removeLibraryPath(pathToRemove) {
  pathsStore.removeLibraryPath(pathToRemove);
}
