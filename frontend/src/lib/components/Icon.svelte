<script module>
	// Trazos de 16×16 con stroke de 1.5, en línea con los íconos ya usados en Alert.
	const PATHS = {
		wallet:
			'M2.75 4.75h9.5a1 1 0 0 1 1 1v6.5a1 1 0 0 1-1 1h-9.5a1 1 0 0 1-1-1v-8a1.5 1.5 0 0 1 1.5-1.5h7.5M10.5 9h.01',
		habits: 'M8 14.25A6.25 6.25 0 1 0 8 1.75a6.25 6.25 0 0 0 0 12.5ZM5.5 8.25l1.75 1.75 3.25-3.5',
		plus: 'M8 3.25v9.5M3.25 8h9.5',
		menu: 'M2.75 4.5h10.5M2.75 8h10.5M2.75 11.5h10.5',
		close: 'm4 4 8 8M12 4l-8 8',
		logout: 'M6.25 13.25h-2.5a1 1 0 0 1-1-1v-8.5a1 1 0 0 1 1-1h2.5M10.5 11l3-3-3-3M13.25 8H6.5',
		income: 'M8 12.75v-9.5M4.25 7 8 3.25 11.75 7',
		expense: 'M8 3.25v9.5M4.25 9 8 12.75 11.75 9',
		receipt: 'M3.75 1.75h8.5v12.5l-2.125-1.25L8 14.25 5.875 13 3.75 14.25V1.75ZM6 5.5h4M6 8.5h4',
		edit: 'M10.75 2.75 13.25 5.25 6 12.5H3.5V10l7.25-7.25ZM9.25 4.25l2.5 2.5',
		trash:
			'M2.75 4.25h10.5M6.25 4.25V2.75h3.5v1.5M4.25 4.25l.6 8.4a1 1 0 0 0 1 .85h4.3a1 1 0 0 0 1-.85l.6-8.4M6.75 7v4M9.25 7v4'
	};
</script>

<script>
	/**
	 * @typedef {object} Props
	 * @property {keyof typeof PATHS} name
	 * @property {string} [class]
	 */

	/** @type {Props} */
	let { name, class: className = 'size-4' } = $props();
</script>

<svg
	class={['shrink-0', className]}
	viewBox="0 0 16 16"
	fill="none"
	stroke="currentColor"
	stroke-width="1.5"
	stroke-linecap="round"
	stroke-linejoin="round"
	aria-hidden="true"
>
	<path d={PATHS[name]} />
</svg>
