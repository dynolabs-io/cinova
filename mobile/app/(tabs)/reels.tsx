/**
 * Reels screen — Full-screen vertical swipe feed
 *
 * Option 3: All 4 WebViews stacked at full screen (all "visible" to iOS).
 * Active video shown via translateY animation. Pan gesture for swiping.
 * All videos autoplay muted; active one gets unmuted.
 */

import React, { useCallback, useRef, useState } from 'react';
import {
  View,
  Text,
  StyleSheet,
  Dimensions,
  ActivityIndicator,
} from 'react-native';
import { GestureDetector, Gesture } from 'react-native-gesture-handler';
import Animated, {
  useSharedValue,
  useAnimatedStyle,
  withSpring,
  runOnJS,
} from 'react-native-reanimated';

const BUILD_VERSION = 'v18-stacked';
import { useFocusEffect } from 'expo-router';
import { useInfiniteQuery } from '@tanstack/react-query';
import ReelItem from '../../components/ui/ReelItem';
import { getDiscoverFeed, saveTitle, rateTitle, dismissTitle } from '../../services/api';
import { useAppStore } from '../../store/useAppStore';
import { Colors } from '../../constants/theme';
import type { Movie } from '../../types';

const { height: SCREEN_HEIGHT } = Dimensions.get('window');
const SWIPE_THRESHOLD = SCREEN_HEIGHT * 0.2;

export default function ReelsScreen() {
  const country = useAppStore((s) => s.country);
  const [savedIds, setSavedIds] = useState<Set<number>>(new Set());
  const [ratings, setRatings] = useState<Record<number, number>>({});
  const [dismissedIds, setDismissedIds] = useState<Set<number>>(new Set());
  const [activeIndex, setActiveIndex] = useState(0);
  const [tabFocused, setTabFocused] = useState(false);

  useFocusEffect(useCallback(() => {
    setTabFocused(true);
    return () => setTabFocused(false);
  }, []));

  const {
    data,
    isLoading,
  } = useInfiniteQuery({
    queryKey: ['reels-feed', country],
    queryFn: ({ pageParam = 1 }) => getDiscoverFeed(country, pageParam as number),
    initialPageParam: 1,
    getNextPageParam: (lastPage, allPages) =>
      lastPage.length === 20 ? allPages.length + 1 : undefined,
  });

  const movies: Movie[] = (data?.pages ?? [])
    .flat()
    .filter((m) => !dismissedIds.has(m.id))
    .slice(0, 4);

  const handleSave = useCallback(async (movie: Movie) => {
    setSavedIds((prev) => new Set(prev).add(movie.id));
    try { await saveTitle(movie.tmdbId); } catch {
      setSavedIds((prev) => { const n = new Set(prev); n.delete(movie.id); return n; });
    }
  }, []);

  const handleRate = useCallback(async (movie: Movie) => {
    const next = (ratings[movie.id] ?? 0) === 0 ? 8 : 0;
    setRatings((prev) => ({ ...prev, [movie.id]: next }));
    if (next > 0) { try { await rateTitle(movie.tmdbId, next); } catch {} }
  }, [ratings]);

  const handleDismiss = useCallback(async (movie: Movie) => {
    setDismissedIds((prev) => new Set(prev).add(movie.id));
    try { await dismissTitle(movie.tmdbId); } catch {}
  }, []);

  // --- Gesture / animation ---
  const translateY = useSharedValue(0);
  const activeIndexRef = useRef(0);

  const goTo = useCallback((idx: number) => {
    activeIndexRef.current = idx;
    setActiveIndex(idx);
  }, []);

  const pan = Gesture.Pan()
    .onUpdate((e) => {
      // Allow drag only within bounds
      const idx = activeIndexRef.current;
      const newY = -idx * SCREEN_HEIGHT + e.translationY;
      const minY = -(movies.length - 1) * SCREEN_HEIGHT;
      translateY.value = Math.max(minY, Math.min(0, newY));
    })
    .onEnd((e) => {
      const idx = activeIndexRef.current;
      let nextIdx = idx;
      if (e.translationY < -SWIPE_THRESHOLD && idx < movies.length - 1) {
        nextIdx = idx + 1;
      } else if (e.translationY > SWIPE_THRESHOLD && idx > 0) {
        nextIdx = idx - 1;
      }
      translateY.value = withSpring(-nextIdx * SCREEN_HEIGHT, {
        damping: 50,
        stiffness: 300,
        mass: 1,
      });
      if (nextIdx !== idx) {
        runOnJS(goTo)(nextIdx);
      }
    });

  const animatedStyle = useAnimatedStyle(() => ({
    transform: [{ translateY: translateY.value }],
  }));

  if (isLoading) {
    return (
      <View style={styles.loading}>
        <ActivityIndicator color={Colors.primary} size="large" />
      </View>
    );
  }

  return (
    <View style={styles.container}>
      <View style={{ position: 'absolute', top: 50, left: 0, right: 0, zIndex: 9999, alignItems: 'center' }}>
        <Text style={{ color: '#0f0', fontSize: 12, fontWeight: '800', backgroundColor: 'rgba(0,0,0,0.7)', paddingHorizontal: 10, paddingVertical: 3, borderRadius: 4 }}>
          {BUILD_VERSION} | idx={activeIndex}
        </Text>
      </View>
      <GestureDetector gesture={pan}>
        <Animated.View style={[styles.stack, animatedStyle]}>
          {movies.map((movie, index) => (
            <View key={movie.id} style={[styles.slide, { top: index * SCREEN_HEIGHT }]}>
              <ReelItem
                movie={movie}
                isActive={index === activeIndex && tabFocused}
                shouldLoad={tabFocused}
                isSaved={savedIds.has(movie.id)}
                userRating={ratings[movie.id]}
                onSave={handleSave}
                onRate={handleRate}
                onDismiss={handleDismiss}
              />
            </View>
          ))}
        </Animated.View>
      </GestureDetector>
    </View>
  );
}

const styles = StyleSheet.create({
  container: {
    flex: 1,
    backgroundColor: Colors.background,
    overflow: 'hidden',
  },
  loading: {
    flex: 1,
    backgroundColor: Colors.background,
    justifyContent: 'center',
    alignItems: 'center',
  },
  stack: {
    position: 'absolute',
    top: 0,
    left: 0,
    right: 0,
    height: SCREEN_HEIGHT * 4,
  },
  slide: {
    position: 'absolute',
    left: 0,
    right: 0,
    height: SCREEN_HEIGHT,
  },
});
