import { page } from 'vitest/browser';
import { describe, expect, it, vi } from 'vitest';
import { render } from 'vitest-browser-svelte';
import { ApiError } from '#lib/api/client.js';
import TelegramCard from './TelegramCard.svelte';

const LINK = {
	code: 'AbC123_-xyz',
	url: 'https://t.me/ByteBalanceBot?start=AbC123_-xyz',
	expires_at: new Date(Date.now() + 10 * 60_000).toISOString()
};

/** @param {Partial<import('#lib/api/telegram.js').TelegramStatus>} status */
function fakeTelegram(status) {
	return {
		status: vi.fn().mockResolvedValue({
			enabled: true,
			linked: false,
			bot_username: 'ByteBalanceBot',
			...status
		}),
		createLink: vi.fn().mockResolvedValue(LINK),
		unlink: vi.fn().mockResolvedValue(undefined),
		sendTest: vi.fn().mockResolvedValue(undefined)
	};
}

describe('TelegramCard', () => {
	it('avisa si el servidor no tiene Telegram configurado', async () => {
		render(TelegramCard, { telegram: fakeTelegram({ enabled: false }) });
		await expect.element(page.getByText(/aún no está disponible/)).toBeVisible();
		await expect
			.element(page.getByRole('button', { name: 'Conectar Telegram' }))
			.not.toBeInTheDocument();
	});

	it('conecta: abre el bot y espera la confirmación', async () => {
		const telegram = fakeTelegram({});
		render(TelegramCard, { telegram, pollIntervalMs: 20 });

		await page.getByRole('button', { name: 'Conectar Telegram' }).click();

		const link = page.getByRole('link', { name: 'Abrir @ByteBalanceBot' });
		await expect.element(link).toHaveAttribute('href', LINK.url);
		await expect.element(link).toHaveAttribute('target', '_blank');
		await expect.element(link).toHaveAttribute('rel', 'noopener noreferrer');
		await expect.element(page.getByText(`/start ${LINK.code}`)).toBeVisible();
		await expect.element(page.getByText(/Esperando confirmación/)).toBeVisible();

		// El bot confirma: la siguiente consulta ya devuelve linked.
		telegram.status.mockResolvedValue({
			enabled: true,
			linked: true,
			bot_username: 'ByteBalanceBot'
		});
		await expect.element(page.getByText('Conectado', { exact: true })).toBeVisible();
		await expect.element(page.getByText(/Telegram quedó conectado/)).toBeVisible();
		await expect
			.element(page.getByRole('button', { name: 'Enviar mensaje de prueba' }))
			.toBeVisible();
	});

	it('si el enlace vence, ofrece generar uno nuevo', async () => {
		const telegram = fakeTelegram({});
		telegram.createLink.mockResolvedValue({
			...LINK,
			expires_at: new Date(Date.now() - 1000).toISOString()
		});
		render(TelegramCard, { telegram });

		await page.getByRole('button', { name: 'Conectar Telegram' }).click();

		await expect.element(page.getByText(/El enlace venció/)).toBeVisible();
		await expect
			.element(page.getByRole('button', { name: 'Generar un enlace nuevo' }))
			.toBeVisible();
	});

	it('muestra el límite de enlaces generados', async () => {
		const telegram = fakeTelegram({});
		telegram.createLink.mockRejectedValue(
			new ApiError({ status: 429, code: 'too_many_requests', message: 'x', retryAfter: 25 })
		);
		render(TelegramCard, { telegram });

		await page.getByRole('button', { name: 'Conectar Telegram' }).click();
		await expect.element(page.getByRole('alert')).toHaveTextContent(/Espera 25 s/);
	});

	it('envía un mensaje de prueba', async () => {
		const telegram = fakeTelegram({ linked: true });
		render(TelegramCard, { telegram });

		await page.getByRole('button', { name: 'Enviar mensaje de prueba' }).click();
		expect(telegram.sendTest).toHaveBeenCalledOnce();
		await expect.element(page.getByText(/Mensaje enviado/)).toBeVisible();
	});

	it('desconecta solo tras confirmar', async () => {
		const telegram = fakeTelegram({ linked: true });
		render(TelegramCard, { telegram });

		await page.getByRole('button', { name: 'Desconectar' }).click();
		const dialog = page.getByRole('dialog');
		await expect.element(dialog.getByText(/Dejarás de recibir recordatorios/)).toBeVisible();
		expect(telegram.unlink).not.toHaveBeenCalled();

		await dialog.getByRole('button', { name: 'Desconectar' }).click();
		expect(telegram.unlink).toHaveBeenCalledOnce();
		await expect.element(page.getByText('Telegram quedó desconectado.')).toBeVisible();
		await expect.element(page.getByRole('button', { name: 'Conectar Telegram' })).toBeVisible();
	});
});
