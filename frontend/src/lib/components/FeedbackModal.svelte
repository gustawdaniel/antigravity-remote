<script lang="ts">
	import type { Approval } from '$lib/types';
	import Icons from './Icons.svelte';
	import { triggerHaptic, playRejectTone } from '$lib/audio';

	interface Props {
		approval: Approval | null;
		onClose: () => void;
		onSubmit: (id: string, reason: string) => Promise<void>;
	}

	let { approval, onClose, onSubmit }: Props = $props();

	let reasonText = $state('');
	let isSubmitting = $state(false);

	const quickChips = [
		'Run tests first',
		'Use pnpm instead of npm',
		'Do not modify this file',
		'Check git diff first',
		'Provide explanation before editing'
	];

	function selectChip(chip: string) {
		triggerHaptic(25);
		if (reasonText.length > 0) {
			reasonText += '; ' + chip;
		} else {
			reasonText = chip;
		}
	}

	async function handleSubmit() {
		if (!approval || isSubmitting) return;
		isSubmitting = true;
		triggerHaptic(60);
		playRejectTone();
		const finalReason = reasonText.trim() || 'Rejected with feedback from user';
		await onSubmit(approval.id, finalReason);
		isSubmitting = false;
		onClose();
	}
</script>

{#if approval}
	<div class="modal-backdrop" onclick={onClose} role="presentation">
		<!-- svelte-ignore a11y_click_events_have_key_events -->
		<div class="modal-sheet" onclick={(e) => e.stopPropagation()} role="dialog" aria-modal="true" tabindex="-1">
			<div class="sheet-handle"></div>

			<div class="sheet-header">
				<div class="header-left">
					<Icons name="message-square" size={18} class="header-icon" />
					<h3 class="sheet-title">Reject with Guidance</h3>
				</div>
				<button type="button" class="close-btn" onclick={onClose} aria-label="Close modal">
					<Icons name="x" size={18} />
				</button>
			</div>

			<p class="sheet-desc">
				Tell Antigravity why this <code class="tool-tag">{approval.toolName}</code> was denied and what it
				should do next:
			</p>

			<!-- Quick Chips -->
			<div class="chips-row">
				{#each quickChips as chip}
					<button type="button" class="chip" onclick={() => selectChip(chip)}>
						+ {chip}
					</button>
				{/each}
			</div>

			<!-- Input Area -->
			<div class="input-wrapper">
				<textarea
					bind:value={reasonText}
					placeholder="e.g. Please run linter first, or check package.json dependencies before installing..."
					rows="3"
					class="reason-textarea"
				></textarea>
			</div>

			<!-- Actions -->
			<div class="sheet-actions">
				<button type="button" class="cancel-btn" onclick={onClose} disabled={isSubmitting}>
					Cancel
				</button>
				<button
					type="button"
					class="send-btn"
					onclick={handleSubmit}
					disabled={isSubmitting || reasonText.trim().length === 0}
				>
					<Icons name="check" size={16} />
					<span>Send Guidance</span>
				</button>
			</div>
		</div>
	</div>
{/if}

<style>
	.modal-backdrop {
		position: fixed;
		inset: 0;
		background: rgba(0, 0, 0, 0.7);
		backdrop-filter: blur(8px);
		-webkit-backdrop-filter: blur(8px);
		z-index: 100;
		display: flex;
		align-items: flex-end;
		justify-content: center;
		animation: fadeIn 0.15s ease-out;
	}

	.modal-sheet {
		width: 100%;
		max-width: 540px;
		background: #0f172a;
		border-top-left-radius: 24px;
		border-top-right-radius: 24px;
		border: 1px solid rgba(255, 255, 255, 0.1);
		border-bottom: none;
		padding: 0.75rem 1.25rem env(safe-area-inset-bottom, 1.25rem) 1.25rem;
		box-shadow: 0 -10px 40px rgba(0, 0, 0, 0.6);
		animation: slideUp 0.2s cubic-bezier(0.16, 1, 0.3, 1);
	}

	.sheet-handle {
		width: 40px;
		height: 4px;
		border-radius: 2px;
		background: rgba(255, 255, 255, 0.2);
		margin: 0 auto 0.75rem auto;
	}

	.sheet-header {
		display: flex;
		align-items: center;
		justify-content: space-between;
		margin-bottom: 0.5rem;
	}

	.header-left {
		display: flex;
		align-items: center;
		gap: 0.5rem;
	}

	:global(.header-icon) {
		color: #818cf8;
	}

	.sheet-title {
		font-family: 'Outfit', sans-serif;
		font-size: 1.1rem;
		font-weight: 700;
		color: #ffffff;
		margin: 0;
	}

	.close-btn {
		background: transparent;
		border: none;
		color: #94a3b8;
		padding: 0.35rem;
		cursor: pointer;
		border-radius: 6px;
	}

	.sheet-desc {
		font-size: 0.8rem;
		color: #94a3b8;
		margin: 0 0 0.85rem 0;
		line-height: 1.4;
	}

	.tool-tag {
		font-family: 'JetBrains Mono', monospace;
		background: rgba(99, 102, 241, 0.2);
		color: #a5b4fc;
		padding: 0.1rem 0.35rem;
		border-radius: 4px;
	}

	.chips-row {
		display: flex;
		gap: 0.4rem;
		overflow-x: auto;
		scrollbar-width: none;
		margin-bottom: 0.85rem;
		padding-bottom: 2px;
	}
	.chips-row::-webkit-scrollbar {
		display: none;
	}

	.chip {
		background: rgba(30, 41, 59, 0.8);
		border: 1px solid rgba(255, 255, 255, 0.08);
		color: #cbd5e1;
		font-size: 0.72rem;
		font-weight: 500;
		padding: 0.3rem 0.6rem;
		border-radius: 9999px;
		white-space: nowrap;
		cursor: pointer;
		transition: all 0.15s ease;
	}

	.chip:active {
		background: rgba(99, 102, 241, 0.3);
		color: #ffffff;
	}

	.input-wrapper {
		margin-bottom: 1rem;
	}

	.reason-textarea {
		width: 100%;
		box-sizing: border-box;
		background: #080c14;
		border: 1px solid rgba(255, 255, 255, 0.12);
		border-radius: 12px;
		color: #ffffff;
		font-family: 'Inter', sans-serif;
		font-size: 0.88rem;
		padding: 0.75rem;
		resize: none;
		outline: none;
		transition: border-color 0.15s ease;
	}

	.reason-textarea:focus {
		border-color: #6366f1;
		box-shadow: 0 0 10px rgba(99, 102, 241, 0.2);
	}

	.sheet-actions {
		display: grid;
		grid-template-columns: 1fr 2fr;
		gap: 0.65rem;
	}

	.cancel-btn {
		background: rgba(30, 41, 59, 0.7);
		border: 1px solid rgba(255, 255, 255, 0.08);
		color: #94a3b8;
		padding: 0.75rem;
		border-radius: 12px;
		font-family: 'Outfit', sans-serif;
		font-size: 0.9rem;
		font-weight: 600;
		cursor: pointer;
	}

	.send-btn {
		display: inline-flex;
		align-items: center;
		justify-content: center;
		gap: 0.45rem;
		background: linear-gradient(135deg, #6366f1 0%, #4f46e5 100%);
		border: none;
		color: #ffffff;
		padding: 0.75rem;
		border-radius: 12px;
		font-family: 'Outfit', sans-serif;
		font-size: 0.92rem;
		font-weight: 700;
		cursor: pointer;
		box-shadow: 0 4px 14px rgba(99, 102, 241, 0.35);
	}

	.send-btn:disabled {
		opacity: 0.4;
		cursor: not-allowed;
	}

	@keyframes fadeIn {
		from {
			opacity: 0;
		}
		to {
			opacity: 1;
		}
	}
	@keyframes slideUp {
		from {
			transform: translateY(100%);
		}
		to {
			transform: translateY(0);
		}
	}
</style>
