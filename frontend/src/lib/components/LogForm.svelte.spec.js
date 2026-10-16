import { page } from 'vitest/browser';
import { describe, expect, it, vi } from 'vitest';
import { render } from 'vitest-browser-svelte';
import LogForm from './LogForm.svelte';

const TODAY = '2026-10-16';

/** @type {import('#lib/api/habits.js').HabitLog[]} */
const LOGS = [
	{ id: 'l1', habit_id: 'h1', date: '2026-10-14', count: 3, note: 'bien' },
	{ id: 'l2', habit_id: 'h1', date: TODAY, count: 1, note: '' }
];

/** @param {Partial<import('svelte').ComponentProps<typeof LogForm>>} [props] */
function renderForm(props = {}) {
	const onSubmit = vi.fn().mockResolvedValue(undefined);
	render(LogForm, { logs: LOGS, today: TODAY, onSubmit, oncancel: vi.fn(), ...props });
	return { onSubmit };
}

describe('LogForm', () => {
	it('precarga lo ya registrado hoy', async () => {
		renderForm();
		await expect.element(page.getByLabelText('Veces')).toHaveValue('1');
		await expect.element(page.getByText(/ya tiene 1 vez registrada/)).toBeVisible();
	});

	it('al elegir otro día precarga su cantidad y nota', async () => {
		renderForm();
		await page.getByLabelText('Día').fill('2026-10-14');
		await expect.element(page.getByLabelText('Veces')).toHaveValue('3');
		await expect.element(page.getByLabelText('Nota')).toHaveValue('bien');
	});

	it('con 0 borra el registro del día', async () => {
		const { onSubmit } = renderForm();
		await page.getByLabelText('Veces').fill('0');
		await page.getByRole('button', { name: 'Borrar registro' }).click();
		expect(onSubmit).toHaveBeenCalledWith({ remove: TODAY });
	});

	it('corrige la cantidad de un día', async () => {
		const { onSubmit } = renderForm();
		await page.getByLabelText('Día').fill('2026-10-14');
		await page.getByLabelText('Veces').fill('2');
		await page.getByRole('button', { name: 'Guardar' }).click();
		expect(onSubmit).toHaveBeenCalledWith({
			input: { date: '2026-10-14', count: 2, note: 'bien' }
		});
	});

	it('no permite borrar un día sin registro', async () => {
		const { onSubmit } = renderForm();
		await page.getByLabelText('Día').fill('2026-10-10');
		await page.getByLabelText('Veces').fill('0');
		await page.getByRole('button', { name: 'Guardar' }).click();
		await expect.element(page.getByText('Ese día no tiene registro que borrar.')).toBeVisible();
		expect(onSubmit).not.toHaveBeenCalled();
	});
});
