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
	import { profile } from '#lib/stores/profile.js';

	const NOTICE_DURATION_MS = 4000;

	// El layout solo muestra esta página con el perfil cargado.
	const user = $derived(/** @type {import('#lib/api/auth.js').User} */ ($profile.user));

	/** @type {import('#lib/api/finance.js').Category[]} */
	let categories = $state([]);
	let categoriesError = $state('');
	let isFormOpen = $state(false);
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
		summary?.refresh();
		recent?.refresh();
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
