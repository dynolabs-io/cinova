/**
 * Discover screen — Staggered masonry grid
 *
 * Two-column layout using FlatList + paired rows (no native modules required).
 * Movies with a vertical_trailer_youtube_key get taller "video" cells (3:4).
 * All others get standard poster cells (2:3).
 * Age-biased ordering from backend (cinova_score × exp(−0.15 × age_years)).
 */

import React, { useCallback, useState, useMemo } from 'react';
import {
  View,
  StyleSheet,
  ActivityIndicator,
  Text,
  TouchableOpacity,
  FlatList,
} from 'react-native';
import { useSafeAreaInsets } from 'react-native-safe-area-context';
import { useInfiniteQuery } from '@tanstack/react-query';
import { useLocalSearchParams, useRouter } from 'expo-router';
import DiscoverGridCard from '../../components/ui/DiscoverGridCard';
import TrailerPlayer from '../../components/ui/TrailerPlayer';
import { getDiscoverFeed, saveTitle } from '../../services/api';
import { useAppStore } from '../../store/useAppStore';
import { Colors, Typography, Spacing, Radius } from '../../constants/theme';
import type { Movie } from '../../types';

type MoviePair = [Movie, Movie | null];

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

  // Pair movies into rows: [[m0,m1],[m2,m3],...]
  const pairs = useMemo<MoviePair[]>(() => {
    const result: MoviePair[] = [];
    for (let i = 0; i < movies.length; i += 2) {
      result.push([movies[i], movies[i + 1] ?? null]);
    }
    return result;
  }, [movies]);

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

  const renderPair = useCallback(
    ({ item }: { item: MoviePair }) => (
      <View style={styles.row}>
        <DiscoverGridCard
          movie={item[0]}
          isSaved={savedIds.has(item[0].id)}
          onSave={handleSave}
          onPlayTrailer={handlePlayTrailer}
        />
        {item[1] ? (
          <DiscoverGridCard
            movie={item[1]}
            isSaved={savedIds.has(item[1].id)}
            onSave={handleSave}
            onPlayTrailer={handlePlayTrailer}
          />
        ) : (
          <View style={styles.emptyCell} />
        )}
      </View>
    ),
    [handleSave, handlePlayTrailer, savedIds]
  );

  if (isLoading) {
    return (
      <View style={styles.loading}>
        <ActivityIndicator color={Colors.primary} size="large" />
        <Text style={styles.loadingText}>Curating your feed…</Text>
      </View>
    );
  }

  const headerHeight = insets.top + 8 + 36;

  return (
    <View style={styles.container}>
      {/* Trailer player — screen-level so it renders outside the list */}
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

      <FlatList
        data={pairs}
        renderItem={renderPair}
        keyExtractor={(_, index) => `pair-${index}`}
        onEndReached={onEndReached}
        onEndReachedThreshold={3}
        removeClippedSubviews
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
  row: {
    flexDirection: 'row',
    alignItems: 'flex-start',
    gap: 10,
    marginBottom: 10,
  },
  emptyCell: {
    flex: 1,
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
