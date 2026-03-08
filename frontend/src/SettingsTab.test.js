import { describe, it, expect, beforeEach } from 'vitest';
import { render, fireEvent } from '@testing-library/svelte';
import SettingsTab from './SettingsTab.svelte';
import { theme } from './lib/theme.js';
import { get } from 'svelte/store';

describe('SettingsTab', () => {
  beforeEach(() => {
    localStorage.clear();
    theme.set('light');
    document.documentElement.setAttribute('data-theme', 'light');
  });

  it('renders Appearance section', () => {
    const { getByText } = render(SettingsTab);
    expect(getByText('Appearance')).toBeTruthy();
  });

  it('renders theme toggle', () => {
    const { getByRole } = render(SettingsTab);
    expect(getByRole('checkbox', { name: /dark mode/i })).toBeTruthy();
  });

  it('toggles theme on checkbox change', async () => {
    const { getByRole } = render(SettingsTab);
    const toggle = getByRole('checkbox', { name: /dark mode/i });
    await fireEvent.click(toggle);
    expect(get(theme)).toBe('dark');
  });

  it('renders About section', () => {
    const { getByText } = render(SettingsTab);
    expect(getByText('About')).toBeTruthy();
  });

  it('shows app version', () => {
    const { getByText } = render(SettingsTab);
    expect(getByText(/version/i)).toBeTruthy();
  });
});
