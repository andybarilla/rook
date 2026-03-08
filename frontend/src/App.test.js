import { describe, it, expect, vi } from 'vitest';
import { render, fireEvent } from '@testing-library/svelte';

vi.mock('../wailsjs/go/main/App.js', () => ({
  ListSites: vi.fn().mockResolvedValue([]),
  AddSite: vi.fn().mockResolvedValue(undefined),
  RemoveSite: vi.fn().mockResolvedValue(undefined),
  DatabaseServices: vi.fn().mockResolvedValue([]),
  StartDatabase: vi.fn().mockResolvedValue(undefined),
  StopDatabase: vi.fn().mockResolvedValue(undefined),
}));

vi.mock('./lib/theme.js', () => {
  const { writable } = require('svelte/store');
  return { initTheme: vi.fn(), toggleTheme: vi.fn(), theme: writable('light') };
});

import App from './App.svelte';

describe('App keyboard shortcuts', () => {
  it('Ctrl+N opens add site form and focuses path input', async () => {
    vi.useFakeTimers();
    const { container } = render(App);
    // Modal should not be visible initially
    expect(container.querySelector('.modal')).toBeNull();
    await fireEvent.keyDown(window, { key: 'n', ctrlKey: true });
    vi.runAllTimers();
    await vi.waitFor(() => {
      const modal = container.querySelector('.modal');
      expect(modal).toBeTruthy();
    });
    vi.useRealTimers();
  });

  it('Escape closes add site form when open', async () => {
    vi.useFakeTimers();
    const { container } = render(App);
    // Open the form first
    await fireEvent.keyDown(window, { key: 'n', ctrlKey: true });
    vi.runAllTimers();
    await vi.waitFor(() => {
      expect(container.querySelector('.modal')).toBeTruthy();
    });
    // Press Escape
    await fireEvent.keyDown(window, { key: 'Escape' });
    await vi.waitFor(() => {
      expect(container.querySelector('.modal')).toBeNull();
    });
    vi.useRealTimers();
  });

  it('Ctrl+Enter submits the add site form when open', async () => {
    vi.useFakeTimers();
    const { AddSite } = await import('../wailsjs/go/main/App.js');
    const { container } = render(App);
    // Open form
    await fireEvent.keyDown(window, { key: 'n', ctrlKey: true });
    vi.runAllTimers();
    await vi.waitFor(() => {
      expect(container.querySelector('.modal')).toBeTruthy();
    });
    // Fill in fields
    const pathInput = container.querySelector('input[placeholder="/home/user/projects/myapp"]');
    const domainInput = container.querySelector('input[placeholder="myapp.test"]');
    await fireEvent.input(pathInput, { target: { value: '/tmp/myapp' } });
    await fireEvent.input(domainInput, { target: { value: 'myapp.test' } });
    // Ctrl+Enter
    await fireEvent.keyDown(window, { key: 'Enter', ctrlKey: true });
    await vi.waitFor(() => {
      expect(AddSite).toHaveBeenCalledWith('/tmp/myapp', 'myapp.test', '', '', false);
    });
    vi.useRealTimers();
  });
});

describe('tab navigation', () => {
  it('renders three tabs: Sites, Services, Settings', async () => {
    const { ListSites, DatabaseServices } = await import('../wailsjs/go/main/App.js');
    ListSites.mockResolvedValue([]);
    DatabaseServices.mockResolvedValue([]);
    const { getByRole } = render(App);
    await vi.waitFor(() => {
      expect(getByRole('tab', { name: 'Sites' })).toBeTruthy();
      expect(getByRole('tab', { name: 'Services' })).toBeTruthy();
      expect(getByRole('tab', { name: 'Settings' })).toBeTruthy();
    });
  });

  it('shows Sites tab as active by default', async () => {
    const { ListSites, DatabaseServices } = await import('../wailsjs/go/main/App.js');
    ListSites.mockResolvedValue([]);
    DatabaseServices.mockResolvedValue([]);
    const { getByRole } = render(App);
    await vi.waitFor(() => {
      expect(getByRole('tab', { name: 'Sites' }).classList.contains('tab-active')).toBe(true);
    });
  });

  it('switches to Services tab on click', async () => {
    const { ListSites, DatabaseServices } = await import('../wailsjs/go/main/App.js');
    ListSites.mockResolvedValue([]);
    DatabaseServices.mockResolvedValue([]);
    const { getByRole } = render(App);
    await vi.waitFor(() => {
      expect(getByRole('tab', { name: 'Services' })).toBeTruthy();
    });
    await fireEvent.click(getByRole('tab', { name: 'Services' }));
    expect(getByRole('tab', { name: 'Services' }).classList.contains('tab-active')).toBe(true);
    expect(getByRole('tab', { name: 'Sites' }).classList.contains('tab-active')).toBe(false);
  });

  it('switches to Settings tab on click', async () => {
    const { ListSites, DatabaseServices } = await import('../wailsjs/go/main/App.js');
    ListSites.mockResolvedValue([]);
    DatabaseServices.mockResolvedValue([]);
    const { getByRole } = render(App);
    await vi.waitFor(() => {
      expect(getByRole('tab', { name: 'Settings' })).toBeTruthy();
    });
    await fireEvent.click(getByRole('tab', { name: 'Settings' }));
    expect(getByRole('tab', { name: 'Settings' }).classList.contains('tab-active')).toBe(true);
  });
});

describe('integration', () => {
  it('Ctrl+N opens Add Site modal', async () => {
    const { ListSites, DatabaseServices } = await import('../wailsjs/go/main/App.js');
    ListSites.mockResolvedValue([]);
    DatabaseServices.mockResolvedValue([]);
    vi.useFakeTimers();
    const { container } = render(App);
    await fireEvent.keyDown(window, { key: 'n', ctrlKey: true });
    vi.runAllTimers();
    await vi.waitFor(() => {
      expect(container.querySelector('.modal-open')).toBeTruthy();
    });
    vi.useRealTimers();
  });

  it('clicking Services tab shows service content', async () => {
    const { ListSites, DatabaseServices } = await import('../wailsjs/go/main/App.js');
    ListSites.mockResolvedValue([]);
    DatabaseServices.mockResolvedValue([
      { type: 'mysql', enabled: true, running: true, autostart: true, port: 3306 },
    ]);
    const { getByRole, getByText } = render(App);
    await vi.waitFor(() => {
      expect(getByRole('tab', { name: 'Services' })).toBeTruthy();
    });
    await fireEvent.click(getByRole('tab', { name: 'Services' }));
    await vi.waitFor(() => {
      expect(getByText('MySQL')).toBeTruthy();
    });
  });

  it('clicking Settings tab shows theme toggle', async () => {
    const { ListSites, DatabaseServices } = await import('../wailsjs/go/main/App.js');
    ListSites.mockResolvedValue([]);
    DatabaseServices.mockResolvedValue([]);
    const { getByRole } = render(App);
    await vi.waitFor(() => {
      expect(getByRole('tab', { name: 'Settings' })).toBeTruthy();
    });
    await fireEvent.click(getByRole('tab', { name: 'Settings' }));
    await vi.waitFor(() => {
      expect(getByRole('checkbox', { name: /dark mode/i })).toBeTruthy();
    });
  });
});
