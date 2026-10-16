<script>
	import { formatShortDate } from '#lib/dates.js';
	import { formatMoney } from '#lib/money.js';
	import Icon from './Icon.svelte';

	/**
	 * Filas de transacciones, más recientes primero.
	 *
	 * @typedef {object} Props
	 * @property {import('#lib/api/finance.js').Transaction[]} transactions
	 * @property {Map<string, import('#lib/api/finance.js').Category>} categoriesById
	 * @property {string} timeZone
	 */

	/** @type {Props} */
	let { transactions, categoriesById, timeZone } = $props();
</script>

<ul class="divide-y divide-zinc-800/70">
	{#each transactions as transaction (transaction.id)}
		{@const isIncome = transaction.type === 'income'}
		{@const category = transaction.category_id
			? categoriesById.get(transaction.category_id)
			: undefined}
		<li class="flex items-center gap-3 px-4 py-3 sm:px-5">
			<span
				class={[
					'grid size-8 shrink-0 place-items-center rounded-md border border-zinc-800 bg-zinc-900',
					isIncome ? 'text-emerald-400' : 'text-zinc-400'
				]}
				aria-hidden="true"
			>
				<Icon name={isIncome ? 'income' : 'expense'} />
			</span>
			<div class="min-w-0 flex-1">
				<p class="truncate text-sm text-zinc-200">
					{transaction.description || category?.name || (isIncome ? 'Ingreso' : 'Gasto')}
				</p>
				<p class="truncate text-xs text-zinc-500">
					{category?.name ?? 'Sin categoría'} · {formatShortDate(transaction.occurred_at, timeZone)}
				</p>
			</div>
			<p
				class={[
					'shrink-0 text-sm font-medium tabular-nums',
					isIncome ? 'text-emerald-400' : 'text-zinc-200'
				]}
			>
				<span class="sr-only">{isIncome ? 'Ingreso de' : 'Gasto de'}</span>
				{formatMoney(
					isIncome ? transaction.amount_cents : -transaction.amount_cents,
					transaction.currency,
					{
						signDisplay: 'always'
					}
				)}
			</p>
		</li>
	{/each}
</ul>
