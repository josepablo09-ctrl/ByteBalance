<script>
	import { goto } from '$app/navigation';
	import { resolve } from '$app/paths';
	import Button from '#lib/components/Button.svelte';
	import Logo from '#lib/components/Logo.svelte';
	import { session } from '#lib/stores/session.js';

	/** @type {{ children: import('svelte').Snippet }} */
	let { children } = $props();

	// Si la sesión termina (logout o token rechazado por la API), vuelve al login.
	$effect(() => {
		if (!$session) goto(resolve('/(auth)/login'), { replaceState: true });
	});
</script>

{#if $session}
	<div class="min-h-dvh">
		<header class="border-b border-zinc-800">
			<div class="mx-auto flex h-14 max-w-5xl items-center justify-between px-6">
				<a href={resolve('/(app)')} aria-label="Inicio">
					<Logo />
				</a>
				<Button variant="ghost" class="h-8 px-3" onclick={() => session.clear()}>
					Cerrar sesión
				</Button>
			</div>
		</header>

		<main class="mx-auto max-w-5xl px-6 py-10">
			{@render children()}
		</main>
	</div>
{/if}
