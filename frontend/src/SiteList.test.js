import { describe, it, expect, vi } from 'vitest';
import { render, fireEvent } from '@testing-library/svelte';
import SiteList from './SiteList.svelte';

const fakeSites = [
  { domain: 'app.test', path: '/home/user/app', php_version: '8.3', node_version: '20', tls: true },
  { domain: 'blog.test', path: '/home/user/blog', php_version: '', node_version: '', tls: false },
];

describe('SiteList', () => {
  it('shows skeleton when not loaded', () => {
    const { container } = render(SiteList, {
      props: { sites: [], loaded: false, onRemove: vi.fn() },
    });
    expect(container.querySelectorAll('.skeleton').length).toBeGreaterThan(0);
  });

  it('shows empty message when no sites', () => {
    const { getByText } = render(SiteList, {
      props: { sites: [], loaded: true, onRemove: vi.fn() },
    });
    expect(getByText(/No sites registered/)).toBeTruthy();
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
});
