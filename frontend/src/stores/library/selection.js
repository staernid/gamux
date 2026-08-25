import { writable } from 'svelte/store';

export const selectedGame = writable(null);
export const activeView = writable('library');
export const showInspector = writable(false);
export const pathPopover = writable(false);
