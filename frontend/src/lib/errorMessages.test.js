import { describe, it, expect } from 'vitest';
import { friendlyError } from './errorMessages.js';

describe('friendlyError', () => {
  it('maps "is not a directory" errors', () => {
    expect(friendlyError('path "/tmp/foo" is not a directory'))
      .toBe('The selected path is not a valid directory.');
  });

  it('maps "already registered" errors', () => {
    expect(friendlyError('domain "myapp.test" is already registered'))
      .toBe('A site with domain "myapp.test" already exists.');
  });

  it('maps "not found" domain errors', () => {
    expect(friendlyError('domain "myapp.test" not found'))
      .toBe('Could not find site "myapp.test".');
  });

  it('passes through unrecognized errors unchanged', () => {
    expect(friendlyError('something unexpected happened'))
      .toBe('something unexpected happened');
  });

  it('handles empty string', () => {
    expect(friendlyError('')).toBe('');
  });
});
