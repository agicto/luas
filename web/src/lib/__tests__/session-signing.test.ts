import { describe, it, expect, beforeAll } from 'vitest';

import { signSession, verifySession } from '../session-signing';

describe('session-signing', () => {
  beforeAll(() => {
    process.env.SESSION_SECRET = 'test-secret-32-chars-long-abcdefghi';
  });

  it('round-trips a payload', async () => {
    const payload = JSON.stringify({ id: 'u1', role: 'admin' });
    const signed = await signSession(payload);
    const got = await verifySession(signed);
    expect(got).toBe(payload);
  });

  it('rejects an empty value', async () => {
    expect(await verifySession('')).toBeNull();
    expect(await verifySession(undefined)).toBeNull();
    expect(await verifySession(null)).toBeNull();
  });

  it('rejects a malformed signature (no dot)', async () => {
    expect(await verifySession('not-a-signed-value')).toBeNull();
  });

  it('rejects a tampered payload', async () => {
    const signed = await signSession(JSON.stringify({ id: 'u1', role: 'user' }));
    const [, sig] = signed.split('.');
    // base64url-encode a different payload, glue on the real signature
    const tamperedPayload = Buffer.from(JSON.stringify({ id: 'u1', role: 'admin' }))
      .toString('base64')
      .replace(/\+/g, '-')
      .replace(/\//g, '_')
      .replace(/=+$/, '');
    expect(await verifySession(`${tamperedPayload}.${sig}`)).toBeNull();
  });

  it('rejects a tampered signature', async () => {
    const signed = await signSession(JSON.stringify({ id: 'u1' }));
    const [body] = signed.split('.');
    expect(await verifySession(`${body}.AAAAAAAAAAAAAAAAAAAAAAAAAAA`)).toBeNull();
  });
});
