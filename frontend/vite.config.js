import tailwindcss from '@tailwindcss/vite';
import { defineConfig } from 'vitest/config';
import { playwright } from '@vitest/browser-playwright';
import adapter from '@sveltejs/adapter-static';
import { sveltekit } from '@sveltejs/kit/vite';
import { loadEnv } from 'vite';
import { normalizeBaseUrl } from './src/lib/api/url.js';

/**
 * Content-Security-Policy del build de producción. SvelteKit la escribe como
 * `<meta>` en index.html y le agrega el hash de su script de arranque. Solo
 * permite recursos propios y llamadas a la API. `frame-ancestors` no funciona en
 * un `<meta>`: CloudFront envía X-Frame-Options (ver DEPLOYMENT.md).
 *
 * @param {string} apiOrigin
 * @returns {import('@sveltejs/kit').Config['csp']}
 */
function productionCsp(apiOrigin) {
	return {
		mode: 'hash',
		directives: {
			'default-src': ['self'],
			'script-src': ['self'],
			// Svelte aplica estilos en línea (style:width de las barras de progreso).
			'style-src': ['self', 'unsafe-inline'],
			'img-src': ['self', 'data:'],
			'font-src': ['self'],
			'connect-src': ['self', apiOrigin],
			'object-src': ['none'],
			'base-uri': ['self'],
			'form-action': ['self']
		}
	};
}

export default defineConfig(({ command, mode }) => {
	const isProductionBuild = command === 'build' && mode === 'production';
	let apiOrigin = '';
	if (isProductionBuild) {
		// La SPA solo lee VITE_API_URL en el navegador: sin esta comprobación, un
		// valor ausente o http se notaría recién con el sitio desplegado (en blanco).
		const { VITE_API_URL } = loadEnv(mode, process.cwd(), 'VITE_');
		apiOrigin = new URL(normalizeBaseUrl(VITE_API_URL, { requireHttps: true })).origin;
	}

	return {
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
				adapter: adapter({ fallback: 'index.html' }),
				csp: isProductionBuild ? productionCsp(apiOrigin) : undefined
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
	};
});
