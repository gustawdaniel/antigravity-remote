<script lang="ts">
	interface Props {
		name:
			| 'terminal'
			| 'file-code'
			| 'file'
			| 'check'
			| 'x'
			| 'message-square'
			| 'radio'
			| 'volume-2'
			| 'volume-x'
			| 'bell'
			| 'bell-off'
			| 'shield'
			| 'refresh-cw'
			| 'layers'
			| 'external-link'
			| 'cpu'
			| 'copy'
			| 'check-check'
			| 'clock'
			| 'folder'
			| 'chevron-right'
			| 'chevron-down'
			| 'sparkles'
			| 'smartphone'
			| 'arrow-down'
			| 'send';
		size?: number;
		class?: string;
	}

	let { name, size = 18, class: className = '' }: Props = $props();
</script>

{#if name === 'terminal'}
	<svg
		xmlns="http://www.w3.org/2000/svg"
		width={size}
		height={size}
		viewBox="0 0 24 24"
		fill="none"
		stroke="currentColor"
		stroke-width="2"
		stroke-linecap="round"
		stroke-linejoin="round"
		class={className}
	>
		<polyline points="4 17 10 11 4 5"></polyline>
		<line x1="12" y1="19" x2="20" y2="19"></line>
	</svg>
{:else if name === 'file-code'}
	<svg
		xmlns="http://www.w3.org/2000/svg"
		width={size}
		height={size}
		viewBox="0 0 24 24"
		fill="none"
		stroke="currentColor"
		stroke-width="2"
		stroke-linecap="round"
		stroke-linejoin="round"
		class={className}
	>
		<path d="M14.5 2H6a2 2 0 0 0-2 2v16a2 2 0 0 0 2 2h12a2 2 0 0 0 2-2V7.5L14.5 2z"></path>
		<polyline points="14 2 14 8 20 8"></polyline>
		<polyline points="10 13 8 15 10 17"></polyline>
		<polyline points="14 13 16 15 14 17"></polyline>
	</svg>
{:else if name === 'file'}
	<svg
		xmlns="http://www.w3.org/2000/svg"
		width={size}
		height={size}
		viewBox="0 0 24 24"
		fill="none"
		stroke="currentColor"
		stroke-width="2"
		stroke-linecap="round"
		stroke-linejoin="round"
		class={className}
	>
		<path d="M14.5 2H6a2 2 0 0 0-2 2v16a2 2 0 0 0 2 2h12a2 2 0 0 0 2-2V7.5L14.5 2z"></path>
		<polyline points="14 2 14 8 20 8"></polyline>
	</svg>
{:else if name === 'check'}
	<svg
		xmlns="http://www.w3.org/2000/svg"
		width={size}
		height={size}
		viewBox="0 0 24 24"
		fill="none"
		stroke="currentColor"
		stroke-width="2.5"
		stroke-linecap="round"
		stroke-linejoin="round"
		class={className}
	>
		<polyline points="20 6 9 17 4 12"></polyline>
	</svg>
{:else if name === 'x'}
	<svg
		xmlns="http://www.w3.org/2000/svg"
		width={size}
		height={size}
		viewBox="0 0 24 24"
		fill="none"
		stroke="currentColor"
		stroke-width="2.5"
		stroke-linecap="round"
		stroke-linejoin="round"
		class={className}
	>
		<line x1="18" y1="6" x2="6" y2="18"></line>
		<line x1="6" y1="6" x2="18" y2="18"></line>
	</svg>
{:else if name === 'message-square'}
	<svg
		xmlns="http://www.w3.org/2000/svg"
		width={size}
		height={size}
		viewBox="0 0 24 24"
		fill="none"
		stroke="currentColor"
		stroke-width="2"
		stroke-linecap="round"
		stroke-linejoin="round"
		class={className}
	>
		<path d="M21 15a2 2 0 0 1-2 2H7l-4 4V5a2 2 0 0 1 2-2h14a2 2 0 0 1 2 2z"></path>
	</svg>
{:else if name === 'radio'}
	<svg
		xmlns="http://www.w3.org/2000/svg"
		width={size}
		height={size}
		viewBox="0 0 24 24"
		fill="none"
		stroke="currentColor"
		stroke-width="2"
		stroke-linecap="round"
		stroke-linejoin="round"
		class={className}
	>
		<circle cx="12" cy="12" r="2"></circle>
		<path d="M16.24 7.76a6 6 0 0 1 0 8.49m-8.48-.01a6 6 0 0 1 0-8.49m11.31-2.82a10 10 0 0 1 0 14.14m-14.14 0a10 10 0 0 1 0-14.14"></path>
	</svg>
{:else if name === 'volume-2'}
	<svg
		xmlns="http://www.w3.org/2000/svg"
		width={size}
		height={size}
		viewBox="0 0 24 24"
		fill="none"
		stroke="currentColor"
		stroke-width="2"
		stroke-linecap="round"
		stroke-linejoin="round"
		class={className}
	>
		<polygon points="11 5 6 9 2 9 2 15 6 15 11 19 11 5"></polygon>
		<path d="M19.07 4.93a10 10 0 0 1 0 14.14M15.54 8.46a5 5 0 0 1 0 7.07"></path>
	</svg>
{:else if name === 'volume-x'}
	<svg
		xmlns="http://www.w3.org/2000/svg"
		width={size}
		height={size}
		viewBox="0 0 24 24"
		fill="none"
		stroke="currentColor"
		stroke-width="2"
		stroke-linecap="round"
		stroke-linejoin="round"
		class={className}
	>
		<polygon points="11 5 6 9 2 9 2 15 6 15 11 19 11 5"></polygon>
		<line x1="23" y1="9" x2="17" y2="15"></line>
		<line x1="17" y1="9" x2="23" y2="15"></line>
	</svg>
{:else if name === 'bell'}
	<svg
		xmlns="http://www.w3.org/2000/svg"
		width={size}
		height={size}
		viewBox="0 0 24 24"
		fill="none"
		stroke="currentColor"
		stroke-width="2"
		stroke-linecap="round"
		stroke-linejoin="round"
		class={className}
	>
		<path d="M18 8A6 6 0 0 0 6 8c0 7-3 9-3 9h18s-3-2-3-9"></path>
		<path d="M13.73 21a2 2 0 0 1-3.46 0"></path>
	</svg>
{:else if name === 'bell-off'}
	<svg
		xmlns="http://www.w3.org/2000/svg"
		width={size}
		height={size}
		viewBox="0 0 24 24"
		fill="none"
		stroke="currentColor"
		stroke-width="2"
		stroke-linecap="round"
		stroke-linejoin="round"
		class={className}
	>
		<path d="M13.73 21a2 2 0 0 1-3.46 0"></path>
		<path d="M18.63 13A17.89 17.89 0 0 1 18 8"></path>
		<path d="M6.26 6.26A5.86 5.86 0 0 0 6 8c0 7-3 9-3 9h14"></path>
		<path d="M18 8a6 6 0 0 0-9.33-5"></path>
		<line x1="1" y1="1" x2="23" y2="23"></line>
	</svg>
{:else if name === 'shield'}
	<svg
		xmlns="http://www.w3.org/2000/svg"
		width={size}
		height={size}
		viewBox="0 0 24 24"
		fill="none"
		stroke="currentColor"
		stroke-width="2"
		stroke-linecap="round"
		stroke-linejoin="round"
		class={className}
	>
		<path d="M12 22s8-4 8-10V5l-8-3-8 3v7c0 6 8 10 8 10z"></path>
	</svg>
{:else if name === 'refresh-cw'}
	<svg
		xmlns="http://www.w3.org/2000/svg"
		width={size}
		height={size}
		viewBox="0 0 24 24"
		fill="none"
		stroke="currentColor"
		stroke-width="2"
		stroke-linecap="round"
		stroke-linejoin="round"
		class={className}
	>
		<polyline points="23 4 23 10 17 10"></polyline>
		<polyline points="1 20 1 14 7 14"></polyline>
		<path d="M3.51 9a9 9 0 0 1 14.85-3.36L23 10M1 14l4.64 4.36A9 9 0 0 0 20.49 15"></path>
	</svg>
{:else if name === 'layers'}
	<svg
		xmlns="http://www.w3.org/2000/svg"
		width={size}
		height={size}
		viewBox="0 0 24 24"
		fill="none"
		stroke="currentColor"
		stroke-width="2"
		stroke-linecap="round"
		stroke-linejoin="round"
		class={className}
	>
		<polygon points="12 2 2 7 12 12 22 7 12 2"></polygon>
		<polyline points="2 17 12 22 22 17"></polyline>
		<polyline points="2 12 12 17 22 12"></polyline>
	</svg>
{:else if name === 'cpu'}
	<svg
		xmlns="http://www.w3.org/2000/svg"
		width={size}
		height={size}
		viewBox="0 0 24 24"
		fill="none"
		stroke="currentColor"
		stroke-width="2"
		stroke-linecap="round"
		stroke-linejoin="round"
		class={className}
	>
		<rect x="4" y="4" width="16" height="16" rx="2"></rect>
		<rect x="9" y="9" width="6" height="6"></rect>
		<line x1="9" y1="1" x2="9" y2="4"></line>
		<line x1="15" y1="1" x2="15" y2="4"></line>
		<line x1="9" y1="20" x2="9" y2="23"></line>
		<line x1="15" y1="20" x2="15" y2="23"></line>
		<line x1="20" y1="9" x2="23" y2="9"></line>
		<line x1="20" y1="14" x2="23" y2="14"></line>
		<line x1="1" y1="9" x2="4" y2="9"></line>
		<line x1="1" y1="14" x2="4" y2="14"></line>
	</svg>
{:else if name === 'copy'}
	<svg
		xmlns="http://www.w3.org/2000/svg"
		width={size}
		height={size}
		viewBox="0 0 24 24"
		fill="none"
		stroke="currentColor"
		stroke-width="2"
		stroke-linecap="round"
		stroke-linejoin="round"
		class={className}
	>
		<rect x="9" y="9" width="13" height="13" rx="2" ry="2"></rect>
		<path d="M5 15H4a2 2 0 0 1-2-2V4a2 2 0 0 1 2-2h9a2 2 0 0 1 2 2v1"></path>
	</svg>
{:else if name === 'check-check'}
	<svg
		xmlns="http://www.w3.org/2000/svg"
		width={size}
		height={size}
		viewBox="0 0 24 24"
		fill="none"
		stroke="currentColor"
		stroke-width="2.5"
		stroke-linecap="round"
		stroke-linejoin="round"
		class={className}
	>
		<path d="M18 6 7 17l-5-5"></path>
		<path d="m22 10-7.5 7.5L13 16"></path>
	</svg>
{:else if name === 'clock'}
	<svg
		xmlns="http://www.w3.org/2000/svg"
		width={size}
		height={size}
		viewBox="0 0 24 24"
		fill="none"
		stroke="currentColor"
		stroke-width="2"
		stroke-linecap="round"
		stroke-linejoin="round"
		class={className}
	>
		<circle cx="12" cy="12" r="10"></circle>
		<polyline points="12 6 12 12 16 14"></polyline>
	</svg>
{:else if name === 'folder'}
	<svg
		xmlns="http://www.w3.org/2000/svg"
		width={size}
		height={size}
		viewBox="0 0 24 24"
		fill="none"
		stroke="currentColor"
		stroke-width="2"
		stroke-linecap="round"
		stroke-linejoin="round"
		class={className}
	>
		<path d="M22 19a2 2 0 0 1-2 2H4a2 2 0 0 1-2-2V5a2 2 0 0 1 2-2h5l2 3h9a2 2 0 0 1 2 2z"></path>
	</svg>
{:else if name === 'chevron-right'}
	<svg
		xmlns="http://www.w3.org/2000/svg"
		width={size}
		height={size}
		viewBox="0 0 24 24"
		fill="none"
		stroke="currentColor"
		stroke-width="2"
		stroke-linecap="round"
		stroke-linejoin="round"
		class={className}
	>
		<polyline points="9 18 15 12 9 6"></polyline>
	</svg>
{:else if name === 'chevron-down'}
	<svg
		xmlns="http://www.w3.org/2000/svg"
		width={size}
		height={size}
		viewBox="0 0 24 24"
		fill="none"
		stroke="currentColor"
		stroke-width="2"
		stroke-linecap="round"
		stroke-linejoin="round"
		class={className}
	>
		<polyline points="6 9 12 15 18 9"></polyline>
	</svg>
{:else if name === 'sparkles'}
	<svg
		xmlns="http://www.w3.org/2000/svg"
		width={size}
		height={size}
		viewBox="0 0 24 24"
		fill="none"
		stroke="currentColor"
		stroke-width="2"
		stroke-linecap="round"
		stroke-linejoin="round"
		class={className}
	>
		<path d="m12 3-1.9 5.8a2 2 0 0 1-1.3 1.3L3 12l5.8 1.9a2 2 0 0 1 1.3 1.3L12 21l1.9-5.8a2 2 0 0 1 1.3-1.3L21 12l-5.8-1.9a2 2 0 0 1-1.3-1.3Z"></path>
	</svg>
{:else if name === 'smartphone'}
	<svg
		xmlns="http://www.w3.org/2000/svg"
		width={size}
		height={size}
		viewBox="0 0 24 24"
		fill="none"
		stroke="currentColor"
		stroke-width="2"
		stroke-linecap="round"
		stroke-linejoin="round"
		class={className}
	>
		<rect x="5" y="2" width="14" height="20" rx="2" ry="2"></rect>
		<line x1="12" y1="18" x2="12.01" y2="18"></line>
	</svg>
{:else if name === 'send'}
	<svg
		xmlns="http://www.w3.org/2000/svg"
		width={size}
		height={size}
		viewBox="0 0 24 24"
		fill="none"
		stroke="currentColor"
		stroke-width="2"
		stroke-linecap="round"
		stroke-linejoin="round"
		class={className}
	>
		<line x1="22" y1="2" x2="11" y2="13"></line>
		<polygon points="22 2 15 22 11 13 2 9 22 2"></polygon>
	</svg>
{:else if name === 'arrow-down'}
	<svg
		xmlns="http://www.w3.org/2000/svg"
		width={size}
		height={size}
		viewBox="0 0 24 24"
		fill="none"
		stroke="currentColor"
		stroke-width="2.5"
		stroke-linecap="round"
		stroke-linejoin="round"
		class={className}
	>
		<line x1="12" y1="5" x2="12" y2="19"></line>
		<polyline points="19 12 12 19 5 12"></polyline>
	</svg>
{/if}
