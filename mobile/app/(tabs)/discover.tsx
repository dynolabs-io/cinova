/**
 * Discover screen — Mosaic / Treemap Layout
 *
 * Movies are grouped into layout templates. Each template places 1–4 cards
 * with different flex widths and row heights — creating a continuous mosaic
 * similar to a treemap or newspaper layout.
 *
 * Base grid: 6 units wide. Templates cycle through a non-repeating sequence.
 * Infinite scroll loads more pages; incomplete trailing groups are held until
 * enough movies arrive to fill the next template.
 */

import React, { useCallback, useState, useMemo } from 'react';
import {
  View,
  Text,
  StyleSheet,
  TouchableOpacity,
  ScrollView,
  Dimensions,
  ActivityIndicator,
  NativeSyntheticEvent,
  NativeScrollEvent,
} from 'react-native';
import { Image } from 'expo-image';
import { LinearGradient } from 'expo-linear-gradient';
import { useSafeAreaInsets } from 'react-native-safe-area-context';
import { useInfiniteQuery } from '@tanstack/react-query';
import { useLocalSearchParams, useRouter } from 'expo-router';
import TrailerPlayer from '../../components/ui/TrailerPlayer';
import CinovaScore from '../../components/ui/CinovaScore';
import { getDiscoverFeed, saveTitle } from '../../services/api';
import { useAppStore } from '../../store/useAppStore';
import { hapticSuccess, hapticMedium } from '../../services/haptics';
import { Colors, Typography, Spacing, Radius } from '../../constants/theme';
import type { Movie } from '../../types';

const { width: SCREEN_WIDTH } = Dimensions.get('window');
const SIDE_PAD = 0;
const GAP = 1; // thin white border between tiles
// One unit = 1/6 of the full width (no side padding, 5 gaps between 6 units)
const U = (SCREEN_WIDTH - GAP * 5) / 6;
const TMDB = 'https://image.tmdb.org/t/p';

// ── MosaicCard ────────────────────────────────────────────────────────────────
// Fills whatever space the parent gives it. Size is driven by the template.

interface CardHandlers {
  savedIds: Set<number>;
  onSave: (m: Movie) => void;
  onPlayTrailer: (m: Movie) => void;
}

interface MosaicCardProps {
  movie: Movie;
  style?: object;
  savedIds: Set<number>;
  onSave: (m: Movie) => void;
  onPlayTrailer: (m: Movie) => void;
}

function MosaicCard({ movie, style, savedIds, onSave, onPlayTrailer }: MosaicCardProps) {
  const router = useRouter();
  const uri = movie.posterPath ? `${TMDB}/w500${movie.posterPath}` : '';

  return (
    <TouchableOpacity
      style={[cardStyles.card, style]}
      activeOpacity={0.88}
      onPress={() => router.push(`/movie/${movie.id}`)}
      onLongPress={() => { hapticSuccess(); onSave(movie); }}
      delayLongPress={400}
    >
      <Image
        source={uri ? { uri } : undefined}
        style={StyleSheet.absoluteFill}
        contentFit="cover"
        transition={200}
        placeholder={{ blurhash: 'L00000fQfQfQfQfQfQfQfQfQfQfQ' }}
      />
      <LinearGradient
        colors={['transparent', 'rgba(0,0,0,0.82)']}
        locations={[0.45, 1]}
        style={StyleSheet.absoluteFill}
      />
      {movie.cinovaScore != null && (
        <View style={cardStyles.score}>
          <CinovaScore score={movie.cinovaScore} size="sm" />
        </View>
      )}
      {savedIds.has(movie.id) && (
        <View style={cardStyles.saved}>
          <Text style={cardStyles.savedTick}>✓</Text>
        </View>
      )}
      {movie.verticalTrailerYoutubeKey && (
        <TouchableOpacity
          style={cardStyles.playBtn}
          onPress={() => { hapticMedium(); onPlayTrailer(movie); }}
          activeOpacity={0.8}
        >
          <Text style={cardStyles.playIcon}>▶</Text>
        </TouchableOpacity>
      )}
      <View style={cardStyles.info}>
        <Text style={cardStyles.title} numberOfLines={2}>{movie.title}</Text>
        <Text style={cardStyles.year}>{movie.year}</Text>
      </View>
    </TouchableOpacity>
  );
}

const cardStyles = StyleSheet.create({
  card: {
    overflow: 'hidden',
    borderRadius: 0,
    backgroundColor: Colors.card,
  },
  score: {
    position: 'absolute',
    top: 4, left: 4,
    backgroundColor: 'rgba(0,0,0,0.55)',
    borderRadius: Radius.full,
    padding: 2,
  },
  saved: {
    position: 'absolute',
    top: 4, right: 4,
    width: 18, height: 18, borderRadius: 9,
    backgroundColor: Colors.primary,
    justifyContent: 'center', alignItems: 'center',
  },
  savedTick: { color: '#fff', fontSize: 9, fontWeight: '700' },
  playBtn: {
    position: 'absolute',
    top: '35%', alignSelf: 'center',
    width: 34, height: 34, borderRadius: 17,
    backgroundColor: 'rgba(0,0,0,0.6)',
    borderWidth: 1.5, borderColor: 'rgba(255,255,255,0.7)',
    justifyContent: 'center', alignItems: 'center',
  },
  playIcon: { color: '#fff', fontSize: 13, marginLeft: 2 },
  info: {
    position: 'absolute',
    bottom: 0, left: 0, right: 0,
    paddingHorizontal: 6, paddingBottom: 6,
  },
  title: {
    color: Colors.textPrimary,
    fontSize: 10, fontWeight: '700', lineHeight: 13,
  },
  year: { color: Colors.textMuted, fontSize: 9 },
});

// ── Layout Templates ──────────────────────────────────────────────────────────
// U = 1 grid unit. Heights and flex values define the mosaic proportions.

type Template = {
  count: number;
  render: (movies: Movie[], h: CardHandlers) => React.ReactElement;
};

// Shorthand builder — creates a MosaicCard spreading handlers
const card = (m: Movie, h: CardHandlers, style: object) => (
  <MosaicCard
    key={m.id}
    movie={m}
    style={style}
    savedIds={h.savedIds}
    onSave={h.onSave}
    onPlayTrailer={h.onPlayTrailer}
  />
);

const TEMPLATES: Template[] = [
  // T0 ── Large left (4u) + 2 stacked right (2u) ── 3 movies
  //  ┌──────────────┬───────┐
  //  │              │   B   │
  //  │      A       ├───────┤
  //  │              │   C   │
  //  └──────────────┴───────┘
  {
    count: 3,
    render: ([a, b, c], h) => (
      <View style={{ flexDirection: 'row', gap: GAP, height: U * 5.5 }}>
        {card(a, h, { flex: 4 })}
        <View style={{ flex: 2, gap: GAP }}>
          {card(b, h, { flex: 1 })}
          {card(c, h, { flex: 1 })}
        </View>
      </View>
    ),
  },

  // T1 ── 3 equal thirds ── 3 movies
  //  ┌──────┬──────┬──────┐
  //  │  A   │  B   │  C   │
  //  └──────┴──────┴──────┘
  {
    count: 3,
    render: ([a, b, c], h) => (
      <View style={{ flexDirection: 'row', gap: GAP, height: U * 3.5 }}>
        {card(a, h, { flex: 2 })}
        {card(b, h, { flex: 2 })}
        {card(c, h, { flex: 2 })}
      </View>
    ),
  },

  // T2 ── Full-width banner ── 1 movie
  //  ┌────────────────────┐
  //  │         A          │
  //  └────────────────────┘
  {
    count: 1,
    render: ([a], h) => (
      <View style={{ height: U * 2.2 }}>
        {card(a, h, { flex: 1 })}
      </View>
    ),
  },

  // T3 ── Wide left (4u) + narrow right (2u) ── 2 movies
  //  ┌──────────────┬──────┐
  //  │      A       │  B   │
  //  └──────────────┴──────┘
  {
    count: 2,
    render: ([a, b], h) => (
      <View style={{ flexDirection: 'row', gap: GAP, height: U * 4 }}>
        {card(a, h, { flex: 4 })}
        {card(b, h, { flex: 2 })}
      </View>
    ),
  },

  // T4 ── 2 stacked left (2u) + large right (4u) ── 3 movies
  //  ┌──────┬──────────────┐
  //  │  A   │              │
  //  ├──────┤      C       │
  //  │  B   │              │
  //  └──────┴──────────────┘
  {
    count: 3,
    render: ([a, b, c], h) => (
      <View style={{ flexDirection: 'row', gap: GAP, height: U * 5 }}>
        <View style={{ flex: 2, gap: GAP }}>
          {card(a, h, { flex: 1 })}
          {card(b, h, { flex: 1 })}
        </View>
        {card(c, h, { flex: 4 })}
      </View>
    ),
  },

  // T5 ── Narrow left (2u) + wide right (4u) ── 2 movies
  //  ┌──────┬──────────────┐
  //  │  A   │      B       │
  //  └──────┴──────────────┘
  {
    count: 2,
    render: ([a, b], h) => (
      <View style={{ flexDirection: 'row', gap: GAP, height: U * 3.5 }}>
        {card(a, h, { flex: 2 })}
        {card(b, h, { flex: 4 })}
      </View>
    ),
  },

  // T6 ── 4 movies: top row 3+3, bottom row 2+4
  //  ┌─────────┬─────────┐
  //  │    A    │    B    │
  //  ├──────┬──┴──────────┤
  //  │  C   │      D     │
  //  └──────┴─────────────┘
  {
    count: 4,
    render: ([a, b, c, d], h) => (
      <View style={{ gap: GAP }}>
        <View style={{ flexDirection: 'row', gap: GAP, height: U * 3 }}>
          {card(a, h, { flex: 3 })}
          {card(b, h, { flex: 3 })}
        </View>
        <View style={{ flexDirection: 'row', gap: GAP, height: U * 3.5 }}>
          {card(c, h, { flex: 2 })}
          {card(d, h, { flex: 4 })}
        </View>
      </View>
    ),
  },

  // T7 ── Tall thin left (2u) + 2 stacked right (4u wide) ── 3 movies
  //  ┌──────┬──────────────┐
  //  │      │      B       │
  //  │  A   ├──────────────┤
  //  │      │      C       │
  //  └──────┴──────────────┘
  {
    count: 3,
    render: ([a, b, c], h) => (
      <View style={{ flexDirection: 'row', gap: GAP, height: U * 6 }}>
        {card(a, h, { flex: 2 })}
        <View style={{ flex: 4, gap: GAP }}>
          {card(b, h, { flex: 3 })}
          {card(c, h, { flex: 2 })}
        </View>
      </View>
    ),
  },

  // T8 ── 2 equal halves (3u + 3u), tall ── 2 movies
  //  ┌─────────┬─────────┐
  //  │         │         │
  //  │    A    │    B    │
  //  │         │         │
  //  └─────────┴─────────┘
  {
    count: 2,
    render: ([a, b], h) => (
      <View style={{ flexDirection: 'row', gap: GAP, height: U * 4.5 }}>
        {card(a, h, { flex: 3 })}
        {card(b, h, { flex: 3 })}
      </View>
    ),
  },

  // T9 ── 4 movies: top 4+2, bottom 3+3
  //  ┌──────────────┬──────┐
  //  │      A       │  B   │
  //  ├─────────┬────┴──────┤
  //  │    C    │     D     │
  //  └─────────┴───────────┘
  {
    count: 4,
    render: ([a, b, c, d], h) => (
      <View style={{ gap: GAP }}>
        <View style={{ flexDirection: 'row', gap: GAP, height: U * 3.8 }}>
          {card(a, h, { flex: 4 })}
          {card(b, h, { flex: 2 })}
        </View>
        <View style={{ flexDirection: 'row', gap: GAP, height: U * 3 }}>
          {card(c, h, { flex: 3 })}
          {card(d, h, { flex: 3 })}
        </View>
      </View>
    ),
  },
];

// Cycling sequence — 20-step non-repeating pattern across all 10 templates
const SEQUENCE = [0, 6, 1, 3, 7, 2, 4, 9, 5, 8, 0, 5, 6, 3, 1, 7, 4, 2, 8, 9];

function groupMovies(movies: Movie[]): { tIdx: number; movies: Movie[] }[] {
  const groups: { tIdx: number; movies: Movie[] }[] = [];
  let i = 0;
  let seqPos = 0;
  while (i < movies.length) {
    const tIdx = SEQUENCE[seqPos % SEQUENCE.length];
    const count = TEMPLATES[tIdx].count;
    if (i + count > movies.length) break; // wait for full group
    groups.push({ tIdx, movies: movies.slice(i, i + count) });
    i += count;
    seqPos++;
  }
  return groups;
}

// ── Screen ────────────────────────────────────────────────────────────────────

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

  const { data, fetchNextPage, hasNextPage, isFetchingNextPage, isLoading } =
    useInfiniteQuery({
      queryKey: ['discover-feed', country, genre, theme, mood],
      queryFn: ({ pageParam = 1 }) => getDiscoverFeed(country, pageParam as number),
      initialPageParam: 1,
      getNextPageParam: (lastPage, allPages) =>
        lastPage.length === 20 ? allPages.length + 1 : undefined,
    });

  const movies: Movie[] = (data?.pages ?? []).flat();
  const groups = useMemo(() => groupMovies(movies), [movies]);

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

  const handleScroll = useCallback(
    ({ nativeEvent }: NativeSyntheticEvent<NativeScrollEvent>) => {
      const { layoutMeasurement, contentOffset, contentSize } = nativeEvent;
      const dist = contentSize.height - contentOffset.y - layoutMeasurement.height;
      if (dist < 800 && hasNextPage && !isFetchingNextPage) fetchNextPage();
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

  const handlers: CardHandlers = { savedIds, onSave: handleSave, onPlayTrailer: handlePlayTrailer };

  return (
    <View style={styles.container}>
      {trailerMovie?.verticalTrailerYoutubeKey && (
        <TrailerPlayer
          youtubeKey={trailerMovie.verticalTrailerYoutubeKey}
          title={trailerMovie.title}
          primaryProvider={trailerMovie.providers?.[0] ?? null}
          tmdbId={trailerMovie.tmdbId}
          onClose={() => setTrailerMovie(null)}
        />
      )}

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
          paddingTop: 0,
          paddingBottom: insets.bottom + 80,
          gap: GAP,
          backgroundColor: '#fff',
        }}
      >
        {groups.map((g, idx) => (
          <View key={idx} style={{ backgroundColor: '#fff', gap: GAP }}>
            {TEMPLATES[g.tIdx].render(g.movies, handlers)}
          </View>
        ))}

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
  filterClear: { padding: Spacing[1] },
  filterClearText: {
    color: Colors.textMuted,
    fontSize: Typography.sm,
  },
});
