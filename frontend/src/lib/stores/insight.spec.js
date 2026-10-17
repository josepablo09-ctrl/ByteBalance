import { get } from 'svelte/store';
import { describe, expect, it } from 'vitest';
import { lastInsight } from './insight.js';

describe('lastInsight', () => {
	it('guarda el último diagnóstico y lo olvida al reiniciar', () => {
		expect(get(lastInsight)).toBeNull();

		const diagnosis = /** @type {import('#lib/api/ai.js').Diagnosis} */ ({ headline: 'Vas bien' });
		lastInsight.set(diagnosis);
		expect(get(lastInsight)).toBe(diagnosis);

		lastInsight.reset();
		expect(get(lastInsight)).toBeNull();
	});
});
