<script>
	import { onMount } from 'svelte';
	import { errorMessage, financeApi } from '#lib/api/index.js';
	import Alert from '#lib/components/Alert.svelte';
	import Button from '#lib/components/Button.svelte';
	import FinanceSummary from '#lib/components/FinanceSummary.svelte';
	import Icon from '#lib/components/Icon.svelte';
	import Modal from '#lib/components/Modal.svelte';
	import RecentTransactions from '#lib/components/RecentTransactions.svelte';
	import TransactionForm from '#lib/components/TransactionForm.svelte';
	import { formatShortDate } from '#lib/dates.js';
	import { formatMoney } from '#lib/money.js';
	import { profile } from '#lib/stores/profile.js';

	const NOTICE_DURATION_MS = 4000;

	// El layout solo muestra esta página con el perfil cargado.
	const user = $derived(/** @type {import('#lib/api/auth.js').User} */ ($profile.user));

	/** @type {import('#lib/api/finance.js').Category[]} */
	let categories = $state([]);
	let categoriesError = $state('');
	let isFormOpen = $state(false);
	/** @type {import('#lib/api/finance.js').Transaction | null} */
	let editTarget = $state(null);
	/** @type {import('#lib/api/finance.js').Transaction | null} */
	let deleteTarget = $state(null);
	let isDeleting = $state(false);
	let deleteError = $state('');
	let notice = $state('');
	/** @type {ReturnType<typeof setTimeout> | undefined} */
	let noticeTimer;

	/** @type {FinanceSummary | undefined} */
	let summary = $state();
	/** @type {RecentTransactions | undefined} */
	let recent = $state();

	async function loadCategories() {
		categoriesError = '';
		try {
			categories = await financeApi.listCategories();
		} catch (err) {
			categoriesError = errorMessage(err);
		}
	}

	/** @param {import('#lib/api/finance.js').RecordTransactionInput} input */
	async function handleRecord(input) {
		await financeApi.recordTransaction(input);
		isFormOpen = false;
		showNotice(input.type === 'income' ? 'Ingreso registrado.' : 'Gasto registrado.');
		refreshAll();
	}

	/** Vuelve a pedir el resumen y la lista tras un cambio. */
	function refreshAll() {
		summary?.refresh();
		recent?.refresh();
	}

	/** @param {import('#lib/api/finance.js').RecordTransactionInput} input */
	async function handleEdit(input) {
		if (!editTarget) return;
		await financeApi.updateTransaction(editTarget.id, input);
		editTarget = null;
		showNotice('Transacción actualizada.');
		refreshAll();
	}

	/** @param {import('#lib/api/finance.js').Transaction} transaction */
	function askDelete(transaction) {
		deleteError = '';
		deleteTarget = transaction;
	}

	async function confirmDelete() {
		if (!deleteTarget) return;
		isDeleting = true;
		deleteError = '';
		try {
			await financeApi.deleteTransaction(deleteTarget.id);
			deleteTarget = null;
			showNotice('Transacción eliminada.');
			refreshAll();
		} catch (err) {
			deleteError = errorMessage(err);
		} finally {
			isDeleting = false;
		}
	}

	/**
	 * @param {string} name
	 * @param {import('#lib/api/finance.js').EntryType} type
	 */
	async function handleCreateCategory(name, type) {
		const category = await financeApi.createCategory(name, type);
		categories = [...categories, category];
		return category;
	}

	/** @param {string} message */
	function showNotice(message) {
		notice = message;
		clearTimeout(noticeTimer);
		noticeTimer = setTimeout(() => (notice = ''), NOTICE_DURATION_MS);
	}

	onMount(() => {
		loadCategories();
		return () => clearTimeout(noticeTimer);
	});
</script>

<svelte:head>
	<title>Finanzas · ByteBalance</title>
</svelte:head>

<div class="space-y-8">
	<div class="flex flex-wrap items-end justify-between gap-4">
		<div>
			<h1 class="text-2xl font-semibold tracking-tight text-zinc-50">Finanzas</h1>
			<p class="mt-1 text-sm text-zinc-400">Tu balance, tus ingresos y tus gastos.</p>
		</div>
		<Button onclick={() => (isFormOpen = true)}>
			<Icon name="plus" />
			Nueva transacción
		</Button>
	</div>

	{#if notice}
		<Alert message={notice} tone="success" />
	{/if}
	{#if categoriesError}
		<Alert message={`No se pudieron cargar las categorías. ${categoriesError}`} />
	{/if}

	<FinanceSummary bind:this={summary} currency={user.default_currency} timeZone={user.timezone} />

	<RecentTransactions
		bind:this={recent}
		{categories}
		timeZone={user.timezone}
		onadd={() => (isFormOpen = true)}
		onedit={(transaction) => (editTarget = transaction)}
		ondelete={askDelete}
	/>
</div>

<Modal
	bind:open={isFormOpen}
	title="Nueva transacción"
	description="Registra un ingreso o un gasto."
>
	<TransactionForm
		{categories}
		defaultCurrency={user.default_currency}
		timeZone={user.timezone}
		onSubmit={handleRecord}
		onCreateCategory={handleCreateCategory}
		oncancel={() => (isFormOpen = false)}
	/>
</Modal>

<Modal
	bind:open={() => editTarget !== null, (open) => !open && (editTarget = null)}
	title="Editar transacción"
	description="Corrige el monto, la categoría, la descripción o la fecha."
>
	{#if editTarget}
		<TransactionForm
			{categories}
			defaultCurrency={user.default_currency}
			timeZone={user.timezone}
			transaction={editTarget}
			onSubmit={handleEdit}
			onCreateCategory={handleCreateCategory}
			oncancel={() => (editTarget = null)}
		/>
	{/if}
</Modal>

<Modal
	bind:open={() => deleteTarget !== null, (open) => !open && (deleteTarget = null)}
	title="Eliminar transacción"
>
	{#if deleteTarget}
		{@const isIncome = deleteTarget.type === 'income'}
		<div class="space-y-4">
			{#if deleteError}
				<Alert message={deleteError} />
			{/if}
			<div class="rounded-lg border border-zinc-800 bg-zinc-900/40 px-4 py-3">
				<p class="text-sm text-zinc-200">
					{deleteTarget.description || (isIncome ? 'Ingreso' : 'Gasto')}
				</p>
				<p class="mt-0.5 text-xs text-zinc-500">
					{formatShortDate(deleteTarget.occurred_at, user.timezone)} ·
					<span class={isIncome ? 'text-emerald-400' : 'text-zinc-300'}>
						{formatMoney(
							isIncome ? deleteTarget.amount_cents : -deleteTarget.amount_cents,
							deleteTarget.currency,
							{ signDisplay: 'always' }
						)}
					</span>
				</p>
			</div>
			<p class="text-sm text-zinc-400">
				Dejará de contar en tu balance. Esta acción no se puede deshacer.
			</p>
		</div>
		<div class="mt-6 flex justify-end gap-2 border-t border-zinc-800 pt-5">
			<Button variant="secondary" onclick={() => (deleteTarget = null)}>Cancelar</Button>
			<Button variant="danger" loading={isDeleting} onclick={confirmDelete}>Eliminar</Button>
		</div>
	{/if}
</Modal>
