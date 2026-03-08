import { describe, it, expect, beforeEach, vi } from 'vitest';
import { get } from 'svelte/store';
import { theme, toggleTheme, initTheme } from './theme.js';

describe('theme store', () => {
  beforeEach(() => {
    localStorage.clear();
    document.documentElement.setAttribute('data-theme', '');
    theme.set('light');
  });

  it('defaults to light', () => {
    expect(get(theme)).toBe('light');
  });

  it('toggleTheme switches light to dark', () => {
    toggleTheme();
    expect(get(theme)).toBe('dark');
  });

  it('toggleTheme switches dark to light', () => {
    theme.set('dark');
    toggleTheme();
    expect(get(theme)).toBe('light');
  });

  it('toggleTheme persists to localStorage', () => {
    toggleTheme();
    expect(localStorage.getItem('flock-theme')).toBe('dark');
  });

  it('toggleTheme sets data-theme attribute on html', () => {
    toggleTheme();
    expect(document.documentElement.getAttribute('data-theme')).toBe('dark');
  });

  it('initTheme reads from localStorage', () => {
    localStorage.setItem('flock-theme', 'dark');
    initTheme();
    expect(get(theme)).toBe('dark');
    expect(document.documentElement.getAttribute('data-theme')).toBe('dark');
  });

  it('initTheme defaults to light when no stored value', () => {
    initTheme();
    expect(get(theme)).toBe('light');
    expect(document.documentElement.getAttribute('data-theme')).toBe('light');
  });
});
