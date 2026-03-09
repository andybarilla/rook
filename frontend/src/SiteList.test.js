import { describe, it, expect, vi } from 'vitest';
import { render, fireEvent } from '@testing-library/svelte';
import SiteList from './SiteList.svelte';

const fakeSites = [
  { domain: 'app.test', path: '/home/user/app', php_version: '8.3', node_version: '20', tls: true },
  { domain: 'blog.test', path: '/home/user/blog', php_version: '', node_version: '', tls: false },
];

const mockSites = [
  { domain: 'app.test', path: '/home/user/app', php_version: '8.3', node_version: '', tls: true },
  { domain: 'api.test', path: '/home/user/api', php_version: '', node_version: '20', tls: false },
];

describe('SiteList', () => {
  it('clicking Remove opens confirmation modal instead of removing immediately', async () => {
    const onRemove = vi.fn();
    const { getByText, getAllByText } = render(SiteList, {
      props: { sites: fakeSites, loaded: true, onRemove },
    });
    const removeButtons = getAllByText('Remove');
    await fireEvent.click(removeButtons[0]);
    // onRemove should NOT have been called yet
    expect(onRemove).not.toHaveBeenCalled();
    // Confirmation modal should be visible with domain name
    expect(getByText('Remove Site')).toBeTruthy();
    expect(getByText(/Are you sure you want to remove "app\.test"/)).toBeTruthy();
  });

  it('confirming the modal calls onRemove with the domain', async () => {
    const onRemove = vi.fn().mockResolvedValue(undefined);
    const { getAllByText, getByText } = render(SiteList, {
      props: { sites: fakeSites, loaded: true, onRemove },
    });
    await fireEvent.click(getAllByText('Remove')[0]);
    // Click the confirm button in the modal
    const confirmBtn = getByText('Yes, Remove');
    await fireEvent.click(confirmBtn);
    expect(onRemove).toHaveBeenCalledWith('app.test');
  });

  it('cancelling the modal does not call onRemove', async () => {
    const onRemove = vi.fn();
    const { getAllByText, getByText } = render(SiteList, {
      props: { sites: fakeSites, loaded: true, onRemove },
    });
    await fireEvent.click(getAllByText('Remove')[0]);
    await fireEvent.click(getByText('Cancel'));
    expect(onRemove).not.toHaveBeenCalled();
  });

  it('returns focus to Remove button after modal cancel', async () => {
    const { getAllByText, getByText } = render(SiteList, {
      props: { sites: fakeSites, loaded: true, onRemove: vi.fn() },
    });
    const removeBtn = getAllByText('Remove')[0];
    removeBtn.focus();
    await fireEvent.click(removeBtn);
    // Modal is open, cancel it
    await fireEvent.click(getByText('Cancel'));
    await vi.waitFor(() => {
      expect(document.activeElement).toBe(removeBtn);
    });
  });

  it('shows skeleton when not loaded', () => {
    const { container } = render(SiteList, {
      props: { sites: [], loaded: false, onRemove: vi.fn() },
    });
    expect(container.querySelectorAll('.skeleton').length).toBeGreaterThan(0);
  });

  it('shows empty state with icon and action when no sites', () => {
    const { getByText, container } = render(SiteList, {
      props: { sites: [], loaded: true, onRemove: vi.fn() },
    });
    expect(getByText('No sites yet')).toBeTruthy();
    expect(getByText('Add your first site to start developing locally.')).toBeTruthy();
    expect(getByText('Add Site')).toBeTruthy();
    expect(container.querySelector('[data-testid="empty-icon"]')).toBeTruthy();
  });

  it('shows Node column header', () => {
    const { container } = render(SiteList, {
      props: { sites: fakeSites, loaded: true, onRemove: vi.fn() },
    });
    const headers = container.querySelectorAll('th');
    const headerTexts = Array.from(headers).map((h) => h.textContent);
    expect(headerTexts).toContain('Node');
  });

  it('displays node_version or dash for each site', () => {
    const { container } = render(SiteList, {
      props: { sites: fakeSites, loaded: true, onRemove: vi.fn() },
    });
    const rows = container.querySelectorAll('tbody tr');
    // First site has node_version '20'
    expect(rows[0].textContent).toContain('20');
    // Second site has empty node_version, should show dash
    expect(rows[1].cells[3].textContent).toBe('—');
  });

  describe('view toggle', () => {
    it('renders view toggle buttons', () => {
      const { getByTitle } = render(SiteList, { props: { sites: mockSites, onRemove: vi.fn() } });
      expect(getByTitle('Table view')).toBeTruthy();
      expect(getByTitle('Card view')).toBeTruthy();
    });

    it('shows table view by default', () => {
      const { container } = render(SiteList, { props: { sites: mockSites, onRemove: vi.fn() } });
      expect(container.querySelector('table')).toBeTruthy();
    });

    it('switches to card view on click', async () => {
      const { getByTitle, container } = render(SiteList, { props: { sites: mockSites, onRemove: vi.fn() } });
      await fireEvent.click(getByTitle('Card view'));
      expect(container.querySelector('table')).toBeNull();
      expect(container.querySelector('[data-testid="site-cards"]')).toBeTruthy();
    });
  });

  describe('edit button', () => {
    beforeEach(() => {
      localStorage.removeItem('rook-view');
    });

    it('renders Edit button in table view for each site', () => {
      const { getAllByTitle } = render(SiteList, {
        props: { sites: fakeSites, loaded: true, onRemove: vi.fn() },
      });
      const editButtons = getAllByTitle('Edit site');
      expect(editButtons.length).toBe(2);
    });

    it('dispatches editsite event with site data when Edit is clicked', async () => {
      const { getAllByTitle, component } = render(SiteList, {
        props: { sites: fakeSites, loaded: true, onRemove: vi.fn() },
      });
      const editSpy = vi.fn();
      component.$on('editsite', editSpy);
      await fireEvent.click(getAllByTitle('Edit site')[0]);
      expect(editSpy).toHaveBeenCalled();
      expect(editSpy.mock.calls[0][0].detail).toEqual(fakeSites[0]);
    });
  });

  describe('search', () => {
    it('renders search input', () => {
      const { getByPlaceholderText } = render(SiteList, { props: { sites: mockSites, onRemove: vi.fn() } });
      expect(getByPlaceholderText(/search/i)).toBeTruthy();
    });

    it('filters sites by domain', async () => {
      const sites = [
        { domain: 'foo.test', path: '/foo', php_version: '', node_version: '', tls: false },
        { domain: 'bar.test', path: '/bar', php_version: '', node_version: '', tls: false },
      ];
      const { getByPlaceholderText, queryByText } = render(SiteList, { props: { sites, onRemove: vi.fn() } });
      await fireEvent.input(getByPlaceholderText(/search/i), { target: { value: 'foo' } });
      expect(queryByText('foo.test')).toBeTruthy();
      expect(queryByText('bar.test')).toBeNull();
    });
  });
});
