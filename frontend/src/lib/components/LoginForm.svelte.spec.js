import { page } from 'vitest/browser';
import { describe, expect, it, vi } from 'vitest';
import { render } from 'vitest-browser-svelte';
import { ApiError } from '#lib/api/client.js';
import LoginForm from './LoginForm.svelte';

const emailInput = () => page.getByLabelText('Email');
const passwordInput = () => page.getByLabelText('Contraseña', { exact: true });
const submitButton = () => page.getByRole('button', { name: 'Iniciar sesión' });

describe('LoginForm', () => {
	it('valida los campos antes de llamar a la API', async () => {
		const onLogin = vi.fn();
		render(LoginForm, { onLogin });

		await submitButton().click();

		await expect.element(page.getByText('Ingresa tu email.')).toBeVisible();
		await expect.element(page.getByText('Ingresa tu contraseña.')).toBeVisible();
		await expect.element(emailInput()).toHaveAttribute('aria-invalid', 'true');
		expect(onLogin).not.toHaveBeenCalled();
	});

	it('envía el email sin espacios y la contraseña tal cual', async () => {
		const onLogin = vi.fn().mockResolvedValue(undefined);
		render(LoginForm, { onLogin });

		await emailInput().fill('  ana@example.com ');
		await passwordInput().fill(' clave con espacios ');
		await submitButton().click();

		expect(onLogin).toHaveBeenCalledWith({
			email: 'ana@example.com',
			password: ' clave con espacios '
		});
	});

	it('muestra el error de credenciales de la API', async () => {
		const onLogin = vi
			.fn()
			.mockRejectedValue(new ApiError({ status: 401, code: 'invalid_credentials', message: 'x' }));
		render(LoginForm, { onLogin });

		await emailInput().fill('ana@example.com');
		await passwordInput().fill('incorrecta');
		await submitButton().click();

		await expect
			.element(page.getByRole('alert'))
			.toHaveTextContent('Email o contraseña incorrectos.');
	});

	it('indica cuánto esperar tras demasiados intentos', async () => {
		const onLogin = vi
			.fn()
			.mockRejectedValue(
				new ApiError({ status: 429, code: 'too_many_requests', message: 'x', retryAfter: 30 })
			);
		render(LoginForm, { onLogin });

		await emailInput().fill('ana@example.com');
		await passwordInput().fill('incorrecta');
		await submitButton().click();

		await expect.element(page.getByRole('alert')).toHaveTextContent(/Espera 30 s/);
	});

	it('deshabilita el botón mientras se envía', async () => {
		/** @type {(value?: unknown) => void} */
		let finish = () => {};
		const onLogin = vi.fn(() => new Promise((resolve) => (finish = resolve)));
		render(LoginForm, { onLogin });

		await emailInput().fill('ana@example.com');
		await passwordInput().fill('clave');
		await submitButton().click();

		const busy = page.getByRole('button', { name: 'Ingresando…' });
		await expect.element(busy).toBeDisabled();
		finish();
		await expect.element(submitButton()).toBeEnabled();
	});

	it('permite mostrar la contraseña', async () => {
		render(LoginForm, { onLogin: vi.fn() });

		await expect.element(passwordInput()).toHaveAttribute('type', 'password');
		await page.getByRole('button', { name: 'Mostrar contraseña' }).click();
		await expect.element(passwordInput()).toHaveAttribute('type', 'text');
	});

	it('muestra el aviso recibido', async () => {
		render(LoginForm, { onLogin: vi.fn(), notice: 'Tu cuenta está lista.' });
		await expect.element(page.getByRole('status')).toHaveTextContent('Tu cuenta está lista.');
	});
});
