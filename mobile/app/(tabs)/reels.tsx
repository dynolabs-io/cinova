/**
 * Reels screen — Full-screen vertical swipe feed (TikTok / Instagram Reels style)
 *
 * Each ReelItem has its own embedded WebView video player that scrolls with
 * the list item — giving the Instagram effect where you see both videos
 * during a half-drag. WebViews are only mounted for items within ±1 of the
 * active index to keep memory usage low.
 */

import React, { useCallback, useRef, useState } from 'react';
import {
  View,
  Text,
  StyleSheet,
  Dimensions,
  FlatList,
  ActivityIndicator,
  ViewToken,
} from 'react-native';

const BUILD_VERSION = 'v12-522a330';
import { useFocusEffect } from 'expo-router';
import { useInfiniteQuery } from '@tanstack/react-query';
import ReelItem from '../../components/ui/ReelItem';
import { getDiscoverFeed, saveTitle, rateTitle, dismissTitle } from '../../services/api';
import { useAppStore } from '../../store/useAppStore';
import { Colors } from '../../constants/theme';
import type { Movie } from '../../types';

const { height: SCREEN_HEIGHT } = Dimensions.get('window');

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
    <View style={styles.container}>
      <View style={{ position: 'absolute', top: 50, left: 0, right: 0, zIndex: 9999, alignItems: 'center' }}>
        <Text style={{ color: '#0f0', fontSize: 12, fontWeight: '800', backgroundColor: 'rgba(0,0,0,0.7)', paddingHorizontal: 10, paddingVertical: 3, borderRadius: 4 }}>
          {BUILD_VERSION}
        </Text>
      </View>
      <FlatList
        data={movies}
        keyExtractor={(item, i) => `${item.id}-${i}`}
        renderItem={({ item, index }) => (
          <ReelItem
            movie={item}
            isActive={index === activeIndex && tabFocused}
            shouldLoad={Math.abs(index - activeIndex) <= 2 && tabFocused}
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
