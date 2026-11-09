<script lang="ts">
	import { tick } from 'svelte';
	import { marked } from 'marked';
	import type { TranscriptStep, WorkspaceSession } from '$lib/types';
	import Icons from './Icons.svelte';

	interface Props {
		sessions: WorkspaceSession[];
		selectedWorkspace: string;
		onSelectWorkspace?: (ws: string) => void;
	}

	let { sessions, selectedWorkspace, onSelectWorkspace }: Props = $props();

	let steps = $state<TranscriptStep[]>([]);
	let loading = $state(false);
	let autoRefresh = $state(true);
	let messageInput = $state('');
	let textareaEl = $state<HTMLTextAreaElement | null>(null);
	let sending = $state(false);
	let feedContainer = $state<HTMLElement | null>(null);
	let copiedIdx = $state<number | null>(null);
	let expandedOutputs = $state<Record<number, boolean>>({});

	function isTouchOrMobileDevice(): boolean {
		if (typeof window === 'undefined') return false;
		return (
			'ontouchstart' in window ||
			navigator.maxTouchPoints > 0 ||
			window.matchMedia('(pointer: coarse)').matches ||
			window.innerWidth <= 850 ||
			/Android|webOS|iPhone|iPad|iPod|BlackBerry|IEMobile|Opera Mini/i.test(navigator.userAgent)
		);
	}

	// Active workspace session
	let targetSession = $derived.by(() => {
		if (selectedWorkspace === 'all' || !selectedWorkspace) {
			return sessions[0] || null;
		}
		return sessions.find((s) => s.workspace === selectedWorkspace) || sessions[0] || null;
	});

	let pendingOptimisticSteps = $state<TranscriptStep[]>([]);
	let isUserScrolledUp = $state(false);
	let unreadCountWhileScrolledUp = $state(0);

	function scrollToBottom(smooth = false) {
		if (!feedContainer) return;
		if (smooth) {
			feedContainer.scrollTo({
				top: feedContainer.scrollHeight,
				behavior: 'smooth'
			});
		} else {
			feedContainer.scrollTop = feedContainer.scrollHeight;
		}
		isUserScrolledUp = false;
		unreadCountWhileScrolledUp = 0;
	}

	function isNearBottom(threshold = 70): boolean {
		if (!feedContainer) return true;
		const distance = feedContainer.scrollHeight - feedContainer.scrollTop - feedContainer.clientHeight;
		return distance <= threshold;
	}

	function handleScroll() {
		if (!feedContainer) return;
		const near = isNearBottom(70);
		isUserScrolledUp = !near;
		if (near) {
			unreadCountWhileScrolledUp = 0;
		}
	}

	function normalizeText(text: string | undefined): string {
		if (!text) return '';
		return text
			.replace(/\r\n/g, '\n')
			.replace(/\r/g, '\n')
			.trim()
			.toLowerCase();
	}

	async function fetchTranscript(forceScroll = false) {
		if (!targetSession) {
			steps = [];
			return;
		}
		loading = true;
		try {
			// Query by workspace name directly
			const res = await fetch(`/api/sessions/${targetSession.workspace}/transcript`);
			if (res.ok) {
				// Record whether user was near bottom right before updating steps
				const wasNear = !isUserScrolledUp && isNearBottom(70);
				const data: TranscriptStep[] = await res.json();

				// Detect if new steps or content changes actually arrived compared to current steps
				const prevNonOptimisticSteps = steps.filter(
					(s) => !pendingOptimisticSteps.some((opt) => opt.step_index === s.step_index)
				);
				const prevCount = prevNonOptimisticSteps.length;
				const prevLast = prevCount > 0 ? prevNonOptimisticSteps[prevCount - 1] : null;
				const newLast = data.length > 0 ? data[data.length - 1] : null;

				const hasNewSteps = data.length > prevCount;
				const lastStepUpdated = !!(
					newLast && prevLast && (
						newLast.step_index !== prevLast.step_index ||
						newLast.content !== prevLast.content ||
						newLast.clean_text !== prevLast.clean_text
					)
				);
				const hasNewContent = hasNewSteps || lastStepUpdated;

				// Filter out optimistic steps that are already present or expired (>60s)
				const now = Date.now();
				if (pendingOptimisticSteps.length > 0) {
					pendingOptimisticSteps = pendingOptimisticSteps.filter((opt) => {
						const optTime = new Date(opt.created_at || 0).getTime();
						if (now - optTime > 60 * 1000) return false;
						const optNorm = normalizeText(opt.clean_text || opt.content);
						return !data.some((s) => {
							if (s.type !== 'USER_INPUT') return false;
							const sCleanNorm = normalizeText(s.clean_text);
							const sContentNorm = normalizeText(s.content);
							return (
								(sCleanNorm !== '' && (sCleanNorm.includes(optNorm) || optNorm.includes(sCleanNorm))) ||
								(sContentNorm !== '' && (sContentNorm.includes(optNorm) || optNorm.includes(sContentNorm)))
							);
						});
					});
				}

				// Combine and sort strictly by timestamp
				const all = [...data, ...pendingOptimisticSteps];
				all.sort((a, b) => {
					const timeA = a.created_at ? new Date(a.created_at).getTime() : 0;
					const timeB = b.created_at ? new Date(b.created_at).getTime() : 0;
					if (timeA && timeB && timeA !== timeB) {
						return timeA - timeB;
					}
					return a.step_index - b.step_index;
				});
				steps = all;

				if (forceScroll) {
					// 1. Initial load / window switch / user sent message: snap to bottom!
					await tick();
					scrollToBottom(false);
					setTimeout(() => scrollToBottom(false), 50);
					setTimeout(() => scrollToBottom(false), 150);
					setTimeout(() => scrollToBottom(false), 300);
					isUserScrolledUp = false;
					unreadCountWhileScrolledUp = 0;
				} else if (hasNewContent) {
					if (wasNear) {
						// 2. New message arrived AND user was already at the bottom: scroll down
						await tick();
						scrollToBottom(false);
						setTimeout(() => scrollToBottom(false), 50);
						isUserScrolledUp = false;
						unreadCountWhileScrolledUp = 0;
					} else {
						// 3. New message arrived BUT user is scrolled up reading: DO NOT SCROLL!
						unreadCountWhileScrolledUp += (hasNewSteps ? data.length - prevCount : 1);
					}
				}
				// Notice: if !hasNewContent and !forceScroll, we DO NOT touch scroll at all!
			}
		} catch (e) {
			console.error('Failed to fetch transcript:', e);
		} finally {
			loading = false;
		}
	}

	let currentWorkspace = '';
	$effect(() => {
		const ws = selectedWorkspace || targetSession?.workspace || '';
		if (ws && ws !== currentWorkspace) {
			currentWorkspace = ws;
			steps = [];
			pendingOptimisticSteps = [];
			isUserScrolledUp = false;
			unreadCountWhileScrolledUp = 0;
			// Switched to a new workspace / window: force scroll to bottom!
			fetchTranscript(true);
		}
	});

	// Polling for live updates when viewing
	$effect(() => {
		if (!autoRefresh || !targetSession) return;
		const interval = setInterval(() => {
			fetchTranscript(false); // background polling: forceScroll is FALSE!
		}, 2500);
		return () => clearInterval(interval);
	});

	async function handleSendMessage() {
		const text = messageInput.trim();
		if (!text || !targetSession || sending) return;

		sending = true;
		messageInput = '';
		if (textareaEl) {
			textareaEl.style.height = 'auto';
		}

		// Optimistic append
		const optimisticStep: TranscriptStep = {
			step_index: steps.length > 0 ? steps[steps.length - 1].step_index + 1 : 1,
			source: 'USER_EXPLICIT',
			type: 'USER_INPUT',
			status: 'DONE',
			content: text,
			clean_text: text,
			created_at: new Date().toISOString()
		};
		pendingOptimisticSteps = [...pendingOptimisticSteps, optimisticStep];
		steps = [...steps, optimisticStep];
		await tick();
		scrollToBottom();

		try {
			const res = await fetch(`/api/sessions/${targetSession.workspace}/message`, {
				method: 'POST',
				headers: { 'Content-Type': 'application/json' },
				body: JSON.stringify({ message: text })
			});
			if (res.ok) {
				const body = await res.json();
				if (body.cascade_id && targetSession.id !== body.cascade_id) {
					targetSession.id = body.cascade_id;
				}
				// Refresh shortly to stream agent reaction
				setTimeout(fetchTranscript, 600);
				setTimeout(fetchTranscript, 1500);
				setTimeout(fetchTranscript, 3000);
			}
		} catch (e) {
			console.error('Failed to send message:', e);
		} finally {
			sending = false;
			await tick();
			scrollToBottom();
		}
	}

	function handleKeyDown(e: KeyboardEvent) {
		// On phone / touch devices: Enter (newline) NEVER sends message.
		// Sending is done explicitly via the Send button, or Ctrl+Enter / Cmd+Enter.
		if (isTouchOrMobileDevice()) {
			if ((e.ctrlKey || e.metaKey) && e.key === 'Enter') {
				e.preventDefault();
				handleSendMessage();
			}
			return; // Default Enter inserts newline
		}

		// On desktop with physical keyboard: Enter sends, Shift+Enter adds newline
		if (e.key === 'Enter' && !e.shiftKey) {
			e.preventDefault();
			handleSendMessage();
		}
	}

	function handleInput() {
		if (!textareaEl) return;
		textareaEl.style.height = 'auto';
		textareaEl.style.height = `${Math.min(textareaEl.scrollHeight, 140)}px`;
	}

	function renderMarkdown(content: string): string {
		try {
			return marked.parse(content) as string;
		} catch {
			return content;
		}
	}

	async function copyToClipboard(text: string, idx: number) {
		try {
			await navigator.clipboard.writeText(text);
			copiedIdx = idx;
			setTimeout(() => {
				copiedIdx = null;
			}, 2000);
		} catch {}
	}

	function toggleOutput(stepIdx: number) {
		expandedOutputs[stepIdx] = !expandedOutputs[stepIdx];
	}

	function formatTime(isoStr?: string): string {
		if (!isoStr) return '';
		try {
			const d = new Date(isoStr);
			return d.toLocaleTimeString([], { hour: '2-digit', minute: '2-digit', second: '2-digit' });
		} catch {
			return '';
		}
	}
</script>

<div class="chat-console">
	<!-- Feed Stream Area -->
	<div class="feed-container" bind:this={feedContainer} onscroll={handleScroll}>
		{#if !targetSession}
			<div class="empty-state">
				<Icons name="layers" size={36} class="empty-icon" />
				<p class="empty-title">No Active Workspace</p>
				<span class="empty-desc">Open a project in Antigravity IDE to stream console logs and chat.</span>
			</div>
		{:else if steps.length === 0}
			<div class="empty-state">
				<Icons name="cpu" size={36} class="empty-icon" />
				<p class="empty-title">Connected to "{targetSession.workspace}"</p>
				<span class="empty-desc">Send a task or command below to start working with the agent.</span>
			</div>
		{:else}
			<div class="stream-list">
				{#each steps as step, i}
					{#if step.type === 'USER_INPUT'}
						<!-- User Message Bubble -->
						<div class="msg-row user-row">
							<div class="user-bubble">
								<div class="bubble-header">
									<span class="sender-name">You</span>
									<div class="header-actions">
										<button
											type="button"
											class="copy-btn copy-inline-btn"
											onclick={() => copyToClipboard(step.clean_text || step.content, step.step_index)}
											title="Copy text"
										>
											{#if copiedIdx === step.step_index}
												<Icons name="check" size={11} class="copy-ok" />
											{:else}
												<Icons name="copy" size={11} />
											{/if}
										</button>
										<span class="bubble-time">{formatTime(step.created_at)}</span>
									</div>
								</div>
								<div class="bubble-text">{step.clean_text || step.content}</div>
							</div>
						</div>
					{:else if step.type === 'RUN_COMMAND'}
						<!-- Command Output Card -->
						<div class="command-block">
							<div class="command-header">
								<div class="cmd-info">
									<Icons name="terminal" size={13} class="cmd-icon" />
									<span class="cmd-line">{step.command_line || 'command'}</span>
								</div>
								<div class="cmd-actions">
									{#if step.exit_code !== undefined && step.exit_code !== null}
										<span class="exit-badge {step.exit_code === 0 ? 'exit-ok' : 'exit-err'}">
											{step.exit_code === 0 ? '✓ 0' : `✗ ${step.exit_code}`}
										</span>
									{/if}
									<button
										type="button"
										class="copy-btn"
										onclick={() => copyToClipboard(step.clean_text || step.content, step.step_index)}
										title="Copy output"
									>
										{#if copiedIdx === step.step_index}
											<Icons name="check" size={12} class="copy-ok" />
										{:else}
											<Icons name="copy" size={12} />
										{/if}
									</button>
								</div>
							</div>

							{#if step.clean_text || step.content}
								{@const output = step.clean_text || step.content}
								{@const lines = output.split('\n')}
								{@const isLong = lines.length > 12}
								{@const isExpanded = expandedOutputs[step.step_index]}

								<div class="terminal-body {isLong && !isExpanded ? 'truncated' : ''}">
									<pre><code>{isLong && !isExpanded ? lines.slice(0, 12).join('\n') + '\n...' : output}</code></pre>
								</div>

								{#if isLong}
									<button
										type="button"
										class="expand-btn"
										onclick={() => toggleOutput(step.step_index)}
									>
										{isExpanded ? 'Collapse output' : `Show full output (${lines.length} lines)`}
									</button>
								{/if}
							{/if}
						</div>
					{:else if step.type === 'PLANNER_RESPONSE' && (step.clean_text || step.content)}
						<!-- Model Markdown Response -->
						<div class="msg-row agent-row">
							<div class="agent-card">
								<div class="agent-header">
									<div class="agent-title">
										<Icons name="sparkles" size={14} class="agent-icon" />
										<span>Agent</span>
									</div>
									<div class="agent-header-actions">
										<button
											type="button"
											class="copy-btn copy-agent-btn"
											onclick={() => copyToClipboard(step.clean_text || step.content, step.step_index)}
											title="Copy response"
										>
											{#if copiedIdx === step.step_index}
												<Icons name="check" size={12} class="copy-ok" />
												<span class="copy-label">Copied</span>
											{:else}
												<Icons name="copy" size={12} />
												<span class="copy-label">Copy</span>
											{/if}
										</button>
										<span class="bubble-time">{formatTime(step.created_at)}</span>
									</div>
								</div>
								<div class="markdown-content">
									{@html renderMarkdown(step.clean_text || step.content)}
								</div>
							</div>
						</div>
					{:else if step.type === 'VIEW_FILE' || step.type === 'CODE_ACTION' || step.type === 'GREP_SEARCH' || step.type === 'LIST_DIRECTORY' || step.type === 'MCP_TOOL'}
						<!-- Tool Operation Pill -->
						<div class="tool-pill">
							<Icons name={step.type === 'MCP_TOOL' ? 'cpu' : 'file-code'} size={12} class="pill-icon" />
							<span class="pill-name">{step.tool_name || step.type.replace('_', ' ')}</span>
							{#if step.command_line}
								<span class="pill-target">{step.command_line}</span>
							{:else if step.content && step.type === 'MCP_TOOL'}
								<span class="pill-target">{step.content.slice(0, 40)}</span>
							{/if}
							<span class="pill-time">{formatTime(step.created_at)}</span>
						</div>
					{/if}
				{/each}
			</div>
		{/if}
	</div>

	<!-- Floating New Messages Pill -->
	{#if isUserScrolledUp && unreadCountWhileScrolledUp > 0}
		<button
			type="button"
			class="scroll-bottom-pill"
			onclick={() => {
				scrollToBottom(true);
			}}
		>
			<Icons name="arrow-down" size={14} class="pill-down-icon" />
			<span>Nowe wiadomości ({unreadCountWhileScrolledUp})</span>
		</button>
	{/if}

	<!-- Bottom Sticky Command / Prompt Input Bar -->
	<div class="input-bar">
		<div class="input-wrapper">
			<textarea
				bind:this={textareaEl}
				bind:value={messageInput}
				oninput={handleInput}
				onkeydown={handleKeyDown}
				placeholder={targetSession ? `Send task or command to ${targetSession.workspace}...` : 'Select workspace...'}
				rows={1}
				disabled={!targetSession || sending}
			></textarea>
			<button
				type="button"
				class="send-btn"
				onclick={handleSendMessage}
				disabled={!messageInput.trim() || sending || !targetSession}
				title="Send to agent"
			>
				{#if sending}
					<Icons name="refresh-cw" size={16} class="spinning" />
				{:else}
					<Icons name="send" size={16} />
				{/if}
			</button>
		</div>
	</div>
</div>

<style>
	.chat-console {
		display: flex;
		flex-direction: column;
		flex: 1;
		min-height: 0;
		height: 100%;
		max-height: 100%;
		position: relative;
		overflow: hidden;
		width: 100%;
		max-width: 100%;
	}

	.feed-container {
		flex: 1;
		min-height: 0;
		overflow-y: auto;
		overflow-x: hidden; /* Conversation itself never scrolls horizontally */
		padding: 0.75rem 0.5rem 1rem;
		display: flex;
		flex-direction: column;
		width: 100%;
		max-width: 100%;
		box-sizing: border-box;
	}

	.stream-list {
		display: flex;
		flex-direction: column;
		gap: 0.85rem;
		max-width: 900px;
		margin: 0 auto;
		width: 100%;
		min-width: 0;
		box-sizing: border-box;
	}

	/* User Message */
	.user-row {
		display: flex;
		justify-content: flex-end;
		width: 100%;
		min-width: 0;
	}

	.user-bubble {
		max-width: 88%;
		min-width: 0;
		background: linear-gradient(135deg, #4f46e5 0%, #6366f1 100%);
		color: #ffffff;
		border-radius: 1.15rem 1.15rem 0.25rem 1.15rem;
		padding: 0.75rem 1rem;
		box-shadow: 0 4px 14px rgba(79, 70, 229, 0.25);
		word-break: break-word;
		overflow-wrap: break-word;
	}

	.bubble-header {
		display: flex;
		align-items: center;
		justify-content: space-between;
		gap: 0.75rem;
		margin-bottom: 0.35rem;
	}

	.sender-name {
		font-size: 0.75rem;
		font-weight: 700;
		text-transform: uppercase;
		letter-spacing: 0.05em;
		opacity: 0.9;
	}

	.bubble-time {
		font-size: 0.68rem;
		opacity: 0.7;
	}

	.bubble-text {
		font-size: 0.92rem;
		line-height: 1.45;
		word-break: break-word;
		white-space: pre-wrap;
	}

	/* Agent Response Card */
	.agent-row {
		display: flex;
		justify-content: flex-start;
		width: 100%;
		min-width: 0;
	}

	.agent-card {
		width: 100%;
		max-width: 100%;
		min-width: 0;
		background: #111827;
		border: 1px solid rgba(255, 255, 255, 0.08);
		border-radius: 1rem;
		padding: 0.85rem 1rem;
		box-shadow: 0 4px 20px rgba(0, 0, 0, 0.35);
		box-sizing: border-box;
		overflow: hidden;
	}

	.agent-header {
		display: flex;
		align-items: center;
		justify-content: space-between;
		margin-bottom: 0.6rem;
		border-bottom: 1px solid rgba(255, 255, 255, 0.06);
		padding-bottom: 0.45rem;
	}

	.agent-title {
		display: flex;
		align-items: center;
		gap: 0.45rem;
		font-size: 0.82rem;
		font-weight: 700;
		color: #c084fc;
	}

	:global(.agent-icon) {
		color: #a855f7;
	}

	.markdown-content {
		font-size: 0.92rem;
		line-height: 1.6;
		color: #e2e8f0;
		min-width: 0;
		max-width: 100%;
		word-break: break-word;
		overflow-wrap: break-word;
	}

	:global(.markdown-content h1, .markdown-content h2, .markdown-content h3) {
		margin-top: 0.85rem;
		margin-bottom: 0.4rem;
		font-family: 'Outfit', sans-serif;
		font-weight: 700;
		color: #f8fafc;
	}

	:global(.markdown-content h1) { font-size: 1.25rem; }
	:global(.markdown-content h2) { font-size: 1.1rem; }
	:global(.markdown-content h3) { font-size: 1rem; }

	:global(.markdown-content p) {
		margin-bottom: 0.65rem;
	}

	:global(.markdown-content ul, .markdown-content ol) {
		margin-left: 1.25rem;
		margin-bottom: 0.65rem;
	}

	:global(.markdown-content pre) {
		background: #090d16;
		border: 1px solid #1e293b;
		border-radius: 0.5rem;
		padding: 0.65rem;
		overflow-x: auto;
		max-width: 100%;
		box-sizing: border-box;
		-webkit-overflow-scrolling: touch;
		font-family: 'JetBrains Mono', 'Fira Code', monospace;
		font-size: 0.82rem;
		margin-bottom: 0.65rem;
	}

	:global(.markdown-content code) {
		font-family: 'JetBrains Mono', monospace;
		background: rgba(255, 255, 255, 0.08);
		padding: 0.15rem 0.35rem;
		border-radius: 0.3rem;
		font-size: 0.85em;
		word-break: break-word;
	}

	:global(.markdown-content table) {
		display: block;
		max-width: 100%;
		overflow-x: auto;
		-webkit-overflow-scrolling: touch;
		border-collapse: collapse;
		margin-bottom: 0.75rem;
		font-size: 0.82rem;
	}

	:global(.markdown-content th, .markdown-content td) {
		border: 1px solid rgba(255, 255, 255, 0.1);
		padding: 0.4rem 0.6rem;
		text-align: left;
	}

	:global(.markdown-content th) {
		background: rgba(255, 255, 255, 0.05);
		font-weight: 600;
	}

	/* Command Execution Block (STDOUT/STDERR) */
	.command-block {
		background: #090d16;
		border: 1px solid rgba(56, 189, 248, 0.2);
		border-radius: 0.85rem;
		overflow: hidden;
		width: 100%;
		max-width: 100%;
		min-width: 0;
		box-sizing: border-box;
		box-shadow: 0 4px 16px rgba(0, 0, 0, 0.4);
	}

	.command-header {
		background: rgba(15, 23, 42, 0.85);
		border-bottom: 1px solid rgba(255, 255, 255, 0.06);
		padding: 0.5rem 0.85rem;
		display: flex;
		align-items: center;
		justify-content: space-between;
		gap: 0.5rem;
		min-width: 0;
	}

	.cmd-info {
		display: flex;
		align-items: center;
		gap: 0.45rem;
		overflow: hidden;
		min-width: 0;
		flex: 1;
	}

	:global(.cmd-icon) {
		color: #38bdf8;
		flex-shrink: 0;
	}

	.cmd-line {
		font-family: 'JetBrains Mono', monospace;
		font-size: 0.82rem;
		font-weight: 600;
		color: #38bdf8;
		white-space: nowrap;
		overflow: hidden;
		text-overflow: ellipsis;
		min-width: 0;
	}

	.cmd-actions {
		display: flex;
		align-items: center;
		gap: 0.45rem;
		flex-shrink: 0;
	}

	.exit-badge {
		font-family: 'JetBrains Mono', monospace;
		font-size: 0.68rem;
		font-weight: 700;
		padding: 0.15rem 0.45rem;
		border-radius: 0.3rem;
	}

	.exit-ok {
		background: rgba(16, 185, 129, 0.15);
		color: #10b981;
		border: 1px solid rgba(16, 185, 129, 0.3);
	}

	.exit-err {
		background: rgba(239, 68, 68, 0.15);
		color: #ef4444;
		border: 1px solid rgba(239, 68, 68, 0.3);
	}

	.copy-btn {
		background: rgba(255, 255, 255, 0.05);
		border: 1px solid rgba(255, 255, 255, 0.1);
		color: #94a3b8;
		border-radius: 0.35rem;
		padding: 0.25rem 0.4rem;
		display: flex;
		align-items: center;
		cursor: pointer;
		transition: all 0.15s ease;
	}

	.copy-btn:hover {
		background: rgba(255, 255, 255, 0.15);
		color: #f8fafc;
	}

	:global(.copy-ok) {
		color: #10b981;
	}

	.terminal-body {
		padding: 0.75rem 0.85rem;
		background: #060911;
		overflow-x: auto;
		max-width: 100%;
		box-sizing: border-box;
		-webkit-overflow-scrolling: touch;
		color: #cbd5e1;
		font-family: 'JetBrains Mono', 'Fira Code', monospace;
		font-size: 0.8rem;
		line-height: 1.45;
	}

	.terminal-body pre {
		margin: 0;
		white-space: pre;
		word-break: normal;
	}

	.expand-btn {
		width: 100%;
		background: rgba(255, 255, 255, 0.03);
		border: none;
		border-top: 1px solid rgba(255, 255, 255, 0.06);
		color: #38bdf8;
		font-size: 0.75rem;
		font-weight: 600;
		padding: 0.4rem;
		cursor: pointer;
		text-align: center;
		transition: background 0.15s ease;
	}

	.expand-btn:hover {
		background: rgba(56, 189, 248, 0.08);
	}

	/* Tool Pill */
	.tool-pill {
		align-self: center;
		display: flex;
		align-items: center;
		gap: 0.45rem;
		background: rgba(255, 255, 255, 0.03);
		border: 1px solid rgba(255, 255, 255, 0.06);
		padding: 0.3rem 0.65rem;
		border-radius: 2rem;
		font-size: 0.75rem;
		color: #94a3b8;
		max-width: 95%;
		min-width: 0;
		box-sizing: border-box;
	}

	:global(.pill-icon) {
		color: #64748b;
	}

	.pill-name {
		font-weight: 600;
		color: #cbd5e1;
	}

	.pill-target {
		font-family: 'JetBrains Mono', monospace;
		font-size: 0.7rem;
		color: #a855f7;
		max-width: 140px;
		overflow: hidden;
		text-overflow: ellipsis;
		white-space: nowrap;
	}

	.pill-time {
		font-size: 0.68rem;
		opacity: 0.6;
	}

	/* Bottom Input Bar */
	.input-bar {
		background: rgba(15, 23, 42, 0.95);
		backdrop-filter: blur(12px);
		border-top: 1px solid rgba(255, 255, 255, 0.1);
		padding: 0.5rem 0.75rem max(0.5rem, env(safe-area-inset-bottom, 0.5rem));
		position: relative;
		z-index: 50;
		flex-shrink: 0;
		transition: padding 0.15s ease;
	}

	.input-bar:focus-within {
		padding-bottom: 0.25rem;
		padding-top: 0.35rem;
	}

	.input-wrapper {
		display: flex;
		align-items: center;
		gap: 0.5rem;
		background: #1e293b;
		border: 1px solid rgba(255, 255, 255, 0.12);
		border-radius: 1.5rem;
		padding: 0.35rem 0.5rem 0.35rem 1rem;
		max-width: 900px;
		margin: 0 auto;
		transition: border-color 0.2s ease;
	}

	.input-wrapper:focus-within {
		border-color: #6366f1;
		box-shadow: 0 0 0 2px rgba(99, 102, 241, 0.2);
	}

	textarea {
		flex: 1;
		background: transparent;
		border: none;
		outline: none;
		color: #f8fafc;
		font-family: inherit;
		font-size: 0.95rem;
		line-height: 1.4;
		resize: none;
		min-height: 24px;
		max-height: 140px;
		overflow-y: auto;
		padding: 0.35rem 0;
	}

	textarea::placeholder {
		color: #64748b;
	}

	.send-btn {
		width: 38px;
		height: 38px;
		border-radius: 50%;
		background: linear-gradient(135deg, #4f46e5 0%, #6366f1 100%);
		border: none;
		color: white;
		display: flex;
		align-items: center;
		justify-content: center;
		cursor: pointer;
		flex-shrink: 0;
		transition: all 0.15s ease;
	}

	.send-btn:hover:not(:disabled) {
		transform: scale(1.05);
		box-shadow: 0 4px 12px rgba(79, 70, 229, 0.4);
	}

	.send-btn:disabled {
		opacity: 0.4;
		cursor: not-allowed;
	}

	/* Empty State */
	.empty-state {
		display: flex;
		flex-direction: column;
		align-items: center;
		justify-content: center;
		height: 100%;
		text-align: center;
		padding: 3rem 1.5rem;
		color: #64748b;
	}

	:global(.empty-icon) {
		margin-bottom: 0.85rem;
		opacity: 0.5;
	}

	.empty-title {
		font-size: 1.1rem;
		font-weight: 700;
		color: #e2e8f0;
		margin-bottom: 0.35rem;
	}

	.empty-desc {
		font-size: 0.85rem;
		max-width: 320px;
	}

	:global(.spinning) {
		animation: spin 1s linear infinite;
	}

	@keyframes spin {
		from { transform: rotate(0deg); }
		to { transform: rotate(360deg); }
	}

	.scroll-bottom-pill {
		position: absolute;
		bottom: 4.8rem;
		left: 50%;
		transform: translateX(-50%);
		z-index: 30;
		display: inline-flex;
		align-items: center;
		gap: 0.45rem;
		padding: 0.45rem 1rem;
		border-radius: 9999px;
		background: #6366f1;
		color: #ffffff;
		border: 1px solid rgba(255, 255, 255, 0.25);
		font-family: 'Outfit', sans-serif;
		font-size: 0.82rem;
		font-weight: 700;
		box-shadow: 0 4px 18px rgba(99, 102, 241, 0.45);
		cursor: pointer;
		animation: bounce-subtle 1.8s infinite ease-in-out;
		transition: all 0.15s ease;
		user-select: none;
		-webkit-tap-highlight-color: transparent;
	}

	.scroll-bottom-pill:active {
		transform: translateX(-50%) scale(0.95);
		background: #4f46e5;
	}

	:global(.pill-down-icon) {
		color: #ffffff;
	}

	@keyframes bounce-subtle {
		0%, 100% {
			transform: translateX(-50%) translateY(0);
		}
		50% {
			transform: translateX(-50%) translateY(-3px);
		}
	}
</style>
