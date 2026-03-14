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
import * as Font from 'expo-font';
// markLoaded is internal but exported — used to populate the JS cache so
// Font.isLoaded() returns true even when the native registration threw 104
// (already registered by Expo Go). Without this, icon components would call
// Font.loadAsync() in componentDidMount, get error 104, and crash unhandled.
// @ts-ignore — internal API, stable within expo-font 14.x
import { markLoaded as fontMarkLoaded } from 'expo-font/build/memory';
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
      // Pre-register Feather font before any icon components mount.
      // Expo Go pre-registers icon fonts in its native binary; a second
      // registration from @expo/vector-icons throws CTFontManagerError 104
      // ("already registered"). We try loading it first; on 104 we manually
      // call markLoaded so Font.isLoaded() returns true and icon components
      // skip their loadAsync call in componentDidMount entirely.
      // Font family name is 'feather' (lowercase) — matches createIconSet() in Feather.js
      await Font.loadAsync({
        // eslint-disable-next-line @typescript-eslint/no-require-imports
        feather: require('@expo/vector-icons/build/vendor/react-native-vector-icons/Fonts/Feather.ttf'),
      }).catch(() => {
        // 104 = font already registered by Expo Go — mark it loaded in JS cache
        fontMarkLoaded('feather');
      });

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
