import { sveltekit } from '@sveltejs/kit/vite';
import topLevelAwait from "vite-plugin-top-level-await";
import wasm from "vite-plugin-wasm";
import { defineConfig } from 'vite';

export default defineConfig({
	optimizeDeps: {
		// exclude: ["brotli-wasm", "brotli-wasm/pkg.bundler/brotli_wasm_bg.wasm"],
	},
	plugins: [
		sveltekit(),
		//wasm(),
	],
	ssr: {
		noExternal: ['brotli-wasm', '*/**.wasm']
	},
	server: {
		fs: {
			allow: ['..']
		}
	}
})
