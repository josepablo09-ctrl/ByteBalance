<script>
	import { goto } from '$app/navigation';
	import { resolve } from '$app/paths';
	import Alert from '#lib/components/Alert.svelte';
	import AppHeader from '#lib/components/AppHeader.svelte';
	import Button from '#lib/components/Button.svelte';
	import Sidebar from '#lib/components/Sidebar.svelte';
	import { lastInsight } from '#lib/stores/insight.js';
	import { profile } from '#lib/stores/profile.js';
	import { session } from '#lib/stores/session.js';

	/** @type {{ children: import('svelte').Snippet }} */
	let { children } = $props();

	let isMenuOpen = $state(false);

	// Si la sesión termina (logout o token rechazado por la API), se olvida el
	// perfil (y su último análisis) y se vuelve al login.
	$effect(() => {
		if (!$session) {
			profile.reset();
			lastInsight.reset();
			goto(resolve('/(auth)/login'), { replaceState: true });
		}
	});

	$effect(() => {
		if ($session && $profile.status === 'idle') profile.load();
	});
</script>

<svelte:window onkeydown={(e) => e.key === 'Escape' && (isMenuOpen = false)} />

{#if $session}
	<div class="min-h-dvh lg:grid lg:grid-cols-[15rem_minmax(0,1fr)]">
		<Sidebar open={isMenuOpen} onclose={() => (isMenuOpen = false)} />

		<div class="flex min-w-0 flex-col">
			<AppHeader
				user={$profile.user}
				{isMenuOpen}
				onmenu={() => (isMenuOpen = true)}
				onlogout={() => session.clear()}
			/>

			<main class="mx-auto w-full max-w-6xl flex-1 px-4 py-8 sm:px-6 lg:px-8">
				{#if $profile.status === 'ready'}
					{@render children()}
				{:else if $profile.status === 'error'}
					<div class="max-w-md space-y-4">
						<Alert message={$profile.error} />
						<Button variant="secondary" onclick={() => profile.load()}>Reintentar</Button>
					</div>
				{:else}
					<div class="animate-pulse space-y-3" aria-busy="true" aria-label="Cargando">
						<div class="h-7 w-48 rounded-md bg-zinc-900"></div>
						<div class="h-4 w-72 rounded-md bg-zinc-900"></div>
					</div>
				{/if}
			</main>
		</div>
	</div>
{/if}
