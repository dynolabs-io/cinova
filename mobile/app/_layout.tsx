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
 * Icon strategy:
 *  TabIcon uses pure View-drawn icons — no font dependency, no @expo/vector-icons.
 *  This sidesteps the Expo Go font conflict entirely.
 */

import React, { useEffect } from 'react';
import { StatusBar } from 'expo-status-bar';
import { SafeAreaProvider } from 'react-native-safe-area-context';
import { GestureHandlerRootView } from 'react-native-gesture-handler';
import { QueryClient, QueryClientProvider } from '@tanstack/react-query';
import { Stack, useRouter } from 'expo-router';
import * as SplashScreen from 'expo-splash-screen';
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
