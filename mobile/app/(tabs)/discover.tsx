/**
 * Discover screen — True staggered masonry (height-tracking)
 *
 * Each card is assigned to the shortest column at the time of insertion,
 * producing genuine Pinterest-style uneven distribution.
 * Supports 2–3 columns; currently uses 2 on phone.
 * No native modules required (pure ScrollView + View columns).
 */

import React, { useCallback, useState, useMemo } from 'react';
import {
  View,
  StyleSheet,
  ActivityIndicator,
  Text,
  TouchableOpacity,
  ScrollView,
  Dimensions,
  NativeSyntheticEvent,
  NativeScrollEvent,
} from 'react-native';
import { useSafeAreaInsets } from 'react-native-safe-area-context';
import { useInfiniteQuery } from '@tanstack/react-query';
import { useLocalSearchParams, useRouter } from 'expo-router';
import DiscoverGridCard, { getCardHeight } from '../../components/ui/DiscoverGridCard';
import TrailerPlayer from '../../components/ui/TrailerPlayer';
import { getDiscoverFeed, saveTitle } from '../../services/api';
import { useAppStore } from '../../store/useAppStore';
import { Colors, Typography, Spacing, Radius } from '../../constants/theme';
import type { Movie } from '../../types';

const { width: SCREEN_WIDTH } = Dimensions.get('window');
const NUM_COLS = SCREEN_WIDTH >= 768 ? 3 : 2; // 3 cols on tablet, 2 on phone
const COL_GAP = 10;
const SIDE_PAD = 12;

/** Distribute movies into columns by always adding to the shortest column */
function distributeToColumns(movies: Movie[], numCols: number): Movie[][] {
  const cols: Movie[][] = Array.from({ length: numCols }, () => []);
  const heights = new Array<number>(numCols).fill(0);
  for (const movie of movies) {
    const cardH = getCardHeight(movie);
    const shortest = heights.indexOf(Math.min(...heights));
    cols[shortest].push(movie);
    heights[shortest] += cardH + COL_GAP;
  }
  return cols;
}

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

  const movies: Movie[] = (data?.pages ?? []).flat();

  // Distribute into columns by shortest-column height tracking
  const columns = useMemo(() => distributeToColumns(movies, NUM_COLS), [movies]);

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

  // Detect scroll near bottom to load next page
  const handleScroll = useCallback(
    ({ nativeEvent }: NativeSyntheticEvent<NativeScrollEvent>) => {
      const { layoutMeasurement, contentOffset, contentSize } = nativeEvent;
      const distanceFromBottom = contentSize.height - contentOffset.y - layoutMeasurement.height;
      if (distanceFromBottom < 600 && hasNextPage && !isFetchingNextPage) {
        fetchNextPage();
      }
    },
    [fetchNextPage, hasNextPage, isFetchingNextPage]
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
      {/* Trailer player — screen-level */}
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

      <ScrollView
        onScroll={handleScroll}
        scrollEventThrottle={200}
        showsVerticalScrollIndicator={false}
        contentContainerStyle={{
          paddingTop: activeFilter ? headerHeight : insets.top + Spacing[3],
          paddingBottom: insets.bottom + 80,
          paddingHorizontal: SIDE_PAD,
        }}
      >
        <View style={styles.columns}>
          {columns.map((colMovies, colIdx) => (
            <View key={colIdx} style={styles.column}>
              {colMovies.map((movie) => (
                <View key={movie.id} style={styles.cardWrapper}>
                  <DiscoverGridCard
                    movie={movie}
                    isSaved={savedIds.has(movie.id)}
                    onSave={handleSave}
                    onPlayTrailer={handlePlayTrailer}
                  />
                </View>
              ))}
            </View>
          ))}
        </View>

        {isFetchingNextPage && (
          <View style={styles.footer}>
            <ActivityIndicator color={Colors.primary} />
          </View>
        )}
      </ScrollView>
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
  columns: {
    flexDirection: 'row',
    gap: COL_GAP,
    alignItems: 'flex-start',
  },
  column: {
    flex: 1,
    gap: COL_GAP,
  },
  cardWrapper: {
    // each card fills column width; height is set by the card itself
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
