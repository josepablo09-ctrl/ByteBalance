<script>
	import { errorMessage } from '#lib/api/errors.js';
	import { currencyOptions } from '#lib/currencies.js';
	import { todayInZone } from '#lib/dates.js';
	import { ENTRY_TYPES } from '#lib/entryTypes.js';
	import { centsToDecimal } from '#lib/money.js';
	import {
		buildTransactionInput,
		MAX_CATEGORY_NAME_LENGTH,
		MAX_DESCRIPTION_LENGTH,
		validateCategoryName
	} from '#lib/transactions.js';
	import Alert from './Alert.svelte';
	import Button from './Button.svelte';
	import SegmentedControl from './SegmentedControl.svelte';
	import SelectField from './SelectField.svelte';
	import TextField from './TextField.svelte';

	/** Valor del selector de categoría que abre la creación de una nueva. */
	const NEW_CATEGORY = '__new__';

	const TYPE_OPTIONS = /** @type {const} */ (['expense', 'income', 'saving', 'withdrawal']).map(
		(value) => ({ value, label: ENTRY_TYPES[value].label })
	);

	/** Explicación de los movimientos de ahorro, que no son ingresos ni gastos. */
	const SAVINGS_HINTS = {
		saving:
			'Sale de tu disponible para gastar y suma a tu ahorro. Tu balance total no cambia: el dinero sigue siendo tuyo.',
		withdrawal: 'Sale de tu ahorro y vuelve a tu disponible para gastar.'
	};

	/**
	 * @typedef {import('#lib/api/finance.js').Category} Category
	 * @typedef {import('#lib/api/finance.js').EntryType} EntryType
	 *
	 * @typedef {object} Props
	 * @property {Category[]} categories
	 * @property {string} defaultCurrency
	 * @property {string} timeZone Zona horaria del usuario.
	 * @property {import('#lib/api/finance.js').Transaction} [transaction] Si se indica, se edita.
	 * @property {(input: import('#lib/api/finance.js').RecordTransactionInput) => Promise<void>} onSubmit
	 *   Registra (o guarda los cambios de) la transacción; si falla debe lanzar el error de la API.
	 * @property {(name: string, type: import('#lib/api/finance.js').CategoryType) => Promise<Category>} onCreateCategory
	 * @property {() => void} oncancel
	 */

	/** @type {Props} */
	let { categories, defaultCurrency, timeZone, transaction, onSubmit, onCreateCategory, oncancel } =
		$props();

	const today = $derived(todayInZone(timeZone));

	// Los valores iniciales se toman una vez: después pertenecen al formulario.
	// svelte-ignore state_referenced_locally
	const initial = transaction;
	// svelte-ignore state_referenced_locally
	const originalDate = initial ? todayInZone(timeZone, new Date(initial.occurred_at)) : undefined;

	let type = $state(/** @type {EntryType} */ (initial?.type ?? 'expense'));
	let amount = $state(initial ? centsToDecimal(initial.amount_cents) : '');
	// svelte-ignore state_referenced_locally
	let currency = $state(initial?.currency ?? defaultCurrency);
	let categoryId = $state(initial?.category_id ?? '');
	let description = $state(initial?.description ?? '');
	// svelte-ignore state_referenced_locally
	let date = $state(originalDate ?? todayInZone(timeZone));

	/** @type {Partial<Record<string, string>>} */
	let errors = $state({});
	let formError = $state('');
	let isSubmitting = $state(false);

	let isCreatingCategory = $state(false);
	let newCategoryName = $state('');
	let categoryError = $state('');
	let isSavingCategory = $state(false);

	const categoryOptions = $derived([
		{ value: '', label: 'Sin categoría' },
		...categories.filter((c) => c.type === type).map((c) => ({ value: c.id, label: c.name })),
		{ value: NEW_CATEGORY, label: '+ Nueva categoría…' }
	]);

	const typeInfo = $derived(ENTRY_TYPES[type]);

	/** @param {string} next */
	function handleTypeChange(next) {
		type = /** @type {EntryType} */ (next);
		// Una categoría de gastos no aplica a un ingreso, y viceversa; el ahorro no lleva categoría.
		if (!categories.some((c) => c.id === categoryId && c.type === type)) categoryId = '';
		isCreatingCategory = false;
	}

	/** @param {string} value */
	function handleCategoryChange(value) {
		if (value !== NEW_CATEGORY) return;
		categoryId = '';
		isCreatingCategory = true;
		newCategoryName = '';
		categoryError = '';
	}

	async function createCategory() {
		categoryError = validateCategoryName(newCategoryName) ?? '';
		if (categoryError) return;
		isSavingCategory = true;
		try {
			const category = await onCreateCategory(
				newCategoryName.trim(),
				/** @type {import('#lib/api/finance.js').CategoryType} */ (type)
			);
			categoryId = category.id;
			isCreatingCategory = false;
		} catch (err) {
			categoryError = errorMessage(err);
		} finally {
			isSavingCategory = false;
		}
	}

	/** @param {KeyboardEvent} event */
	function handleCategoryKeydown(event) {
		// Enter crea la categoría en lugar de enviar el formulario completo.
		if (event.key === 'Enter') {
			event.preventDefault();
			createCategory();
		}
	}

	/** @param {SubmitEvent} event */
	async function handleSubmit(event) {
		event.preventDefault();
		formError = '';
		const result = buildTransactionInput(
			{ type, amount, currency, categoryId, description, date },
			{ timeZone, today, originalDate }
		);
		if (result.errors) {
			errors = result.errors;
			return;
		}
		errors = {};
		isSubmitting = true;
		try {
			await onSubmit(result.input);
		} catch (err) {
			formError = errorMessage(err);
		} finally {
			isSubmitting = false;
		}
	}
</script>

<form onsubmit={handleSubmit} novalidate class="space-y-5">
	{#if formError}
		<Alert message={formError} />
	{/if}

	<SegmentedControl
		label="Tipo de movimiento"
		options={TYPE_OPTIONS}
		value={type}
		onchange={handleTypeChange}
		fullWidth
	/>

	<div class="grid grid-cols-[minmax(0,1fr)_7.5rem] gap-3">
		<TextField
			id="amount"
			label="Monto"
			inputmode="decimal"
			placeholder="0.00"
			bind:value={amount}
			error={errors.amount}
			oninput={() => (errors.amount = undefined)}
		/>
		<SelectField
			id="currency"
			label="Moneda"
			options={currencyOptions(initial?.currency ?? defaultCurrency).map((c) => ({
				value: c.value,
				label: c.value
			}))}
			bind:value={currency}
			error={errors.currency}
		/>
	</div>

	{#if type === 'saving' || type === 'withdrawal'}
		<p
			class="rounded-lg border border-zinc-800 bg-zinc-900/40 px-3 py-2.5 text-xs leading-relaxed text-zinc-400"
		>
			{SAVINGS_HINTS[type]}
		</p>
	{:else if isCreatingCategory}
		<div class="space-y-2 rounded-lg border border-zinc-800 bg-zinc-900/40 p-3">
			<TextField
				id="new_category"
				label={type === 'income' ? 'Nueva categoría de ingresos' : 'Nueva categoría de gastos'}
				placeholder={type === 'income' ? 'Salario' : 'Supermercado'}
				maxlength={MAX_CATEGORY_NAME_LENGTH}
				bind:value={newCategoryName}
				error={categoryError}
				oninput={() => (categoryError = '')}
				onkeydown={handleCategoryKeydown}
			/>
			<div class="flex justify-end gap-2">
				<Button variant="ghost" class="h-8 px-3" onclick={() => (isCreatingCategory = false)}>
					Cancelar
				</Button>
				<Button
					variant="secondary"
					class="h-8 px-3"
					loading={isSavingCategory}
					onclick={createCategory}
				>
					Crear categoría
				</Button>
			</div>
		</div>
	{:else}
		<SelectField
			id="category"
			label="Categoría"
			options={categoryOptions}
			bind:value={categoryId}
			onchange={handleCategoryChange}
		/>
	{/if}

	<TextField
		id="description"
		label="Descripción"
		placeholder={typeInfo.placeholder}
		maxlength={MAX_DESCRIPTION_LENGTH}
		bind:value={description}
		error={errors.description}
		hint="Opcional."
	/>

	<TextField
		id="date"
		label="Fecha"
		type="date"
		bind:value={date}
		error={errors.date}
		oninput={() => (errors.date = undefined)}
	/>

	<div class="flex justify-end gap-2 border-t border-zinc-800 pt-5">
		<Button variant="secondary" onclick={oncancel}>Cancelar</Button>
		<Button type="submit" loading={isSubmitting}>
			{isSubmitting ? 'Guardando…' : transaction ? 'Guardar cambios' : typeInfo.submitLabel}
		</Button>
	</div>
</form>
