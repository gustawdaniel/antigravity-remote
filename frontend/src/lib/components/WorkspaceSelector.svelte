<script lang="ts">
	import type { WorkspaceSession, Approval } from '$lib/types';
	import { triggerHaptic } from '$lib/audio';

	interface Props {
		sessions: WorkspaceSession[];
		approvals: Approval[];
		selectedWorkspace: string;
		onSelect: (ws: string) => void;
	}

	let { sessions, approvals, selectedWorkspace, onSelect }: Props = $props();

	// Derive workspaces strictly from active sessions (actual IDE windows)
	let workspaces = $derived.by(() => {
		const set = new Set<string>();
		sessions.forEach((s) => {
			if (s.workspace && s.workspace !== 'default') {
				set.add(s.workspace);
			}
		});
		return Array.from(set).sort();
	});

	function getPendingCount(ws: string): number {
		if (ws === 'all') {
			return approvals.filter((a) => a.status === 'pending').length;
		}
		return approvals.filter((a) => a.status === 'pending' && a.workspace === ws).length;
	}

	function isSessionRunning(ws: string): boolean {
		return sessions.some((s) => s.workspace === ws && s.status === 'running');
	}
</script>

<div class="workspace-bar">
	<div class="pill-scroll">
		<!-- All Pill -->
		<button
			type="button"
			class="ws-pill {selectedWorkspace === 'all' ? 'active' : ''}"
			onclick={() => {
				triggerHaptic(30);
				onSelect('all');
			}}
		>
			<span class="ws-name">All Windows</span>
			{#if getPendingCount('all') > 0}
				<span class="badge-count animate-bounce">{getPendingCount('all')}</span>
			{/if}
		</button>

		<!-- Individual Workspace Pills -->
		{#each workspaces as ws}
			{@const pending = getPendingCount(ws)}
			{@const running = isSessionRunning(ws)}
			<button
				type="button"
				class="ws-pill {selectedWorkspace === ws ? 'active' : ''}"
				onclick={() => {
					triggerHaptic(30);
					onSelect(ws);
				}}
			>
				<span class="status-indicator {pending > 0 ? 'alert' : running ? 'running' : 'idle'}"></span>
				<span class="ws-name">{ws}</span>
				{#if pending > 0}
					<span class="badge-count">{pending}</span>
				{/if}
			</button>
		{/each}
	</div>
</div>

<style>
	.workspace-bar {
		padding: 0.6rem 1rem;
		background: rgba(15, 23, 42, 0.4);
		border-bottom: 1px solid rgba(255, 255, 255, 0.05);
		max-width: 100%;
		overflow: hidden;
		box-sizing: border-box;
	}

	.pill-scroll {
		display: flex;
		gap: 0.5rem;
		overflow-x: auto;
		scrollbar-width: none;
		-ms-overflow-style: none;
		padding-bottom: 2px;
	}
	.pill-scroll::-webkit-scrollbar {
		display: none;
	}

	.ws-pill {
		display: inline-flex;
		align-items: center;
		gap: 0.45rem;
		padding: 0.4rem 0.8rem;
		border-radius: 9999px;
		font-size: 0.8rem;
		font-weight: 600;
		color: #94a3b8;
		background: rgba(30, 41, 59, 0.5);
		border: 1px solid rgba(255, 255, 255, 0.07);
		cursor: pointer;
		white-space: nowrap;
		transition: all 0.15s ease;
		user-select: none;
		-webkit-tap-highlight-color: transparent;
	}

	.ws-pill:active {
		transform: scale(0.95);
	}

	.ws-pill.active {
		color: #ffffff;
		background: linear-gradient(135deg, rgba(99, 102, 241, 0.25) 0%, rgba(168, 85, 247, 0.25) 100%);
		border-color: rgba(168, 85, 247, 0.5);
		box-shadow: 0 0 16px rgba(139, 92, 246, 0.2);
	}

	.status-indicator {
		width: 7px;
		height: 7px;
		border-radius: 50%;
	}
	.status-indicator.alert {
		background: #f43f5e;
		box-shadow: 0 0 6px #f43f5e;
		animation: pulse-red 1.2s infinite;
	}
	.status-indicator.running {
		background: #06b6d4;
		box-shadow: 0 0 6px #06b6d4;
	}
	.status-indicator.idle {
		background: #64748b;
	}

	.badge-count {
		font-size: 0.68rem;
		font-weight: 700;
		background: #f43f5e;
		color: #ffffff;
		padding: 0.1rem 0.4rem;
		border-radius: 9999px;
		line-height: 1;
	}

	@keyframes pulse-red {
		0%,
		100% {
			transform: scale(1);
			opacity: 1;
		}
		50% {
			transform: scale(1.3);
			opacity: 0.6;
		}
	}
</style>
