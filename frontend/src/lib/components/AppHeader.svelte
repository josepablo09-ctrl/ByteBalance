<script>
	import Icon from './Icon.svelte';

	/**
	 * Barra superior: botón de menú (móvil), perfil del usuario y cierre de sesión.
	 *
	 * @typedef {object} Props
	 * @property {import('#lib/api/auth.js').User | null} user Nulo mientras carga.
	 * @property {boolean} isMenuOpen
	 * @property {() => void} onmenu
	 * @property {() => void} onlogout
	 */

	/** @type {Props} */
	let { user, isMenuOpen, onmenu, onlogout } = $props();

	const initials = $derived(
		(user?.full_name ?? '')
			.split(/\s+/)
			.filter(Boolean)
			.slice(0, 2)
			.map((word) => word[0].toUpperCase())
			.join('')
	);
</script>

<header
	class="sticky top-0 z-20 flex h-14 shrink-0 items-center gap-3 border-b border-zinc-800 bg-zinc-950/95 px-4 backdrop-blur sm:px-6 lg:px-8"
>
	<button
		type="button"
		onclick={onmenu}
		aria-label="Abrir menú"
		aria-controls="app-sidebar"
		aria-expanded={isMenuOpen}
		class="-ml-1.5 rounded-md p-1.5 text-zinc-400 transition-colors hover:bg-zinc-900 hover:text-zinc-100 focus-visible:ring-2 focus-visible:ring-emerald-500/40 focus-visible:outline-none lg:hidden"
	>
		<Icon name="menu" class="size-5" />
	</button>

	<div class="ml-auto flex items-center gap-3">
		{#if user}
			<div class="hidden text-right sm:block">
				<p class="text-sm leading-tight font-medium text-zinc-200">{user.full_name}</p>
				<p class="text-xs leading-tight text-zinc-500">{user.email}</p>
			</div>
			<span
				class="grid size-8 place-items-center rounded-md border border-zinc-800 bg-zinc-900 text-xs font-medium text-zinc-300"
				aria-hidden="true"
			>
				{initials}
			</span>
		{:else}
			<div class="h-8 w-40 animate-pulse rounded-md bg-zinc-900" aria-hidden="true"></div>
		{/if}

		<span class="h-5 w-px bg-zinc-800" aria-hidden="true"></span>

		<button
			type="button"
			onclick={onlogout}
			class="inline-flex h-8 items-center gap-2 rounded-lg px-2.5 text-sm text-zinc-400 transition-colors hover:bg-zinc-900 hover:text-zinc-100 focus-visible:ring-2 focus-visible:ring-emerald-500/40 focus-visible:outline-none"
		>
			<Icon name="logout" />
			<span class="hidden sm:inline">Cerrar sesión</span>
			<span class="sr-only sm:hidden">Cerrar sesión</span>
		</button>
	</div>
</header>
