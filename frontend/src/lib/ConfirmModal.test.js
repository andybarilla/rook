import { describe, it, expect, vi } from 'vitest';
import { render, fireEvent } from '@testing-library/svelte';
import ConfirmModal from './ConfirmModal.svelte';

describe('ConfirmModal', () => {
  it('renders nothing when open is false', () => {
    const { container } = render(ConfirmModal, {
      props: { open: false, title: 'Delete?', message: 'Are you sure?', onConfirm: vi.fn(), onCancel: vi.fn() },
    });
    expect(container.querySelector('.modal')).toBeNull();
  });

  it('renders modal when open is true', () => {
    const { getByText } = render(ConfirmModal, {
      props: { open: true, title: 'Delete?', message: 'Are you sure?', onConfirm: vi.fn(), onCancel: vi.fn() },
    });
    expect(getByText('Delete?')).toBeTruthy();
    expect(getByText('Are you sure?')).toBeTruthy();
  });

  it('shows custom confirm label and class', () => {
    const { getByText } = render(ConfirmModal, {
      props: {
        open: true,
        title: 'Remove',
        message: 'Gone forever',
        confirmLabel: 'Yes, remove',
        confirmClass: 'btn-error',
        onConfirm: vi.fn(),
        onCancel: vi.fn(),
      },
    });
    const btn = getByText('Yes, remove');
    expect(btn.className).toContain('btn-error');
  });

  it('fires onConfirm when confirm button clicked', async () => {
    const onConfirm = vi.fn();
    const { getByText } = render(ConfirmModal, {
      props: { open: true, title: 'T', message: 'M', onConfirm, onCancel: vi.fn() },
    });
    await fireEvent.click(getByText('Confirm'));
    expect(onConfirm).toHaveBeenCalledOnce();
  });

  it('fires onCancel when cancel button clicked', async () => {
    const onCancel = vi.fn();
    const { getByText } = render(ConfirmModal, {
      props: { open: true, title: 'T', message: 'M', onConfirm: vi.fn(), onCancel },
    });
    await fireEvent.click(getByText('Cancel'));
    expect(onCancel).toHaveBeenCalledOnce();
  });

  it('fires onCancel when backdrop clicked', async () => {
    const onCancel = vi.fn();
    const { container } = render(ConfirmModal, {
      props: { open: true, title: 'T', message: 'M', onConfirm: vi.fn(), onCancel },
    });
    const backdrop = container.querySelector('.modal-backdrop');
    await fireEvent.click(backdrop);
    expect(onCancel).toHaveBeenCalledOnce();
  });

  it('fires onCancel when Escape key is pressed', async () => {
    const onCancel = vi.fn();
    render(ConfirmModal, {
      props: { open: true, title: 'T', message: 'M', onConfirm: vi.fn(), onCancel },
    });
    await fireEvent.keyDown(window, { key: 'Escape' });
    expect(onCancel).toHaveBeenCalledOnce();
  });

  it('defaults confirmLabel to "Confirm"', () => {
    const { getByText } = render(ConfirmModal, {
      props: { open: true, title: 'T', message: 'M', onConfirm: vi.fn(), onCancel: vi.fn() },
    });
    expect(getByText('Confirm')).toBeTruthy();
  });
});
