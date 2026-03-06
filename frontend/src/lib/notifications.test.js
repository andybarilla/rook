import { describe, it, expect, vi, beforeEach } from 'vitest';
import { notifications, notifySuccess, notifyError, notifyInfo, dismiss, dismissLatest } from './notifications.js';
import { get } from 'svelte/store';

beforeEach(() => {
  // Clear all notifications before each test
  notifications.set([]);
});

describe('notifications store', () => {
  it('starts empty', () => {
    expect(get(notifications)).toEqual([]);
  });

  it('notifySuccess adds a success notification', () => {
    notifySuccess('Site added');
    const items = get(notifications);
    expect(items).toHaveLength(1);
    expect(items[0].type).toBe('success');
    expect(items[0].message).toBe('Site added');
    expect(items[0].timeout).toBe(3000);
  });

  it('notifyError adds an error notification', () => {
    notifyError('Something failed');
    const items = get(notifications);
    expect(items).toHaveLength(1);
    expect(items[0].type).toBe('error');
    expect(items[0].message).toBe('Something failed');
    expect(items[0].timeout).toBe(8000);
  });

  it('notifyInfo adds an info notification', () => {
    notifyInfo('Heads up');
    const items = get(notifications);
    expect(items).toHaveLength(1);
    expect(items[0].type).toBe('info');
    expect(items[0].timeout).toBe(3000);
  });

  it('dismiss removes a notification by id', () => {
    notifySuccess('one');
    notifyError('two');
    const items = get(notifications);
    dismiss(items[0].id);
    expect(get(notifications)).toHaveLength(1);
    expect(get(notifications)[0].message).toBe('two');
  });

  it('each notification gets a unique id', () => {
    notifySuccess('a');
    notifySuccess('b');
    const items = get(notifications);
    expect(items[0].id).not.toBe(items[1].id);
  });

  it('dismissLatest removes the most recent notification', () => {
    notifySuccess('First');
    notifySuccess('Second');
    const result = dismissLatest();
    expect(result).toBe(true);
    const items = get(notifications);
    expect(items).toHaveLength(1);
    expect(items[0].message).toBe('First');
  });

  it('dismissLatest returns false when no notifications', () => {
    const result = dismissLatest();
    expect(result).toBe(false);
  });
});
