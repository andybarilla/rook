import { describe, it, expect, vi } from 'vitest';
import { render, fireEvent } from '@testing-library/svelte';
import AddSiteForm from './AddSiteForm.svelte';

describe('AddSiteForm', () => {
  it('modal is not visible when open is false', () => {
    const { container } = render(AddSiteForm);
    const modal = container.querySelector('.modal');
    expect(modal).toBeNull();
  });

  it('modal is visible when open is true', () => {
    const { container } = render(AddSiteForm, {
      props: { open: true },
    });
    const modal = container.querySelector('.modal');
    expect(modal).toBeTruthy();
    expect(modal.classList.contains('modal-open')).toBe(true);
  });

  it('has a required section with Path and Domain fields', () => {
    const { container } = render(AddSiteForm, {
      props: { open: true },
    });
    const requiredSection = container.querySelector('[data-section="required"]');
    expect(requiredSection).toBeTruthy();
    expect(requiredSection.textContent).toContain('Path');
    expect(requiredSection.textContent).toContain('Domain');
  });

  it('required fields use input-md for visual prominence', () => {
    const { container } = render(AddSiteForm, {
      props: { open: true },
    });
    const requiredSection = container.querySelector('[data-section="required"]');
    const inputs = requiredSection.querySelectorAll('input[type="text"]');
    inputs.forEach((input) => {
      expect(input.classList.contains('input-md')).toBe(true);
    });
  });

  it('has a divider separating required and optional sections', () => {
    const { container } = render(AddSiteForm, {
      props: { open: true },
    });
    const divider = container.querySelector('.divider');
    expect(divider).toBeTruthy();
    expect(divider.textContent).toContain('Options');
  });

  it('has an optional section with PHP, Node, TLS', () => {
    const { container } = render(AddSiteForm, {
      props: { open: true },
    });
    const optionalSection = container.querySelector('[data-section="optional"]');
    expect(optionalSection).toBeTruthy();
    expect(optionalSection.textContent).toContain('PHP Version');
    expect(optionalSection.textContent).toContain('Node Version');
    expect(optionalSection.textContent).toContain('TLS');
  });

  it('optional fields use input-sm to de-emphasize', () => {
    const { container } = render(AddSiteForm, {
      props: { open: true },
    });
    const optionalSection = container.querySelector('[data-section="optional"]');
    const inputs = optionalSection.querySelectorAll('input[type="text"]');
    inputs.forEach((input) => {
      expect(input.classList.contains('input-sm')).toBe(true);
    });
  });

  it('dispatches close event after successful submission', async () => {
    const onAdd = vi.fn().mockResolvedValue(undefined);
    const { container, getByPlaceholderText, component } = render(AddSiteForm, {
      props: { onAdd, open: true },
    });
    const closeSpy = vi.fn();
    component.$on('close', closeSpy);
    const pathInput = getByPlaceholderText('/home/user/projects/myapp');
    const domainInput = getByPlaceholderText('myapp.test');
    await fireEvent.input(pathInput, { target: { value: '/tmp/app' } });
    await fireEvent.input(domainInput, { target: { value: 'app.test' } });
    const form = container.querySelector('form');
    await fireEvent.submit(form);
    await vi.waitFor(() => {
      expect(closeSpy).toHaveBeenCalled();
    });
  });

  it('Cancel button dispatches close event', async () => {
    const { getByText, component } = render(AddSiteForm, {
      props: { open: true },
    });
    const closeSpy = vi.fn();
    component.$on('close', closeSpy);
    const cancelBtn = getByText('Cancel');
    await fireEvent.click(cancelBtn);
    expect(closeSpy).toHaveBeenCalled();
  });

  it('exposes focusPathInput method that focuses the path input', async () => {
    const { container, component } = render(AddSiteForm, {
      props: { open: true },
    });
    component.focusPathInput();
    const pathInput = container.querySelector('input[placeholder="/home/user/projects/myapp"]');
    expect(document.activeElement).toBe(pathInput);
  });

  it('has proper aria attributes for accessibility', () => {
    const { container } = render(AddSiteForm, {
      props: { open: true },
    });
    const modal = container.querySelector('.modal');
    expect(modal.getAttribute('role')).toBe('dialog');
    expect(modal.getAttribute('aria-modal')).toBe('true');
    expect(modal.getAttribute('aria-labelledby')).toBe('add-site-title');
    const title = container.querySelector('#add-site-title');
    expect(title).toBeTruthy();
    expect(title.textContent).toBe('Add Site');
  });
});
