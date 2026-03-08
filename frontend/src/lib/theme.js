import { writable, get } from 'svelte/store';

export const theme = writable('light');

function applyTheme(value) {
  document.documentElement.setAttribute('data-theme', value);
  localStorage.setItem('rook-theme', value);
}

export function toggleTheme() {
  const next = get(theme) === 'light' ? 'dark' : 'light';
  theme.set(next);
  applyTheme(next);
}

export function initTheme() {
  const stored = localStorage.getItem('rook-theme') || 'light';
  theme.set(stored);
  applyTheme(stored);
}
