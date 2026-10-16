<script>
	import { onMount } from 'svelte';
	import { errorMessage } from '#lib/api/errors.js';
	import Alert from './Alert.svelte';
	import Button from './Button.svelte';
	import Icon from './Icon.svelte';
	import Modal from './Modal.svelte';

	/**
	 * Conectar y desconectar Telegram. Al conectar se abre el bot con un enlace de
	 * un solo uso y se consulta el estado hasta que el bot confirme la vinculación.
	 *
	 * @typedef {object} Props
	 * @property {Pick<import('#lib/api/telegram.js').TelegramApi, 'status' | 'createLink' | 'unlink' | 'sendTest'>} telegram
	 * @property {number} [pollIntervalMs] Cada cuánto se consulta si ya se vinculó.
	 */

	/** @type {Props} */
	let { telegram, pollIntervalMs = 3000 } = $props();

	let status = $state(/** @type {import('#lib/api/telegram.js').TelegramStatus | null} */ (null));
	let isLoading = $state(true);
	let loadError = $state('');
	let pendingLink = $state(
		/** @type {import('#lib/api/telegram.js').TelegramLink | null} */ (null)
	);
	let now = $state(Date.now());
	let actionError = $state('');
	let notice = $state('');
	let busyAction = $state(/** @type {'' | 'link' | 'test' | 'unlink'} */ (''));
	let isConfirmingUnlink = $state(false);

	const secondsLeft = $derived(
		pendingLink ? Math.max(0, Math.ceil((Date.parse(pendingLink.expires_at) - now) / 1000)) : 0
	);
	const isExpired = $derived(pendingLink !== null && secondsLeft === 0);
	const countdown = $derived(
		`${Math.floor(secondsLeft / 60)}:${String(secondsLeft % 60).padStart(2, '0')}`
	);

	async function loadStatus() {
		isLoading = true;
		loadError = '';
		try {
			status = await telegram.status();
		} catch (err) {
			loadError = errorMessage(err);
		} finally {
			isLoading = false;
		}
	}

	async function connect() {
		busyAction = 'link';
		actionError = '';
		notice = '';
		try {
			pendingLink = await telegram.createLink();
			now = Date.now();
		} catch (err) {
			actionError = errorMessage(err);
		} finally {
			busyAction = '';
		}
	}

	async function sendTest() {
		busyAction = 'test';
		actionError = '';
		notice = '';
		try {
			await telegram.sendTest();
			notice = 'Mensaje enviado. Revisa tu chat con el bot.';
		} catch (err) {
			actionError = errorMessage(err);
		} finally {
			busyAction = '';
		}
	}

	async function unlink() {
		busyAction = 'unlink';
		actionError = '';
		notice = '';
		try {
			await telegram.unlink();
			isConfirmingUnlink = false;
			status = status && { ...status, linked: false };
			notice = 'Telegram quedó desconectado.';
		} catch (err) {
			isConfirmingUnlink = false;
			actionError = errorMessage(err);
		} finally {
			busyAction = '';
		}
	}

	/** @param {string} text */
	async function copy(text) {
		try {
			await navigator.clipboard.writeText(text);
			notice = 'Comando copiado.';
		} catch {
			actionError = 'No se pudo copiar. Selecciona el texto y cópialo a mano.';
		}
	}

	// Mientras hay un enlace vigente: cuenta regresiva y consulta del estado.
	$effect(() => {
		if (!pendingLink || isExpired) return;
		const clock = setInterval(() => (now = Date.now()), 1000);
		const poll = setInterval(async () => {
			try {
				const next = await telegram.status();
				if (next.linked) {
					status = next;
					pendingLink = null;
					notice = 'Telegram quedó conectado. Ahí recibirás tus recordatorios.';
				}
			} catch {
				// Un fallo puntual no interrumpe la espera; se reintenta en el próximo ciclo.
			}
		}, pollIntervalMs);
		return () => {
			clearInterval(clock);
			clearInterval(poll);
		};
	});

	onMount(loadStatus);
</script>

<section aria-labelledby="telegram-title" class="rounded-lg border border-zinc-800 bg-zinc-900/30">
	<div class="flex items-start gap-3 border-b border-zinc-800 px-5 py-4">
		<span
			class="grid size-9 shrink-0 place-items-center rounded-lg border border-zinc-800 bg-zinc-900 text-zinc-300"
			aria-hidden="true"
		>
			<Icon name="send" />
		</span>
		<div class="min-w-0 flex-1">
			<div class="flex flex-wrap items-center gap-2">
				<h2 id="telegram-title" class="text-sm font-medium text-zinc-100">Telegram</h2>
				{#if status?.linked}
					<span
						class="rounded border border-emerald-500/30 bg-emerald-500/10 px-1.5 py-px text-[11px] font-medium text-emerald-300"
					>
						Conectado
					</span>
				{/if}
			</div>
			<p class="mt-0.5 text-sm text-zinc-400">
				Recibe en Telegram los recordatorios de tus hábitos a la hora que elijas.
			</p>
		</div>
	</div>

	<div class="space-y-4 px-5 py-4">
		{#if notice}
			<Alert message={notice} tone="success" />
		{/if}
		{#if actionError}
			<Alert message={actionError} />
		{/if}

		{#if isLoading}
			<div class="h-9 w-48 animate-pulse rounded-lg bg-zinc-800/70" aria-label="Cargando"></div>
		{:else if loadError}
			<div class="flex flex-wrap items-center gap-3">
				<Alert message={loadError} />
				<Button variant="secondary" class="h-9" onclick={loadStatus}>Reintentar</Button>
			</div>
		{:else if status && !status.enabled}
			<p class="text-sm text-zinc-500">
				La integración con Telegram aún no está disponible en este servidor.
			</p>
		{:else if status?.linked}
			<p class="text-sm text-zinc-400">
				Tu cuenta está vinculada con
				<span class="font-mono text-zinc-200">@{status.bot_username}</span>. Para dejar de recibir
				mensajes puedes desconectarla aquí o enviar <span class="font-mono">/stop</span> al bot.
			</p>
			<div class="flex flex-wrap gap-2">
				<Button variant="secondary" loading={busyAction === 'test'} onclick={sendTest}>
					Enviar mensaje de prueba
				</Button>
				<Button
					variant="ghost"
					class="hover:text-red-300"
					disabled={busyAction !== ''}
					onclick={() => (isConfirmingUnlink = true)}
				>
					Desconectar
				</Button>
			</div>
		{:else if pendingLink && !isExpired}
			<ol class="space-y-3 text-sm text-zinc-300">
				<li class="flex gap-3">
					<span class="font-mono text-zinc-500">1.</span>
					<div class="space-y-2">
						<p>Abre el bot en Telegram y pulsa <span class="font-medium">Iniciar</span>.</p>
						<!-- eslint-disable svelte/no-navigation-without-resolve -- enlace externo a t.me, no una ruta de la app -->
						<a
							href={pendingLink.url}
							target="_blank"
							rel="noopener noreferrer"
							class="inline-flex h-10 items-center gap-2 rounded-lg bg-emerald-500 px-4 text-sm font-medium text-zinc-950 transition-colors hover:bg-emerald-400 focus-visible:ring-2 focus-visible:ring-emerald-500/50 focus-visible:ring-offset-2 focus-visible:ring-offset-zinc-950 focus-visible:outline-none"
						>
							<Icon name="send" />
							Abrir @{status?.bot_username}
						</a>
						<!-- eslint-enable svelte/no-navigation-without-resolve -->
						<p class="text-xs text-zinc-500">
							¿Telegram está en otro dispositivo? Envía al bot:
							<button
								type="button"
								onclick={() => pendingLink && copy(`/start ${pendingLink.code}`)}
								class="ml-1 rounded border border-zinc-800 bg-zinc-900 px-1.5 py-0.5 font-mono text-zinc-300 transition-colors hover:border-zinc-700"
								aria-label="Copiar comando /start"
							>
								/start {pendingLink.code}
							</button>
						</p>
					</div>
				</li>
				<li class="flex gap-3">
					<span class="font-mono text-zinc-500">2.</span>
					<p class="flex items-center gap-2 text-zinc-400" role="status">
						<span class="size-2 animate-pulse rounded-full bg-emerald-400" aria-hidden="true"
						></span>
						Esperando confirmación del bot… el enlace vence en
						<span class="font-mono text-zinc-200 tabular-nums">{countdown}</span>
					</p>
				</li>
			</ol>
			<Button variant="ghost" class="h-8 px-3" onclick={() => (pendingLink = null)}>Cancelar</Button
			>
		{:else}
			{#if isExpired}
				<p class="text-sm text-zinc-400">El enlace venció sin confirmarse. Genera uno nuevo.</p>
			{/if}
			<Button loading={busyAction === 'link'} onclick={connect}>
				<Icon name="send" />
				{isExpired ? 'Generar un enlace nuevo' : 'Conectar Telegram'}
			</Button>
		{/if}
	</div>
</section>

<Modal bind:open={isConfirmingUnlink} title="Desconectar Telegram">
	<p class="text-sm text-zinc-300">
		Dejarás de recibir recordatorios en Telegram. Puedes volver a conectarlo cuando quieras.
	</p>
	<div class="mt-6 flex justify-end gap-2 border-t border-zinc-800 pt-5">
		<Button variant="secondary" onclick={() => (isConfirmingUnlink = false)}>Cancelar</Button>
		<Button variant="danger" loading={busyAction === 'unlink'} onclick={unlink}>Desconectar</Button>
	</div>
</Modal>
