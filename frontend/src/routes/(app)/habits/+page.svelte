<script>
	import { onMount } from 'svelte';
	import { SvelteSet } from 'svelte/reactivity';
	import { errorMessage, habitsApi } from '#lib/api/index.js';
	import Alert from '#lib/components/Alert.svelte';
	import Button from '#lib/components/Button.svelte';
	import HabitCard from '#lib/components/HabitCard.svelte';
	import HabitForm from '#lib/components/HabitForm.svelte';
	import Icon from '#lib/components/Icon.svelte';
	import LogForm from '#lib/components/LogForm.svelte';
	import Modal from '#lib/components/Modal.svelte';
	import SegmentedControl from '#lib/components/SegmentedControl.svelte';
	import SummaryCard from '#lib/components/SummaryCard.svelte';
	import { todayInZone } from '#lib/dates.js';
	import { countOn, dailyStreak, logsRange, periodProgress } from '#lib/habits.js';
	import { profile } from '#lib/stores/profile.js';

	/**
	 * @typedef {import('#lib/api/habits.js').Habit} Habit
	 * @typedef {import('#lib/api/habits.js').HabitLog} HabitLog
	 */

	const NOTICE_DURATION_MS = 4000;

	// El layout solo muestra esta página con el perfil cargado.
	const user = $derived(/** @type {import('#lib/api/auth.js').User} */ ($profile.user));
	// "Hoy" en la zona del usuario, fijado al cargar la página.
	// svelte-ignore state_referenced_locally
	const today = todayInZone(user.timezone);
	const range = logsRange(today);

	/** @type {Habit[]} */
	let habits = $state([]);
	/** @type {Record<string, HabitLog[]>} */
	let logsByHabit = $state({});
	let isLoading = $state(true);
	let loadError = $state('');
	let view = $state('active');
	const busyIds = new SvelteSet();

	let notice = $state('');
	let actionError = $state('');
	/** @type {ReturnType<typeof setTimeout> | undefined} */
	let noticeTimer;

	/** @type {{ habit?: Habit } | null} Hábito en edición; `{}` = crear. */
	let formTarget = $state(null);
	/** @type {Habit | null} */
	let logTarget = $state(null);
	/** @type {Habit | null} */
	let deleteTarget = $state(null);
	let isDeleting = $state(false);

	const active = $derived(habits.filter((h) => h.is_active));
	const paused = $derived(habits.filter((h) => !h.is_active));
	const visible = $derived(view === 'active' ? active : paused);
	const viewOptions = $derived([
		{ value: 'active', label: `Activos · ${active.length}` },
		{ value: 'paused', label: `Pausados · ${paused.length}` }
	]);

	/** @param {Habit} habit */
	const logsOf = (habit) => logsByHabit[habit.id] ?? [];

	const stats = $derived.by(() => {
		const daily = active.filter((h) => h.frequency === 'daily');
		return {
			dailyDone: daily.filter((h) => periodProgress(h, logsOf(h), today).isComplete).length,
			dailyTotal: daily.length,
			goalsDone: active.filter((h) => periodProgress(h, logsOf(h), today).isComplete).length,
			goalsTotal: active.length,
			bestStreak: Math.max(0, ...daily.map((h) => dailyStreak(h, logsOf(h), today)))
		};
	});

	async function load() {
		isLoading = true;
		loadError = '';
		try {
			const list = await habitsApi.listHabits();
			const logs = await Promise.all(list.map((h) => habitsApi.listLogs(h.id, range)));
			habits = list;
			logsByHabit = Object.fromEntries(list.map((h, i) => [h.id, logs[i]]));
		} catch (err) {
			loadError = errorMessage(err);
		} finally {
			isLoading = false;
		}
	}

	/** @param {string} message */
	function showNotice(message) {
		actionError = '';
		notice = message;
		clearTimeout(noticeTimer);
		noticeTimer = setTimeout(() => (notice = ''), NOTICE_DURATION_MS);
	}

	/** @param {Habit} updated */
	function replaceHabit(updated) {
		habits = habits.map((h) => (h.id === updated.id ? updated : h));
	}

	/** Guarda el registro devuelto por la API, reemplazando el del mismo día. */
	function storeLog(/** @type {HabitLog} */ log) {
		if (log.date < range.from) return; // Fuera de lo que muestra la pantalla.
		const others = (logsByHabit[log.habit_id] ?? []).filter((l) => l.date !== log.date);
		logsByHabit[log.habit_id] = [...others, log].sort((a, b) => a.date.localeCompare(b.date));
	}

	/**
	 * Ejecuta una acción sobre un hábito marcándolo como ocupado; un error se
	 * muestra arriba de la lista.
	 *
	 * @param {Habit} habit
	 * @param {() => Promise<void>} action
	 */
	async function withBusy(habit, action) {
		busyIds.add(habit.id);
		actionError = '';
		try {
			await action();
		} catch (err) {
			actionError = errorMessage(err);
		} finally {
			busyIds.delete(habit.id);
		}
	}

	/** @param {Habit} habit */
	function checkIn(habit) {
		return withBusy(habit, async () => {
			// El backend guarda un total por día: se envía el de hoy más uno.
			const log = await habitsApi.logCompletion(habit.id, {
				count: countOn(logsOf(habit), today) + 1
			});
			storeLog(log);
		});
	}

	/** @param {Habit} habit */
	function toggleActive(habit) {
		return withBusy(habit, async () => {
			const updated = await habitsApi.updateHabit(habit.id, { is_active: !habit.is_active });
			replaceHabit(updated);
			showNotice(
				updated.is_active ? `"${updated.name}" se reanudó.` : `"${updated.name}" quedó en pausa.`
			);
		});
	}

	/** @param {import('#lib/api/habits.js').HabitInput} input */
	async function saveHabit(input) {
		const editing = formTarget?.habit;
		if (editing) {
			replaceHabit(await habitsApi.updateHabit(editing.id, input));
			showNotice('Cambios guardados.');
		} else {
			const created = await habitsApi.createHabit(input);
			habits = [...habits, created];
			logsByHabit[created.id] = [];
			view = 'active';
			showNotice(`"${created.name}" se creó.`);
		}
		formTarget = null;
	}

	/** @param {import('#lib/api/habits.js').LogInput} input */
	async function saveLog(input) {
		if (!logTarget) return;
		storeLog(await habitsApi.logCompletion(logTarget.id, input));
		logTarget = null;
		showNotice('Registro guardado.');
	}

	async function confirmDelete() {
		const habit = deleteTarget;
		if (!habit) return;
		isDeleting = true;
		try {
			await habitsApi.deleteHabit(habit.id);
			habits = habits.filter((h) => h.id !== habit.id);
			delete logsByHabit[habit.id];
			deleteTarget = null;
			showNotice(`"${habit.name}" se eliminó.`);
		} catch (err) {
			deleteTarget = null;
			actionError = errorMessage(err);
		} finally {
			isDeleting = false;
		}
	}

	onMount(() => {
		load();
		return () => clearTimeout(noticeTimer);
	});
</script>

<svelte:head>
	<title>Hábitos · ByteBalance</title>
</svelte:head>

<div class="space-y-8">
	<div class="flex flex-wrap items-end justify-between gap-4">
		<div>
			<h1 class="text-2xl font-semibold tracking-tight text-zinc-50">Hábitos</h1>
			<p class="mt-1 text-sm text-zinc-400">Tus metas diarias, semanales y mensuales.</p>
		</div>
		<Button onclick={() => (formTarget = {})}>
			<Icon name="plus" />
			Nuevo hábito
		</Button>
	</div>

	{#if notice}
		<Alert message={notice} tone="success" />
	{/if}
	{#if actionError}
		<Alert message={actionError} />
	{/if}

	{#if loadError}
		<div class="flex flex-wrap items-center gap-3">
			<Alert message={loadError} />
			<Button variant="secondary" class="h-9" onclick={load}>Reintentar</Button>
		</div>
	{:else}
		<section aria-label="Resumen" class="grid gap-4 sm:grid-cols-3">
			<SummaryCard
				label="Completados hoy"
				value={`${stats.dailyDone} / ${stats.dailyTotal}`}
				hint="Hábitos diarios activos"
				tone={stats.dailyTotal > 0 && stats.dailyDone === stats.dailyTotal ? 'positive' : 'neutral'}
				{isLoading}
			/>
			<SummaryCard
				label="Metas cumplidas"
				value={`${stats.goalsDone} / ${stats.goalsTotal}`}
				hint="En el periodo de cada hábito"
				{isLoading}
			/>
			<SummaryCard
				label="Mejor racha"
				value={stats.bestStreak === 1 ? '1 día' : `${stats.bestStreak} días`}
				hint="Días seguidos cumpliendo la meta"
				{isLoading}
			/>
		</section>

		<section aria-labelledby="habits-title" class="space-y-4">
			<div class="flex flex-wrap items-center justify-between gap-3">
				<h2 id="habits-title" class="text-sm font-medium text-zinc-200">Tus hábitos</h2>
				{#if habits.length > 0}
					<SegmentedControl label="Mostrar" options={viewOptions} bind:value={view} />
				{/if}
			</div>

			{#if isLoading}
				<div class="grid gap-4 md:grid-cols-2" aria-busy="true" aria-label="Cargando hábitos">
					{#each { length: 2 }, i (i)}
						<div class="h-52 animate-pulse rounded-lg border border-zinc-800 bg-zinc-900/30"></div>
					{/each}
				</div>
			{:else if habits.length === 0}
				<div
					class="flex flex-col items-center rounded-lg border border-dashed border-zinc-800 px-6 py-14 text-center"
				>
					<span
						class="grid size-10 place-items-center rounded-lg border border-zinc-800 bg-zinc-900 text-zinc-500"
						aria-hidden="true"
					>
						<Icon name="habits" class="size-5" />
					</span>
					<p class="mt-4 text-sm font-medium text-zinc-200">Aún no tienes hábitos</p>
					<p class="mt-1 max-w-xs text-sm text-zinc-500">
						Define una meta pequeña y regístrala cada día para construir una racha.
					</p>
					<Button class="mt-5" onclick={() => (formTarget = {})}>
						<Icon name="plus" />
						Crear mi primer hábito
					</Button>
				</div>
			{:else if visible.length === 0}
				<p
					class="rounded-lg border border-dashed border-zinc-800 px-6 py-10 text-center text-sm text-zinc-500"
				>
					{view === 'active'
						? 'No tienes hábitos activos. Reanuda uno pausado o crea uno nuevo.'
						: 'No tienes hábitos pausados.'}
				</p>
			{:else}
				<div class="grid gap-4 md:grid-cols-2">
					{#each visible as habit (habit.id)}
						<HabitCard
							{habit}
							logs={logsOf(habit)}
							{today}
							isBusy={busyIds.has(habit.id)}
							oncheckin={() => checkIn(habit)}
							onlog={() => (logTarget = habit)}
							onedit={() => (formTarget = { habit })}
							ontoggle={() => toggleActive(habit)}
							ondelete={() => (deleteTarget = habit)}
						/>
					{/each}
				</div>
			{/if}
		</section>
	{/if}
</div>

<Modal
	bind:open={() => formTarget !== null, (open) => !open && (formTarget = null)}
	title={formTarget?.habit ? 'Editar hábito' : 'Nuevo hábito'}
	description={formTarget?.habit
		? formTarget.habit.name
		: 'Define qué quieres repetir y con qué frecuencia.'}
>
	<HabitForm habit={formTarget?.habit} onSubmit={saveHabit} oncancel={() => (formTarget = null)} />
</Modal>

<Modal
	bind:open={() => logTarget !== null, (open) => !open && (logTarget = null)}
	title="Registrar un día"
	description={logTarget?.name}
>
	{#if logTarget}
		<LogForm
			logs={logsOf(logTarget)}
			{today}
			onSubmit={saveLog}
			oncancel={() => (logTarget = null)}
		/>
	{/if}
</Modal>

<Modal
	bind:open={() => deleteTarget !== null, (open) => !open && (deleteTarget = null)}
	title="Eliminar hábito"
>
	{#if deleteTarget}
		<p class="text-sm text-zinc-300">
			Se eliminará <span class="font-medium text-zinc-100">"{deleteTarget.name}"</span> junto con su historial.
			Si solo quieres dejarlo por un tiempo, puedes pausarlo.
		</p>
		<div class="mt-6 flex justify-end gap-2 border-t border-zinc-800 pt-5">
			<Button variant="secondary" onclick={() => (deleteTarget = null)}>Cancelar</Button>
			<Button variant="danger" loading={isDeleting} onclick={confirmDelete}>Eliminar</Button>
		</div>
	{/if}
</Modal>
