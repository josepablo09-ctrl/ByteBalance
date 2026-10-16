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
	 * @property {(transaction: import('#lib/api/finance.js').Transaction) => void} onedit
	 * @property {(transaction: import('#lib/api/finance.js').Transaction) => void} ondelete
	 */

	/** @type {Props} */
	let { transactions, categoriesById, timeZone, onedit, ondelete } = $props();

	const ACTION_CLASS =
		'grid size-8 place-items-center rounded-md text-zinc-500 transition-colors hover:bg-zinc-800/70 focus-visible:ring-2 focus-visible:ring-emerald-500/40 focus-visible:outline-none';
</script>

<ul class="divide-y divide-zinc-800/70">
	{#each transactions as transaction (transaction.id)}
		{@const isIncome = transaction.type === 'income'}
		{@const category = transaction.category_id
			? categoriesById.get(transaction.category_id)
			: undefined}
		{@const title = transaction.description || category?.name || (isIncome ? 'Ingreso' : 'Gasto')}
		<li class="flex items-center gap-3 py-3 pr-2 pl-4 sm:pl-5">
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
				<p class="truncate text-sm text-zinc-200">{title}</p>
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
			<div class="flex shrink-0">
				<button
					type="button"
					class={[ACTION_CLASS, 'hover:text-zinc-100']}
					aria-label="Editar {title}"
					onclick={() => onedit(transaction)}
				>
					<Icon name="edit" />
				</button>
				<button
					type="button"
					class={[ACTION_CLASS, 'hover:text-red-300']}
					aria-label="Eliminar {title}"
					onclick={() => ondelete(transaction)}
				>
					<Icon name="trash" />
				</button>
			</div>
		</li>
	{/each}
</ul>
