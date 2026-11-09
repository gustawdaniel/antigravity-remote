export interface ToolCall {
	name: string;
	args: Record<string, any>;
}

export interface Approval {
	id: string;
	workspace: string;
	conversationId: string;
	stepIdx: number;
	toolName: string;
	toolArgs: Record<string, any>;
	commandLine?: string;
	filePath?: string;
	diff?: string;
	description?: string;
	status: 'pending' | 'allowed' | 'denied' | 'timed_out';
	isCritical?: boolean;
	autoApproved?: boolean;
	createdAt: string;
	decidedAt?: string;
	decisionReason?: string;
}

export interface ServerSettings {
	mode: 'smart' | 'auto' | 'manual';
}

export interface WorkspaceSession {
	id: string; // conversationId
	workspace: string;
	workspacePath: string;
	transcriptPath: string;
	lastToolName: string;
	status: 'idle' | 'running' | 'waiting_approval';
	lastActive: string;
}

export interface TranscriptStep {
	step_index: number;
	source: string;
	type: string;
	status: string;
	content: string;
	clean_text?: string;
	exit_code?: number;
	created_at?: string;
	command_line?: string;
	tool_name?: string;
	tool_calls?: Array<{
		name: string;
		command_line?: string;
		file_path?: string;
		toolSummary?: string;
		toolAction?: string;
	}>;
	is_truncated?: boolean;
}

export interface WSMessage {
	type:
		| 'init'
		| 'approval_created'
		| 'approval_resolved'
		| 'session_updated'
		| 'workspaces'
		| 'user_message_sent'
		| 'settings_updated'
		| 'transcript_append'
		| 'agent_finished'
		| 'agent_notice';
	payload: any;
}
