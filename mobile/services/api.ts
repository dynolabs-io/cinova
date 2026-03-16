/**
 * Cinova API client — all paths match Go backend routes exactly.
 */

import axios, {
  AxiosInstance,
  AxiosRequestConfig,
  InternalAxiosRequestConfig,
} from 'axios';
import { getToken, saveToken, clearToken, getSessionId } from './session';
import type { Movie, TVShow, Person, AuthResponse, SearchResult, WatchProvider, ScoringProfile } from '../types';

export const BASE_URL = 'https://api.cinova.openova.io';

// Recursively converts snake_case keys to camelCase to match frontend types
function toCamel(s: string): string {
  return s.replace(/_([a-z])/g, (_, c) => c.toUpperCase());
}

function camelizeKeys(obj: unknown): unknown {
  if (Array.isArray(obj)) return obj.map(camelizeKeys);
  if (obj !== null && typeof obj === 'object') {
    return Object.fromEntries(
      Object.entries(obj as Record<string, unknown>).map(([k, v]) => [toCamel(k), camelizeKeys(v)])
    );
  }
  return obj;
}

function normalizeMedia(item: unknown): unknown {
  const m = camelizeKeys(item) as Record<string, unknown>;
  // ensure id === tmdbId (both needed by components)
  if (m.tmdbId) { m.id = m.tmdbId; }
  else if (m.id) { m.tmdbId = m.id; }
  // guard: if still missing, set a sentinel so keyExtractors never see undefined
  if (!m.id && !m.tmdbId) { m.id = 0; m.tmdbId = 0; }
  // derive year from releaseDate / firstAirDate if missing
  if (!m.year) {
    const dateStr = (m.releaseDate ?? m.firstAirDate) as string | undefined;
    if (dateStr) m.year = parseInt(dateStr.slice(0, 4), 10);
  }
  // normalise language field
  if (!m.language && m.originalLanguage) m.language = m.originalLanguage;
  // ensure array fields are always arrays (list endpoints omit detail fields)
  if (!Array.isArray(m.themes)) m.themes = [];
  if (!Array.isArray(m.moods)) m.moods = [];
  if (!Array.isArray(m.cast)) m.cast = [];
  if (!Array.isArray(m.genres)) m.genres = [];
  if (!Array.isArray(m.providers)) m.providers = [];
  if (!Array.isArray(m.awards)) m.awards = [];
  if (!Array.isArray(m.keywords)) m.keywords = [];
  return m;
}

const api: AxiosInstance = axios.create({
  baseURL: BASE_URL,
  timeout: 15_000,
  headers: { 'Content-Type': 'application/json', Accept: 'application/json' },
});

// Request interceptor: attach JWT + Session-ID
api.interceptors.request.use(
  async (config: InternalAxiosRequestConfig) => {
    const [token, sessionId] = await Promise.all([getToken(), getSessionId()]);
    if (token) config.headers.Authorization = `Bearer ${token}`;
    if (sessionId) config.headers['X-Session-ID'] = sessionId;
    return config;
  },
  (err) => Promise.reject(err)
);

// Response interceptor: refresh JWT on 401
let isRefreshing = false;
let failedQueue: Array<{ resolve: (t: string) => void; reject: (e: unknown) => void }> = [];

function processQueue(error: unknown, token: string | null = null) {
  failedQueue.forEach(({ resolve, reject }) => (error ? reject(error) : resolve(token!)));
  failedQueue = [];
}

api.interceptors.response.use(
  (res) => res,
  async (error) => {
    const req: AxiosRequestConfig & { _retry?: boolean } = error.config;
    if (error.response?.status !== 401 || req._retry) return Promise.reject(error);
    if (isRefreshing) {
      return new Promise<string>((resolve, reject) => failedQueue.push({ resolve, reject }))
        .then((token) => { if (req.headers) req.headers.Authorization = `Bearer ${token}`; return api(req); });
    }
    req._retry = true;
    isRefreshing = true;
    try {
      const sessionId = await getSessionId();
      const res = await authAnonymous(sessionId ?? undefined);
      await saveToken(res.access_token);
      processQueue(null, res.access_token);
      if (req.headers) req.headers.Authorization = `Bearer ${res.access_token}`;
      return api(req);
    } catch (e) {
      processQueue(e, null);
      await clearToken();
      return Promise.reject(e);
    } finally {
      isRefreshing = false;
    }
  }
);

// ── Auth ─────────────────────────────────────────────────────────────────────

export async function authAnonymous(deviceId?: string): Promise<AuthResponse> {
  const { data } = await api.post<AuthResponse>('/api/v1/auth/anonymous', { device_id: deviceId ?? 'unknown' });
  return data;
}

export async function authSignup(email: string, password: string, username: string, sessionUuid?: string): Promise<AuthResponse> {
  const { data } = await api.post<AuthResponse>('/api/v1/auth/signup', { email, password, username, session_uuid: sessionUuid });
  return data;
}

export async function authLogin(email: string, password: string, sessionUuid?: string): Promise<AuthResponse> {
  const { data } = await api.post<AuthResponse>('/api/v1/auth/login', { email, password, session_uuid: sessionUuid });
  return data;
}

export async function authRefresh(refreshToken: string): Promise<AuthResponse> {
  const { data } = await api.post<AuthResponse>('/api/v1/auth/refresh', { refresh_token: refreshToken });
  return data;
}

// ── Discovery ─────────────────────────────────────────────────────────────────

export async function getTrending(country = 'US', limit = 20): Promise<Movie[]> {
  const { data } = await api.get<{ results: Movie[]; total: number }>('/api/v1/trending', { params: { country, limit } });
  return (data.results ?? []).map(normalizeMedia) as Movie[];
}

export async function getPopular(country = 'US', limit = 20, page = 1): Promise<Movie[]> {
  const { data } = await api.get<{ results: Movie[]; total: number }>('/api/v1/popular', { params: { country, limit, page } });
  return (data.results ?? []).map(normalizeMedia) as Movie[];
}

export async function getDiscoverFeed(country = 'US', page = 1): Promise<Movie[]> {
  const { data } = await api.get<{ reels: Movie[]; total: number }>('/api/v1/discover/reels', { params: { country, page, limit: 20 } });
  return (data.reels ?? []).map(normalizeMedia) as Movie[];
}

export async function getRecommendations(country = 'US', limit = 20): Promise<Movie[]> {
  const { data } = await api.get<{ results: Movie[]; total: number }>('/api/v1/recommend', { params: { country, limit } });
  return (data.results ?? []).map(normalizeMedia) as Movie[];
}

// ── Content Detail ────────────────────────────────────────────────────────────

export async function getMovie(id: number, country = 'US'): Promise<Movie> {
  const { data } = await api.get<Movie>(`/api/v1/movie/${id}`, { params: { country } });
  return normalizeMedia(data as Record<string, unknown>) as unknown as Movie;
}

export async function getMovieProviders(id: number, country = 'US'): Promise<WatchProvider[]> {
  const { data } = await api.get<{ providers: WatchProvider[]; tmdb_id: number; country: string }>(`/api/v1/movie/${id}/providers`, { params: { country } });
  return data.providers ?? [];
}

export async function getTV(id: number, country = 'US'): Promise<TVShow> {
  const { data } = await api.get<TVShow>(`/api/v1/tv/${id}`, { params: { country } });
  return normalizeMedia(data) as unknown as TVShow;
}

export async function getPerson(id: number): Promise<Person> {
  const { data } = await api.get<Person>(`/api/v1/person/${id}`);
  return camelizeKeys(data) as unknown as Person;
}

// ── Search ────────────────────────────────────────────────────────────────────

export async function search(q: string, country = 'US'): Promise<SearchResult> {
  const { data } = await api.get<{ results: (Movie | TVShow)[]; query: string; country: string; total: number }>('/api/v1/search', { params: { q, country } });
  return {
    items: (data.results ?? []).map(normalizeMedia) as (Movie | TVShow)[],
    total: data.total ?? 0,
    page: 1,
    hasMore: false,
    query: data.query ?? q,
  };
}

// ── Interactions (require auth) ───────────────────────────────────────────────

// rating accepts 'like'|'dislike' or a numeric score (≥6 = like, <6 = dislike)
export async function rateTitle(tmdbId: number, ratingOrMediaType: 'like' | 'dislike' | 'movie' | 'tv' | number = 'like', ratingArg?: 'like' | 'dislike' | number): Promise<void> {
  let mediaType: 'movie' | 'tv' = 'movie';
  let rating: 'like' | 'dislike' = 'like';
  if (ratingOrMediaType === 'movie' || ratingOrMediaType === 'tv') {
    mediaType = ratingOrMediaType;
    const r = ratingArg ?? 'like';
    rating = typeof r === 'number' ? (r >= 6 ? 'like' : 'dislike') : r;
  } else {
    rating = typeof ratingOrMediaType === 'number' ? (ratingOrMediaType >= 6 ? 'like' : 'dislike') : ratingOrMediaType as 'like' | 'dislike';
  }
  await api.post('/api/v1/me/rate', { tmdb_id: tmdbId, media_type: mediaType, rating });
}

export async function saveTitle(tmdbId: number, mediaType: 'movie' | 'tv' = 'movie'): Promise<void> {
  await api.post('/api/v1/me/save', { tmdb_id: tmdbId, media_type: mediaType });
}

export async function unsaveTitle(tmdbId: number, mediaType: 'movie' | 'tv' = 'movie'): Promise<void> {
  await api.delete('/api/v1/me/save', { data: { tmdb_id: tmdbId, media_type: mediaType } });
}

export async function dismissTitle(tmdbId: number, mediaType: 'movie' | 'tv' = 'movie'): Promise<void> {
  await api.post('/api/v1/me/dismiss', { tmdb_id: tmdbId, media_type: mediaType });
}

export async function getWatchlist(page = 1, limit = 20): Promise<Movie[]> {
  const { data } = await api.get<{ results: Movie[] | null; total: number }>('/api/v1/me/watchlist', { params: { page, limit } });
  return (data.results ?? []).map(normalizeMedia) as Movie[];
}

/** Alias used by home screen — maps to /api/v1/popular */
export async function getNewOnNetflix(country = 'US', limit = 20): Promise<Movie[]> {
  return getPopular(country, limit);
}

/** Alias used by home screen — maps to /api/v1/trending */
export async function getTopRated(country = 'US', limit = 20): Promise<Movie[]> {
  return getTrending(country, limit);
}

/** Alias used by search screen */
export async function searchMovies(q: string, country = 'US'): Promise<SearchResult> {
  return search(q, country);
}

// ── Scoring Profile (require auth) ────────────────────────────────────────────

export async function getScoringProfile(): Promise<ScoringProfile> {
  const { data } = await api.get<ScoringProfile>('/api/v1/me/scoring-profile');
  return data;
}

export async function setScoringProfile(profile: Pick<ScoringProfile, 'preset' | 'audience' | 'critic' | 'award' | 'prestige' | 'commercial'>): Promise<ScoringProfile> {
  const { data } = await api.put<ScoringProfile>('/api/v1/me/scoring-profile', profile);
  return data;
}

// ── Chat (require auth) ───────────────────────────────────────────────────────

export interface ChatSuggestion {
  tmdbId: number;
  mediaType: string;
  title: string;
  posterPath?: string;
  releaseYear?: string;
  cinovaScore?: number;
  overview?: string;
  genres?: { id: number; name: string }[];
  providers?: { providerId: number; providerName: string; logoPath?: string; type: string }[];
  cinovaSynopsis?: string;
  reason?: string;
}

export interface ChatApiResponse {
  reply: string;
  suggestions?: ChatSuggestion[];
  convId: string;
}

export async function sendChatMessage(
  message: string,
  convId?: string,
  country = 'US'
): Promise<ChatApiResponse> {
  const { data } = await api.post<ChatApiResponse>(
    '/api/v1/me/chat',
    { message, conv_id: convId ?? '' },
    { params: { country }, timeout: 90_000 }
  );
  return camelizeKeys(data) as unknown as ChatApiResponse;
}

/**
 * SSE streaming chat — fires callbacks as events arrive.
 * Returns an abort function.
 */
export function streamChatMessage(
  message: string,
  convId: string | undefined,
  country: string,
  onDelta: (text: string) => void,
  onSuggestions: (suggestions: ChatSuggestion[], convId: string) => void,
  onDone: () => void,
  onError: (err: Error) => void,
  onStatus?: (text: string) => void,
): () => void {
  const xhr = new XMLHttpRequest();
  let lastLen = 0;
  let aborted = false;

  const processChunk = (responseText: string) => {
    const newText = responseText.slice(lastLen);
    lastLen = responseText.length;
    const lines = newText.split('\n');
    for (const line of lines) {
      if (!line.startsWith('data: ')) continue;
      const data = line.slice(6).trim();
      if (!data) continue;
      try {
        const event = JSON.parse(data) as { type: string; text?: string; items?: unknown[]; conv_id?: string };
        if (event.type === 'status' && event.text) {
          onStatus?.(event.text);
        } else if (event.type === 'delta' && event.text) {
          onDelta(event.text);
        } else if (event.type === 'suggestions') {
          const items = (event.items ?? []).map(normalizeMedia) as unknown as ChatSuggestion[];
          onSuggestions(items, event.conv_id ?? '');
        } else if (event.type === 'done') {
          onDone();
        }
      } catch { /* ignore malformed chunk */ }
    }
  };

  const setup = async () => {
    const [token, sessionId] = await Promise.all([getToken(), getSessionId()]);
    if (aborted) return;

    xhr.open('POST', `${BASE_URL}/api/v1/me/chat/stream?country=${encodeURIComponent(country)}`);
    xhr.setRequestHeader('Content-Type', 'application/json');
    xhr.setRequestHeader('Accept', 'text/event-stream');
    if (token) xhr.setRequestHeader('Authorization', `Bearer ${token}`);
    if (sessionId) xhr.setRequestHeader('X-Session-ID', sessionId);
    xhr.timeout = 120_000;

    xhr.onprogress = () => processChunk(xhr.responseText);
    xhr.onload = () => processChunk(xhr.responseText);
    xhr.onerror = () => onError(new Error('Network error'));
    xhr.ontimeout = () => onError(new Error('Request timed out'));

    xhr.send(JSON.stringify({ message, conv_id: convId ?? '' }));
  };

  setup().catch(onError);
  return () => { aborted = true; xhr.abort(); };
}

/** Aliases matching screen import names */
export const login = authLogin;
export const signUp = authSignup;

export default api;
