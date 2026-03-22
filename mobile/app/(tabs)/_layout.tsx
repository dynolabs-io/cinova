/**
 * Tab navigator layout
 *
 * Five tabs: Home · Reels · Discover · Watchlist · Profile
 * Chat is a floating bubble on all tab screens — tap to open chat.
 */

import React, { useEffect } from 'react';
import { View, TouchableOpacity, StyleSheet, Platform } from 'react-native';
import { Tabs, useRouter } from 'expo-router';
import { useSafeAreaInsets } from 'react-native-safe-area-context';
import { useQueryClient } from '@tanstack/react-query';
import TabIcon from '../../components/ui/TabIcon';
import { Colors, Layout } from '../../constants/theme';
import { getDiscoverFeed } from '../../services/api';
import { useAppStore } from '../../store/useAppStore';


function ChatFAB() {
  const router = useRouter();
  const insets = useSafeAreaInsets();
  return (
    <TouchableOpacity
      style={[styles.fab, { bottom: Layout.tabBarHeight + insets.bottom + 16 }]}
      onPress={() => router.push('/chat')}
      activeOpacity={0.85}
    >
      <View style={styles.fabInner}>
        <TabIcon name="chat" color="#fff" size={22} />
      </View>
    </TouchableOpacity>
  );
}

export default function TabLayout() {
  const insets = useSafeAreaInsets();
  const queryClient = useQueryClient();
  const country = useAppStore((s) => s.country);

  // Prefetch first page of reels immediately so data is ready before user taps the tab
  useEffect(() => {
    queryClient.prefetchInfiniteQuery({
      queryKey: ['reels-feed', country],
      queryFn: ({ pageParam = 1 }) => getDiscoverFeed(country, pageParam as number),
      initialPageParam: 1,
      pages: 1,
    });
  }, [queryClient, country]);

  return (
    <View style={{ flex: 1 }}>
      <Tabs
        screenOptions={{
          headerShown: false,
          lazy: false,
          tabBarActiveTintColor: Colors.tabBarActive,
          tabBarInactiveTintColor: Colors.tabBarInactive,
          tabBarStyle: {
            backgroundColor: Colors.tabBarBackground,
            borderTopColor: 'rgba(255,255,255,0.06)',
            borderTopWidth: 1,
            height: Layout.tabBarHeight + (Platform.OS === 'android' ? insets.bottom : 0),
            paddingBottom: Platform.OS === 'android' ? insets.bottom : 6,
            paddingTop: 6,
            elevation: 0,
          },
          tabBarShowLabel: false,
        }}
      >
        <Tabs.Screen
          name="index"
          options={{
            title: 'Home',
            tabBarIcon: ({ color }) => <TabIcon name="home" color={color} size={24} />,
          }}
        />
        <Tabs.Screen
          name="reels"
          options={{
            title: 'Reels',
            tabBarIcon: ({ color }) => <TabIcon name="reels" color={color} size={24} />,
          }}
        />
        <Tabs.Screen
          name="discover"
          options={{
            title: 'Discover',
            tabBarIcon: ({ color }) => <TabIcon name="discover" color={color} size={26} />,
          }}
        />
        <Tabs.Screen
          name="watchlist"
          options={{
            title: 'Watchlist',
            tabBarIcon: ({ color }) => <TabIcon name="watchlist" color={color} size={24} />,
          }}
        />
        <Tabs.Screen
          name="profile"
          options={{
            title: 'Profile',
            tabBarIcon: ({ color }) => <TabIcon name="profile" color={color} size={24} />,
          }}
        />
        {/* Hidden — not in tab bar */}
        <Tabs.Screen name="chat" options={{ href: null }} />
        <Tabs.Screen name="search" options={{ href: null }} />
      </Tabs>

      {/* Floating chat bubble — visible on all tab screens */}
      <ChatFAB />

    </View>
  );
}

const styles = StyleSheet.create({
  fab: {
    position: 'absolute',
    right: 12,
    zIndex: 100,
  },
  fabInner: {
    width: 52,
    height: 52,
    borderRadius: 26,
    backgroundColor: Colors.primary,
    justifyContent: 'center',
    alignItems: 'center',
    shadowColor: Colors.primary,
    shadowOffset: { width: 0, height: 4 },
    shadowOpacity: 0.4,
    shadowRadius: 8,
    elevation: 8,
  },
});
