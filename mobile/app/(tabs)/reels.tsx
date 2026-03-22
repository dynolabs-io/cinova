/**
 * Reels screen — Full-screen vertical swipe feed (TikTok / Instagram Reels style)
 *
 * Video is handled by a single PersistentPlayer WebView that stays alive for the
 * whole session. Switching reels injects player.loadVideoById() — no per-video
 * WebView initialization overhead after the first load.
 */

import React, { useCallback, useRef, useState } from 'react';
import {
  View,
  StyleSheet,
  Dimensions,
  FlatList,
  ActivityIndicator,
  ViewToken,
} from 'react-native';
import { useFocusEffect } from 'expo-router';
import { useInfiniteQuery } from '@tanstack/react-query';
import { useSafeAreaInsets } from 'react-native-safe-area-context';
import ReelItem from '../../components/ui/ReelItem';
import PersistentPlayer from '../../components/ui/PersistentPlayer';
import { getDiscoverFeed, saveTitle, rateTitle, dismissTitle } from '../../services/api';
import { useAppStore } from '../../store/useAppStore';
import { Colors } from '../../constants/theme';
import type { Movie } from '../../types';

const { height: SCREEN_HEIGHT } = Dimensions.get('window');

function validKey(m: Movie | undefined): string | null {
  if (!m) return null;
  const k = m.verticalTrailerYoutubeKey;
  return k && k !== 'NOT_FOUND' ? k : null;
}

export default function ReelsScreen() {
  const country = useAppStore((s) => s.country);
  const insets = useSafeAreaInsets();
  const [savedIds, setSavedIds] = useState<Set<number>>(new Set());
  const [ratings, setRatings] = useState<Record<number, number>>({});
  const [dismissedIds, setDismissedIds] = useState<Set<number>>(new Set());
  const [activeIndex, setActiveIndex] = useState(0);
  const [tabFocused, setTabFocused] = useState(false);

  // initialVideoKey is set once (first non-null key) — the WebView URL never changes
  const [initialVideoKey, setInitialVideoKey] = useState<string | null>(null);

  useFocusEffect(useCallback(() => {
    setTabFocused(true);
    return () => setTabFocused(false);
  }, []));

  const viewabilityConfig = useRef({ viewAreaCoveragePercentThreshold: 60 });
  const onViewableItemsChanged = useRef(({ viewableItems }: { viewableItems: ViewToken[] }) => {
    if (viewableItems.length > 0) setActiveIndex(viewableItems[0].index ?? 0);
  });

  const {
    data,
    fetchNextPage,
    hasNextPage,
    isFetchingNextPage,
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
    .filter((m) => !dismissedIds.has(m.id));

  // Set initialVideoKey once when data first arrives
  if (!initialVideoKey && movies.length > 0) {
    const k = validKey(movies[0]);
    if (k) setInitialVideoKey(k);
  }

  const activeVideoKey = validKey(movies[activeIndex]);

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

  const onEndReached = useCallback(() => {
    if (hasNextPage && !isFetchingNextPage) fetchNextPage();
  }, [fetchNextPage, hasNextPage, isFetchingNextPage]);

  if (isLoading) {
    return (
      <View style={styles.loading}>
        <ActivityIndicator color={Colors.primary} size="large" />
      </View>
    );
  }

  return (
    <View style={[styles.container, { paddingBottom: insets.bottom }]}>

      {/* Single persistent video player — always behind the FlatList content */}
      {initialVideoKey && (
        <PersistentPlayer
          initialVideoKey={initialVideoKey}
          videoKey={activeVideoKey}
          playing={tabFocused}
        />
      )}

      <FlatList
        data={movies}
        keyExtractor={(item, i) => `${item.id}-${i}`}
        renderItem={({ item, index }) => (
          <ReelItem
            movie={item}
            isActive={index === activeIndex}
            isSaved={savedIds.has(item.id)}
            userRating={ratings[item.id]}
            onSave={handleSave}
            onRate={handleRate}
            onDismiss={handleDismiss}
          />
        )}
        pagingEnabled
        showsVerticalScrollIndicator={false}
        snapToInterval={SCREEN_HEIGHT}
        snapToAlignment="start"
        decelerationRate="fast"
        onEndReached={onEndReached}
        onEndReachedThreshold={2}
        onViewableItemsChanged={onViewableItemsChanged.current}
        viewabilityConfig={viewabilityConfig.current}
        getItemLayout={(_, index) => ({
          length: SCREEN_HEIGHT,
          offset: SCREEN_HEIGHT * index,
          index,
        })}
        removeClippedSubviews={false}
      />
    </View>
  );
}

const styles = StyleSheet.create({
  container: {
    flex: 1,
    backgroundColor: Colors.background,
  },
  loading: {
    flex: 1,
    backgroundColor: Colors.background,
    justifyContent: 'center',
    alignItems: 'center',
  },
});
