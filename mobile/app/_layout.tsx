/**
 * Root layout — expo-router entry point
 *
 * Responsibilities:
 *  - StatusBar dark (light text on dark background)
 *  - SafeAreaProvider
 *  - QueryClientProvider (react-query)
 *  - Load Feather font before splash hides (prevents "?" icons)
 *  - Anonymous session initialisation on first mount
 */

import React, { useEffect } from 'react';
import { StatusBar } from 'expo-status-bar';
import { SafeAreaProvider } from 'react-native-safe-area-context';
import { GestureHandlerRootView } from 'react-native-gesture-handler';
import { QueryClient, QueryClientProvider } from '@tanstack/react-query';
import { Stack, useRouter } from 'expo-router';
import * as SplashScreen from 'expo-splash-screen';
import * as Font from 'expo-font';
import { Ionicons } from '@expo/vector-icons';
// Patch ExpoFontLoader.loadAsync to swallow ALL errors.
// In Expo Go the vector-icon fonts are pre-registered in the host binary;
// a second registration attempt throws code 104 ("already registered").
// By swallowing ALL errors here, Font.loadAsync() resolves successfully and
// Font.isLoaded() returns true — icons render correctly on first paint.
// Without this patch, icons render as "?" glyphs on device.
// @ts-ignore — internal path, stable within expo-font 14.x
import ExpoFontLoader from 'expo-font/build/ExpoFontLoader';
{
  const _orig = ExpoFontLoader.loadAsync.bind(ExpoFontLoader);
  ExpoFontLoader.loadAsync = async (name: string, uri: string) => {
    try {
      return await _orig(name, uri);
    } catch {
      return; // swallow all errors so Font.isLoaded() returns true
    }
  };
}
import { initSession } from '../services/session';
import { useAppStore } from '../store/useAppStore';
import { Colors } from '../constants/theme';

// Keep splash visible until fonts + session are ready
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
      // Load Ionicons font BEFORE hiding splash so tab icons are ready on
      // first render. The ExpoFontLoader patch above swallows the 104 error
      // in Expo Go, so this always resolves successfully.
      try {
        await Font.loadAsync(Ionicons.font);
      } catch {
        // Ignore — patch already handles this
      }

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
