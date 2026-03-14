/**
 * Cinova session management
 *
 * Anonymous sessions allow full app usage before sign-up.
 * Credentials stored in expo-secure-store (hardware-backed on iOS,
 * Android Keystore on Android).
 */

import * as SecureStore from 'expo-secure-store';
import axios from 'axios';

const API_BASE = 'https://api.cinova.openova.io';

async function fetchAnonymousToken(deviceId: string): Promise<string> {
  const res = await axios.post(`${API_BASE}/api/v1/auth/anonymous`, { device_id: deviceId });
  return res.data.access_token;
}

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
  let sessionId: string | null = null;
  try { sessionId = await SecureStore.getItemAsync(KEY_SESSION_ID); } catch { /* web */ }

  if (!sessionId) {
    sessionId = uuidv4();
    try { await SecureStore.setItemAsync(KEY_SESSION_ID, sessionId); } catch { /* web */ }
  }

  let existingToken: string | null = null;
  try { existingToken = await SecureStore.getItemAsync(KEY_JWT); } catch { /* web */ }

  if (!existingToken) {
    try {
      const token = await fetchAnonymousToken(sessionId);
      try { await SecureStore.setItemAsync(KEY_JWT, token); } catch { /* web */ }
    } catch {
      // Fail silently — app will retry on first authenticated request
    }
  }

  return sessionId;
}

/** Returns the UUID session ID from SecureStore, or null if not initialised. */
export async function getSessionId(): Promise<string | null> {
  try { return await SecureStore.getItemAsync(KEY_SESSION_ID); } catch { return null; }
}

/** Returns the current JWT from SecureStore, or null if not present. */
export async function getToken(): Promise<string | null> {
  try { return await SecureStore.getItemAsync(KEY_JWT); } catch { return null; }
}

/** Persist a new JWT (called after login / signup / token refresh). */
export async function saveToken(token: string): Promise<void> {
  try { await SecureStore.setItemAsync(KEY_JWT, token); } catch { /* web fallback */ }
}

/** Remove JWT only (keeps session UUID). Called on 401 refresh failure. */
export async function clearToken(): Promise<void> {
  try { await SecureStore.deleteItemAsync(KEY_JWT); } catch { /* web fallback */ }
}

/**
 * Clears all session data (called on logout).
 * The session UUID is cleared so the next initSession() generates a fresh one.
 */
export async function clearSession(): Promise<void> {
  try {
    await Promise.all([
      SecureStore.deleteItemAsync(KEY_SESSION_ID),
      SecureStore.deleteItemAsync(KEY_JWT),
    ]);
  } catch { /* web fallback */ }
}
