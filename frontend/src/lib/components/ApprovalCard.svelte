<script lang="ts">
	import type { Approval } from '$lib/types';
	import Icons from './Icons.svelte';
	import { playSuccessChime, playRejectTone, triggerHaptic } from '$lib/audio';

	interface Props {
		approval: Approval;
		onDecide: (id: string, decision: 'allow' | 'deny', reason?: string) => Promise<void>;
		onOpenFeedback: (approval: Approval) => void;
	}

	let { approval, onDecide, onOpenFeedback }: Props = $props();

	let copying = $state(false);
	let isSubmitting = $state(false);
	let diffExpanded = $state(true);

	// Elapsed time ticker
	let elapsedText = $state('');

	$effect(() => {
		function updateElapsed() {
			const diffMs = Date.now() - new Date(approval.createdAt).getTime();
			const secs = Math.floor(diffMs / 1000);
			if (secs < 60) {
				elapsedText = `${secs}s ago`;
			} else {
				const mins = Math.floor(secs / 60);
				elapsedText = `${mins}m ago`;
			}
		}

		updateElapsed();
		const timer = setInterval(updateElapsed, 2000);
		return () => clearInterval(timer);
	});

	async function handleApprove() {
		if (isSubmitting) return;
		isSubmitting = true;
		triggerHaptic([60, 40, 80]);
		playSuccessChime();
		await onDecide(approval.id, 'allow');
		isSubmitting = false;
	}

	async function handleReject() {
		if (isSubmitting) return;
		isSubmitting = true;
		triggerHaptic(80);
		playRejectTone();
		await onDecide(approval.id, 'deny', 'Denied by user via phone');
		isSubmitting = false;
	}

	function handleFeedback() {
		triggerHaptic(40);
		onOpenFeedback(approval);
	}

	async function copyCommand() {
		if (!approval.commandLine) return;
		try {
			await navigator.clipboard.writeText(approval.commandLine);
			copying = true;
			triggerHaptic(30);
			setTimeout(() => {
				copying = false;
			}, 1800);
		} catch {}
	}

	// Tool badge styling
	let toolBadgeColor = $derived.by(() => {
		switch (approval.toolName) {
			case 'run_command':
				return 'tool-command';
			case 'replace_file_content':
				return 'tool-replace';
			case 'write_to_file':
				return 'tool-write';
			default:
				return 'tool-generic';
		}
	});

	// Diff lines breakdown
	let diffLines = $derived.by(() => {
		if (!approval.diff) return [];
		return approval.diff.split('\n').filter((l) => l.trim().length > 0);
	});
</script>

<div class="approval-card {approval.status}">
	<!-- Top Bar -->
	<div class="card-header">
		<div class="badges">
			<span class="tool-badge {toolBadgeColor}">
				{#if approval.toolName === 'run_command'}
					<Icons name="terminal" size={13} />
				{:else if approval.toolName === 'replace_file_content'}
					<Icons name="file-code" size={13} />
				{:else}
					<Icons name="file" size={13} />
				{/if}
				{approval.toolName}
			</span>

			<span class="ws-badge">
				<Icons name="folder" size={12} />
				{approval.workspace}
			</span>

			{#if approval.autoApproved}
				<span class="auto-badge">
					<Icons name="sparkles" size={11} />
					Auto-Approved
				</span>
			{/if}

			{#if approval.isCritical}
				<span class="crit-badge">
					<Icons name="shield" size={11} />
					CRITICAL
				</span>
			{/if}
		</div>

		<div class="time-meta">
			<Icons name="clock" size={12} />
			<span>{elapsedText}</span>
		</div>
	</div>

	<!-- Description / Summary -->
	{#if approval.description}
		<p class="description-text">{approval.description}</p>
	{/if}

	<!-- Command Line Box -->
	{#if approval.commandLine}
		<div class="cmd-box">
			<div class="cmd-header">
				<span class="cmd-label">BASH COMMAND</span>
				<button type="button" class="copy-btn" onclick={copyCommand} title="Copy command">
					{#if copying}
						<Icons name="check-check" size={13} class="copied-icon" />
						<span class="copied-text">Copied</span>
					{:else}
						<Icons name="copy" size={13} />
						<span>Copy</span>
					{/if}
				</button>
			</div>
			<div class="cmd-body">
				<span class="cmd-prompt">$</span>
				<code class="cmd-code">{approval.commandLine}</code>
			</div>
		</div>
	{/if}

	<!-- File & Diff Box -->
	{#if approval.filePath}
		<div class="file-box">
			<div class="file-header">
				<div class="file-path-row">
					<Icons name="file-code" size={14} />
					<span class="file-path">{approval.filePath}</span>
				</div>
				{#if approval.diff}
					<button
						type="button"
						class="toggle-diff-btn"
						onclick={() => (diffExpanded = !diffExpanded)}
					>
						{diffExpanded ? 'Collapse' : 'Show Diff'}
						<Icons name={diffExpanded ? 'chevron-down' : 'chevron-right'} size={13} />
					</button>
				{/if}
			</div>

			{#if approval.diff && diffExpanded}
				<div class="diff-container">
					{#each diffLines as line}
						<div class="diff-line {line.startsWith('+') ? 'add' : line.startsWith('-') ? 'del' : ''}">
							<span class="line-marker">{line.slice(0, 1)}</span>
							<span class="line-content">{line.slice(1)}</span>
						</div>
					{/each}
				</div>
			{/if}
		</div>
	{/if}

	<!-- Other Tool Arguments -->
	{#if !approval.commandLine && !approval.filePath && approval.toolArgs}
		<div class="args-box">
			<span class="args-label">PARAMETERS</span>
			<pre class="args-code">{JSON.stringify(approval.toolArgs, null, 2)}</pre>
		</div>
	{/if}

	<!-- Decision Actions or Status -->
	{#if approval.status === 'pending'}
		<div class="actions-grid">
			<button
				type="button"
				class="action-btn approve-btn"
				onclick={handleApprove}
				disabled={isSubmitting}
			>
				<Icons name="check" size={18} />
				<span>Approve</span>
			</button>

			<button
				type="button"
				class="action-btn reject-btn"
				onclick={handleReject}
				disabled={isSubmitting}
			>
				<Icons name="x" size={18} />
				<span>Reject</span>
			</button>

			<button
				type="button"
				class="action-btn feedback-btn"
				onclick={handleFeedback}
				disabled={isSubmitting}
				title="Reject and provide guidance instructions to agent"
			>
				<Icons name="message-square" size={17} />
				<span>Feedback</span>
			</button>
		</div>
	{:else}
		<div class="resolution-row {approval.status}">
			<span class="resolution-label">
				{#if approval.autoApproved}
					<Icons name="sparkles" size={15} /> Auto-Approved
				{:else if approval.status === 'allowed'}
					<Icons name="check" size={15} /> Approved
				{:else if approval.status === 'denied'}
					<Icons name="x" size={15} /> Rejected
				{:else}
					<Icons name="clock" size={15} /> Timed Out
				{/if}
			</span>
			{#if approval.decisionReason}
				<span class="decision-reason">"{approval.decisionReason}"</span>
			{/if}
		</div>
	{/if}
</div>

<style>
	.approval-card {
		background: rgba(18, 24, 38, 0.75);
		backdrop-filter: blur(14px);
		-webkit-backdrop-filter: blur(14px);
		border: 1px solid rgba(255, 255, 255, 0.08);
		border-radius: 16px;
		padding: 1rem;
		margin-bottom: 1rem;
		box-shadow: 0 8px 32px rgba(0, 0, 0, 0.35);
		transition: all 0.2s cubic-bezier(0.4, 0, 0.2, 1);
		position: relative;
		overflow: hidden;
	}

	.approval-card.pending {
		border-color: rgba(99, 102, 241, 0.45);
		box-shadow:
			0 0 20px rgba(99, 102, 241, 0.15),
			0 8px 32px rgba(0, 0, 0, 0.4);
	}

	.approval-card.pending::before {
		content: '';
		position: absolute;
		top: 0;
		left: 0;
		right: 0;
		height: 3px;
		background: linear-gradient(90deg, #6366f1, #a855f7, #06b6d4);
	}

	.approval-card.allowed {
		opacity: 0.78;
		border-color: rgba(16, 185, 129, 0.25);
	}

	.approval-card.denied {
		opacity: 0.78;
		border-color: rgba(239, 68, 68, 0.25);
	}

	.card-header {
		display: flex;
		align-items: center;
		justify-content: space-between;
		margin-bottom: 0.75rem;
	}

	.badges {
		display: flex;
		align-items: center;
		gap: 0.45rem;
		flex-wrap: wrap;
	}

	.tool-badge {
		display: inline-flex;
		align-items: center;
		gap: 0.35rem;
		padding: 0.25rem 0.55rem;
		border-radius: 6px;
		font-family: 'JetBrains Mono', monospace;
		font-size: 0.75rem;
		font-weight: 600;
	}

	.tool-badge.tool-command {
		background: rgba(168, 85, 247, 0.15);
		color: #c084fc;
		border: 1px solid rgba(168, 85, 247, 0.35);
	}

	.tool-badge.tool-replace {
		background: rgba(16, 185, 129, 0.15);
		color: #34d399;
		border: 1px solid rgba(16, 185, 129, 0.35);
	}

	.tool-badge.tool-write {
		background: rgba(245, 158, 11, 0.15);
		color: #fbbf24;
		border: 1px solid rgba(245, 158, 11, 0.35);
	}

	.tool-badge.tool-generic {
		background: rgba(56, 189, 248, 0.15);
		color: #38bdf8;
		border: 1px solid rgba(56, 189, 248, 0.35);
	}

	.ws-badge {
		display: inline-flex;
		align-items: center;
		gap: 0.3rem;
		padding: 0.25rem 0.5rem;
		border-radius: 6px;
		font-size: 0.72rem;
		font-weight: 600;
		color: #94a3b8;
		background: rgba(30, 41, 59, 0.6);
		border: 1px solid rgba(255, 255, 255, 0.06);
	}

	.auto-badge {
		display: inline-flex;
		align-items: center;
		gap: 0.25rem;
		padding: 0.25rem 0.5rem;
		border-radius: 6px;
		font-size: 0.7rem;
		font-weight: 700;
		color: #34d399;
		background: rgba(16, 185, 129, 0.15);
		border: 1px solid rgba(16, 185, 129, 0.35);
	}

	.crit-badge {
		display: inline-flex;
		align-items: center;
		gap: 0.25rem;
		padding: 0.25rem 0.5rem;
		border-radius: 6px;
		font-size: 0.7rem;
		font-weight: 800;
		color: #ffffff;
		background: #f43f5e;
		box-shadow: 0 0 10px rgba(244, 63, 94, 0.5);
		animation: pulse-ring 1.5s infinite;
	}

	.time-meta {
		display: flex;
		align-items: center;
		gap: 0.3rem;
		font-size: 0.72rem;
		color: #64748b;
	}

	.description-text {
		font-size: 0.85rem;
		color: #cbd5e1;
		line-height: 1.45;
		margin: 0 0 0.75rem 0;
	}

	/* Terminal Command Box */
	.cmd-box {
		background: #080c14;
		border: 1px solid rgba(255, 255, 255, 0.08);
		border-radius: 10px;
		overflow: hidden;
		margin-bottom: 0.85rem;
	}

	.cmd-header {
		display: flex;
		align-items: center;
		justify-content: space-between;
		padding: 0.4rem 0.75rem;
		background: rgba(255, 255, 255, 0.03);
		border-bottom: 1px solid rgba(255, 255, 255, 0.06);
	}

	.cmd-label {
		font-size: 0.65rem;
		font-weight: 700;
		letter-spacing: 0.08em;
		color: #64748b;
	}

	.copy-btn {
		display: inline-flex;
		align-items: center;
		gap: 0.3rem;
		background: transparent;
		border: none;
		color: #94a3b8;
		font-size: 0.7rem;
		font-weight: 500;
		cursor: pointer;
		padding: 0.2rem 0.4rem;
		border-radius: 4px;
		transition: all 0.15s ease;
	}

	.copy-btn:active {
		transform: scale(0.94);
		background: rgba(255, 255, 255, 0.08);
	}

	.copied-icon {
		color: #10b981;
	}
	.copied-text {
		color: #34d399;
	}

	.cmd-body {
		display: flex;
		align-items: flex-start;
		gap: 0.5rem;
		padding: 0.65rem 0.75rem;
		overflow-x: auto;
	}

	.cmd-prompt {
		color: #ec4899;
		font-family: 'JetBrains Mono', monospace;
		font-weight: 700;
		user-select: none;
	}

	.cmd-code {
		font-family: 'JetBrains Mono', monospace;
		font-size: 0.8rem;
		color: #f1f5f9;
		white-space: pre-wrap;
		word-break: break-all;
	}

	/* File Box */
	.file-box {
		background: #080c14;
		border: 1px solid rgba(255, 255, 255, 0.08);
		border-radius: 10px;
		overflow: hidden;
		margin-bottom: 0.85rem;
	}

	.file-header {
		display: flex;
		align-items: center;
		justify-content: space-between;
		padding: 0.45rem 0.75rem;
		background: rgba(255, 255, 255, 0.03);
		border-bottom: 1px solid rgba(255, 255, 255, 0.06);
	}

	.file-path-row {
		display: flex;
		align-items: center;
		gap: 0.45rem;
		color: #38bdf8;
		font-family: 'JetBrains Mono', monospace;
		font-size: 0.75rem;
		font-weight: 500;
		overflow: hidden;
		text-overflow: ellipsis;
		white-space: nowrap;
	}

	.file-path {
		overflow: hidden;
		text-overflow: ellipsis;
	}

	.toggle-diff-btn {
		display: inline-flex;
		align-items: center;
		gap: 0.25rem;
		background: transparent;
		border: none;
		color: #94a3b8;
		font-size: 0.7rem;
		cursor: pointer;
		padding: 0.2rem 0.4rem;
	}

	.diff-container {
		max-height: 240px;
		overflow-y: auto;
		font-family: 'JetBrains Mono', monospace;
		font-size: 0.72rem;
		line-height: 1.5;
	}

	.diff-line {
		display: flex;
		align-items: flex-start;
		padding: 0.12rem 0.6rem;
		white-space: pre-wrap;
		word-break: break-all;
	}

	.diff-line.add {
		background: rgba(16, 185, 129, 0.15);
		color: #6ee7b7;
	}

	.diff-line.del {
		background: rgba(239, 68, 68, 0.15);
		color: #fca5a5;
	}

	.line-marker {
		width: 14px;
		flex-shrink: 0;
		font-weight: 700;
		user-select: none;
	}

	.line-content {
		flex: 1;
	}

	/* Args Box */
	.args-box {
		background: #080c14;
		border: 1px solid rgba(255, 255, 255, 0.08);
		border-radius: 10px;
		padding: 0.6rem 0.75rem;
		margin-bottom: 0.85rem;
	}

	.args-label {
		display: block;
		font-size: 0.65rem;
		font-weight: 700;
		letter-spacing: 0.08em;
		color: #64748b;
		margin-bottom: 0.35rem;
	}

	.args-code {
		font-family: 'JetBrains Mono', monospace;
		font-size: 0.72rem;
		color: #cbd5e1;
		margin: 0;
		white-space: pre-wrap;
		max-height: 140px;
		overflow-y: auto;
	}

	/* Action Buttons */
	.actions-grid {
		display: grid;
		grid-template-columns: 2fr 1.3fr 1.3fr;
		gap: 0.55rem;
		margin-top: 0.85rem;
	}

	.action-btn {
		display: inline-flex;
		align-items: center;
		justify-content: center;
		gap: 0.45rem;
		padding: 0.75rem 0.6rem;
		border-radius: 12px;
		font-family: 'Outfit', sans-serif;
		font-size: 0.95rem;
		font-weight: 700;
		border: none;
		cursor: pointer;
		user-select: none;
		-webkit-tap-highlight-color: transparent;
		transition: all 0.15s cubic-bezier(0.4, 0, 0.2, 1);
	}

	.action-btn:active {
		transform: scale(0.96);
	}

	.action-btn:disabled {
		opacity: 0.5;
		cursor: not-allowed;
	}

	.approve-btn {
		background: linear-gradient(135deg, #10b981 0%, #059669 100%);
		color: #ffffff;
		box-shadow: 0 4px 14px rgba(16, 185, 129, 0.35);
	}

	.approve-btn:active {
		box-shadow: 0 2px 6px rgba(16, 185, 129, 0.2);
	}

	.reject-btn {
		background: linear-gradient(135deg, #ef4444 0%, #dc2626 100%);
		color: #ffffff;
		box-shadow: 0 4px 14px rgba(239, 68, 68, 0.3);
	}

	.feedback-btn {
		background: rgba(30, 41, 59, 0.9);
		color: #cbd5e1;
		border: 1px solid rgba(255, 255, 255, 0.12);
	}

	.feedback-btn:active {
		background: rgba(51, 65, 85, 0.9);
	}

	/* Resolved Row */
	.resolution-row {
		display: flex;
		align-items: center;
		justify-content: space-between;
		padding: 0.55rem 0.8rem;
		border-radius: 8px;
		font-size: 0.8rem;
		font-weight: 600;
		margin-top: 0.5rem;
	}

	.resolution-row.allowed {
		background: rgba(16, 185, 129, 0.12);
		color: #34d399;
		border: 1px solid rgba(16, 185, 129, 0.25);
	}

	.resolution-row.denied {
		background: rgba(239, 68, 68, 0.12);
		color: #f87171;
		border: 1px solid rgba(239, 68, 68, 0.25);
	}

	.resolution-label {
		display: flex;
		align-items: center;
		gap: 0.35rem;
	}

	.decision-reason {
		font-size: 0.72rem;
		font-style: italic;
		color: #94a3b8;
		max-width: 60%;
		overflow: hidden;
		text-overflow: ellipsis;
		white-space: nowrap;
	}
</style>
