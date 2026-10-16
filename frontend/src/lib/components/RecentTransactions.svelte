<script>
	import { onMount } from 'svelte';
	import { errorMessage, financeApi } from '#lib/api/index.js';
	import { monthRange } from '#lib/dates.js';
	import Alert from './Alert.svelte';
	import Button from './Button.svelte';
	import Icon from './Icon.svelte';
	import SegmentedControl from './SegmentedControl.svelte';
	import TransactionList from './TransactionList.svelte';

	const PAGE_SIZE = 20;

	const TYPE_FILTERS = [
		{ value: 'all', label: 'Todas' },
		{ value: 'income', label: 'Ingresos' },
		{ value: 'expense', label: 'Gastos' }
	];

	const PERIOD_FILTERS = [
		{ value: 'month', label: 'Este mes' },
		{ value: 'previous', label: 'Mes anterior' },
		{ value: 'all', label: 'Todo' }
	];

	/**
	 * Transacciones recientes con filtros rápidos por tipo y periodo, y carga
	 * incremental de páginas.
	 *
	 * @typedef {object} Props
	 * @property {import('#lib/api/finance.js').Category[]} categories
	 * @property {string} timeZone
	 * @property {() => void} onadd Abre el formulario de nueva transacción.
	 * @property {(transaction: import('#lib/api/finance.js').Transaction) => void} onedit
	 * @property {(transaction: import('#lib/api/finance.js').Transaction) => void} ondelete
	 */

	/** @type {Props} */
	let { categories, timeZone, onadd, onedit, ondelete } = $props();

	const DEFAULT_TYPE = 'all';
	const DEFAULT_PERIOD = 'month';

	let typeFilter = $state(DEFAULT_TYPE);
	let periodFilter = $state(DEFAULT_PERIOD);

	/** @type {import('#lib/api/finance.js').Transaction[]} */
	let transactions = $state([]);
	let isLoading = $state(true);
	let isLoadingMore = $state(false);
	let hasMore = $state(false);
	let loadError = $state('');
	let requestId = 0;

	const categoriesById = $derived(new Map(categories.map((c) => [c.id, c])));
	// Con los filtros por defecto (todo, este mes) la lista vacía invita a registrar;
	// con otros, sugiere cambiar el filtro.
	const isDefaultView = $derived(typeFilter === DEFAULT_TYPE && periodFilter === DEFAULT_PERIOD);
	const emptyState = $derived(
		isDefaultView
			? {
					title: 'No hay transacciones este mes',
					text: 'Registra un ingreso o un gasto para ver tu balance.'
				}
			: typeFilter === 'all' && periodFilter === 'all'
				? {
						title: 'Aún no registras transacciones',
						text: 'Registra tu primer ingreso o gasto para ver tu balance.'
					}
				: {
						title: 'No hay transacciones con estos filtros',
						text: 'Prueba con otro periodo o tipo de movimiento.'
					}
	);
	const canAddFromEmpty = $derived(typeFilter === 'all' && periodFilter !== 'previous');

	/** @returns {import('#lib/api/finance.js').TransactionFilter} */
	function currentFilter() {
		/** @type {import('#lib/api/finance.js').TransactionFilter} */
		const filter = { limit: PAGE_SIZE };
		if (typeFilter === 'income' || typeFilter === 'expense') filter.type = typeFilter;
		if (periodFilter !== 'all') {
			Object.assign(filter, monthRange(timeZone, new Date(), periodFilter === 'previous' ? -1 : 0));
		}
		return filter;
	}

	/** Recarga la primera página con los filtros actuales. */
	export async function refresh() {
		const current = ++requestId;
		isLoading = true;
		loadError = '';
		try {
			const page = await financeApi.listTransactions(currentFilter());
			if (current !== requestId) return;
			transactions = page;
			hasMore = page.length === PAGE_SIZE;
		} catch (err) {
			if (current === requestId) loadError = errorMessage(err);
		} finally {
			if (current === requestId) isLoading = false;
		}
	}

	async function loadMore() {
		const current = requestId;
		isLoadingMore = true;
		try {
			const page = await financeApi.listTransactions({
				...currentFilter(),
				offset: transactions.length
			});
			if (current !== requestId) return;
			transactions = [...transactions, ...page];
			hasMore = page.length === PAGE_SIZE;
		} catch (err) {
			if (current === requestId) loadError = errorMessage(err);
		} finally {
			isLoadingMore = false;
		}
	}

	onMount(refresh);
</script>

<section aria-labelledby="recent-title" class="rounded-lg border border-zinc-800 bg-zinc-900/20">
	<div
		class="flex flex-col gap-3 border-b border-zinc-800 px-4 py-3.5 sm:flex-row sm:items-center sm:justify-between sm:px-5"
	>
		<h2 id="recent-title" class="text-sm font-medium text-zinc-200">Transacciones recientes</h2>
		<div class="flex flex-wrap gap-2">
			<SegmentedControl
				label="Filtrar por tipo"
				options={TYPE_FILTERS}
				bind:value={typeFilter}
				onchange={refresh}
			/>
			<SegmentedControl
				label="Filtrar por periodo"
				options={PERIOD_FILTERS}
				bind:value={periodFilter}
				onchange={refresh}
			/>
		</div>
	</div>

	{#if isLoading}
		<ul class="divide-y divide-zinc-800/70" aria-busy="true" aria-label="Cargando transacciones">
			{#each { length: 4 }, i (i)}
				<li class="flex animate-pulse items-center gap-3 px-4 py-3 sm:px-5">
					<span class="size-8 rounded-md bg-zinc-800/70"></span>
					<span class="flex-1 space-y-1.5">
						<span class="block h-3.5 w-40 rounded bg-zinc-800/70"></span>
						<span class="block h-3 w-24 rounded bg-zinc-800/50"></span>
					</span>
					<span class="h-3.5 w-20 rounded bg-zinc-800/70"></span>
				</li>
			{/each}
		</ul>
	{:else if loadError && transactions.length === 0}
		<div class="flex flex-wrap items-center gap-3 p-5">
			<Alert message={loadError} />
			<Button variant="secondary" class="h-9" onclick={refresh}>Reintentar</Button>
		</div>
	{:else if transactions.length === 0}
		<div class="flex flex-col items-center px-6 py-14 text-center">
			<span
				class="grid size-10 place-items-center rounded-lg border border-zinc-800 bg-zinc-900 text-zinc-500"
				aria-hidden="true"
			>
				<Icon name="receipt" class="size-5" />
			</span>
			<p class="mt-4 text-sm font-medium text-zinc-200">
				{emptyState.title}
			</p>
			<p class="mt-1 max-w-xs text-sm text-zinc-500">{emptyState.text}</p>
			{#if canAddFromEmpty}
				<Button class="mt-5" onclick={onadd}>
					<Icon name="plus" />
					Nueva transacción
				</Button>
			{/if}
		</div>
	{:else}
		<TransactionList {transactions} {categoriesById} {timeZone} {onedit} {ondelete} />
		{#if loadError}
			<div class="border-t border-zinc-800 px-5 py-3"><Alert message={loadError} /></div>
		{/if}
		{#if hasMore}
			<div class="border-t border-zinc-800 px-4 py-3 text-center">
				<Button variant="ghost" class="h-8 px-3" loading={isLoadingMore} onclick={loadMore}>
					Cargar más
				</Button>
			</div>
		{/if}
	{/if}
</section>
