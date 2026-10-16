import { page } from 'vitest/browser';
import { describe, expect, it, vi } from 'vitest';
import { render } from 'vitest-browser-svelte';
import HabitCard from './HabitCard.svelte';

const TODAY = '2026-10-16';

/**
 * @param {Partial<import('#lib/api/habits.js').Habit>} habit
 * @param {Record<string, number>} [counts]
 */
function renderCard(habit, counts = {}) {
	const handlers = {
		oncheckin: vi.fn(),
		onlog: vi.fn(),
		onedit: vi.fn(),
		ontoggle: vi.fn(),
		ondelete: vi.fn()
	};
	render(HabitCard, {
		habit: {
			id: 'h1',
			name: 'Leer',
			description: '',
			frequency: 'daily',
			target_count: 1,
			reminder_time: null,
			is_active: true,
			created_at: '',
			...habit
		},
		logs: Object.entries(counts).map(([date, count]) => ({
			id: date,
			habit_id: 'h1',
			date,
			count,
			note: ''
		})),
		today: TODAY,
		...handlers
	});
	return handlers;
}

describe('HabitCard', () => {
	it('un hábito diario de una vez se marca como hecho', async () => {
		const { oncheckin } = renderCard({});
		await page.getByRole('button', { name: 'Marcar hecho' }).click();
		expect(oncheckin).toHaveBeenCalledOnce();
	});

	it('muestra "Hecho hoy" y la racha cuando ya se cumplió', async () => {
		renderCard({}, { '2026-10-14': 1, '2026-10-15': 1, '2026-10-16': 1 });
		await expect.element(page.getByText('Hecho hoy')).toBeVisible();
		await expect
			.element(page.getByRole('button', { name: 'Marcar hecho' }))
			.not.toBeInTheDocument();
		await expect.element(page.getByText('días seguidos')).toBeVisible();
		await expect.element(page.getByText('3', { exact: true })).toBeVisible();
	});

	it('un hábito semanal suma de a uno y muestra el progreso de la semana', async () => {
		renderCard({ frequency: 'weekly', target_count: 3 }, { '2026-10-12': 1, '2026-10-16': 1 });
		await expect.element(page.getByText('3 veces por semana')).toBeVisible();
		await expect.element(page.getByText('2 / 3')).toBeVisible();
		await expect
			.element(page.getByRole('progressbar', { name: 'Progreso esta semana' }))
			.toHaveAttribute('aria-valuenow', '2');
		await expect.element(page.getByRole('button', { name: 'Sumar 1 hoy' })).toBeVisible();
	});

	it('un hábito pausado no admite registros y ofrece reanudar', async () => {
		const { ontoggle } = renderCard({ is_active: false });
		await expect.element(page.getByText('Pausado')).toBeVisible();
		await expect
			.element(page.getByRole('button', { name: 'Marcar hecho' }))
			.not.toBeInTheDocument();
		await page.getByRole('button', { name: 'Reanudar' }).click();
		expect(ontoggle).toHaveBeenCalledOnce();
	});

	it('expone editar y eliminar', async () => {
		const { onedit, ondelete } = renderCard({});
		await page.getByRole('button', { name: 'Editar' }).click();
		await page.getByRole('button', { name: 'Eliminar' }).click();
		expect(onedit).toHaveBeenCalledOnce();
		expect(ondelete).toHaveBeenCalledOnce();
	});
});
