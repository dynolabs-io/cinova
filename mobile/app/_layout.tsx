/**
 * Root layout — expo-router entry point
 *
 * Responsibilities:
 *  - StatusBar dark (light text on dark background)
 *  - SafeAreaProvider
 *  - QueryClientProvider (react-query)
 *  - Anonymous session initialisation on first mount
 */

import React, { useEffect } from 'react';
import { StatusBar } from 'expo-status-bar';
import { SafeAreaProvider } from 'react-native-safe-area-context';
import { GestureHandlerRootView } from 'react-native-gesture-handler';
import { QueryClient, QueryClientProvider } from '@tanstack/react-query';
import { Stack, useRouter } from 'expo-router';
import * as SplashScreen from 'expo-splash-screen';
// ── Font loading fixes ────────────────────────────────────────────────────────
//
// Problem 1: CTFontManagerError 104 in Expo Go
//   Expo Go pre-registers vector-icon fonts in the native binary.  A second
//   registration attempt throws code 104 ("already registered"). Swallow it
//   so loadSingleFontAsync completes and expo-font calls markLoaded().
//
// Problem 2: Icons show "?" on first render
//   @expo/vector-icons checks Font.isLoaded() synchronously on first render.
//   Even after the async load succeeds, the component may not re-render.
//   Fix: call markLoaded() for every vector-icon font at module level so that
//   Font.isLoaded() returns true immediately — before any icon renders.
//   In Expo Go these fonts ARE natively available (pre-bundled), so the glyphs
//   will render correctly. In production the fonts load normally via loadAsync.
//
// @ts-ignore — internal paths, stable within expo-font 14.x
import ExpoFontLoader from 'expo-font/build/ExpoFontLoader';
// @ts-ignore
import { markLoaded } from 'expo-font/build/memory';
{
  // Mark all @expo/vector-icons fonts as loaded upfront so icon components
  // never see isLoaded()=false on first render.
  const vectorIconFonts = [
    'feather', 'ionicons', 'material-icons', 'material-community',
    'font-awesome', 'font-awesome-5', 'ant-design', 'entypo',
    'evilicons', 'foundation', 'octicons', 'simple-line-icons', 'zocial',
  ];
  vectorIconFonts.forEach((name) => {
    try { markLoaded(name); } catch { /* ignore */ }
  });

  // Also patch loadAsync to swallow duplicate-registration errors (104) so
  // subsequent explicit loadAsync calls don't throw.
  const _orig = ExpoFontLoader.loadAsync.bind(ExpoFontLoader);
  ExpoFontLoader.loadAsync = async (name: string, uri: string) => {
    try {
      return await _orig(name, uri);
    } catch {
      return;
    }
  };
}
import { initSession } from '../services/session';
import { useAppStore } from '../store/useAppStore';
import { Colors } from '../constants/theme';

// Keep splash visible until session is ready
SplashScreen.preventAutoHideAsync();

const queryClient = new QueryClient({
  defaultOptions: {
    queries: {
      staleTime: 1000 * 60 * 5,       // 5 min
      gcTime: 1000 * 60 * 30,         // 30 min
      retry: 2,
      refetchOnWindowFocus: false,
    },
  },
});

export default function RootLayout() {
  const router = useRouter();
  const setSessionId = useAppStore((s) => s.setSessionId);
  const hasOnboarded = useAppStore((s) => s.hasOnboarded);

  useEffect(() => {
    async function bootstrap() {
      try {
        const sessionId = await initSession();
        setSessionId(sessionId);
      } catch {
        // Session init failed — app still works in degraded mode
      } finally {
        await SplashScreen.hideAsync();
        if (!hasOnboarded) {
          router.replace('/onboarding');
        }
      }
    }
    bootstrap();
  }, [setSessionId]);

  return (
    <GestureHandlerRootView style={{ flex: 1 }}>
      <SafeAreaProvider>
        <QueryClientProvider client={queryClient}>
          <StatusBar style="light" backgroundColor={Colors.background} />
          <Stack
            screenOptions={{
              headerShown: false,
              contentStyle: { backgroundColor: Colors.background },
              animation: 'slide_from_right',
            }}
          >
            <Stack.Screen name="onboarding" options={{ headerShown: false, animation: 'none' }} />
            <Stack.Screen name="(tabs)" options={{ headerShown: false }} />
            <Stack.Screen
              name="movie/[id]"
              options={{ headerShown: false, animation: 'slide_from_bottom' }}
            />
            <Stack.Screen
              name="person/[id]"
              options={{ headerShown: false, animation: 'slide_from_right' }}
            />
          </Stack>
        </QueryClientProvider>
      </SafeAreaProvider>
    </GestureHandlerRootView>
  );
}
