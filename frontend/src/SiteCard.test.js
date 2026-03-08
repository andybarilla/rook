import { describe, it, expect, vi } from 'vitest';
import { render, fireEvent } from '@testing-library/svelte';
import SiteCard from './SiteCard.svelte';

const mockSite = {
  domain: 'myapp.test',
  path: '/home/user/projects/myapp',
  php_version: '8.3',
  node_version: '20',
  tls: true,
};

describe('SiteCard', () => {
  it('renders site domain', () => {
    const { getByText } = render(SiteCard, { props: { site: mockSite, onRemove: vi.fn() } });
    expect(getByText('myapp.test')).toBeTruthy();
  });

  it('renders site path', () => {
    const { container } = render(SiteCard, { props: { site: mockSite, onRemove: vi.fn() } });
    expect(container.textContent).toContain('/home/user/projects/myapp');
  });

  it('renders PHP version badge', () => {
    const { getByText } = render(SiteCard, { props: { site: mockSite, onRemove: vi.fn() } });
    expect(getByText('PHP 8.3')).toBeTruthy();
  });

  it('renders Node version badge', () => {
    const { getByText } = render(SiteCard, { props: { site: mockSite, onRemove: vi.fn() } });
    expect(getByText('Node 20')).toBeTruthy();
  });

  it('renders TLS badge when enabled', () => {
    const { getByText } = render(SiteCard, { props: { site: mockSite, onRemove: vi.fn() } });
    expect(getByText('TLS')).toBeTruthy();
  });

  it('does not render TLS badge when disabled', () => {
    const site = { ...mockSite, tls: false };
    const { queryByText } = render(SiteCard, { props: { site, onRemove: vi.fn() } });
    expect(queryByText('TLS')).toBeNull();
  });

  it('calls onRemove with domain on remove button click', async () => {
    const onRemove = vi.fn();
    const { getByTitle } = render(SiteCard, { props: { site: mockSite, onRemove } });
    await fireEvent.click(getByTitle('Remove site'));
    expect(onRemove).toHaveBeenCalledWith('myapp.test');
  });

  it('handles missing php_version gracefully', () => {
    const site = { ...mockSite, php_version: '' };
    const { queryByText } = render(SiteCard, { props: { site, onRemove: vi.fn() } });
    expect(queryByText(/PHP/)).toBeNull();
  });

  it('handles missing node_version gracefully', () => {
    const site = { ...mockSite, node_version: '' };
    const { queryByText } = render(SiteCard, { props: { site, onRemove: vi.fn() } });
    expect(queryByText(/Node/)).toBeNull();
  });

  describe('runtime warnings', () => {
    it('shows warning badge when runtime is not installed', () => {
      const { getByText } = render(SiteCard, {
        props: {
          site: { domain: 'app.test', path: '/tmp', php_version: '8.3', tls: false },
          runtimeStatuses: [
            { tool: 'php', version: '8.3', installed: false, domain: 'app.test' },
          ],
          miseAvailable: true,
        },
      });
      expect(getByText('PHP 8.3 not found')).toBeTruthy();
    });

    it('shows Install button when mise is available', () => {
      const { getByRole } = render(SiteCard, {
        props: {
          site: { domain: 'app.test', path: '/tmp', php_version: '8.3', tls: false },
          runtimeStatuses: [
            { tool: 'php', version: '8.3', installed: false, domain: 'app.test' },
          ],
          miseAvailable: true,
        },
      });
      expect(getByRole('button', { name: /install/i })).toBeTruthy();
    });

    it('hides Install button when mise is unavailable', () => {
      const { queryByRole } = render(SiteCard, {
        props: {
          site: { domain: 'app.test', path: '/tmp', php_version: '8.3', tls: false },
          runtimeStatuses: [
            { tool: 'php', version: '8.3', installed: false, domain: 'app.test' },
          ],
          miseAvailable: false,
        },
      });
      expect(queryByRole('button', { name: /install/i })).toBeNull();
    });

    it('shows no warning when runtime is installed', () => {
      const { queryByText } = render(SiteCard, {
        props: {
          site: { domain: 'app.test', path: '/tmp', php_version: '8.3', tls: false },
          runtimeStatuses: [
            { tool: 'php', version: '8.3', installed: true, domain: 'app.test' },
          ],
          miseAvailable: true,
        },
      });
      expect(queryByText(/not found/)).toBeNull();
    });
  });
});
