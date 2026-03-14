/**
 * Cinova session management
 *
 * Anonymous sessions allow full app usage before sign-up.
 * Credentials stored in expo-secure-store (hardware-backed on iOS,
 * Android Keystore on Android).
 */

import * as SecureStore from 'expo-secure-store';
import { createAnonymousSession } from './api';

const KEY_SESSION_ID = 'cinova_session_id';
const KEY_JWT = 'cinova_jwt';

// ── UUID v4 (no crypto module needed — pure JS) ───────────────────────────────

function uuidv4(): string {
  let dt = Date.now();
  return 'xxxxxxxx-xxxx-4xxx-yxxx-xxxxxxxxxxxx'.replace(/[xy]/g, (c) => {
    const r = (dt + Math.random() * 16) % 16 | 0;
    dt = Math.floor(dt / 16);
    return (c === 'x' ? r : (r & 0x3) | 0x8).toString(16);
  });
}

// ── Public API ────────────────────────────────────────────────────────────────

/**
 * Initialise session on first launch:
 *   1. Generate UUID and persist to SecureStore
 *   2. Exchange UUID for anonymous JWT from the API
 *   3. Persist JWT to SecureStore
 *
 * If a session already exists, this is a no-op and returns the
 * existing session ID.
 */
export async function initSession(): Promise<string> {
  let sessionId = await SecureStore.getItemAsync(KEY_SESSION_ID);

  if (!sessionId) {
    sessionId = uuidv4();
    await SecureStore.setItemAsync(KEY_SESSION_ID, sessionId);
  }

  const existingToken = await SecureStore.getItemAsync(KEY_JWT);
  if (!existingToken) {
    try {
      const { token } = await createAnonymousSession();
      await SecureStore.setItemAsync(KEY_JWT, token);
    } catch {
      // Fail silently — app will retry on first authenticated request
    }
  }

  return sessionId;
}

/** Returns the UUID session ID from SecureStore, or null if not initialised. */
export async function getSessionId(): Promise<string | null> {
  return SecureStore.getItemAsync(KEY_SESSION_ID);
}

/** Returns the current JWT from SecureStore, or null if not present. */
export async function getToken(): Promise<string | null> {
  return SecureStore.getItemAsync(KEY_JWT);
}

/** Persist a new JWT (called after login / signup / token refresh). */
export async function saveToken(token: string): Promise<void> {
  await SecureStore.setItemAsync(KEY_JWT, token);
}

/**
 * Clears all session data (called on logout).
 * The session UUID is cleared so the next initSession() generates a fresh one.
 */
export async function clearSession(): Promise<void> {
  await Promise.all([
    SecureStore.deleteItemAsync(KEY_SESSION_ID),
    SecureStore.deleteItemAsync(KEY_JWT),
  ]);
}
