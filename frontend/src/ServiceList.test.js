import { describe, it, expect, vi } from 'vitest';
import { render, fireEvent } from '@testing-library/svelte';
import ServiceList from './ServiceList.svelte';

describe('ServiceList', () => {
  const mockServices = [
    { type: 'mysql', enabled: true, running: true, autostart: true, port: 3306 },
    { type: 'postgres', enabled: true, running: false, autostart: false, port: 5432 },
  ];

  it('renders service names', () => {
    const { getByText } = render(ServiceList, { props: { services: mockServices } });
    expect(getByText('MySQL')).toBeTruthy();
    expect(getByText('PostgreSQL')).toBeTruthy();
  });

  it('renders running status badge', () => {
    const { getByText } = render(ServiceList, { props: { services: mockServices } });
    expect(getByText('Running')).toBeTruthy();
    expect(getByText('Stopped')).toBeTruthy();
  });

  it('renders port numbers', () => {
    const { getByText } = render(ServiceList, { props: { services: mockServices } });
    expect(getByText('3306')).toBeTruthy();
    expect(getByText('5432')).toBeTruthy();
  });

  it('renders header with running count', () => {
    const { getByText } = render(ServiceList, { props: { services: mockServices } });
    expect(getByText('Services')).toBeTruthy();
    expect(getByText('1 of 2 running')).toBeTruthy();
  });

  it('calls onStop when stop button clicked', async () => {
    const onStop = vi.fn();
    const { getByTitle } = render(ServiceList, { props: { services: mockServices, onStop, onStart: vi.fn() } });
    await fireEvent.click(getByTitle('Stop service'));
    expect(onStop).toHaveBeenCalledWith('mysql');
  });

  it('calls onStart when start button clicked', async () => {
    const onStart = vi.fn();
    const { getByTitle } = render(ServiceList, { props: { services: mockServices, onStart, onStop: vi.fn() } });
    await fireEvent.click(getByTitle('Start service'));
    expect(onStart).toHaveBeenCalledWith('postgres');
  });

  it('renders empty state when no services', () => {
    const { getByText } = render(ServiceList, { props: { services: [] } });
    expect(getByText('No services available')).toBeTruthy();
  });

  it('renders loading skeleton when not loaded', () => {
    const { container } = render(ServiceList, { props: { loaded: false, services: [] } });
    expect(container.querySelectorAll('.skeleton').length).toBeGreaterThan(0);
  });

  it('renders card grid layout', () => {
    const { container } = render(ServiceList, { props: { services: mockServices } });
    expect(container.querySelector('.grid')).toBeTruthy();
  });
});
