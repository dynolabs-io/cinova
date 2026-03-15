/**
 * Cinova global app store — Zustand + AsyncStorage persistence
 */

import { create } from 'zustand';
import { persist, createJSONStorage } from 'zustand/middleware';
import AsyncStorage from '@react-native-async-storage/async-storage';
import type { User } from '../types';

interface AppState {
  // Auth
  user: User | null;
  isAnonymous: boolean;
  sessionId: string | null;

  // Preferences
  country: string;
  scoringPreset: string;

  // UI state
  hasOnboarded: boolean;
}

interface AppActions {
  setUser: (user: User | null) => void;
  setCountry: (country: string) => void;
  setScoringPreset: (preset: string) => void;
  setSessionId: (sessionId: string) => void;
  setHasOnboarded: (value: boolean) => void;
  logout: () => void;
}

const initialState: AppState = {
  user: null,
  isAnonymous: true,
  sessionId: null,
  country: 'US',
  scoringPreset: 'mainstream',
  hasOnboarded: false,
};

export const useAppStore = create<AppState & AppActions>()(
  persist(
    (set) => ({
      ...initialState,

      setUser: (user) =>
        set({
          user,
          isAnonymous: user === null,
        }),

      setCountry: (country) => set({ country }),

      setScoringPreset: (scoringPreset) => set({ scoringPreset }),

      setSessionId: (sessionId) => set({ sessionId }),

      setHasOnboarded: (hasOnboarded) => set({ hasOnboarded }),

      logout: () =>
        set({
          user: null,
          isAnonymous: true,
          // Keep sessionId and country — those survive logout
        }),
    }),
    {
      name: 'cinova-app-store',
      storage: createJSONStorage(() => AsyncStorage),
      // Only persist non-sensitive state — JWT lives in SecureStore
      partialize: (state) => ({
        country: state.country,
        scoringPreset: state.scoringPreset,
        hasOnboarded: state.hasOnboarded,
        // user email/displayName for display — no tokens here
        user: state.user
          ? {
              id: state.user.id,
              email: state.user.email,
              displayName: state.user.displayName,
              avatarUrl: state.user.avatarUrl,
              createdAt: state.user.createdAt,
              country: state.user.country,
              isPremium: state.user.isPremium,
              stats: state.user.stats,
            }
          : null,
        isAnonymous: state.isAnonymous,
        sessionId: state.sessionId,
      }),
    }
  )
);
