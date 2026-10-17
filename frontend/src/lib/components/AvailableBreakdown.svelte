<script>
	import { formatMoney } from '#lib/money.js';

	/**
	 * Separa el balance total en lo que se puede gastar y lo apartado en el
	 * ahorro: disponible = balance total − ahorro acumulado.
	 *
	 * @typedef {object} Props
	 * @property {number} netCents Balance total (ingresos − gastos).
	 * @property {number} savingsCents Ahorro acumulado.
	 * @property {number} availableCents Disponible real para gastar.
	 * @property {string} currency
	 * @property {boolean} [isLoading]
	 */

	/** @type {Props} */
	let { netCents, savingsCents, availableCents, currency, isLoading = false } = $props();

	// Parte del balance que está apartada; solo tiene sentido con un balance positivo.
	const savingsShare = $derived(
		netCents > 0 ? Math.min(100, Math.max(0, (savingsCents / netCents) * 100)) : null
	);
</script>

<section
	aria-labelledby="available-title"
	class="rounded-lg border border-zinc-800 bg-zinc-900/40 px-5 py-4"
>
	<div class="flex flex-wrap items-end justify-between gap-x-6 gap-y-3">
		<div>
			<h2 id="available-title" class="text-sm text-zinc-400">Disponible para gastar</h2>
			{#if isLoading}
				<div
					class="mt-2.5 h-8 w-44 animate-pulse rounded-md bg-zinc-800/70"
					aria-hidden="true"
				></div>
			{:else}
				<p
					class={[
						'mt-1.5 text-3xl font-semibold tracking-tight tabular-nums',
						availableCents < 0 ? 'text-red-400' : 'text-emerald-400'
					]}
				>
					{formatMoney(availableCents, currency)}
				</p>
			{/if}
		</div>

		{#if !isLoading}
			<dl class="grid grid-cols-[auto_auto] gap-x-4 gap-y-1 text-sm tabular-nums">
				<dt class="text-zinc-500">Balance total</dt>
				<dd class="text-right text-zinc-200">{formatMoney(netCents, currency)}</dd>
				<dt class="text-zinc-500">Apartado en ahorro</dt>
				<dd class="text-right text-zinc-200">
					{formatMoney(-savingsCents, currency, { signDisplay: 'exceptZero' })}
				</dd>
				<dt class="border-t border-zinc-800 pt-1 text-zinc-400">Disponible</dt>
				<dd class="border-t border-zinc-800 pt-1 text-right font-medium text-zinc-50">
					{formatMoney(availableCents, currency)}
				</dd>
			</dl>
		{/if}
	</div>

	{#if !isLoading && savingsShare !== null}
		<div class="mt-4 flex h-2 overflow-hidden rounded-full bg-zinc-800" aria-hidden="true">
			<div class="h-full bg-emerald-500/80" style:width={`${100 - savingsShare}%`}></div>
			<div class="h-full bg-zinc-500" style:width={`${savingsShare}%`}></div>
		</div>
		<div class="mt-2 flex flex-wrap gap-x-4 gap-y-1 text-xs text-zinc-500">
			<span class="flex items-center gap-1.5">
				<span class="size-2 rounded-full bg-emerald-500/80" aria-hidden="true"></span>
				Para gastar
			</span>
			<span class="flex items-center gap-1.5">
				<span class="size-2 rounded-full bg-zinc-500" aria-hidden="true"></span>
				Ahorro (no se cuenta como disponible)
			</span>
		</div>
	{/if}

	{#if !isLoading && availableCents < 0}
		<p class="mt-3 text-xs text-red-300">
			Apartaste más de lo que tienes disponible. Registra tus ingresos o retira parte del ahorro.
		</p>
	{/if}
</section>
