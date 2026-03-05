import { describe, it, expect, vi } from 'vitest';
import { render, fireEvent } from '@testing-library/svelte';
import AddSiteForm from './AddSiteForm.svelte';

describe('AddSiteForm', () => {
  it('is collapsed by default', () => {
    const { container } = render(AddSiteForm);
    const collapse = container.querySelector('.collapse');
    expect(collapse).toBeTruthy();
    const checkbox = collapse.querySelector('input[type="checkbox"]');
    expect(checkbox.checked).toBe(false);
  });

  it('expands when the collapse title is clicked', async () => {
    const { container } = render(AddSiteForm);
    const checkbox = container.querySelector('.collapse input[type="checkbox"]');
    await fireEvent.click(checkbox);
    expect(checkbox.checked).toBe(true);
  });

  it('has a required section with Path and Domain fields', () => {
    const { container } = render(AddSiteForm);
    const requiredSection = container.querySelector('[data-section="required"]');
    expect(requiredSection).toBeTruthy();
    expect(requiredSection.textContent).toContain('Path');
    expect(requiredSection.textContent).toContain('Domain');
  });

  it('required fields use input-md for visual prominence', () => {
    const { container } = render(AddSiteForm);
    const requiredSection = container.querySelector('[data-section="required"]');
    const inputs = requiredSection.querySelectorAll('input[type="text"]');
    inputs.forEach((input) => {
      expect(input.classList.contains('input-md')).toBe(true);
    });
  });

  it('has a divider separating required and optional sections', () => {
    const { container } = render(AddSiteForm);
    const divider = container.querySelector('.divider');
    expect(divider).toBeTruthy();
    expect(divider.textContent).toContain('Options');
  });

  it('has an optional section with PHP, Node, TLS', () => {
    const { container } = render(AddSiteForm);
    const optionalSection = container.querySelector('[data-section="optional"]');
    expect(optionalSection).toBeTruthy();
    expect(optionalSection.textContent).toContain('PHP Version');
    expect(optionalSection.textContent).toContain('Node Version');
    expect(optionalSection.textContent).toContain('TLS');
  });

  it('optional fields use input-sm to de-emphasize', () => {
    const { container } = render(AddSiteForm);
    const optionalSection = container.querySelector('[data-section="optional"]');
    const inputs = optionalSection.querySelectorAll('input[type="text"]');
    inputs.forEach((input) => {
      expect(input.classList.contains('input-sm')).toBe(true);
    });
  });

  it('auto-collapses after successful submission', async () => {
    const onAdd = vi.fn().mockResolvedValue(undefined);
    const { container, getByPlaceholderText } = render(AddSiteForm, {
      props: { onAdd },
    });
    const checkbox = container.querySelector('.collapse input[type="checkbox"]');
    await fireEvent.click(checkbox);
    expect(checkbox.checked).toBe(true);
    const pathInput = getByPlaceholderText('/home/user/projects/myapp');
    const domainInput = getByPlaceholderText('myapp.test');
    await fireEvent.input(pathInput, { target: { value: '/tmp/app' } });
    await fireEvent.input(domainInput, { target: { value: 'app.test' } });
    const form = container.querySelector('form');
    await fireEvent.submit(form);
    await vi.waitFor(() => {
      expect(checkbox.checked).toBe(false);
    });
  });

  it('can be opened externally via collapseOpen prop', () => {
    const { container } = render(AddSiteForm, {
      props: { collapseOpen: true },
    });
    const checkbox = container.querySelector('.collapse input[type="checkbox"]');
    expect(checkbox.checked).toBe(true);
  });
});
