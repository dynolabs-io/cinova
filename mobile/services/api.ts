/**
 * Cinova API client
 *
 * Axios instance with JWT auth, session ID header, and automatic
 * token refresh on 401.
 */

import axios, {
  AxiosInstance,
  AxiosRequestConfig,
  InternalAxiosRequestConfig,
} from 'axios';
import { getToken, saveToken, getSessionId } from './session';
import type {
  Movie,
  SearchResult,
  AuthResponse,
} from '../types';

const BASE_URL = 'https://api.cinova.openova.io';

const apiClient: AxiosInstance = axios.create({
  baseURL: BASE_URL,
  timeout: 15000,
  headers: {
    'Content-Type': 'application/json',
    Accept: 'application/json',
  },
});

// ── Request interceptor ──────────────────────────────────────────────────────

apiClient.interceptors.request.use(
  async (config: InternalAxiosRequestConfig) => {
    const [token, sessionId] = await Promise.all([getToken(), getSessionId()]);

    if (token) {
      config.headers.Authorization = `Bearer ${token}`;
    }
    if (sessionId) {
      config.headers['X-Session-ID'] = sessionId;
    }

    return config;
  },
  (error) => Promise.reject(error)
);

// ── Response interceptor — refresh on 401 ────────────────────────────────────

let isRefreshing = false;
let failedQueue: Array<{
  resolve: (value: string) => void;
  reject: (reason: unknown) => void;
}> = [];

function processQueue(error: unknown, token: string | null = null) {
  failedQueue.forEach(({ resolve, reject }) => {
    if (error) {
      reject(error);
    } else {
      resolve(token as string);
    }
  });
  failedQueue = [];
}

apiClient.interceptors.response.use(
  (response) => response,
  async (error) => {
    const originalRequest: AxiosRequestConfig & { _retry?: boolean } =
      error.config;

    if (error.response?.status !== 401 || originalRequest._retry) {
      return Promise.reject(error);
    }

    if (isRefreshing) {
      return new Promise<string>((resolve, reject) => {
        failedQueue.push({ resolve, reject });
      })
        .then((token) => {
          if (originalRequest.headers) {
            originalRequest.headers.Authorization = `Bearer ${token}`;
          }
          return apiClient(originalRequest);
        })
        .catch((err) => Promise.reject(err));
    }

    originalRequest._retry = true;
    isRefreshing = true;

    try {
      // Re-create anonymous session if no refresh token available
      const { createAnonymousSession } = await import('./api');
      const { token } = await createAnonymousSession();
      await saveToken(token);

      processQueue(null, token);

      if (originalRequest.headers) {
        originalRequest.headers.Authorization = `Bearer ${token}`;
      }
      return apiClient(originalRequest);
    } catch (refreshError) {
      processQueue(refreshError, null);
      return Promise.reject(refreshError);
    } finally {
      isRefreshing = false;
    }
  }
);

// ── API functions ─────────────────────────────────────────────────────────────

export async function createAnonymousSession(): Promise<{ token: string; sessionId: string }> {
  const response = await apiClient.post<{ token: string; sessionId: string }>(
    '/api/v1/auth/anonymous'
  );
  return response.data;
}

export async function signUp(
  email: string,
  password: string,
  sessionId: string
): Promise<AuthResponse> {
  const response = await apiClient.post<AuthResponse>('/api/v1/auth/signup', {
    email,
    password,
    sessionId,
  });
  return response.data;
}

export async function login(
  email: string,
  password: string,
  sessionId: string
): Promise<AuthResponse> {
  const response = await apiClient.post<AuthResponse>('/api/v1/auth/login', {
    email,
    password,
    sessionId,
  });
  return response.data;
}

export async function searchMovies(
  q: string,
  country: string
): Promise<SearchResult> {
  const response = await apiClient.get<SearchResult>('/api/v1/search', {
    params: { q, country },
  });
  return response.data;
}

export async function getMovie(id: number, country: string): Promise<Movie> {
  const response = await apiClient.get<Movie>(`/api/v1/movies/${id}`, {
    params: { country },
  });
  return response.data;
}

export async function getTrending(country: string): Promise<Movie[]> {
  const response = await apiClient.get<Movie[]>('/api/v1/movies/trending', {
    params: { country },
  });
  return response.data;
}

export async function getNewOnNetflix(country: string): Promise<Movie[]> {
  const response = await apiClient.get<Movie[]>('/api/v1/movies/new-on-netflix', {
    params: { country },
  });
  return response.data;
}

export async function getTopRated(country: string): Promise<Movie[]> {
  const response = await apiClient.get<Movie[]>('/api/v1/movies/top-rated', {
    params: { country },
  });
  return response.data;
}

export async function getRecommendations(country: string): Promise<Movie[]> {
  const response = await apiClient.get<Movie[]>('/api/v1/movies/recommended', {
    params: { country },
  });
  return response.data;
}

export async function getDiscoverFeed(
  country: string,
  page: number = 1
): Promise<Movie[]> {
  const response = await apiClient.get<Movie[]>('/api/v1/movies/discover', {
    params: { country, page, limit: 20 },
  });
  return response.data;
}

export async function saveTitle(tmdbId: number): Promise<void> {
  await apiClient.post('/api/v1/user/watchlist', { tmdbId });
}

export async function unsaveTitle(tmdbId: number): Promise<void> {
  await apiClient.delete(`/api/v1/user/watchlist/${tmdbId}`);
}

export async function rateTitle(
  tmdbId: number,
  score: number
): Promise<void> {
  await apiClient.post('/api/v1/user/ratings', { tmdbId, score });
}

export async function dismissTitle(tmdbId: number): Promise<void> {
  await apiClient.post('/api/v1/user/dismissed', { tmdbId });
}

export async function getWatchlist(): Promise<Movie[]> {
  const response = await apiClient.get<Movie[]>('/api/v1/user/watchlist');
  return response.data;
}

export async function getPerson(id: number): Promise<import('../types').Person> {
  const response = await apiClient.get<import('../types').Person>(
    `/api/v1/people/${id}`
  );
  return response.data;
}

export default apiClient;
