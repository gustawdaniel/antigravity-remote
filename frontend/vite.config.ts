import adapter from '@sveltejs/adapter-static';
import { sveltekit } from '@sveltejs/kit/vite';
import { defineConfig } from 'vite';

export default defineConfig({
	plugins: [
		sveltekit({
			compilerOptions: {
				runes: ({ filename }) =>
					filename.split(/[/\\]/).includes('node_modules') ? undefined : true
			},
			adapter: adapter({
				fallback: 'index.html',
				pages: 'build',
				assets: 'build',
				strict: false
			})
		})
	],
	server: {
		proxy: {
			'/api': 'http://127.0.0.1:8765',
			'/ws': {
				target: 'ws://127.0.0.1:8765',
				ws: true
			}
		}
	}
});
