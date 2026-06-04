import { sveltekit } from '@sveltejs/kit/vite';
import { defineConfig } from 'vite';

export default defineConfig(({ command }) => ({
	plugins: [sveltekit()],
	resolve: {
		tsconfigPaths: true
	},
	css: {
		devSourcemap: false,
		transformer: 'lightningcss'
	},
	server: {
		port: 4500,
		host: true,
		proxy:
			command === 'serve'
				? {
						'/api': {
							target: 'http://localhost:8080',
							changeOrigin: true
						},
						'/data': {
							target: 'http://localhost:8080',
							changeOrigin: true
						},
						'/health': {
							target: 'http://localhost:8080',
							changeOrigin: true
						}
					}
				: undefined
	},
	build: {
		sourcemap: false,
		reportCompressedSize: false,
		target: 'es2019',
		minify: 'terser',
		terserOptions: {
			compress: {
				passes: 2,
				drop_console: true
			},
			format: {
				comments: false
			}
		} as any,
		rollupOptions: {
			output: {
				manualChunks(id) {
					const normalized = id.split('\\').join('/');

					if (normalized.includes('/src/lib/services/')) return 'app-services';
					if (normalized.includes('/src/lib/store/')) return 'app-store';
					if (normalized.includes('/src/lib/utils/')) return 'app-utils';
					if (normalized.includes('/src/lib/config/')) return 'app-config';
					if (normalized.includes('/src/lib/components/')) return 'app-components';

					if (!normalized.includes('node_modules')) return;
					if (normalized.includes('svelte')) return 'vendor-svelte';
					return 'vendor';
				}
			}
		}
	}
}));
