<script>
	import { telegramApi } from '#lib/api/index.js';
	import TelegramCard from '#lib/components/TelegramCard.svelte';
	import { profile } from '#lib/stores/profile.js';

	// El layout solo muestra esta página con el perfil cargado.
	const user = $derived(/** @type {import('#lib/api/auth.js').User} */ ($profile.user));
</script>

<svelte:head>
	<title>Ajustes · ByteBalance</title>
</svelte:head>

<div class="max-w-2xl space-y-8">
	<div>
		<h1 class="text-2xl font-semibold tracking-tight text-zinc-50">Ajustes</h1>
		<p class="mt-1 text-sm text-zinc-400">Tu cuenta y tus notificaciones.</p>
	</div>

	<section aria-labelledby="account-title" class="rounded-lg border border-zinc-800 bg-zinc-900/30">
		<h2
			id="account-title"
			class="border-b border-zinc-800 px-5 py-4 text-sm font-medium text-zinc-100"
		>
			Cuenta
		</h2>
		<dl class="divide-y divide-zinc-800/70 text-sm">
			<div class="flex justify-between gap-4 px-5 py-3">
				<dt class="text-zinc-500">Nombre</dt>
				<dd class="truncate text-zinc-200">{user.full_name}</dd>
			</div>
			<div class="flex justify-between gap-4 px-5 py-3">
				<dt class="text-zinc-500">Email</dt>
				<dd class="truncate text-zinc-200">{user.email}</dd>
			</div>
			<div class="flex justify-between gap-4 px-5 py-3">
				<dt class="text-zinc-500">Moneda principal</dt>
				<dd class="font-mono text-zinc-200">{user.default_currency}</dd>
			</div>
			<div class="flex justify-between gap-4 px-5 py-3">
				<dt class="text-zinc-500">Zona horaria</dt>
				<dd class="font-mono text-zinc-200">{user.timezone}</dd>
			</div>
		</dl>
	</section>

	<TelegramCard telegram={telegramApi} />
</div>
