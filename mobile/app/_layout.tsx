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
// Patch ExpoFontLoader.loadAsync at module level (synchronously, before any
// Feather icon component can call componentDidMount). When running in Expo Go,
// vector-icon fonts are pre-registered in the native binary; a second
// registration attempt throws CTFontManagerError code 104. By swallowing 104
// here, loadSingleFontAsync completes normally and expo-font calls markLoaded,
// so Font.isLoaded() returns true and subsequent icon components skip loading.
// @ts-ignore — internal path, stable within expo-font 14.x
import ExpoFontLoader from 'expo-font/build/ExpoFontLoader';
{
  const _orig = ExpoFontLoader.loadAsync.bind(ExpoFontLoader);
  ExpoFontLoader.loadAsync = async (name: string, uri: string) => {
    try {
      return await _orig(name, uri);
    } catch {
      // Swallow font registration errors. Two cases:
      //  • Expo Go: font already pre-registered by the host app → CTFontManagerError 104.
      //    The font IS available in Core Text, expo-font just can't register it again.
      //  • Production (first launch): call succeeds, this catch never fires.
      // Either way, expo-font calls markLoaded() after this returns, so Font.isLoaded()
      // becomes true and icon components render correctly.
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
