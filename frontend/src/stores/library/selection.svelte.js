export class SelectionState {
  selectedGame = $state(null);
  activeView = $state('library');
  showInspector = $state(false);
  pathPopover = $state(false);

  #selectedSubscribers = new Set();
  #activeViewSubscribers = new Set();
  #showInspectorSubscribers = new Set();
  #pathPopoverSubscribers = new Set();

  setSelectedGame(game) {
    this.selectedGame = game;
    this.#notifySelected();
  }

  setActiveView(view) {
    this.activeView = view;
    this.#notifyActiveView();
  }

  setShowInspector(show) {
    this.showInspector = Boolean(show);
    this.#notifyShowInspector();
  }

  setPathPopover(open) {
    this.pathPopover = Boolean(open);
    this.#notifyPathPopover();
  }

  #notifySelected() {
    for (const fn of this.#selectedSubscribers) {
      try { fn(this.selectedGame); } catch {}
    }
  }

  #notifyActiveView() {
    for (const fn of this.#activeViewSubscribers) {
      try { fn(this.activeView); } catch {}
    }
  }

  #notifyShowInspector() {
    for (const fn of this.#showInspectorSubscribers) {
      try { fn(this.showInspector); } catch {}
    }
  }

  #notifyPathPopover() {
    for (const fn of this.#pathPopoverSubscribers) {
      try { fn(this.pathPopover); } catch {}
    }
  }

  get _selectedGameBridge() {
    return {
      subscribe: (fn) => {
        this.#selectedSubscribers.add(fn);
        fn(this.selectedGame);
        return () => this.#selectedSubscribers.delete(fn);
      },
      set: (val) => {
        this.selectedGame = val;
        this.#notifySelected();
      },
      update: (fn) => {
        this.selectedGame = fn(this.selectedGame);
        this.#notifySelected();
      }
    };
  }

  get _activeViewBridge() {
    return {
      subscribe: (fn) => {
        this.#activeViewSubscribers.add(fn);
        fn(this.activeView);
        return () => this.#activeViewSubscribers.delete(fn);
      },
      set: (val) => {
        this.activeView = val;
        this.#notifyActiveView();
      },
      update: (fn) => {
        this.activeView = fn(this.activeView);
        this.#notifyActiveView();
      }
    };
  }

  get _showInspectorBridge() {
    return {
      subscribe: (fn) => {
        this.#showInspectorSubscribers.add(fn);
        fn(this.showInspector);
        return () => this.#showInspectorSubscribers.delete(fn);
      },
      set: (val) => {
        this.showInspector = val;
        this.#notifyShowInspector();
      },
      update: (fn) => {
        this.showInspector = fn(this.showInspector);
        this.#notifyShowInspector();
      }
    };
  }

  get _pathPopoverBridge() {
    return {
      subscribe: (fn) => {
        this.#pathPopoverSubscribers.add(fn);
        fn(this.pathPopover);
        return () => this.#pathPopoverSubscribers.delete(fn);
      },
      set: (val) => {
        this.pathPopover = val;
        this.#notifyPathPopover();
      },
      update: (fn) => {
        this.pathPopover = fn(this.pathPopover);
        this.#notifyPathPopover();
      }
    };
  }
}

export const selectionStore = new SelectionState();
export const selectedGame = selectionStore._selectedGameBridge;
export const activeView = selectionStore._activeViewBridge;
export const showInspector = selectionStore._showInspectorBridge;
export const pathPopover = selectionStore._pathPopoverBridge;
