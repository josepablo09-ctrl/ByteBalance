import { page } from 'vitest/browser';
import { beforeEach, describe, expect, it, vi } from 'vitest';
import { render } from 'vitest-browser-svelte';
import { ApiError } from '#lib/api/client.js';
import { lastInsight } from '#lib/stores/insight.js';
import AIInsightCard from './AIInsightCard.svelte';

/** @type {import('#lib/api/ai.js').Diagnosis} */
const DIAGNOSIS = {
	headline: 'Ahorras el 63 % de tus ingresos',
	summary: 'Tus gastos subieron este mes.\n\nTu racha de meditación va en 3 días.',
	finance_insights: ['Comida es tu mayor gasto: 180 000 CRC.'],
	habit_insights: ['Meditar: 12 de 29 días.'],
	suggestions: [
		{ title: 'Pon un tope a Comida', detail: 'Fija 150 000 CRC para el próximo mes.' },
		{ title: 'Registra tras meditar', detail: 'Anota tus gastos después de meditar.' }
	],
	period: { from: '2026-09-17', to: '2026-10-16' },
	generated_at: '2026-10-16T20:30:00Z'
};

/** @param {boolean} enabled */
function fakeAI(enabled = true) {
	return {
		status: vi.fn().mockResolvedValue({ enabled }),
		diagnose: vi.fn().mockResolvedValue(DIAGNOSIS)
	};
}

/** Una promesa que el test resuelve cuando quiere. */
function deferred() {
	/** @type {(value: import('#lib/api/ai.js').Diagnosis) => void} */
	let resolve = () => {};
	const promise = new Promise((r) => (resolve = r));
	return { promise, resolve };
}

describe('AIInsightCard', () => {
	beforeEach(() => lastInsight.reset());

	it('avisa si el servidor no tiene IA configurada', async () => {
		render(AIInsightCard, { ai: fakeAI(false), timeZone: 'UTC' });
		await expect.element(page.getByText(/aún no está disponible/)).toBeVisible();
		await expect
			.element(page.getByRole('button', { name: 'Generar análisis' }))
			.not.toBeInTheDocument();
	});

	it('muestra el estado de carga y luego el diagnóstico', async () => {
		const ai = fakeAI();
		const pending = deferred();
		ai.diagnose.mockReturnValue(pending.promise);
		render(AIInsightCard, { ai, timeZone: 'America/Costa_Rica' });

		await page.getByRole('button', { name: 'Generar análisis' }).click();
		await expect.element(page.getByText('Analizando tus finanzas y hábitos…')).toBeVisible();

		pending.resolve(DIAGNOSIS);

		await expect.element(page.getByText(DIAGNOSIS.headline)).toBeVisible();
		await expect.element(page.getByText('Tus gastos subieron este mes.')).toBeVisible();
		await expect.element(page.getByText('Tu racha de meditación va en 3 días.')).toBeVisible();
		await expect.element(page.getByText(DIAGNOSIS.finance_insights[0])).toBeVisible();
		await expect.element(page.getByText(DIAGNOSIS.habit_insights[0])).toBeVisible();
		await expect.element(page.getByText('Pon un tope a Comida')).toBeVisible();
		await expect.element(page.getByText(/Generado a las\s+14:30/)).toBeVisible();
		await expect.element(page.getByRole('button', { name: 'Actualizar análisis' })).toBeVisible();
		expect(ai.diagnose).toHaveBeenCalledTimes(1);
	});

	it('avisa cuando la IA tarda más de lo habitual', async () => {
		const ai = fakeAI();
		ai.diagnose.mockReturnValue(deferred().promise);
		render(AIInsightCard, { ai, timeZone: 'UTC', slowAfterMs: 10 });

		await page.getByRole('button', { name: 'Generar análisis' }).click();
		await expect.element(page.getByText(/Está tardando más de lo habitual/)).toBeVisible();
	});

	it('conserva el último análisis sin pedir otro', async () => {
		lastInsight.set(DIAGNOSIS);
		const ai = fakeAI();
		render(AIInsightCard, { ai, timeZone: 'UTC' });

		await expect.element(page.getByText(DIAGNOSIS.headline)).toBeVisible();
		expect(ai.diagnose).not.toHaveBeenCalled();
	});

	it('muestra el límite de análisis y permite reintentar', async () => {
		const ai = fakeAI();
		ai.diagnose.mockRejectedValueOnce(
			new ApiError({ status: 429, code: 'too_many_requests', message: '', retryAfter: 1200 })
		);
		render(AIInsightCard, { ai, timeZone: 'UTC' });

		await page.getByRole('button', { name: 'Generar análisis' }).click();
		await expect.element(page.getByRole('alert')).toHaveTextContent(/Espera 20 minutos/);

		await page.getByRole('button', { name: 'Generar análisis' }).click();
		await expect.element(page.getByText(DIAGNOSIS.headline)).toBeVisible();
		await expect.element(page.getByRole('alert')).not.toBeInTheDocument();
	});

	it('explica cuando aún no hay datos', async () => {
		const ai = fakeAI();
		ai.diagnose.mockRejectedValue(
			new ApiError({
				status: 422,
				code: 'not_enough_data',
				message: 'aún no hay datos para analizar: registra algunas transacciones o crea un hábito'
			})
		);
		render(AIInsightCard, { ai, timeZone: 'UTC' });

		await page.getByRole('button', { name: 'Generar análisis' }).click();
		await expect
			.element(page.getByRole('alert'))
			.toHaveTextContent(/registra algunas transacciones/);
	});
});
