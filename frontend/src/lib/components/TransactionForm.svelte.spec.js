import { page } from 'vitest/browser';
import { describe, expect, it, vi } from 'vitest';
import { render } from 'vitest-browser-svelte';
import { ApiError } from '#lib/api/client.js';
import { todayInZone } from '#lib/dates.js';
import TransactionForm from './TransactionForm.svelte';

const TIME_ZONE = 'America/Costa_Rica';

/** @type {import('#lib/api/finance.js').Category[]} */
const CATEGORIES = [
	{ id: 'c-food', name: 'Supermercado', type: 'expense', created_at: '' },
	{ id: 'c-salary', name: 'Salario', type: 'income', created_at: '' }
];

/** @param {Partial<import('svelte').ComponentProps<typeof TransactionForm>>} [props] */
function renderForm(props = {}) {
	const onSubmit = vi.fn().mockResolvedValue(undefined);
	const onCreateCategory = vi.fn();
	render(TransactionForm, {
		categories: CATEGORIES,
		defaultCurrency: 'CRC',
		timeZone: TIME_ZONE,
		onSubmit,
		onCreateCategory,
		oncancel: vi.fn(),
		...props
	});
	return { onSubmit, onCreateCategory, ...props };
}

const amountInput = () => page.getByLabelText('Monto');
const categorySelect = () => page.getByLabelText('Categoría');

describe('TransactionForm', () => {
	it('registra un gasto de hoy en la moneda del usuario', async () => {
		const { onSubmit } = renderForm();

		await amountInput().fill('1250,5');
		await categorySelect().selectOptions('Supermercado');
		await page.getByLabelText('Descripción').fill('Compra semanal');
		await page.getByRole('button', { name: 'Registrar gasto' }).click();

		expect(onSubmit).toHaveBeenCalledWith({
			type: 'expense',
			amount_cents: 125050,
			currency: 'CRC',
			category_id: 'c-food',
			description: 'Compra semanal'
		});
	});

	it('valida el monto antes de enviar', async () => {
		const { onSubmit } = renderForm();

		await amountInput().fill('1.250,50');
		await page.getByRole('button', { name: 'Registrar gasto' }).click();

		await expect.element(page.getByText(/Ingresa un monto válido/)).toBeVisible();
		await expect.element(amountInput()).toHaveAttribute('aria-invalid', 'true');
		expect(onSubmit).not.toHaveBeenCalled();
	});

	it('solo ofrece categorías del tipo elegido', async () => {
		renderForm();
		await expect.element(page.getByRole('option', { name: 'Supermercado' })).toBeInTheDocument();
		await expect.element(page.getByRole('option', { name: 'Salario' })).not.toBeInTheDocument();

		await page.getByRole('button', { name: 'Ingreso' }).click();

		await expect.element(page.getByRole('option', { name: 'Salario' })).toBeInTheDocument();
		await expect
			.element(page.getByRole('option', { name: 'Supermercado' }))
			.not.toBeInTheDocument();
		await expect.element(page.getByRole('button', { name: 'Registrar ingreso' })).toBeVisible();
	});

	it('quita la categoría si deja de aplicar al cambiar de tipo', async () => {
		const { onSubmit } = renderForm();

		await categorySelect().selectOptions('Supermercado');
		await page.getByRole('button', { name: 'Ingreso' }).click();
		await amountInput().fill('100');
		await page.getByRole('button', { name: 'Registrar ingreso' }).click();

		expect(onSubmit).toHaveBeenCalledWith(expect.objectContaining({ category_id: null }));
	});

	it('crea una categoría sin salir del formulario y la selecciona', async () => {
		const onCreateCategory = vi
			.fn()
			.mockResolvedValue({ id: 'c-new', name: 'Transporte', type: 'expense', created_at: '' });
		const { onSubmit } = renderForm({ onCreateCategory });

		await categorySelect().selectOptions('+ Nueva categoría…');
		const nameInput = page.getByLabelText('Nueva categoría de gastos');
		await nameInput.fill('Transporte');
		await page.getByRole('button', { name: 'Crear categoría' }).click();

		expect(onCreateCategory).toHaveBeenCalledWith('Transporte', 'expense');
		await amountInput().fill('3000');
		await page.getByRole('button', { name: 'Registrar gasto' }).click();
		expect(onSubmit).toHaveBeenCalledWith(expect.objectContaining({ category_id: 'c-new' }));
	});

	it('fecha un día pasado a las 12:00 del usuario', async () => {
		const { onSubmit } = renderForm();

		await amountInput().fill('50');
		await page.getByLabelText('Fecha').fill('2026-10-01');
		await page.getByRole('button', { name: 'Registrar gasto' }).click();

		expect(onSubmit).toHaveBeenCalledWith(
			expect.objectContaining({ occurred_at: '2026-10-01T18:00:00.000Z' })
		);
	});

	it('propone la fecha de hoy en la zona del usuario', async () => {
		renderForm();
		await expect.element(page.getByLabelText('Fecha')).toHaveValue(todayInZone(TIME_ZONE));
	});

	it('muestra el error de la API sin cerrar el formulario', async () => {
		const onSubmit = vi.fn().mockRejectedValue(
			new ApiError({
				status: 422,
				code: 'invalid_input',
				message: 'una categoría de tipo "income" no aplica a una transacción de tipo "expense"'
			})
		);
		renderForm({ onSubmit });

		await amountInput().fill('10');
		await page.getByRole('button', { name: 'Registrar gasto' }).click();

		await expect.element(page.getByRole('alert')).toHaveTextContent(/no aplica a una transacción/);
	});
});
