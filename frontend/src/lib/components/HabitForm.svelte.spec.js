import { page } from 'vitest/browser';
import { describe, expect, it, vi } from 'vitest';
import { render } from 'vitest-browser-svelte';
import HabitForm from './HabitForm.svelte';

/** @type {import('#lib/api/habits.js').Habit} */
const EXISTING = {
	id: 'h1',
	name: 'Leer',
	description: '20 páginas',
	frequency: 'weekly',
	target_count: 3,
	reminder_time: '21:30',
	is_active: true,
	created_at: ''
};

describe('HabitForm', () => {
	it('crea un hábito diario sin recordatorio por defecto', async () => {
		const onSubmit = vi.fn().mockResolvedValue(undefined);
		render(HabitForm, { onSubmit, oncancel: vi.fn() });

		await page.getByLabelText('Nombre').fill('Meditar');
		await page.getByRole('button', { name: 'Crear hábito' }).click();

		expect(onSubmit).toHaveBeenCalledWith({
			name: 'Meditar',
			description: '',
			frequency: 'daily',
			target_count: 1,
			reminder_time: null
		});
	});

	it('elige frecuencia, meta y recordatorio', async () => {
		const onSubmit = vi.fn().mockResolvedValue(undefined);
		render(HabitForm, { onSubmit, oncancel: vi.fn() });

		await page.getByLabelText('Nombre').fill('Correr');
		await page.getByRole('button', { name: 'Semanal' }).click();
		await page.getByLabelText('Meta por periodo').fill('3');
		await expect.element(page.getByText('Meta: 3 veces por semana.')).toBeVisible();
		await page.getByRole('switch', { name: 'Recordatorio' }).click();
		await page.getByLabelText('Hora').fill('06:45');
		await page.getByRole('button', { name: 'Crear hábito' }).click();

		expect(onSubmit).toHaveBeenCalledWith(
			expect.objectContaining({ frequency: 'weekly', target_count: 3, reminder_time: '06:45' })
		);
	});

	it('precarga el hábito al editar y permite quitar el recordatorio', async () => {
		const onSubmit = vi.fn().mockResolvedValue(undefined);
		render(HabitForm, { habit: EXISTING, onSubmit, oncancel: vi.fn() });

		await expect.element(page.getByLabelText('Nombre')).toHaveValue('Leer');
		await expect.element(page.getByLabelText('Hora')).toHaveValue('21:30');
		await page.getByRole('switch', { name: 'Recordatorio' }).click();
		await page.getByRole('button', { name: 'Guardar cambios' }).click();

		expect(onSubmit).toHaveBeenCalledWith({
			name: 'Leer',
			description: '20 páginas',
			frequency: 'weekly',
			target_count: 3,
			reminder_time: null
		});
	});

	it('valida antes de enviar', async () => {
		const onSubmit = vi.fn();
		render(HabitForm, { onSubmit, oncancel: vi.fn() });

		await page.getByLabelText('Meta por periodo').fill('0');
		await page.getByRole('button', { name: 'Crear hábito' }).click();

		await expect.element(page.getByText('Ponle un nombre.')).toBeVisible();
		await expect.element(page.getByText(/entero entre 1 y 1000/)).toBeVisible();
		expect(onSubmit).not.toHaveBeenCalled();
	});
});
