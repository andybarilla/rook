import { describe, it, expect, vi } from 'vitest';
import { render, fireEvent } from '@testing-library/svelte';
import ServiceCard from './ServiceCard.svelte';

describe('ServiceCard', () => {
  const mockService = { type: 'mysql', enabled: true, running: true, autostart: true, port: 3306 };

  it('renders service name', () => {
    const { getByText } = render(ServiceCard, { props: { service: mockService, onStart: vi.fn(), onStop: vi.fn() } });
    expect(getByText('MySQL')).toBeTruthy();
  });

  it('renders running badge when running', () => {
    const { getByText } = render(ServiceCard, { props: { service: mockService, onStart: vi.fn(), onStop: vi.fn() } });
    expect(getByText('Running')).toBeTruthy();
  });

  it('renders stopped badge when not running', () => {
    const svc = { ...mockService, running: false };
    const { getByText } = render(ServiceCard, { props: { service: svc, onStart: vi.fn(), onStop: vi.fn() } });
    expect(getByText('Stopped')).toBeTruthy();
  });

  it('renders port number', () => {
    const { getByText } = render(ServiceCard, { props: { service: mockService, onStart: vi.fn(), onStop: vi.fn() } });
    expect(getByText('3306')).toBeTruthy();
  });

  it('calls onStop when stop button clicked for running service', async () => {
    const onStop = vi.fn();
    const { getByTitle } = render(ServiceCard, { props: { service: mockService, onStart: vi.fn(), onStop } });
    await fireEvent.click(getByTitle('Stop service'));
    expect(onStop).toHaveBeenCalledWith('mysql');
  });

  it('calls onStart when start button clicked for stopped service', async () => {
    const svc = { ...mockService, running: false };
    const onStart = vi.fn();
    const { getByTitle } = render(ServiceCard, { props: { service: svc, onStart, onStop: vi.fn() } });
    await fireEvent.click(getByTitle('Start service'));
    expect(onStart).toHaveBeenCalledWith('mysql');
  });

  it('shows dimmed state for disabled service', () => {
    const svc = { ...mockService, enabled: false };
    const { container } = render(ServiceCard, { props: { service: svc, onStart: vi.fn(), onStop: vi.fn() } });
    expect(container.querySelector('.opacity-50')).toBeTruthy();
  });

  it('shows auto-start badge when autostart enabled', () => {
    const { getByText } = render(ServiceCard, { props: { service: mockService, onStart: vi.fn(), onStop: vi.fn() } });
    expect(getByText('Auto-start')).toBeTruthy();
  });

  it('renders postgres service correctly', () => {
    const svc = { type: 'postgres', enabled: true, running: false, autostart: false, port: 5432 };
    const { getByText } = render(ServiceCard, { props: { service: svc, onStart: vi.fn(), onStop: vi.fn() } });
    expect(getByText('PostgreSQL')).toBeTruthy();
    expect(getByText('5432')).toBeTruthy();
  });

  it('renders redis service correctly', () => {
    const svc = { type: 'redis', enabled: true, running: true, autostart: true, port: 6379 };
    const { getByText } = render(ServiceCard, { props: { service: svc, onStart: vi.fn(), onStop: vi.fn() } });
    expect(getByText('Redis')).toBeTruthy();
  });
});
