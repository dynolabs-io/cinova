/**
 * Discover screen — Full-screen vertical Reels feed
 *
 * TikTok / Instagram Reels-style vertical paging.
 * Prefetches adjacent items. Loads more as user scrolls toward end.
 */

import React, { useCallback, useRef, useState } from 'react';
import {
  View,
  FlatList,
  StyleSheet,
  Dimensions,
  ActivityIndicator,
  Text,
  Alert,
  TouchableOpacity,
  ViewToken,
} from 'react-native';
import { useSafeAreaInsets } from 'react-native-safe-area-context';
import { useInfiniteQuery } from '@tanstack/react-query';
import { useLocalSearchParams, useRouter } from 'expo-router';
import ReelItem from '../../components/ui/ReelItem';
import { getDiscoverFeed, saveTitle, rateTitle, dismissTitle } from '../../services/api';
import { useAppStore } from '../../store/useAppStore';
import { Colors, Typography, Spacing, Radius } from '../../constants/theme';
import type { Movie } from '../../types';

const { height: SCREEN_HEIGHT, width: SCREEN_WIDTH } = Dimensions.get('window');

export default function DiscoverScreen() {
  const insets = useSafeAreaInsets();
  const router = useRouter();
  const country = useAppStore((s) => s.country);
  const { genre, theme, mood } = useLocalSearchParams<{ genre?: string; theme?: string; mood?: string }>();
  const activeFilter = genre ? { type: 'genre', value: genre } : theme ? { type: 'theme', value: theme } : mood ? { type: 'mood', value: mood } : null;
  const [savedIds, setSavedIds] = useState<Set<number>>(new Set());
  const [ratedIds, setRatedIds] = useState<Map<number, number>>(new Map());
  const [dismissedIds, setDismissedIds] = useState<Set<number>>(new Set());

  const {
    data,
    fetchNextPage,
    hasNextPage,
    isFetchingNextPage,
    isLoading,
  } = useInfiniteQuery({
    queryKey: ['discover-feed', country, genre, theme, mood],
    queryFn: ({ pageParam = 1 }) => getDiscoverFeed(country, pageParam as number),
    initialPageParam: 1,
    getNextPageParam: (lastPage, allPages) =>
      lastPage.length === 20 ? allPages.length + 1 : undefined,
  });

  const movies: Movie[] = (data?.pages ?? [])
    .flat()
    .filter((m) => !dismissedIds.has(m.id));

  const onEndReached = useCallback(() => {
    if (hasNextPage && !isFetchingNextPage) {
      fetchNextPage();
    }
  }, [fetchNextPage, hasNextPage, isFetchingNextPage]);

  const handleSave = useCallback(async (movie: Movie) => {
    setSavedIds((prev) => {
      const next = new Set(prev);
      next.add(movie.id);
      return next;
    });
    try {
      await saveTitle(movie.tmdbId);
    } catch {
      setSavedIds((prev) => {
        const next = new Set(prev);
        next.delete(movie.id);
        return next;
      });
    }
  }, []);

  const handleRate = useCallback((movie: Movie) => {
    Alert.alert(
      `Rate "${movie.title}"`,
      'How would you rate it?',
      [
        { text: 'Cancel', style: 'cancel' },
        ...[1,2,3,4,5,6,7,8,9,10].map((score) => ({
          text: `${score}/10`,
          onPress: async () => {
            try {
              await rateTitle(movie.tmdbId, score);
              setRatedIds((prev) => new Map(prev).set(movie.id, score));
            } catch {
              Alert.alert('Error', 'Could not save rating. Please try again.');
            }
          },
        })),
      ]
    );
  }, []);

  const handleDismiss = useCallback(async (movie: Movie) => {
    setDismissedIds((prev) => new Set(prev).add(movie.id));
    try {
      await dismissTitle(movie.tmdbId);
    } catch {
      setDismissedIds((prev) => {
        const next = new Set(prev);
        next.delete(movie.id);
        return next;
      });
    }
  }, []);

  const getItemLayout = useCallback(
    (_: unknown, index: number) => ({
      length: SCREEN_HEIGHT,
      offset: SCREEN_HEIGHT * index,
      index,
    }),
    []
  );

  const renderItem = useCallback(
    ({ item }: { item: Movie }) => (
      <ReelItem
        movie={item}
        isSaved={savedIds.has(item.id)}
        userRating={ratedIds.get(item.id)}
        onSave={handleSave}
        onRate={handleRate}
        onDismiss={handleDismiss}
      />
    ),
    [handleDismiss, handleRate, handleSave, savedIds, ratedIds]
  );

  if (isLoading) {
    return (
      <View style={styles.loading}>
        <ActivityIndicator color={Colors.primary} size="large" />
        <Text style={styles.loadingText}>Loading your feed…</Text>
      </View>
    );
  }

  return (
    <View style={[styles.container, { paddingTop: 0 }]}>
      {activeFilter && (
        <View style={[styles.filterBanner, { top: insets.top + 8 }]}>
          <Text style={styles.filterLabel}>{activeFilter.type}: {activeFilter.value}</Text>
          <TouchableOpacity onPress={() => router.replace('/(tabs)/discover')} style={styles.filterClear}>
            <Text style={styles.filterClearText}>✕</Text>
          </TouchableOpacity>
        </View>
      )}
      <FlatList
        data={movies}
        renderItem={renderItem}
        keyExtractor={(item, index) => String(item.tmdbId ?? item.id ?? index)}
        pagingEnabled
        showsVerticalScrollIndicator={false}
        snapToInterval={SCREEN_HEIGHT}
        snapToAlignment="start"
        decelerationRate="fast"
        getItemLayout={getItemLayout}
        onEndReached={onEndReached}
        onEndReachedThreshold={3}
        initialNumToRender={3}
        maxToRenderPerBatch={3}
        windowSize={5}
        removeClippedSubviews
        ListFooterComponent={
          isFetchingNextPage ? (
            <View style={styles.footer}>
              <ActivityIndicator color={Colors.primary} />
            </View>
          ) : null
        }
      />
    </View>
  );
}

const styles = StyleSheet.create({
  container: {
    flex: 1,
    backgroundColor: Colors.background,
    width: SCREEN_WIDTH,
  },
  loading: {
    flex: 1,
    backgroundColor: Colors.background,
    justifyContent: 'center',
    alignItems: 'center',
    gap: 12,
  },
  loadingText: {
    color: Colors.textSecondary,
    fontSize: Typography.base,
  },
  footer: {
    height: 60,
    justifyContent: 'center',
    alignItems: 'center',
    backgroundColor: Colors.background,
  },
  filterBanner: {
    position: 'absolute',
    left: Spacing[4],
    right: Spacing[4],
    zIndex: 20,
    flexDirection: 'row',
    alignItems: 'center',
    backgroundColor: 'rgba(0,0,0,0.75)',
    borderRadius: Radius.full ?? 999,
    paddingHorizontal: Spacing[4],
    paddingVertical: Spacing[2],
    gap: Spacing[2],
  },
  filterLabel: {
    flex: 1,
    color: Colors.textPrimary,
    fontSize: Typography.sm,
    fontWeight: '600',
    textTransform: 'capitalize',
  },
  filterClear: {
    padding: Spacing[1],
  },
  filterClearText: {
    color: Colors.textMuted,
    fontSize: Typography.sm,
  },
});
