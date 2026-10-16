import tailwindcss from '@tailwindcss/vite';
import { defineConfig } from 'vitest/config';
import { playwright } from '@vitest/browser-playwright';
import adapter from '@sveltejs/adapter-static';
import { sveltekit } from '@sveltejs/kit/vite';

export default defineConfig({
	plugins: [
		tailwindcss(),
		sveltekit({
			compilerOptions: {
				// Force runes mode for the project, except for libraries. Can be removed in svelte 6.
				runes: ({ filename }) =>
					filename.split(/[/\\]/).includes('node_modules') ? undefined : true
			},
			// SPA: el backend Go es una API separada, así que el frontend se compila a
			// archivos estáticos y todas las rutas caen en index.html (enrutado en el cliente).
			adapter: adapter({ fallback: 'index.html' })
		})
	],
	server: {
		// Puerto fijo: debe coincidir con CORS_ALLOWED_ORIGINS del backend.
		port: 5173,
		strictPort: true
	},
	test: {
		expect: { requireAssertions: true },
		// Los tests no dependen de frontend/.env (que no existe en CI).
		env: { VITE_API_URL: 'http://localhost:8080/api/v1' },
		projects: [
			{
				extends: './vite.config.js',
				test: {
					name: 'client',
					browser: {
						enabled: true,
						provider: playwright(),
						instances: [{ browser: 'chromium', headless: true }]
					},
					include: ['src/**/*.svelte.{test,spec}.{js,ts}'],
					exclude: ['src/lib/server/**']
				}
			},

			{
				extends: './vite.config.js',
				test: {
					name: 'server',
					environment: 'node',
					include: ['src/**/*.{test,spec}.{js,ts}'],
					exclude: ['src/**/*.svelte.{test,spec}.{js,ts}']
				}
			}
		]
	}
});
