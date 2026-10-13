<script>
	import { resolve } from '$app/paths';
	import { page } from '$app/state';
	import Logo from '#lib/components/Logo.svelte';

	const isNotFound = $derived(page.status === 404);
</script>

<svelte:head>
	<title>{isNotFound ? 'Página no encontrada' : 'Error'} · ByteBalance</title>
</svelte:head>

<div class="flex min-h-dvh flex-col items-center justify-center px-6 text-center">
	<Logo />
	<p class="mt-10 font-mono text-sm text-emerald-400">{page.status}</p>
	<h1 class="mt-2 text-2xl font-semibold tracking-tight text-zinc-50">
		{isNotFound ? 'Esta página no existe' : 'Algo salió mal'}
	</h1>
	<p class="mt-2 max-w-sm text-sm text-zinc-400">
		{isNotFound
			? 'Revisa la dirección o vuelve al inicio.'
			: 'Ocurrió un error inesperado. Inténtalo de nuevo en unos minutos.'}
	</p>
	<a
		href={resolve('/(app)')}
		class="mt-8 inline-flex h-10 items-center rounded-lg border border-zinc-800 bg-zinc-900 px-4 text-sm font-medium text-zinc-200 transition-colors hover:border-zinc-700 hover:bg-zinc-800/70"
	>
		Volver al inicio
	</a>
</div>
