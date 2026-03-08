import { describe, it, expect, beforeEach, vi } from 'vitest';
import { render, fireEvent } from '@testing-library/svelte';
import SettingsTab from './SettingsTab.svelte';
import { theme } from './lib/theme.js';
import { get } from 'svelte/store';

vi.mock('../wailsjs/go/main/App.js', () => ({
  MiseStatus: vi.fn().mockResolvedValue({ available: false, version: '' }),
}));

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
    expect(getByText('1.0.0')).toBeTruthy();
  });

  describe('mise status', () => {
    it('shows detected badge when mise is available', async () => {
      const { MiseStatus } = await import('../wailsjs/go/main/App.js');
      MiseStatus.mockResolvedValue({ available: true, version: 'mise 2024.12.0' });

      const { getByText } = render(SettingsTab);
      await vi.waitFor(() => {
        expect(getByText('mise 2024.12.0')).toBeTruthy();
      });
    });

    it('shows green Detected badge when mise available', async () => {
      const { MiseStatus } = await import('../wailsjs/go/main/App.js');
      MiseStatus.mockResolvedValue({ available: true, version: 'mise 2024.12.0' });

      const { getByText } = render(SettingsTab);
      await vi.waitFor(() => {
        expect(getByText('Detected')).toBeTruthy();
      });
    });

    it('shows not-found message when mise is unavailable', async () => {
      const { MiseStatus } = await import('../wailsjs/go/main/App.js');
      MiseStatus.mockResolvedValue({ available: false, version: '' });

      const { getByText } = render(SettingsTab);
      await vi.waitFor(() => {
        expect(getByText(/Install mise/i)).toBeTruthy();
      });
    });

    it('shows link to mise.jdx.dev when unavailable', async () => {
      const { MiseStatus } = await import('../wailsjs/go/main/App.js');
      MiseStatus.mockResolvedValue({ available: false, version: '' });

      const { getByText } = render(SettingsTab);
      await vi.waitFor(() => {
        const link = getByText('mise.jdx.dev');
        expect(link.getAttribute('href')).toBe('https://mise.jdx.dev');
      });
    });
  });
});
