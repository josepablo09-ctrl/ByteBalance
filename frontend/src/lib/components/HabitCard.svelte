<script>
	import { LOCALE } from '#lib/locale.js';
	import {
		countOn,
		dailyStreak,
		describeGoal,
		MAX_COUNT,
		periodProgress,
		recentDays
	} from '#lib/habits.js';
	import Button from './Button.svelte';
	import Icon from './Icon.svelte';

	/**
	 * Tarjeta de un hábito: meta, progreso del periodo, historial reciente y acciones.
	 *
	 * @typedef {object} Props
	 * @property {import('#lib/api/habits.js').Habit} habit
	 * @property {import('#lib/api/habits.js').HabitLog[]} logs
	 * @property {string} today `YYYY-MM-DD` en la zona del usuario.
	 * @property {boolean} [isBusy] Hay una acción en curso sobre este hábito.
	 * @property {() => void} oncheckin Suma una vez al día de hoy.
	 * @property {() => void} onundo Resta una vez al día de hoy (en 1 lo deja en cero).
	 * @property {() => void} onlog Abre el registro de otro día.
	 * @property {() => void} onedit
	 * @property {() => void} ontoggle Pausa o reanuda.
	 * @property {() => void} ondelete
	 */

	/** @type {Props} */
	let {
		habit,
		logs,
		today,
		isBusy = false,
		oncheckin,
		onundo,
		onlog,
		onedit,
		ontoggle,
		ondelete
	} = $props();

	const progress = $derived(periodProgress(habit, logs, today));
	const todayCount = $derived(countOn(logs, today));
	const streak = $derived(dailyStreak(habit, logs, today));
	const days = $derived(recentDays(logs, today));
	const percent = $derived(Math.min(100, Math.round((progress.done / progress.target) * 100)));
	// Un hábito diario de 1 vez se marca "hecho"; el resto suma de a una vez.
	const isSingleDaily = $derived(habit.frequency === 'daily' && habit.target_count === 1);
	const isDoneToday = $derived(isSingleDaily && todayCount >= 1);

	const dayFormatter = new Intl.DateTimeFormat(LOCALE, {
		weekday: 'short',
		day: 'numeric',
		month: 'short',
		timeZone: 'UTC'
	});

	/** @param {{ date: string, count: number }} day */
	function dayClass(day) {
		if (day.count === 0) return 'bg-zinc-800/70';
		if (habit.frequency !== 'daily' || day.count >= habit.target_count) return 'bg-emerald-400';
		return 'bg-emerald-500/40';
	}

	/** @param {{ date: string, count: number }} day */
	function dayTitle(day) {
		const label = dayFormatter.format(new Date(`${day.date}T00:00:00Z`));
		return `${label}: ${day.count === 0 ? 'sin registro' : day.count === 1 ? '1 vez' : `${day.count} veces`}`;
	}
</script>

<article
	aria-labelledby="habit-{habit.id}"
	class={[
		'flex flex-col rounded-lg border bg-zinc-900/30',
		habit.is_active ? 'border-zinc-800' : 'border-dashed border-zinc-800'
	]}
>
	<div class="flex items-start gap-3 px-5 pt-4">
		<div class="min-w-0 flex-1">
			<div class="flex items-center gap-2">
				<h3 id="habit-{habit.id}" class="truncate text-sm font-medium text-zinc-100">
					{habit.name}
				</h3>
				{#if !habit.is_active}
					<span
						class="rounded border border-zinc-700 px-1.5 py-px font-mono text-[10px] text-zinc-400 uppercase"
					>
						Pausado
					</span>
				{/if}
			</div>
			<p class="mt-0.5 text-xs text-zinc-500">
				{describeGoal(habit)}{habit.reminder_time ? ` · Recordatorio ${habit.reminder_time}` : ''}
			</p>
			{#if habit.description}
				<p class="mt-2 line-clamp-2 text-sm text-zinc-400">{habit.description}</p>
			{/if}
		</div>
		{#if streak > 1}
			<p class="shrink-0 text-right">
				<span class="block text-sm font-semibold text-zinc-100 tabular-nums">{streak}</span>
				<span class="block text-[11px] text-zinc-500">días seguidos</span>
			</p>
		{/if}
	</div>

	<div class="mt-4 px-5">
		<div class="flex items-baseline justify-between text-xs">
			<span class="text-zinc-400">Progreso {progress.periodName}</span>
			<span class={['tabular-nums', progress.isComplete ? 'text-emerald-400' : 'text-zinc-300']}>
				{progress.done} / {progress.target}
			</span>
		</div>
		<div
			class="mt-1.5 h-1.5 overflow-hidden rounded-full bg-zinc-800"
			role="progressbar"
			aria-label="Progreso {progress.periodName}"
			aria-valuemin="0"
			aria-valuemax={progress.target}
			aria-valuenow={Math.min(progress.done, progress.target)}
		>
			<div
				class={[
					'h-full rounded-full transition-[width]',
					progress.isComplete ? 'bg-emerald-400' : 'bg-emerald-500/70'
				]}
				style:width="{percent}%"
			></div>
		</div>
	</div>

	<div class="mt-4 px-5">
		<p class="sr-only">
			Últimos {days.length} días: {days.filter((d) => d.count > 0).length} con registro.
		</p>
		<ol class="flex gap-1" aria-hidden="true">
			{#each days as day (day.date)}
				<li
					class={[
						'h-5 flex-1 rounded-sm',
						dayClass(day),
						day.date === today && 'ring-1 ring-zinc-500'
					]}
					title={dayTitle(day)}
				></li>
			{/each}
		</ol>
	</div>

	<div class="mt-4 flex flex-wrap items-center gap-2 border-t border-zinc-800 px-3 py-2.5">
		{#if habit.is_active}
			{#if isDoneToday}
				<span
					class="inline-flex h-8 items-center gap-1.5 rounded-lg border border-emerald-500/30 bg-emerald-500/10 px-3 text-xs font-medium text-emerald-300"
				>
					<Icon name="habits" class="size-3.5" />
					Hecho hoy
				</span>
			{:else}
				<Button
					class="h-8 px-3 text-xs"
					loading={isBusy}
					disabled={todayCount >= MAX_COUNT}
					onclick={oncheckin}
				>
					{#if isSingleDaily}
						<Icon name="habits" class="size-3.5" />
						Marcar hecho
					{:else}
						<Icon name="plus" class="size-3.5" />
						Sumar 1 hoy
					{/if}
				</Button>
			{/if}
			{#if todayCount > 0}
				<Button
					variant="ghost"
					class="h-8 px-2.5 text-xs"
					disabled={isBusy}
					onclick={onundo}
					aria-label={isSingleDaily ? 'Deshacer el registro de hoy' : 'Quitar 1 al registro de hoy'}
				>
					{isSingleDaily ? 'Deshacer' : 'Quitar 1'}
				</Button>
			{/if}
			<Button variant="ghost" class="h-8 px-2.5 text-xs" onclick={onlog}>Registrar otro día</Button>
		{/if}

		<div class="ml-auto flex items-center gap-1">
			<Button variant="ghost" class="h-8 px-2.5 text-xs" onclick={onedit}>Editar</Button>
			<Button variant="ghost" class="h-8 px-2.5 text-xs" disabled={isBusy} onclick={ontoggle}>
				{habit.is_active ? 'Pausar' : 'Reanudar'}
			</Button>
			<Button
				variant="ghost"
				class="h-8 px-2.5 text-xs hover:text-red-300"
				disabled={isBusy}
				onclick={ondelete}
			>
				Eliminar
			</Button>
		</div>
	</div>
</article>
