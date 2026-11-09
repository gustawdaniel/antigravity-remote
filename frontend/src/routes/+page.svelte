<script lang="ts">
	import { onMount } from 'svelte';
	import type { Approval, WorkspaceSession, WSMessage } from '$lib/types';
	import {
		playAgentFinishedChime,
		playApprovalChime,
		triggerHaptic,
		speakText,
		showBrowserNotification
	} from '$lib/audio';
	import Navbar from '$lib/components/Navbar.svelte';
	import WorkspaceSelector from '$lib/components/WorkspaceSelector.svelte';
	import ApprovalCard from '$lib/components/ApprovalCard.svelte';
	import FeedbackModal from '$lib/components/FeedbackModal.svelte';
	import LiveFeed from '$lib/components/LiveFeed.svelte';
	import SessionsList from '$lib/components/SessionsList.svelte';
	import Icons from '$lib/components/Icons.svelte';

	let approvals = $state<Approval[]>([]);
	let sessions = $state<WorkspaceSession[]>([]);
	let selectedWorkspace = $state<string>('');
	let activeTab = $state<'feed' | 'windows' | 'approvals'>('feed'); // Console & Chat is default!
	let connectionStatus = $state<'connected' | 'connecting' | 'disconnected'>('connecting');
	let feedbackApproval = $state<Approval | null>(null);
	let mode = $state<'smart' | 'auto' | 'manual'>('auto'); // Default full auto

	let ws: WebSocket | null = null;
	let reconnectTimer: any = null;

	// Active workspace fallback
	$effect(() => {
		if (!selectedWorkspace && sessions.length > 0) {
			selectedWorkspace = sessions[0].workspace;
		}
	});

	// Filtered approvals based on workspace
	let filteredApprovals = $derived.by(() => {
		if (selectedWorkspace === 'all' || !selectedWorkspace) {
			return approvals;
		}
		return approvals.filter((a) => a.workspace === selectedWorkspace);
	});

	let pendingApprovals = $derived.by(() => {
		return filteredApprovals.filter((a) => a.status === 'pending');
	});

	let resolvedApprovals = $derived.by(() => {
		return filteredApprovals.filter((a) => a.status !== 'pending');
	});

	let totalPendingCount = $derived.by(() => {
		return approvals.filter((a) => a.status === 'pending').length;
	});

	// WebSocket connection
	function connectWebSocket() {
		if (typeof window === 'undefined') return;

		connectionStatus = 'connecting';
		const protocol = window.location.protocol === 'https:' ? 'wss:' : 'ws:';
		const wsUrl = `${protocol}//${window.location.host}/ws`;

		try {
			ws = new WebSocket(wsUrl);

			ws.onopen = () => {
				connectionStatus = 'connected';
				clearTimeout(reconnectTimer);
			};

			ws.onmessage = (event) => {
				try {
					const msg: WSMessage = JSON.parse(event.data);
					handleWSMessage(msg);
				} catch (e) {
					console.error('Failed to parse WS message:', e);
				}
			};

			ws.onclose = () => {
				connectionStatus = 'disconnected';
				scheduleReconnect();
			};

			ws.onerror = () => {
				connectionStatus = 'disconnected';
				ws?.close();
			};
		} catch {
			connectionStatus = 'disconnected';
			scheduleReconnect();
		}
	}

	function scheduleReconnect() {
		clearTimeout(reconnectTimer);
		reconnectTimer = setTimeout(() => {
			connectWebSocket();
		}, 3000);
	}

	function handleWSMessage(msg: WSMessage) {
		switch (msg.type) {
			case 'init':
				if (Array.isArray(msg.payload)) {
					approvals = msg.payload;
				}
				break;

			case 'workspaces':
				if (Array.isArray(msg.payload)) {
					sessions = msg.payload;
					if (!selectedWorkspace && sessions.length > 0) {
						selectedWorkspace = sessions[0].workspace;
					}
				}
				break;

			case 'approval_created': {
				const newApp: Approval = msg.payload;
				const exists = approvals.some((a) => a.id === newApp.id);
				if (!exists) {
					approvals = [newApp, ...approvals];
				}

				// ONLY notify/chime if TRULY pending human approval (in manual mode).
				// Never spam user when auto-approved!
				if (newApp.status === 'pending' && !newApp.autoApproved) {
					playApprovalChime();
					triggerHaptic([120, 80, 120]);
					showBrowserNotification(`Antigravity Approval [${newApp.workspace}]`, {
						body: `${newApp.toolName}: ${newApp.commandLine || newApp.filePath || 'Wymagane zatwierdzenie'}`,
						icon: '/icon.svg'
					});
				}
				break;
			}

			case 'approval_resolved': {
				const resolved: Approval = msg.payload;
				approvals = approvals.map((a) => (a.id === resolved.id ? resolved : a));
				break;
			}

			case 'agent_finished': {
				playAgentFinishedChime();
				triggerHaptic([80, 50, 80]);
				speakText(`Zadanie zakończone. Antigravity w projekcie ${msg.payload?.workspace || ''} skończył pracę.`);
				showBrowserNotification(`Antigravity [${msg.payload?.workspace || 'Gotowe'}]`, {
					body: 'Agent skończył pracę nad zadaniem!',
					icon: '/icon.svg'
				});
				break;
			}

			case 'agent_notice': {
				triggerHaptic([200, 100, 200]);
				showBrowserNotification(`Antigravity [${msg.payload?.workspace || ''}]`, {
					body: msg.payload?.message || 'Agent wymaga uwagi',
					icon: '/icon.svg'
				});
				break;
			}

			case 'session_updated': {
				const updatedSess: WorkspaceSession = msg.payload;
				const idx = sessions.findIndex((s) => s.workspace === updatedSess.workspace);
				if (idx >= 0) {
					sessions[idx] = updatedSess;
					sessions = [...sessions];
				} else {
					sessions = [...sessions, updatedSess];
				}
				if (!selectedWorkspace) {
					selectedWorkspace = updatedSess.workspace;
				}
				break;
			}

			case 'settings_updated': {
				if (msg.payload && msg.payload.mode) {
					mode = msg.payload.mode;
				}
				break;
			}
		}
	}

	async function fetchInitialData() {
		try {
			const [appRes, wsRes, setRes] = await Promise.all([
				fetch('/api/approvals'),
				fetch('/api/workspaces'),
				fetch('/api/settings')
			]);
			if (appRes.ok) {
				approvals = await appRes.json();
			}
			if (wsRes.ok) {
				sessions = await wsRes.json();
				if (!selectedWorkspace && sessions.length > 0) {
					selectedWorkspace = sessions[0].workspace;
				}
			}
			if (setRes.ok) {
				const settings = await setRes.json();
				if (settings.mode) mode = settings.mode;
			}
		} catch (e) {
			console.error('Fetch initial data failed:', e);
		}
	}

	async function handleToggleMode(newMode: 'smart' | 'auto' | 'manual') {
		mode = newMode;
		try {
			await fetch('/api/settings', {
				method: 'POST',
				headers: { 'Content-Type': 'application/json' },
				body: JSON.stringify({ mode: newMode })
			});
		} catch (e) {
			console.error('Failed to update mode:', e);
		}
	}

	async function handleDecide(id: string, decision: 'allow' | 'deny', reason?: string) {
		try {
			const res = await fetch(`/api/approvals/${id}/decide`, {
				method: 'POST',
				headers: { 'Content-Type': 'application/json' },
				body: JSON.stringify({ decision, reason: reason || '' })
			});
			if (res.ok) {
				const updated: Approval = await res.json();
				approvals = approvals.map((a) => (a.id === updated.id ? updated : a));
			}
		} catch (e) {
			alert('Failed to send decision: ' + e);
		}
	}

	function openFeedbackModal(approval: Approval) {
		feedbackApproval = approval;
	}

	function closeFeedbackModal() {
		feedbackApproval = null;
	}

	onMount(() => {
		fetchInitialData();
		connectWebSocket();

		return () => {
			clearTimeout(reconnectTimer);
			ws?.close();
		};
	});
</script>

<svelte:head>
	<title>
		{totalPendingCount > 0 ? `(${totalPendingCount}) ` : ''}Antigravity Remote
	</title>
</svelte:head>

<div class="page-container">
	<Navbar onRefresh={fetchInitialData} />

	<WorkspaceSelector
		{sessions}
		{approvals}
		{selectedWorkspace}
		onSelect={(ws) => (selectedWorkspace = ws)}
	/>

	<!-- Main Tab Navigation -->
	<div class="tabs-nav">
		<button
			type="button"
			class="tab-btn {activeTab === 'feed' ? 'active' : ''}"
			onclick={() => {
				triggerHaptic(25);
				activeTab = 'feed';
			}}
		>
			<Icons name="terminal" size={15} />
			<span>Console & Chat</span>
		</button>

		<button
			type="button"
			class="tab-btn {activeTab === 'windows' ? 'active' : ''}"
			onclick={() => {
				triggerHaptic(25);
				activeTab = 'windows';
			}}
		>
			<Icons name="layers" size={15} />
			<span>Windows</span>
			{#if sessions.length > 0}
				<span class="tab-badge neutral">{sessions.length}</span>
			{/if}
		</button>

		<button
			type="button"
			class="tab-btn {activeTab === 'approvals' ? 'active' : ''}"
			onclick={() => {
				triggerHaptic(25);
				activeTab = 'approvals';
			}}
		>
			<Icons name="shield" size={15} />
			<span>Approvals</span>
			{#if pendingApprovals.length > 0}
				<span class="tab-badge">{pendingApprovals.length}</span>
			{/if}
		</button>
	</div>

	<!-- Main Content Area -->
	<main class="content-view {activeTab === 'feed' ? 'feed-view' : ''}">
		{#if activeTab === 'feed'}
			{#key selectedWorkspace}
				<LiveFeed
					{sessions}
					{selectedWorkspace}
					onSelectWorkspace={(ws) => (selectedWorkspace = ws)}
				/>
			{/key}
		{:else if activeTab === 'windows'}
			<SessionsList
				{sessions}
				onSelectWorkspace={(ws) => {
					selectedWorkspace = ws;
					activeTab = 'feed'; // Jump directly to chat for that window!
				}}
			/>
		{:else if activeTab === 'approvals'}
			<!-- Pending Approvals Section -->
			{#if pendingApprovals.length > 0}
				<div class="section-title-row">
					<span class="pulse-indicator"></span>
					<h2 class="section-title">Pending Approvals ({pendingApprovals.length})</h2>
				</div>

				<div class="approvals-list">
					{#each pendingApprovals as app (app.id)}
						<ApprovalCard
							approval={app}
							onDecide={handleDecide}
							onOpenFeedback={openFeedbackModal}
						/>
					{/each}
				</div>
			{:else}
				<div class="all-clear-card">
					<div class="orbit-radar">
						<div class="radar-sweep"></div>
						<div class="radar-dot dot-1"></div>
						<div class="radar-dot dot-2"></div>
						<Icons name="shield" size={38} class="radar-center-icon" />
					</div>
					<h3 class="all-clear-title">Full Auto Active</h3>
					<p class="all-clear-text">
						All agent actions run automatically without blocking or interrupting your workflow.
					</p>
					<span class="all-clear-hint">Use Console & Chat tab to interact and inspect full command outputs.</span>
				</div>
			{/if}

			<!-- Resolved History Section -->
			{#if resolvedApprovals.length > 0}
				<div class="history-divider">
					<span>Recent Action History</span>
				</div>

				<div class="approvals-list resolved">
					{#each resolvedApprovals.slice(0, 15) as app (app.id)}
						<ApprovalCard
							approval={app}
							onDecide={handleDecide}
							onOpenFeedback={openFeedbackModal}
						/>
					{/each}
				</div>
			{/if}
		{/if}
	</main>

	<!-- Feedback Modal -->
	<FeedbackModal
		approval={feedbackApproval}
		onClose={closeFeedbackModal}
		onSubmit={async (id, reason) => {
			await handleDecide(id, 'deny', reason);
		}}
	/>
</div>

<style>
	.page-container {
		display: flex;
		flex-direction: column;
		height: 100vh;
		height: 100dvh;
		overflow: hidden;
	}

	/* Tab Navigation */
	.tabs-nav {
		display: flex;
		background: rgba(15, 23, 42, 0.7);
		border-bottom: 1px solid rgba(255, 255, 255, 0.08);
		padding: 0.35rem 0.75rem;
		gap: 0.4rem;
		backdrop-filter: blur(16px);
		-webkit-backdrop-filter: blur(16px);
		flex-shrink: 0;
	}

	.tab-btn {
		flex: 1;
		display: inline-flex;
		align-items: center;
		justify-content: center;
		gap: 0.45rem;
		padding: 0.5rem 0.4rem;
		border-radius: 8px;
		background: transparent;
		border: 1px solid transparent;
		color: #94a3b8;
		font-family: 'Outfit', sans-serif;
		font-size: 0.85rem;
		font-weight: 600;
		cursor: pointer;
		transition: all 0.15s ease;
	}

	.tab-btn.active {
		color: #ffffff;
		background: rgba(30, 41, 59, 0.85);
		border-color: rgba(255, 255, 255, 0.12);
		box-shadow: 0 2px 8px rgba(0, 0, 0, 0.25);
	}

	.tab-badge {
		font-size: 0.65rem;
		font-weight: 700;
		background: #f43f5e;
		color: #ffffff;
		padding: 0.1rem 0.4rem;
		border-radius: 9999px;
		line-height: 1;
	}

	.tab-badge.neutral {
		background: rgba(100, 116, 139, 0.35);
		color: #cbd5e1;
	}

	/* Content Area */
	.content-view {
		flex: 1;
		overflow-y: auto;
		display: flex;
		flex-direction: column;
	}

	.content-view:not(.feed-view) {
		padding: 1rem;
		max-width: 680px;
		width: 100%;
		margin: 0 auto;
		box-sizing: border-box;
	}

	.feed-view {
		padding: 0;
		overflow: hidden;
	}

	.section-title-row {
		display: flex;
		align-items: center;
		gap: 0.5rem;
		margin-bottom: 0.85rem;
	}

	.pulse-indicator {
		width: 8px;
		height: 8px;
		border-radius: 50%;
		background: #f43f5e;
		box-shadow: 0 0 10px #f43f5e;
	}

	.section-title {
		font-family: 'Outfit', sans-serif;
		font-size: 0.95rem;
		font-weight: 700;
		color: #f8fafc;
	}

	.approvals-list {
		display: flex;
		flex-direction: column;
		gap: 0.75rem;
	}

	.all-clear-card {
		background: rgba(15, 23, 42, 0.6);
		border: 1px solid rgba(255, 255, 255, 0.06);
		border-radius: 16px;
		padding: 2.5rem 1.5rem;
		text-align: center;
		display: flex;
		flex-direction: column;
		align-items: center;
		justify-content: center;
	}

	.orbit-radar {
		position: relative;
		width: 80px;
		height: 80px;
		border-radius: 50%;
		border: 1px dashed rgba(56, 189, 248, 0.25);
		display: flex;
		align-items: center;
		justify-content: center;
		margin-bottom: 1.25rem;
	}

	:global(.radar-center-icon) {
		color: #38bdf8;
	}

	.all-clear-title {
		font-family: 'Outfit', sans-serif;
		font-size: 1.15rem;
		font-weight: 700;
		color: #ffffff;
		margin-bottom: 0.35rem;
	}

	.all-clear-text {
		font-size: 0.88rem;
		color: #94a3b8;
		max-width: 320px;
		margin-bottom: 0.5rem;
		line-height: 1.45;
	}

	.all-clear-hint {
		font-size: 0.75rem;
		color: #64748b;
	}

	.history-divider {
		display: flex;
		align-items: center;
		margin: 1.5rem 0 0.85rem;
		color: #64748b;
		font-size: 0.75rem;
		font-weight: 700;
		text-transform: uppercase;
		letter-spacing: 0.08em;
	}

	.history-divider::before,
	.history-divider::after {
		content: '';
		flex: 1;
		height: 1px;
		background: rgba(255, 255, 255, 0.06);
	}

	.history-divider span {
		padding: 0 0.75rem;
	}
</style>
