/**
 * Discover screen — Staggered masonry grid
 *
 * Two-column Instagram-style masonry layout.
 * Movies with a vertical_trailer_youtube_key get taller "video" cells (3:4).
 * All others get standard poster cells (2:3).
 * Age-biased ordering from backend (cinova_score × exp(−0.15 × age_years)).
 */

import React, { useCallback, useState } from 'react';
import {
  View,
  StyleSheet,
  ActivityIndicator,
  Text,
  TouchableOpacity,
} from 'react-native';
import { MasonryFlashList } from '@shopify/flash-list';
import { useSafeAreaInsets } from 'react-native-safe-area-context';
import { useInfiniteQuery } from '@tanstack/react-query';
import { useLocalSearchParams, useRouter } from 'expo-router';
import DiscoverGridCard, { VIDEO_CARD_HEIGHT, POSTER_CARD_HEIGHT } from '../../components/ui/DiscoverGridCard';
import TrailerPlayer from '../../components/ui/TrailerPlayer';
import { getDiscoverFeed, saveTitle, dismissTitle } from '../../services/api';
import { useAppStore } from '../../store/useAppStore';
import { Colors, Typography, Spacing, Radius } from '../../constants/theme';
import type { Movie } from '../../types';

export default function DiscoverScreen() {
  const insets = useSafeAreaInsets();
  const router = useRouter();
  const country = useAppStore((s) => s.country);
  const { genre, theme, mood } = useLocalSearchParams<{ genre?: string; theme?: string; mood?: string }>();
  const activeFilter = genre
    ? { type: 'genre', value: genre }
    : theme
    ? { type: 'theme', value: theme }
    : mood
    ? { type: 'mood', value: mood }
    : null;

  const [savedIds, setSavedIds] = useState<Set<number>>(new Set());
  const [dismissedIds, setDismissedIds] = useState<Set<number>>(new Set());
  const [trailerMovie, setTrailerMovie] = useState<Movie | null>(null);

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
    if (hasNextPage && !isFetchingNextPage) fetchNextPage();
  }, [fetchNextPage, hasNextPage, isFetchingNextPage]);

  const handleSave = useCallback(async (movie: Movie) => {
    setSavedIds((prev) => new Set(prev).add(movie.id));
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

  const handlePlayTrailer = useCallback((movie: Movie) => {
    setTrailerMovie(movie);
  }, []);

  const renderItem = useCallback(
    ({ item }: { item: Movie }) => (
      <DiscoverGridCard
        movie={item}
        isSaved={savedIds.has(item.id)}
        onSave={handleSave}
        onPlayTrailer={handlePlayTrailer}
      />
    ),
    [handleSave, handlePlayTrailer, savedIds]
  );

  const getItemType = useCallback(
    (item: Movie) => (item.verticalTrailerYoutubeKey ? 'video' : 'poster'),
    []
  );

  if (isLoading) {
    return (
      <View style={styles.loading}>
        <ActivityIndicator color={Colors.primary} size="large" />
        <Text style={styles.loadingText}>Curating your feed…</Text>
      </View>
    );
  }

  const headerHeight = insets.top + 8 + 36; // safe area + padding + banner height

  return (
    <View style={styles.container}>
      {/* Trailer player — rendered at screen level to avoid AutoLayoutView crash inside list cells */}
      {trailerMovie?.verticalTrailerYoutubeKey && (
        <TrailerPlayer
          youtubeKey={trailerMovie.verticalTrailerYoutubeKey}
          title={trailerMovie.title}
          primaryProvider={trailerMovie.providers?.[0] ?? null}
          tmdbId={trailerMovie.tmdbId}
          onClose={() => setTrailerMovie(null)}
        />
      )}

      {/* Filter banner */}
      {activeFilter && (
        <View style={[styles.filterBanner, { top: insets.top + 8 }]}>
          <Text style={styles.filterLabel}>{activeFilter.type}: {activeFilter.value}</Text>
          <TouchableOpacity
            onPress={() => router.replace('/(tabs)/discover')}
            style={styles.filterClear}
          >
            <Text style={styles.filterClearText}>✕</Text>
          </TouchableOpacity>
        </View>
      )}

      <MasonryFlashList
        data={movies}
        numColumns={2}
        renderItem={renderItem}
        keyExtractor={(item, index) => `${item.tmdbId ?? item.id ?? index}-${index}`}
        getItemType={getItemType}
        estimatedItemSize={POSTER_CARD_HEIGHT}
        onEndReached={onEndReached}
        onEndReachedThreshold={3}
        contentContainerStyle={{
          paddingTop: activeFilter ? headerHeight : insets.top + Spacing[3],
          paddingBottom: insets.bottom + 80,
          paddingHorizontal: 12,
        }}
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
