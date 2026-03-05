import { describe, it, expect, vi } from 'vitest';
import { render, fireEvent } from '@testing-library/svelte';
import EmptyState from './EmptyState.svelte';

describe('EmptyState', () => {
  it('renders the message', () => {
    const { getByText } = render(EmptyState, {
      props: { message: 'No items yet' },
    });
    expect(getByText('No items yet')).toBeTruthy();
  });

  it('renders the subtitle when provided', () => {
    const { getByText } = render(EmptyState, {
      props: { message: 'No items yet', subtitle: 'Add one to get started.' },
    });
    expect(getByText('Add one to get started.')).toBeTruthy();
  });

  it('does not render subtitle element when not provided', () => {
    const { container } = render(EmptyState, {
      props: { message: 'No items yet' },
    });
    expect(container.querySelector('[data-testid="empty-subtitle"]')).toBeNull();
  });

  it('renders action button when actionLabel is provided', () => {
    const { getByText } = render(EmptyState, {
      props: { message: 'No items', actionLabel: 'Add Item' },
    });
    expect(getByText('Add Item')).toBeTruthy();
  });

  it('does not render action button when actionLabel is not provided', () => {
    const { container } = render(EmptyState, {
      props: { message: 'No items' },
    });
    expect(container.querySelector('button')).toBeNull();
  });

  it('renders the icon slot content', () => {
    const { container } = render(EmptyState, {
      props: { message: 'No items', icon: '🌐' },
    });
    expect(container.querySelector('[data-testid="empty-icon"]')).toBeTruthy();
  });
});
