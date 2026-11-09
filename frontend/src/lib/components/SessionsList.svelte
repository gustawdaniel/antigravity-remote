<script lang="ts">
	import type { WorkspaceSession } from '$lib/types';
	import Icons from './Icons.svelte';
	import { triggerHaptic } from '$lib/audio';

	interface Props {
		sessions: WorkspaceSession[];
		onSelectWorkspace: (ws: string) => void;
	}

	let { sessions, onSelectWorkspace }: Props = $props();

</script>

<div class="sessions-view">
	<!-- Active Windows Section -->
	<div class="section-header">
		<Icons name="layers" size={16} class="sec-icon" />
		<h3 class="sec-title">Connected Antigravity Windows</h3>
	</div>

	{#if sessions.length === 0}
		<div class="no-sessions-card">
			<Icons name="cpu" size={28} class="cpu-icon" />
			<h4>No Active Sessions Detected</h4>
			<p>When Antigravity runs any tool with the remote hook configured, sessions will appear here automatically.</p>
		</div>
	{:else}
		<div class="sessions-grid">
			{#each sessions as s}
				<div
					class="session-card {s.status}"
					onclick={() => {
						triggerHaptic(30);
						onSelectWorkspace(s.workspace);
					}}
					role="button"
					tabindex="0"
					onkeydown={(e) => e.key === 'Enter' && onSelectWorkspace(s.workspace)}
				>
					<div class="session-top">
						<div class="session-info">
							<span class="session-status-dot {s.status}"></span>
							<span class="session-name">{s.workspace}</span>
						</div>
						<span class="status-badge {s.status}">{s.status.replace('_', ' ')}</span>
					</div>

					<div class="session-details">
						<div class="detail-row">
							<Icons name="folder" size={12} />
							<span class="detail-val">{s.workspacePath || 'Workspace default'}</span>
						</div>
						{#if s.lastToolName}
							<div class="detail-row">
								<Icons name="terminal" size={12} />
								<span class="detail-val">Last tool: <code>{s.lastToolName}</code></span>
							</div>
						{/if}
					</div>
				</div>
			{/each}
		</div>
	{/if}

</div>

<style>
	.sessions-view {
		padding: 0.25rem 0;
	}

	.section-header {
		display: flex;
		align-items: center;
		gap: 0.45rem;
		margin-bottom: 0.75rem;
		padding: 0 0.25rem;
	}

	:global(.sec-icon) {
		color: #38bdf8;
	}

	.sec-title {
		font-family: 'Outfit', sans-serif;
		font-size: 0.95rem;
		font-weight: 700;
		color: #e2e8f0;
		margin: 0;
	}

	.no-sessions-card {
		text-align: center;
		padding: 2rem 1.25rem;
		background: rgba(18, 24, 38, 0.5);
		border: 1px solid rgba(255, 255, 255, 0.08);
		border-radius: 14px;
		margin-bottom: 1.5rem;
	}

	:global(.cpu-icon) {
		color: #64748b;
		margin-bottom: 0.5rem;
	}

	.no-sessions-card h4 {
		margin: 0 0 0.35rem 0;
		color: #cbd5e1;
		font-size: 0.95rem;
	}

	.no-sessions-card p {
		margin: 0;
		font-size: 0.78rem;
		color: #64748b;
		line-height: 1.4;
	}

	.sessions-grid {
		display: flex;
		flex-direction: column;
		gap: 0.65rem;
		margin-bottom: 1.5rem;
	}

	.session-card {
		background: rgba(18, 24, 38, 0.7);
		border: 1px solid rgba(255, 255, 255, 0.08);
		border-radius: 14px;
		padding: 0.85rem 1rem;
		cursor: pointer;
		transition: all 0.15s ease;
	}

	.session-card:active {
		transform: scale(0.98);
		background: rgba(30, 41, 59, 0.8);
	}

	.session-top {
		display: flex;
		align-items: center;
		justify-content: space-between;
		margin-bottom: 0.45rem;
	}

	.session-info {
		display: flex;
		align-items: center;
		gap: 0.5rem;
	}

	.session-status-dot {
		width: 8px;
		height: 8px;
		border-radius: 50%;
	}
	.session-status-dot.waiting_approval {
		background: #f43f5e;
		box-shadow: 0 0 6px #f43f5e;
	}
	.session-status-dot.running {
		background: #06b6d4;
		box-shadow: 0 0 6px #06b6d4;
	}
	.session-status-dot.idle {
		background: #64748b;
	}

	.session-name {
		font-family: 'Outfit', sans-serif;
		font-size: 0.95rem;
		font-weight: 700;
		color: #f1f5f9;
	}

	.status-badge {
		font-size: 0.65rem;
		font-weight: 700;
		text-transform: uppercase;
		letter-spacing: 0.05em;
		padding: 0.15rem 0.45rem;
		border-radius: 4px;
	}
	.status-badge.waiting_approval {
		background: rgba(244, 63, 94, 0.15);
		color: #fb7185;
		border: 1px solid rgba(244, 63, 94, 0.3);
	}
	.status-badge.running {
		background: rgba(6, 182, 212, 0.15);
		color: #22d3ee;
		border: 1px solid rgba(6, 182, 212, 0.3);
	}
	.status-badge.idle {
		background: rgba(100, 116, 139, 0.15);
		color: #94a3b8;
		border: 1px solid rgba(100, 116, 139, 0.25);
	}

	.session-details {
		display: flex;
		flex-direction: column;
		gap: 0.25rem;
	}

	.detail-row {
		display: flex;
		align-items: center;
		gap: 0.35rem;
		color: #94a3b8;
		font-size: 0.72rem;
		overflow: hidden;
		text-overflow: ellipsis;
		white-space: nowrap;
	}

	.detail-val {
		overflow: hidden;
		text-overflow: ellipsis;
	}
</style>
