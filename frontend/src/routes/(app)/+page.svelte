<script>
	import { onMount } from 'svelte';
	import { authApi, errorMessage } from '#lib/api/index.js';
	import Alert from '#lib/components/Alert.svelte';
	import Button from '#lib/components/Button.svelte';

	let profile = $state(/** @type {import('#lib/api/auth.js').User | null} */ (null));
	let isLoading = $state(true);
	let loadError = $state('');

	const firstName = $derived(profile?.full_name.split(' ')[0] ?? '');
	const memberSince = $derived(
		profile
			? new Intl.DateTimeFormat('es', { dateStyle: 'long' }).format(new Date(profile.created_at))
			: ''
	);

	async function loadProfile() {
		isLoading = true;
		loadError = '';
		try {
			profile = await authApi.me();
		} catch (err) {
			loadError = errorMessage(err);
		} finally {
			isLoading = false;
		}
	}

	onMount(loadProfile);
</script>

<svelte:head>
	<title>Inicio · ByteBalance</title>
</svelte:head>

{#if isLoading}
	<div class="animate-pulse space-y-4" aria-busy="true" aria-label="Cargando perfil">
		<div class="h-7 w-48 rounded-md bg-zinc-800/70"></div>
		<div class="h-4 w-72 rounded-md bg-zinc-800/50"></div>
		<div class="mt-8 h-40 rounded-lg border border-zinc-800 bg-zinc-900/40"></div>
	</div>
{:else if loadError}
	<div class="max-w-md space-y-4">
		<Alert message={loadError} />
		<Button variant="secondary" onclick={loadProfile}>Reintentar</Button>
	</div>
{:else if profile}
	<div class="space-y-1.5">
		<h1 class="text-2xl font-semibold tracking-tight text-zinc-50">Hola, {firstName}</h1>
		<p class="text-sm text-zinc-400">Este es el resumen de tu cuenta.</p>
	</div>

	<div class="mt-8 grid gap-6 lg:grid-cols-[minmax(0,1fr)_minmax(0,1.4fr)]">
		<section class="rounded-lg border border-zinc-800 bg-zinc-900/40">
			<h2 class="border-b border-zinc-800 px-5 py-3.5 text-sm font-medium text-zinc-200">Perfil</h2>
			<dl class="divide-y divide-zinc-800/70 text-sm">
				<div class="flex justify-between gap-4 px-5 py-3">
					<dt class="text-zinc-500">Email</dt>
					<dd class="truncate text-zinc-200">{profile.email}</dd>
				</div>
				<div class="flex justify-between gap-4 px-5 py-3">
					<dt class="text-zinc-500">Moneda principal</dt>
					<dd class="font-mono text-zinc-200">{profile.default_currency}</dd>
				</div>
				<div class="flex justify-between gap-4 px-5 py-3">
					<dt class="text-zinc-500">Zona horaria</dt>
					<dd class="font-mono text-zinc-200">{profile.timezone}</dd>
				</div>
				<div class="flex justify-between gap-4 px-5 py-3">
					<dt class="text-zinc-500">Telegram</dt>
					<dd class={profile.telegram_linked ? 'text-emerald-400' : 'text-zinc-400'}>
						{profile.telegram_linked ? 'Vinculado' : 'Sin vincular'}
					</dd>
				</div>
				<div class="flex justify-between gap-4 px-5 py-3">
					<dt class="text-zinc-500">Miembro desde</dt>
					<dd class="text-zinc-200">{memberSince}</dd>
				</div>
			</dl>
		</section>

		<section
			class="flex flex-col justify-center rounded-lg border border-dashed border-zinc-800 px-6 py-10"
		>
			<p class="font-mono text-xs tracking-wider text-zinc-500 uppercase">Próximamente</p>
			<h2 class="mt-2 text-base font-medium text-zinc-200">Finanzas y hábitos</h2>
			<p class="mt-1.5 max-w-sm text-sm text-zinc-400">
				Aquí verás tu balance del mes, tus transacciones recientes y el progreso de tus hábitos.
			</p>
		</section>
	</div>
{/if}
