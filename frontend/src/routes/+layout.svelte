<script lang="ts">
	import { onMount } from 'svelte';
	import favicon from '$lib/assets/favicon.svg';

	let { children } = $props();

	onMount(() => {
		if (typeof window !== 'undefined' && 'serviceWorker' in navigator) {
			navigator.serviceWorker.register('/sw.js').catch((err) => {
				console.warn('Service worker registration failed:', err);
			});
		}
	});
</script>

<svelte:head>
	<link rel="icon" href={favicon} />
	<meta name="apple-mobile-web-app-status-bar-style" content="black-translucent" />
</svelte:head>

<div class="app-root">
	{@render children()}
</div>

<style>
	:global(:root) {
		--bg-dark: #090d16;
		--bg-card: rgba(18, 24, 38, 0.75);
		--border-subtle: rgba(255, 255, 255, 0.08);
		--accent-primary: #6366f1;
		--accent-cyan: #06b6d4;
		--accent-emerald: #10b981;
		--accent-rose: #f43f5e;
		--text-main: #f8fafc;
		--text-muted: #94a3b8;
	}

	:global(*) {
		box-sizing: border-box;
	}

	:global(body) {
		margin: 0;
		padding: 0;
		background-color: #090d16;
		color: #f8fafc;
		font-family: 'Inter', -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, sans-serif;
		-webkit-font-smoothing: antialiased;
		-moz-osx-font-smoothing: grayscale;
		overflow-x: hidden;
		-webkit-tap-highlight-color: transparent;
	}

	:global(code, pre) {
		font-family: 'JetBrains Mono', 'Fira Code', monospace;
	}

	.app-root {
		min-height: 100vh;
		min-height: 100dvh;
		display: flex;
		flex-direction: column;
		background:
			radial-gradient(ellipse at 80% 0%, rgba(99, 102, 241, 0.12) 0%, transparent 60%),
			radial-gradient(ellipse at 20% 90%, rgba(6, 182, 212, 0.08) 0%, transparent 50%),
			#090d16;
	}
</style>
