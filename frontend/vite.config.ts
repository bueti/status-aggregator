import { sveltekit } from '@sveltejs/kit/vite';
import tailwindcss from '@tailwindcss/vite';
import { defineConfig } from 'vite';

export default defineConfig({
	plugins: [tailwindcss(), sveltekit()],
	server: {
		proxy: {
			'/api': 'http://localhost:8080',
			'/openapi.json': 'http://localhost:8080',
			'/docs': 'http://localhost:8080',
			'/schemas': 'http://localhost:8080'
		}
	}
});
