/**
 * Reels screen — Full-screen vertical swipe feed
 *
 * All WebViews at top:0, no transform — all visible to iOS.
 * Active = high zIndex. Swipe animates ONLY the current slide away,
 * revealing the next one already underneath.
 */

import React, { useCallback, useState } from 'react';
import {
  View,
  Text,
  StyleSheet,
  Dimensions,
  ActivityIndicator,
} from 'react-native';
import { Gesture, GestureDetector } from 'react-native-gesture-handler';
import Animated, {
  useSharedValue,
  useAnimatedStyle,
  withTiming,
  runOnJS,
  Easing,
} from 'react-native-reanimated';
import { useRouter } from 'expo-router';

const BUILD_VERSION = 'v20-flat';
import { useFocusEffect } from 'expo-router';
import { useInfiniteQuery } from '@tanstack/react-query';
import ReelItem from '../../components/ui/ReelItem';
import { getDiscoverFeed, saveTitle, rateTitle, dismissTitle } from '../../services/api';
import { useAppStore } from '../../store/useAppStore';
import { Colors } from '../../constants/theme';
import type { Movie } from '../../types';

const { height: SCREEN_HEIGHT } = Dimensions.get('window');
const SWIPE_THRESHOLD = SCREEN_HEIGHT * 0.15;

export default function ReelsScreen() {
  const router = useRouter();
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

  // --- Swipe state ---
  const dragY = useSharedValue(0);
  const swiping = useSharedValue(false);
  const activeRef = useSharedValue(0);

  const finishSwipe = useCallback((nextIdx: number) => {
    activeRef.value = nextIdx;
    setActiveIndex(nextIdx);
    dragY.value = 0;
    swiping.value = false;
  }, [activeRef, dragY, swiping]);

  const cancelSwipe = useCallback(() => {
    dragY.value = 0;
    swiping.value = false;
  }, [dragY, swiping]);

  const navigateToMovie = useCallback((idx: number) => {
    if (movies[idx]) {
      router.push(`/movie/${movies[idx].id}`);
    }
  }, [movies, router]);

  const pan = Gesture.Pan()
    .activeOffsetY([-10, 10])
    .onStart(() => {
      swiping.value = true;
    })
    .onUpdate((e) => {
      dragY.value = e.translationY;
    })
    .onEnd((e) => {
      const idx = activeRef.value;
      const len = movies.length;

      if (e.translationY < -SWIPE_THRESHOLD && idx < len - 1) {
        // Swipe up — go to next
        dragY.value = withTiming(-SCREEN_HEIGHT, {
          duration: 200,
          easing: Easing.out(Easing.cubic),
        }, () => {
          runOnJS(finishSwipe)(idx + 1);
        });
      } else if (e.translationY > SWIPE_THRESHOLD && idx > 0) {
        // Swipe down — go to prev
        dragY.value = withTiming(SCREEN_HEIGHT, {
          duration: 200,
          easing: Easing.out(Easing.cubic),
        }, () => {
          runOnJS(finishSwipe)(idx - 1);
        });
      } else {
        // Snap back
        dragY.value = withTiming(0, { duration: 150 }, () => {
          runOnJS(cancelSwipe)();
        });
      }
    });

  const tap = Gesture.Tap()
    .onEnd(() => {
      runOnJS(navigateToMovie)(activeRef.value);
    });

  const gesture = Gesture.Race(pan, tap);

  if (isLoading) {
    return (
      <View style={styles.loading}>
        <ActivityIndicator color={Colors.primary} size="large" />
      </View>
    );
  }

  return (
    <View style={styles.container}>
      <View style={styles.badge}>
        <Text style={styles.badgeText}>
          {BUILD_VERSION} | idx={activeIndex} | {movies.length} vids
        </Text>
      </View>

      {/* All WebViews at top:0 — truly overlapping, all visible to iOS */}
      {movies.map((movie, index) => (
        <ReelSlide
          key={movie.id}
          movie={movie}
          index={index}
          activeIndex={activeIndex}
          dragY={dragY}
          swiping={swiping}
          tabFocused={tabFocused}
          savedIds={savedIds}
          ratings={ratings}
          onSave={handleSave}
          onRate={handleRate}
          onDismiss={handleDismiss}
        />
      ))}

      {/* Gesture layer on top of everything */}
      <GestureDetector gesture={gesture}>
        <Animated.View style={styles.gestureLayer} />
      </GestureDetector>
    </View>
  );
}

// Separate component so each slide gets its own animated style
function ReelSlide({
  movie, index, activeIndex, dragY, swiping, tabFocused,
  savedIds, ratings, onSave, onRate, onDismiss,
}: any) {
  const isCurrent = index === activeIndex;
  const isNext = index === activeIndex + 1;
  const isPrev = index === activeIndex - 1;

  const animStyle = useAnimatedStyle(() => {
    if (isCurrent && swiping.value) {
      // Current slide moves with drag
      return { transform: [{ translateY: dragY.value }], zIndex: 10 };
    }
    if (isCurrent) {
      return { transform: [{ translateY: 0 }], zIndex: 10 };
    }
    if (isNext) {
      // Next slide sits underneath, no transform
      return { transform: [{ translateY: 0 }], zIndex: 5 };
    }
    if (isPrev) {
      // Prev slide sits underneath, no transform
      return { transform: [{ translateY: 0 }], zIndex: 5 };
    }
    // All others: still at top:0 but behind everything
    return { transform: [{ translateY: 0 }], zIndex: 1 };
  });

  return (
    <Animated.View style={[styles.slide, animStyle]}>
      <ReelItem
        movie={movie}
        isActive={isCurrent && tabFocused}
        shouldLoad={tabFocused}
        isSaved={savedIds.has(movie.id)}
        userRating={ratings[movie.id]}
        onSave={onSave}
        onRate={onRate}
        onDismiss={onDismiss}
      />
    </Animated.View>
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
  slide: {
    position: 'absolute',
    top: 0,
    left: 0,
    right: 0,
    height: SCREEN_HEIGHT,
  },
  gestureLayer: {
    ...StyleSheet.absoluteFillObject,
    zIndex: 100,
  },
  badge: {
    position: 'absolute',
    top: 50,
    left: 0,
    right: 0,
    zIndex: 9999,
    alignItems: 'center',
  },
  badgeText: {
    color: '#0f0',
    fontSize: 12,
    fontWeight: '800',
    backgroundColor: 'rgba(0,0,0,0.7)',
    paddingHorizontal: 10,
    paddingVertical: 3,
    borderRadius: 4,
  },
});
