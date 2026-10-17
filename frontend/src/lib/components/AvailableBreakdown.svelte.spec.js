import { page } from 'vitest/browser';
import { describe, expect, it } from 'vitest';
import { render } from 'vitest-browser-svelte';
import { formatMoney } from '#lib/money.js';
import AvailableBreakdown from './AvailableBreakdown.svelte';

/**
 * Monto formateado con espacios normales: toHaveTextContent normaliza los del
 * texto (incluido el NBSP de los montos) pero no los del valor esperado.
 *
 * @param {Parameters<typeof formatMoney>} args
 */
const money = (...args) => formatMoney(...args).replace(/\s+/g, ' ');

describe('AvailableBreakdown', () => {
	it('resta el ahorro del balance total', async () => {
		render(AvailableBreakdown, {
			netCents: 475_000_00,
			savingsCents: 120_000_00,
			availableCents: 355_000_00,
			currency: 'CRC'
		});

		const terms = page.getByRole('definition');
		await expect.element(terms.nth(0)).toHaveTextContent(money(475_000_00, 'CRC'));
		await expect
			.element(terms.nth(1))
			.toHaveTextContent(money(-120_000_00, 'CRC', { signDisplay: 'exceptZero' }));
		await expect.element(terms.nth(2)).toHaveTextContent(money(355_000_00, 'CRC'));
		await expect.element(page.getByText(/Apartaste más/)).not.toBeInTheDocument();
	});

	it('avisa si se apartó más de lo disponible', async () => {
		render(AvailableBreakdown, {
			netCents: 200,
			savingsCents: 500,
			availableCents: -300,
			currency: 'USD'
		});

		await expect.element(page.getByText(/Apartaste más de lo que tienes disponible/)).toBeVisible();
	});
});
