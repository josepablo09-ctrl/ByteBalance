import { page } from 'vitest/browser';
import { describe, expect, it, vi } from 'vitest';
import { render } from 'vitest-browser-svelte';
import { ApiError } from '#lib/api/client.js';
import { formatMoney } from '#lib/money.js';
import SavingsCard from './SavingsCard.svelte';

/** @param {Partial<import('svelte').ComponentProps<typeof SavingsCard>>} [props] */
function renderCard(props = {}) {
	const onSaveGoal = vi.fn().mockResolvedValue(undefined);
	render(SavingsCard, {
		monthCents: 60_000_00,
		totalCents: 340_000_00,
		currency: 'CRC',
		monthLabel: 'octubre de 2026',
		goalCents: 100_000_00,
		onSaveGoal,
		...props
	});
	return { onSaveGoal, ...props };
}

describe('SavingsCard', () => {
	it('muestra el ahorro del mes y el progreso hacia la meta', async () => {
		renderCard();

		await expect.element(page.getByText(formatMoney(60_000_00, 'CRC'))).toBeVisible();
		await expect.element(page.getByRole('progressbar')).toHaveAttribute('aria-valuenow', '60');
		await expect.element(page.getByText('60 %')).toBeVisible();
		await expect
			.element(page.getByText(new RegExp(`Acumulado ${formatMoney(340_000_00, 'CRC')}`)))
			.toBeVisible();
	});

	it('sin meta no muestra barra e invita a fijarla', async () => {
		const { onSaveGoal } = renderCard({ goalCents: null });

		await expect.element(page.getByRole('progressbar')).not.toBeInTheDocument();
		await page.getByRole('button', { name: 'Fijar meta' }).click();
		await page.getByLabelText('Meta mensual (CRC)').fill('150000');
		await page.getByRole('button', { name: 'Guardar meta' }).click();

		expect(onSaveGoal).toHaveBeenCalledWith(150_000_00);
		await expect.element(page.getByRole('dialog')).not.toBeInTheDocument();
	});

	it('permite quitar la meta', async () => {
		const { onSaveGoal } = renderCard();

		await page.getByRole('button', { name: 'Editar meta' }).click();
		await expect.element(page.getByLabelText('Meta mensual (CRC)')).toHaveValue('100000.00');
		await page.getByRole('button', { name: 'Quitar meta' }).click();

		expect(onSaveGoal).toHaveBeenCalledWith(null);
	});

	it('valida el monto y muestra los errores de la API', async () => {
		const onSaveGoal = vi
			.fn()
			.mockRejectedValue(
				new ApiError({ status: 422, code: 'invalid_input', message: 'la meta es demasiado grande' })
			);
		renderCard({ goalCents: null, onSaveGoal });

		await page.getByRole('button', { name: 'Fijar meta' }).click();
		await page.getByRole('button', { name: 'Guardar meta' }).click();
		await expect.element(page.getByText('Ingresa un monto.')).toBeVisible();
		expect(onSaveGoal).not.toHaveBeenCalled();

		await page.getByLabelText('Meta mensual (CRC)').fill('999');
		await page.getByRole('button', { name: 'Guardar meta' }).click();
		await expect.element(page.getByRole('alert')).toHaveTextContent('La meta es demasiado grande.');
	});
});
