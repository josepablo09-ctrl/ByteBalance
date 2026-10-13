import { page } from 'vitest/browser';
import { describe, expect, it, vi } from 'vitest';
import { render } from 'vitest-browser-svelte';
import { ApiError } from '#lib/api/client.js';
import { browserTimezone } from '#lib/validation.js';
import RegisterForm from './RegisterForm.svelte';

async function fillValidForm() {
	await page.getByLabelText('Nombre completo').fill(' Ana Pérez ');
	await page.getByLabelText('Email').fill('ana@example.com');
	await page.getByLabelText('Contraseña', { exact: true }).fill('contraseña-segura');
}

const submitButton = () => page.getByRole('button', { name: 'Crear cuenta' });

describe('RegisterForm', () => {
	it('envía los datos con la moneda elegida y la zona horaria del navegador', async () => {
		const onRegister = vi.fn().mockResolvedValue(undefined);
		render(RegisterForm, { onRegister });

		await fillValidForm();
		await page.getByLabelText('Moneda principal').selectOptions('CRC');
		await submitButton().click();

		expect(onRegister).toHaveBeenCalledWith({
			full_name: 'Ana Pérez',
			email: 'ana@example.com',
			password: 'contraseña-segura',
			default_currency: 'CRC',
			timezone: browserTimezone()
		});
	});

	it('exige una contraseña de al menos 8 caracteres', async () => {
		const onRegister = vi.fn();
		render(RegisterForm, { onRegister });

		await fillValidForm();
		await page.getByLabelText('Contraseña', { exact: true }).fill('corta');
		await submitButton().click();

		await expect.element(page.getByText('Usa al menos 8 caracteres.')).toBeVisible();
		expect(onRegister).not.toHaveBeenCalled();
	});

	it('muestra el email duplicado junto al campo', async () => {
		const onRegister = vi
			.fn()
			.mockRejectedValue(
				new ApiError({ status: 409, code: 'email_taken', message: 'el email ya está registrado' })
			);
		render(RegisterForm, { onRegister });

		await fillValidForm();
		await submitButton().click();

		await expect.element(page.getByText('Ya existe una cuenta con este email.')).toBeVisible();
		await expect.element(page.getByLabelText('Email')).toHaveAttribute('aria-invalid', 'true');
		await expect.element(page.getByRole('alert')).not.toBeInTheDocument();
	});

	it('muestra el detalle de validación del servidor', async () => {
		const onRegister = vi.fn().mockRejectedValue(
			new ApiError({
				status: 422,
				code: 'invalid_input',
				message: 'zona horaria "Foo/Bar" desconocida'
			})
		);
		render(RegisterForm, { onRegister });

		await fillValidForm();
		await submitButton().click();

		await expect
			.element(page.getByRole('alert'))
			.toHaveTextContent('Zona horaria "Foo/Bar" desconocida.');
	});
});
