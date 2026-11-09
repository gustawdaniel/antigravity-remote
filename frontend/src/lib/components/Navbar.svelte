<script lang="ts">
	import Icons from './Icons.svelte';
	import { isSoundEnabled, setSoundEnabled, triggerHaptic } from '$lib/audio';

	interface Props {
		onRefresh: () => void;
	}

	let { onRefresh }: Props = $props();

	let soundOn = $state(true);
	let notificationsAllowed = $state(false);

	$effect(() => {
		soundOn = isSoundEnabled();
		if (typeof window !== 'undefined' && 'Notification' in window) {
			notificationsAllowed = Notification.permission === 'granted';
		}
	});

	function toggleSound() {
		triggerHaptic(40);
		soundOn = !soundOn;
		setSoundEnabled(soundOn);
	}

	async function requestNotifications() {
		triggerHaptic(40);
		if (typeof window !== 'undefined' && 'Notification' in window) {
			if (Notification.permission === 'default') {
				const res = await Notification.requestPermission();
				notificationsAllowed = res === 'granted';
			} else if (Notification.permission === 'denied') {
				alert('Notifications are blocked in your browser settings. Please enable them in site permissions to receive pocket alerts.');
			}
		}
	}
</script>

<header class="navbar">
	<div class="brand">
		<div class="logo-box">
			<img src="/icon.svg" alt="Antigravity Logo" class="brand-logo" />
			<span class="logo-glow"></span>
		</div>
		<div class="brand-text">
			<div class="title-row">
				<h1 class="brand-name">Antigravity</h1>
				<span class="badge-rc">RC</span>
			</div>
			<span class="brand-sub">Tailscale Remote</span>
		</div>
	</div>

	<div class="controls">
		<!-- Sound Chime Toggle -->
		<button
			type="button"
			class="icon-btn {soundOn ? 'active' : ''}"
			onclick={toggleSound}
			aria-label={soundOn ? 'Agent completion chime ON' : 'Agent completion chime OFF'}
			title={soundOn ? 'Chime on task complete is ON' : 'Chime on task complete is MUTED'}
		>
			{#if soundOn}
				<Icons name="volume-2" size={17} />
			{:else}
				<Icons name="volume-x" size={17} />
			{/if}
		</button>

		<!-- Notifications Toggle -->
		<button
			type="button"
			class="icon-btn {notificationsAllowed ? 'active' : ''}"
			onclick={requestNotifications}
			aria-label="Browser notifications"
			title={notificationsAllowed ? 'Browser notifications active' : 'Enable browser notifications'}
		>
			{#if notificationsAllowed}
				<Icons name="bell" size={17} />
			{:else}
				<Icons name="bell-off" size={17} />
			{/if}
		</button>

		<!-- Podcast Feed Link -->
		<a
			href="/podcast"
			target="_blank"
			class="icon-btn"
			aria-label="Brain Audio Feed"
			title="Otwórz Brain Audio Feed Podcast"
			style="text-decoration: none;"
		>
			<Icons name="radio" size={17} />
		</a>

		<!-- Refresh Button -->
		<button
			type="button"
			class="icon-btn"
			onclick={() => {
				triggerHaptic(30);
				onRefresh();
			}}
			aria-label="Refresh data"
			title="Refresh data"
		>
			<Icons name="refresh-cw" size={16} />
		</button>
	</div>
</header>

<style>
	.navbar {
		position: sticky;
		top: 0;
		z-index: 50;
		display: flex;
		align-items: center;
		justify-content: space-between;
		padding: env(safe-area-inset-top, 0.75rem) 1rem 0.75rem 1rem;
		background: rgba(11, 15, 25, 0.82);
		backdrop-filter: blur(18px);
		-webkit-backdrop-filter: blur(18px);
		border-bottom: 1px solid rgba(255, 255, 255, 0.07);
	}

	.brand {
		display: flex;
		align-items: center;
		gap: 0.65rem;
	}

	.logo-box {
		position: relative;
		width: 36px;
		height: 36px;
		display: flex;
		align-items: center;
		justify-content: center;
	}

	.brand-logo {
		width: 34px;
		height: 34px;
		border-radius: 8px;
		position: relative;
		z-index: 2;
	}

	.logo-glow {
		position: absolute;
		inset: 0;
		border-radius: 50%;
		background: radial-gradient(circle, rgba(99, 102, 241, 0.5) 0%, transparent 70%);
		filter: blur(6px);
		z-index: 1;
	}

	.brand-text {
		display: flex;
		flex-direction: column;
	}

	.title-row {
		display: flex;
		align-items: center;
		gap: 0.35rem;
	}

	.brand-name {
		font-family: 'Outfit', sans-serif;
		font-size: 1.05rem;
		font-weight: 700;
		letter-spacing: -0.02em;
		background: linear-gradient(135deg, #ffffff 40%, #a5b4fc 100%);
		-webkit-background-clip: text;
		-webkit-text-fill-color: transparent;
		margin: 0;
	}

	.badge-rc {
		font-size: 0.62rem;
		font-weight: 800;
		padding: 0.1rem 0.35rem;
		border-radius: 4px;
		background: rgba(99, 102, 241, 0.2);
		color: #818cf8;
		border: 1px solid rgba(99, 102, 241, 0.4);
		letter-spacing: 0.05em;
	}

	.brand-sub {
		font-size: 0.68rem;
		color: #64748b;
		font-weight: 500;
	}

	.controls {
		display: flex;
		align-items: center;
		gap: 0.45rem;
	}
	.icon-btn {
		background: rgba(30, 41, 59, 0.6);
		border: 1px solid rgba(255, 255, 255, 0.08);
		color: #94a3b8;
		width: 34px;
		height: 34px;
		border-radius: 8px;
		display: flex;
		align-items: center;
		justify-content: center;
		cursor: pointer;
		transition: all 0.15s ease;
		padding: 0;
	}

	.icon-btn:active {
		transform: scale(0.92);
		background: rgba(51, 65, 85, 0.8);
	}

	.icon-btn.active {
		color: #38bdf8;
		border-color: rgba(56, 189, 248, 0.4);
		background: rgba(56, 189, 248, 0.12);
	}
</style>
