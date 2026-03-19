/**
 * Cinova session management
 *
 * Anonymous sessions allow full app usage before sign-up.
 * Credentials stored in expo-secure-store (hardware-backed on iOS,
 * Android Keystore on Android). On web, localStorage is used as fallback.
 */

import * as SecureStore from 'expo-secure-store';
import axios from 'axios';
import { Platform } from 'react-native';

const API_BASE = 'https://api.cinova.openova.io';

async function fetchAnonymousToken(deviceId: string): Promise<string> {
  const res = await axios.post(`${API_BASE}/api/v1/auth/anonymous`, { device_id: deviceId });
  return res.data.access_token;
}

const KEY_SESSION_ID = 'cinova_session_id';
const KEY_JWT = 'cinova_jwt';

// ── Web localStorage fallback ─────────────────────────────────────────────────

function webGet(key: string): string | null {
  try { return typeof localStorage !== 'undefined' ? localStorage.getItem(key) : null; } catch { return null; }
}
function webSet(key: string, value: string): void {
  try { if (typeof localStorage !== 'undefined') localStorage.setItem(key, value); } catch { /* ignore */ }
}
function webDel(key: string): void {
  try { if (typeof localStorage !== 'undefined') localStorage.removeItem(key); } catch { /* ignore */ }
}

async function storeGet(key: string): Promise<string | null> {
  if (Platform.OS === 'web') return webGet(key);
  try { return await SecureStore.getItemAsync(key); } catch { return webGet(key); }
}

async function storeSet(key: string, value: string): Promise<void> {
  if (Platform.OS === 'web') { webSet(key, value); return; }
  try { await SecureStore.setItemAsync(key, value); } catch { webSet(key, value); }
}

async function storeDel(key: string): Promise<void> {
  if (Platform.OS === 'web') { webDel(key); return; }
  try { await SecureStore.deleteItemAsync(key); } catch { webDel(key); }
}

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
 *   1. Generate UUID and persist to SecureStore / localStorage
 *   2. Exchange UUID for anonymous JWT from the API
 *   3. Persist JWT
 *
 * If a session already exists, this is a no-op and returns the
 * existing session ID.
 */
export async function initSession(): Promise<string> {
  let sessionId = await storeGet(KEY_SESSION_ID);

  if (!sessionId) {
    sessionId = uuidv4();
    await storeSet(KEY_SESSION_ID, sessionId);
  }

  const existingToken = await storeGet(KEY_JWT);

  if (!existingToken) {
    try {
      const token = await fetchAnonymousToken(sessionId);
      await storeSet(KEY_JWT, token);
    } catch {
      // Fail silently — app will retry on first authenticated request
    }
  }

  return sessionId;
}

/** Returns the UUID session ID, or null if not initialised. */
export async function getSessionId(): Promise<string | null> {
  return storeGet(KEY_SESSION_ID);
}

/** Returns the current JWT, or null if not present. */
export async function getToken(): Promise<string | null> {
  return storeGet(KEY_JWT);
}

/** Persist a new JWT (called after login / signup / token refresh). */
export async function saveToken(token: string): Promise<void> {
  await storeSet(KEY_JWT, token);
}

/** Remove JWT only (keeps session UUID). Called on 401 refresh failure. */
export async function clearToken(): Promise<void> {
  await storeDel(KEY_JWT);
}

/**
 * Clears all session data (called on logout).
 * The session UUID is cleared so the next initSession() generates a fresh one.
 */
export async function clearSession(): Promise<void> {
  await Promise.all([storeDel(KEY_SESSION_ID), storeDel(KEY_JWT)]);
}
