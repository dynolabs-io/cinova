/**
 * Reels screen — Single WebView player with gesture swipe
 *
 * One WebView stays on screen permanently. On swipe, we call
 * switchVideo(key) to load the next video in the same player.
 * No WebView remounting, no iOS autoplay restrictions.
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
  withSequence,
  runOnJS,
  Easing,
} from 'react-native-reanimated';
import { WebView } from 'react-native-webview';
import { useRouter } from 'expo-router';

const BUILD_VERSION = 'v23-swipe';
import { useFocusEffect } from 'expo-router';
import { useInfiniteQuery } from '@tanstack/react-query';
import { getDiscoverFeed, saveTitle, rateTitle, dismissTitle } from '../../services/api';
import { useAppStore } from '../../store/useAppStore';
import { Colors, Typography, Spacing, Radius } from '../../constants/theme';
import CinovaScore from '../../components/ui/CinovaScore';
import StreamingBadge from '../../components/ui/StreamingBadge';
import type { Movie } from '../../types';

const { width: SCREEN_WIDTH, height: SCREEN_HEIGHT } = Dimensions.get('window');
const EMBED_BASE = 'https://api.cinova.openova.io/api/v1/embed';
const SWIPE_THRESHOLD = SCREEN_HEIGHT * 0.15;

function getVideoKey(movie: Movie): string | null {
  const k = movie.verticalTrailerYoutubeKey;
  return k && k !== 'NOT_FOUND' ? k : null;
}

export default function ReelsScreen() {
  const router = useRouter();
  const country = useAppStore((s) => s.country);
  const [savedIds, setSavedIds] = useState<Set<number>>(new Set());
  const [ratings, setRatings] = useState<Record<number, number>>({});
  const [dismissedIds, setDismissedIds] = useState<Set<number>>(new Set());
  const [activeIndex, setActiveIndex] = useState(0);
  const [tabFocused, setTabFocused] = useState(false);
  const [playerReady, setPlayerReady] = useState(false);
  const [debugState, setDebugState] = useState('INIT');

  const webViewRef = useRef<WebView>(null);

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

  // WebView message handler
  const onMessage = useCallback((e: any) => {
    try {
      const msg = JSON.parse(e.nativeEvent.data);
      if (msg.type === 'playerReady') {
        setPlayerReady(true);
        setDebugState('READY');
        // Unmute after player is ready
        webViewRef.current?.injectJavaScript('player.unMute(); true;');
      }
      if (msg.type === 'playerPlaying') {
        setDebugState(`PLAYING:${msg.videoKey?.slice(0, 6) || '?'}`);
        // Always ensure unmuted when playing
        webViewRef.current?.injectJavaScript('player.unMute(); true;');
      }
    } catch {}
  }, []);

  // --- Swipe logic ---
  const activeRef = useSharedValue(0);
  const dragY = useSharedValue(0);
  const isAnimating = useSharedValue(false);

  const doSwitch = useCallback((nextIdx: number, direction: 'up' | 'down') => {
    const movie = movies[nextIdx];
    if (!movie) return;
    const key = getVideoKey(movie);
    if (!key) return;
    activeRef.value = nextIdx;
    setActiveIndex(nextIdx);
    webViewRef.current?.injectJavaScript(`switchVideo('${key}'); true;`);
    setDebugState(`SWITCH:${key.slice(0, 6)}`);
    // Snap in from opposite direction
    dragY.value = direction === 'up' ? SCREEN_HEIGHT * 0.3 : -SCREEN_HEIGHT * 0.3;
    dragY.value = withTiming(0, { duration: 200, easing: Easing.out(Easing.cubic) }, () => {
      isAnimating.value = false;
    });
  }, [movies, activeRef, dragY, isAnimating]);

  const snapBack = useCallback(() => {
    isAnimating.value = false;
  }, [isAnimating]);

  const pan = Gesture.Pan()
    .activeOffsetY([-10, 10])
    .onUpdate((e) => {
      if (!isAnimating.value) {
        dragY.value = e.translationY;
      }
    })
    .onEnd((e) => {
      if (isAnimating.value) return;
      const idx = activeRef.value;
      const len = movies.length;

      if (e.translationY < -SWIPE_THRESHOLD && idx < len - 1) {
        // Swipe up → slide out up, then switch and slide in from below
        isAnimating.value = true;
        dragY.value = withTiming(-SCREEN_HEIGHT, {
          duration: 200,
          easing: Easing.in(Easing.cubic),
        }, () => {
          runOnJS(doSwitch)(idx + 1, 'up');
        });
      } else if (e.translationY > SWIPE_THRESHOLD && idx > 0) {
        // Swipe down → slide out down, then switch and slide in from above
        isAnimating.value = true;
        dragY.value = withTiming(SCREEN_HEIGHT, {
          duration: 200,
          easing: Easing.in(Easing.cubic),
        }, () => {
          runOnJS(doSwitch)(idx - 1, 'down');
        });
      } else {
        // Snap back
        dragY.value = withTiming(0, { duration: 150 }, () => {
          runOnJS(snapBack)();
        });
      }
    });

  const tap = Gesture.Tap()
    .onEnd(() => {
      const idx = activeRef.value;
      const movie = movies[idx];
      if (movie) runOnJS(router.push)(`/movie/${movie.id}`);
    });

  const gesture = Gesture.Race(pan, tap);

  // Animate the entire content (WebView + overlay) together
  const contentStyle = useAnimatedStyle(() => ({
    transform: [{ translateY: dragY.value }],
  }));

  if (isLoading || movies.length === 0) {
    return (
      <View style={styles.loading}>
        <ActivityIndicator color={Colors.primary} size="large" />
      </View>
    );
  }

  const firstKey = getVideoKey(movies[0]);
  const currentMovie = movies[activeIndex];
  const genreLabel = currentMovie?.genres.slice(0, 2).map((g: any) => g.name).join(' · ') || '';
  const runtimeLabel = currentMovie?.runtime ? `${currentMovie.runtime}m` : '';

  return (
    <View style={styles.container}>
      {/* Animated content — WebView + overlay move together on swipe */}
      <Animated.View style={[StyleSheet.absoluteFill, contentStyle]}>
        {/* Single WebView — always on screen, never remounted */}
        {firstKey && (
          <WebView
            ref={webViewRef}
            source={{ uri: `${EMBED_BASE}/${firstKey}?autoplay=1&controls=0&mute=1` }}
            style={StyleSheet.absoluteFill}
            allowsInlineMediaPlayback
            mediaPlaybackRequiresUserAction={false}
            scrollEnabled={false}
            bounces={false}
            startInLoadingState={false}
            onMessage={onMessage}
            pointerEvents="none"
          />
        )}

        {/* Movie info overlay */}
        <View style={StyleSheet.absoluteFill} pointerEvents="none">
        {/* Debug badges */}
        <View style={styles.badge}>
          <Text style={styles.badgeText}>
            {BUILD_VERSION} | idx={activeIndex} | {debugState}
          </Text>
        </View>

        {/* CinovaScore */}
        {currentMovie?.cinovaScore != null && (
          <View style={styles.scoreContainer}>
            <CinovaScore score={currentMovie.cinovaScore} size="md" />
          </View>
        )}

        {/* Bottom content */}
        <View style={styles.bottomContent}>
          {currentMovie?.providers && currentMovie.providers.length > 0 && (
            <View style={styles.providerRow}>
              {currentMovie.providers.slice(0, 4).map((p: any, i: number) => (
                <StreamingBadge key={`${p.providerId}-${i}`} provider={p} variant="icon" size={32} />
              ))}
            </View>
          )}
          <Text style={styles.title} numberOfLines={2}>{currentMovie?.title}</Text>
          <Text style={styles.meta}>
            {[currentMovie?.year, genreLabel, runtimeLabel].filter(Boolean).join(' · ')}
          </Text>
          {(currentMovie?.cinovaSynopsis ?? currentMovie?.aiDescription ?? currentMovie?.overview) ? (
            <Text style={styles.synopsis} numberOfLines={2}>
              {currentMovie.cinovaSynopsis ?? currentMovie.aiDescription ?? currentMovie.overview}
            </Text>
          ) : null}
        </View>
      </View>
      </Animated.View>

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
    backgroundColor: '#000',
  },
  loading: {
    flex: 1,
    backgroundColor: Colors.background,
    justifyContent: 'center',
    alignItems: 'center',
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
  scoreContainer: {
    position: 'absolute',
    top: 60,
    right: Spacing[4],
    backgroundColor: 'rgba(0,0,0,0.6)',
    borderRadius: Radius.full,
    padding: Spacing[1.5],
  },
  bottomContent: {
    position: 'absolute',
    bottom: 0,
    left: 0,
    right: 0,
    paddingHorizontal: Spacing[4],
    paddingBottom: Spacing[10],
  },
  providerRow: {
    flexDirection: 'row',
    gap: Spacing[2],
    marginBottom: Spacing[3],
  },
  title: {
    color: Colors.textPrimary,
    fontSize: Typography['2xl'],
    fontWeight: Typography.black,
    letterSpacing: Typography.tighter,
    lineHeight: Typography['2xl'] * 1.15,
    marginBottom: Spacing[1.5],
  },
  meta: {
    color: Colors.textSecondary,
    fontSize: Typography.sm,
    fontWeight: Typography.medium,
    marginBottom: Spacing[2],
  },
  synopsis: {
    color: Colors.textSecondary,
    fontSize: Typography.sm,
    fontStyle: 'italic',
    lineHeight: Typography.sm * 1.5,
  },
});
