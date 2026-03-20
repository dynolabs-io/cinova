/**
 * Root layout — expo-router entry point
 *
 * Responsibilities:
 *  - StatusBar dark (light text on dark background)
 *  - SafeAreaProvider
 *  - QueryClientProvider (react-query)
 *  - Load CinovaIcons font (Ionicons TTF under a unique name to avoid Expo Go conflict)
 *  - Anonymous session initialisation on first mount
 *
 * Icon font strategy:
 *  Expo Go pre-bundles its own version of Ionicons in the host binary. If we try to load
 *  the font under the name "Ionicons", the native registration either fails (code 104 —
 *  "already registered") or succeeds but the host's version (different glyph map) is used.
 *  Either way, @expo/vector-icons renders "?" glyphs because the code points from our
 *  installed version don't match the host's font.
 *
 *  Solution: load the TTF bundled with @expo/vector-icons under a UNIQUE name "CinovaIcons".
 *  No conflict with Expo Go. TabIcon renders Text with fontFamily="CinovaIcons" + the
 *  correct Unicode code points from the installed glyph map. Always correct, always works.
 */

import React, { useEffect } from 'react';
import { StatusBar } from 'expo-status-bar';
import { SafeAreaProvider } from 'react-native-safe-area-context';
import { GestureHandlerRootView } from 'react-native-gesture-handler';
import { QueryClient, QueryClientProvider } from '@tanstack/react-query';
import { Stack, useRouter } from 'expo-router';
import * as SplashScreen from 'expo-splash-screen';
import * as Font from 'expo-font';
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
      // Load Ionicons TTF under unique name "CinovaIcons" — avoids conflict with
      // Expo Go's pre-bundled Ionicons. TabIcon renders glyphs using this font.
      try {
        await Font.loadAsync({
          // eslint-disable-next-line @typescript-eslint/no-require-imports
          CinovaIcons: require('@expo/vector-icons/build/vendor/react-native-vector-icons/Fonts/Ionicons.ttf'),
        });
      } catch {
        // Non-fatal — icons fall back to text labels if font fails to load
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
