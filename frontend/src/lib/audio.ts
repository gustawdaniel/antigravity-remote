// Synthesized sound effects using the Web Audio API (zero external assets required)

let audioCtx: AudioContext | null = null;
let soundEnabled = true;

if (typeof window !== 'undefined') {
	const stored = localStorage.getItem('agy_sound_enabled');
	if (stored !== null) {
		soundEnabled = stored === 'true';
	}
}

export function isSoundEnabled(): boolean {
	return soundEnabled;
}

export function setSoundEnabled(enabled: boolean) {
	soundEnabled = enabled;
	if (typeof window !== 'undefined') {
		localStorage.setItem('agy_sound_enabled', enabled ? 'true' : 'false');
	}
}

function getAudioContext(): AudioContext | null {
	if (typeof window === 'undefined') return null;
	if (!audioCtx) {
		const AudioContextClass = window.AudioContext || (window as any).webkitAudioContext;
		if (AudioContextClass) {
			audioCtx = new AudioContextClass();
		}
	}
	if (audioCtx && audioCtx.state === 'suspended') {
		audioCtx.resume().catch(() => {});
	}
	return audioCtx;
}

/**
 * High-priority notification chime when a new approval request arrives
 */
export function playApprovalChime() {
	if (!soundEnabled) return;
	try {
		const ctx = getAudioContext();
		if (!ctx) return;

		const now = ctx.currentTime;

		// Note 1: 587.33 Hz (D5)
		const osc1 = ctx.createOscillator();
		const gain1 = ctx.createGain();
		osc1.type = 'sine';
		osc1.frequency.setValueAtTime(587.33, now);
		gain1.gain.setValueAtTime(0, now);
		gain1.gain.linearRampToValueAtTime(0.25, now + 0.03);
		gain1.gain.exponentialRampToValueAtTime(0.001, now + 0.35);
		osc1.connect(gain1);
		gain1.connect(ctx.destination);
		osc1.start(now);
		osc1.stop(now + 0.35);

		// Note 2: 880 Hz (A5)
		const osc2 = ctx.createOscillator();
		const gain2 = ctx.createGain();
		osc2.type = 'sine';
		osc2.frequency.setValueAtTime(880, now + 0.12);
		gain2.gain.setValueAtTime(0, now + 0.12);
		gain2.gain.linearRampToValueAtTime(0.3, now + 0.15);
		gain2.gain.exponentialRampToValueAtTime(0.001, now + 0.65);
		osc2.connect(gain2);
		gain2.connect(ctx.destination);
		osc2.start(now + 0.12);
		osc2.stop(now + 0.65);

		// Note 3: 1174.66 Hz (D6) shimmer
		const osc3 = ctx.createOscillator();
		const gain3 = ctx.createGain();
		osc3.type = 'sine';
		osc3.frequency.setValueAtTime(1174.66, now + 0.24);
		gain3.gain.setValueAtTime(0, now + 0.24);
		gain3.gain.linearRampToValueAtTime(0.2, now + 0.27);
		gain3.gain.exponentialRampToValueAtTime(0.0001, now + 0.8);
		osc3.connect(gain3);
		gain3.connect(ctx.destination);
		osc3.start(now + 0.24);
		osc3.stop(now + 0.8);
	} catch (e) {
		console.warn('Audio play failed:', e);
	}
}

let lastFinishedChimeTime = 0;

/**
 * Calm, pleasant chime played when the agent finishes working on a task
 */
export function playAgentFinishedChime() {
	if (!soundEnabled) return;
	const now = Date.now();
	if (now - lastFinishedChimeTime < 8000) return;
	lastFinishedChimeTime = now;

	try {
		const ctx = getAudioContext();
		if (!ctx) return;
		const now = ctx.currentTime;

		// Gentle 2-tone melodic chime: G5 (784Hz) -> C6 (1046.5Hz)
		const osc1 = ctx.createOscillator();
		const gain1 = ctx.createGain();
		osc1.type = 'sine';
		osc1.frequency.setValueAtTime(783.99, now);
		gain1.gain.setValueAtTime(0, now);
		gain1.gain.linearRampToValueAtTime(0.2, now + 0.04);
		gain1.gain.exponentialRampToValueAtTime(0.001, now + 0.4);
		osc1.connect(gain1);
		gain1.connect(ctx.destination);
		osc1.start(now);
		osc1.stop(now + 0.4);

		const osc2 = ctx.createOscillator();
		const gain2 = ctx.createGain();
		osc2.type = 'sine';
		osc2.frequency.setValueAtTime(1046.5, now + 0.14);
		gain2.gain.setValueAtTime(0, now + 0.14);
		gain2.gain.linearRampToValueAtTime(0.25, now + 0.18);
		gain2.gain.exponentialRampToValueAtTime(0.0001, now + 0.85);
		osc2.connect(gain2);
		gain2.connect(ctx.destination);
		osc2.start(now + 0.14);
		osc2.stop(now + 0.85);
	} catch (e) {
		console.warn('Audio play failed:', e);
	}
}

/**
 * Positive confirmation chime
 */
export function playSuccessChime() {
	if (!soundEnabled) return;
	try {
		const ctx = getAudioContext();
		if (!ctx) return;
		const now = ctx.currentTime;

		const osc = ctx.createOscillator();
		const gain = ctx.createGain();
		osc.type = 'triangle';
		osc.frequency.setValueAtTime(523.25, now); // C5
		osc.frequency.exponentialRampToValueAtTime(783.99, now + 0.12); // G5
		gain.gain.setValueAtTime(0.18, now);
		gain.gain.exponentialRampToValueAtTime(0.001, now + 0.3);
		osc.connect(gain);
		gain.connect(ctx.destination);
		osc.start(now);
		osc.stop(now + 0.3);
	} catch {}
}

/**
 * Negative / Denied click
 */
export function playRejectTone() {
	if (!soundEnabled) return;
	try {
		const ctx = getAudioContext();
		if (!ctx) return;
		const now = ctx.currentTime;

		const osc = ctx.createOscillator();
		const gain = ctx.createGain();
		osc.type = 'sawtooth';
		osc.frequency.setValueAtTime(260, now);
		osc.frequency.exponentialRampToValueAtTime(140, now + 0.18);
		gain.gain.setValueAtTime(0.15, now);
		gain.gain.exponentialRampToValueAtTime(0.001, now + 0.22);
		osc.connect(gain);
		gain.connect(ctx.destination);
		osc.start(now);
		osc.stop(now + 0.22);
	} catch {}
}

/**
 * Haptic feedback helper
 */
export function triggerHaptic(pattern: number | number[] = 60) {
	if (typeof window !== 'undefined' && 'vibrate' in navigator) {
		try {
			navigator.vibrate(pattern);
		} catch {}
	}
}

let lastSpokenText = '';
let lastSpokenTime = 0;

/**
 * Voice announcement using Web Speech API (Text-to-Speech)
 */
export function speakText(text: string) {
	if (!soundEnabled) return;
	if (typeof window === 'undefined' || !('speechSynthesis' in window)) return;

	const now = Date.now();
	// Deduplicate identical speech announcements or completion phrases within 10 seconds
	if (now - lastSpokenTime < 10000 && (text === lastSpokenText || text.includes('Zadanie zakończone'))) {
		return;
	}
	lastSpokenText = text;
	lastSpokenTime = now;

	try {
		window.speechSynthesis.cancel();
		const utterance = new SpeechSynthesisUtterance(text);
		utterance.lang = 'pl-PL';
		utterance.rate = 1.0;
		utterance.pitch = 1.0;
		window.speechSynthesis.speak(utterance);
	} catch (e) {
		console.warn('Speech synthesis failed:', e);
	}
}

let lastNotificationTitle = '';
let lastNotificationTime = 0;

/**
 * Mobile-friendly notification helper using ServiceWorker if available
 */
export async function showBrowserNotification(title: string, options?: NotificationOptions) {
	if (typeof window === 'undefined' || !('Notification' in window)) return;
	if (Notification.permission !== 'granted') return;

	const now = Date.now();
	if (now - lastNotificationTime < 8000 && title === lastNotificationTitle) return;
	lastNotificationTitle = title;
	lastNotificationTime = now;

	try {
		if ('serviceWorker' in navigator) {
			const reg = await navigator.serviceWorker.ready;
			if (reg && reg.showNotification) {
				await reg.showNotification(title, options);
				return;
			}
		}
	} catch (e) {
		console.warn('ServiceWorker showNotification failed, trying fallback:', e);
	}

	try {
		new Notification(title, options);
	} catch (e) {
		console.warn('Notification constructor failed:', e);
	}
}

