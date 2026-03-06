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

import App from './App.svelte';

describe('App keyboard shortcuts', () => {
  it('Ctrl+N opens add site form and focuses path input', async () => {
    vi.useFakeTimers();
    const { container } = render(App);
    await vi.waitFor(() => {
      expect(container.querySelector('.collapse')).toBeTruthy();
    });
    await fireEvent.keyDown(window, { key: 'n', ctrlKey: true });
    vi.runAllTimers();
    await vi.waitFor(() => {
      const checkbox = container.querySelector('.collapse input[type="checkbox"]');
      expect(checkbox.checked).toBe(true);
    });
    vi.useRealTimers();
  });

  it('Escape closes add site form when open', async () => {
    vi.useFakeTimers();
    const { container } = render(App);
    await vi.waitFor(() => {
      expect(container.querySelector('.collapse')).toBeTruthy();
    });
    // Open the form first
    await fireEvent.keyDown(window, { key: 'n', ctrlKey: true });
    vi.runAllTimers();
    await vi.waitFor(() => {
      const checkbox = container.querySelector('.collapse input[type="checkbox"]');
      expect(checkbox.checked).toBe(true);
    });
    // Press Escape
    await fireEvent.keyDown(window, { key: 'Escape' });
    await vi.waitFor(() => {
      const checkbox = container.querySelector('.collapse input[type="checkbox"]');
      expect(checkbox.checked).toBe(false);
    });
    vi.useRealTimers();
  });

  it('Ctrl+Enter submits the add site form when open', async () => {
    vi.useFakeTimers();
    const { AddSite } = await import('../wailsjs/go/main/App.js');
    const { container } = render(App);
    await vi.waitFor(() => {
      expect(container.querySelector('.collapse')).toBeTruthy();
    });
    // Open form
    await fireEvent.keyDown(window, { key: 'n', ctrlKey: true });
    vi.runAllTimers();
    await vi.waitFor(() => {
      const checkbox = container.querySelector('.collapse input[type="checkbox"]');
      expect(checkbox.checked).toBe(true);
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
