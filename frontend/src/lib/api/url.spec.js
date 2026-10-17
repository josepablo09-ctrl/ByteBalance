import { describe, expect, it } from 'vitest';
import { buildUrl, normalizeBaseUrl } from './url.js';

describe('normalizeBaseUrl', () => {
	it.each([
		['http://localhost:8080/api/v1', 'http://localhost:8080/api/v1'],
		['  https://api.bytebalance.com/api/v1/  ', 'https://api.bytebalance.com/api/v1'],
		['http://localhost:8080/api/v1///', 'http://localhost:8080/api/v1']
	])('normaliza %j', (raw, want) => {
		expect(normalizeBaseUrl(raw)).toBe(want);
	});

	it.each([
		[undefined, /no está definida/],
		['', /no está definida/],
		['localhost:8080', /http o https|no es una URL válida/],
		['ftp://example.com', /http o https/],
		['no es una url', /no es una URL válida/]
	])('rechaza %j', (raw, message) => {
		expect(() => normalizeBaseUrl(raw)).toThrow(message);
	});

	describe('en producción (requireHttps)', () => {
		it.each([
			'https://api.bytebalance.com/api/v1',
			'http://localhost:8080/api/v1',
			'http://127.0.0.1:8080/api/v1'
		])('admite %j', (raw) => {
			expect(normalizeBaseUrl(raw, { requireHttps: true })).toBe(raw);
		});

		it.each(['http://api.bytebalance.com/api/v1', 'http://localhost.evil.com/api/v1'])(
			'rechaza http remoto %j',
			(raw) => {
				expect(() => normalizeBaseUrl(raw, { requireHttps: true })).toThrow(/https en producción/);
			}
		);
	});
});

describe('buildUrl', () => {
	const base = 'http://localhost:8080/api/v1';

	it('une la ruta con o sin barra inicial', () => {
		expect(buildUrl(base, '/users/me')).toBe(`${base}/users/me`);
		expect(buildUrl(base, 'users/me')).toBe(`${base}/users/me`);
	});

	it('codifica los parámetros y omite los vacíos', () => {
		expect(
			buildUrl(base, '/habits', { active: true, from: '2026-10-01', to: null, x: undefined })
		).toBe(`${base}/habits?active=true&from=2026-10-01`);
		expect(buildUrl(base, '/transactions', { q: 'café & pan' })).toBe(
			`${base}/transactions?q=caf%C3%A9+%26+pan`
		);
	});

	it('no añade "?" si no quedan parámetros', () => {
		expect(buildUrl(base, '/habits', { active: undefined })).toBe(`${base}/habits`);
	});
});
