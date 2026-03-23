/**
 * Reels screen — Full-screen vertical swipe feed
 *
 * All 4 WebViews stacked at the SAME position (top:0) so iOS considers
 * them all "on screen" and allows muted autoplay. Active video shown via
 * zIndex. Pan gesture for swipe, tap gesture for movie detail navigation.
 */

import React, { useCallback, useRef, useState } from 'react';
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

const BUILD_VERSION = 'v19-overlap';
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

  // --- Animation state ---
  const activeIndexRef = useRef(0);
  // Animated values for each slide's translateY (for swipe transition)
  const transitionY = useSharedValue(0); // drag offset during gesture
  const [transitioning, setTransitioning] = useState(false);
  const directionRef = useRef<'up' | 'down' | null>(null);

  const goTo = useCallback((idx: number) => {
    activeIndexRef.current = idx;
    setActiveIndex(idx);
    setTransitioning(false);
    transitionY.value = 0;
  }, [transitionY]);

  const handleTap = useCallback((idx: number) => {
    if (movies[idx]) {
      router.push(`/movie/${movies[idx].id}`);
    }
  }, [movies, router]);

  // Combined gesture: pan for swipe, tap for navigation
  const pan = Gesture.Pan()
    .onStart(() => {
      runOnJS(setTransitioning)(true);
    })
    .onUpdate((e) => {
      transitionY.value = e.translationY;
    })
    .onEnd((e) => {
      const idx = activeIndexRef.current;
      let nextIdx = idx;
      if (e.translationY < -SWIPE_THRESHOLD && idx < movies.length - 1) {
        nextIdx = idx + 1;
        directionRef.current = 'up';
      } else if (e.translationY > SWIPE_THRESHOLD && idx > 0) {
        nextIdx = idx - 1;
        directionRef.current = 'down';
      }

      if (nextIdx !== idx) {
        // Animate current slide out, then switch
        const target = e.translationY < 0 ? -SCREEN_HEIGHT : SCREEN_HEIGHT;
        transitionY.value = withTiming(target, {
          duration: 250,
          easing: Easing.out(Easing.cubic),
        }, () => {
          runOnJS(goTo)(nextIdx);
        });
      } else {
        // Snap back
        transitionY.value = withTiming(0, { duration: 200 });
        runOnJS(setTransitioning)(false);
      }
    });

  const tap = Gesture.Tap()
    .onEnd(() => {
      const idx = activeIndexRef.current;
      runOnJS(handleTap)(idx);
    });

  const gesture = Gesture.Race(pan, tap);

  // Animated style for the CURRENT active slide (moves with drag)
  const currentSlideStyle = useAnimatedStyle(() => ({
    transform: [{ translateY: transitionY.value }],
  }));

  // Animated style for the NEXT slide (peeks from below during upward swipe)
  const nextSlideStyle = useAnimatedStyle(() => ({
    transform: [{ translateY: transitionY.value < 0
      ? SCREEN_HEIGHT + transitionY.value
      : SCREEN_HEIGHT }],
  }));

  // Animated style for the PREV slide (peeks from above during downward swipe)
  const prevSlideStyle = useAnimatedStyle(() => ({
    transform: [{ translateY: transitionY.value > 0
      ? -SCREEN_HEIGHT + transitionY.value
      : -SCREEN_HEIGHT }],
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
          {BUILD_VERSION} | idx={activeIndex} | {movies.length} vids
        </Text>
      </View>

      {/* All WebViews rendered at position absolute top:0 — all "visible" to iOS */}
      {movies.map((movie, index) => {
        // Determine z-order and animation
        const isCurrent = index === activeIndex;
        const isNext = index === activeIndex + 1;
        const isPrev = index === activeIndex - 1;

        let zIndex = 1;
        let animStyle = undefined;

        if (isCurrent) {
          zIndex = 10;
          animStyle = transitioning ? currentSlideStyle : undefined;
        } else if (isNext) {
          zIndex = 5;
          animStyle = transitioning ? nextSlideStyle : undefined;
        } else if (isPrev) {
          zIndex = 5;
          animStyle = transitioning ? prevSlideStyle : undefined;
        }

        const Wrapper = animStyle ? Animated.View : View;
        const wrapperStyle = animStyle
          ? [styles.slide, { zIndex }, animStyle]
          : [styles.slide, { zIndex, transform: [{ translateY: isCurrent ? 0 : SCREEN_HEIGHT * 2 }] }];

        return (
          <Wrapper key={movie.id} style={wrapperStyle}>
            <ReelItem
              movie={movie}
              isActive={isCurrent && tabFocused}
              shouldLoad={tabFocused}
              isSaved={savedIds.has(movie.id)}
              userRating={ratings[movie.id]}
              onSave={handleSave}
              onRate={handleRate}
              onDismiss={handleDismiss}
            />
          </Wrapper>
        );
      })}

      {/* Gesture layer on top */}
      <GestureDetector gesture={gesture}>
        <Animated.View style={styles.gestureLayer} />
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
});
