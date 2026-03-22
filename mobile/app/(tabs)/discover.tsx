/**
 * Discover screen — Mosaic / Treemap Layout
 *
 * Movies are grouped into layout templates. Each template places 1–4 cards
 * with different flex widths and row heights — creating a continuous mosaic
 * similar to a treemap or newspaper layout.
 *
 * Every 3rd movie slot is a VIDEO tile (auto-plays muted, no controls).
 * All other slots are POSTER tiles (TMDB poster image).
 * Tapping any tile — video or poster — navigates to the movie detail page.
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
  LayoutChangeEvent,
} from 'react-native';
import { Image } from 'expo-image';
import { LinearGradient } from 'expo-linear-gradient';
import { useSafeAreaInsets } from 'react-native-safe-area-context';
import { useInfiniteQuery } from '@tanstack/react-query';
import { useLocalSearchParams, useRouter } from 'expo-router';
import YoutubeIframe from 'react-native-youtube-iframe';
import CinovaScore from '../../components/ui/CinovaScore';
import { getDiscoverFeed, saveTitle } from '../../services/api';
import { useAppStore } from '../../store/useAppStore';
import { hapticSuccess } from '../../services/haptics';
import { Colors, Typography, Spacing, Radius } from '../../constants/theme';
import type { Movie } from '../../types';

const { width: SCREEN_WIDTH } = Dimensions.get('window');
const GAP = 1; // thin gap between tiles
const U = (SCREEN_WIDTH - GAP * 5) / 6;
const TMDB = 'https://image.tmdb.org/t/p';

// ── MosaicCard ────────────────────────────────────────────────────────────────

interface MosaicCardProps {
  movie: Movie;
  style?: object;
  isVideoTile: boolean;
  savedIds: Set<number>;
  onSave: (m: Movie) => void;
}

function MosaicCard({ movie, style, isVideoTile, savedIds, onSave }: MosaicCardProps) {
  const router = useRouter();
  const [cardSize, setCardSize] = useState({ width: 0, height: 0 });

  // Video key: prefer vertical trailer, fall back to regular TMDB trailer
  const videoKey = movie.trailerYoutubeKey || movie.verticalTrailerYoutubeKey || null;
  const showVideo = isVideoTile && !!videoKey && cardSize.width > 0;

  const handlePress = useCallback(() => {
    router.push(`/movie/${movie.id}`);
  }, [movie.id, router]);

  const handleLayout = useCallback((e: LayoutChangeEvent) => {
    const { width, height } = e.nativeEvent.layout;
    if (width > 0 && height > 0) setCardSize({ width, height });
  }, []);

  return (
    <View style={[cardStyles.card, style]} onLayout={handleLayout}>
      {/* Video or poster background */}
      {showVideo ? (
        <YoutubeIframe
          videoId={videoKey!}
          width={cardSize.width}
          height={cardSize.height}
          play
          mute
          initialPlayerParams={{ controls: 0, rel: 0, modestbranding: 1, loop: 1, playlist: videoKey! }}
          webViewStyle={{ backgroundColor: '#000' }}
          webViewProps={{
            userAgent: 'Mozilla/5.0 (iPhone; CPU iPhone OS 17_0 like Mac OS X) AppleWebKit/605.1.15 (KHTML, like Gecko) Version/17.0 Mobile/15E148 Safari/604.1',
            allowsInlineMediaPlayback: true,
            mediaPlaybackRequiresUserAction: false,
          }}
        />
      ) : (
        <Image
          source={movie.posterPath ? { uri: `${TMDB}/w500${movie.posterPath}` } : undefined}
          style={StyleSheet.absoluteFill}
          contentFit="cover"
          transition={200}
          placeholder={{ blurhash: 'L00000fQfQfQfQfQfQfQfQfQfQfQ' }}
        />
      )}

      <LinearGradient
        colors={['transparent', 'rgba(0,0,0,0.75)']}
        locations={[0.5, 1]}
        style={StyleSheet.absoluteFill}
        pointerEvents="none"
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
      <View style={cardStyles.info} pointerEvents="none">
        <Text style={cardStyles.title} numberOfLines={2}>{movie.title}</Text>
        <Text style={cardStyles.year}>{movie.year}</Text>
      </View>

      {/* Full-area tap target — sits on top of WebView to capture all taps */}
      <TouchableOpacity
        style={StyleSheet.absoluteFill}
        activeOpacity={0}
        onPress={handlePress}
        onLongPress={() => { hapticSuccess(); onSave(movie); }}
        delayLongPress={400}
      />
    </View>
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

type MovieItem = { movie: Movie; isVideo: boolean };

type CardHandlers = {
  savedIds: Set<number>;
  onSave: (m: Movie) => void;
};

type Template = {
  count: number;
  render: (items: MovieItem[], h: CardHandlers) => React.ReactElement;
};

const card = (item: MovieItem, h: CardHandlers, style: object) => (
  <MosaicCard
    key={item.movie.id}
    movie={item.movie}
    style={style}
    isVideoTile={item.isVideo}
    savedIds={h.savedIds}
    onSave={h.onSave}
  />
);

const TEMPLATES: Template[] = [
  // T0 ── Large left (4u) + 2 stacked right (2u) ── 3 movies
  {
    count: 3,
    render: (items, h) => (
      <View style={{ flexDirection: 'row', gap: GAP, height: U * 5.5 }}>
        {card(items[0], h, { flex: 4 })}
        <View style={{ flex: 2, gap: GAP }}>
          {card(items[1], h, { flex: 1 })}
          {card(items[2], h, { flex: 1 })}
        </View>
      </View>
    ),
  },
  // T1 ── 3 equal thirds ── 3 movies
  {
    count: 3,
    render: (items, h) => (
      <View style={{ flexDirection: 'row', gap: GAP, height: U * 3.5 }}>
        {card(items[0], h, { flex: 2 })}
        {card(items[1], h, { flex: 2 })}
        {card(items[2], h, { flex: 2 })}
      </View>
    ),
  },
  // T2 ── Full-width banner ── 1 movie
  {
    count: 1,
    render: (items, h) => (
      <View style={{ height: U * 2.2 }}>
        {card(items[0], h, { flex: 1 })}
      </View>
    ),
  },
  // T3 ── Wide left (4u) + narrow right (2u) ── 2 movies
  {
    count: 2,
    render: (items, h) => (
      <View style={{ flexDirection: 'row', gap: GAP, height: U * 4 }}>
        {card(items[0], h, { flex: 4 })}
        {card(items[1], h, { flex: 2 })}
      </View>
    ),
  },
  // T4 ── 2 stacked left (2u) + large right (4u) ── 3 movies
  {
    count: 3,
    render: (items, h) => (
      <View style={{ flexDirection: 'row', gap: GAP, height: U * 5 }}>
        <View style={{ flex: 2, gap: GAP }}>
          {card(items[0], h, { flex: 1 })}
          {card(items[1], h, { flex: 1 })}
        </View>
        {card(items[2], h, { flex: 4 })}
      </View>
    ),
  },
  // T5 ── Narrow left (2u) + wide right (4u) ── 2 movies
  {
    count: 2,
    render: (items, h) => (
      <View style={{ flexDirection: 'row', gap: GAP, height: U * 3.5 }}>
        {card(items[0], h, { flex: 2 })}
        {card(items[1], h, { flex: 4 })}
      </View>
    ),
  },
  // T6 ── 4 movies: top row 3+3, bottom row 2+4
  {
    count: 4,
    render: (items, h) => (
      <View style={{ gap: GAP }}>
        <View style={{ flexDirection: 'row', gap: GAP, height: U * 3 }}>
          {card(items[0], h, { flex: 3 })}
          {card(items[1], h, { flex: 3 })}
        </View>
        <View style={{ flexDirection: 'row', gap: GAP, height: U * 3.5 }}>
          {card(items[2], h, { flex: 2 })}
          {card(items[3], h, { flex: 4 })}
        </View>
      </View>
    ),
  },
  // T7 ── Tall thin left (2u) + 2 stacked right (4u wide) ── 3 movies
  {
    count: 3,
    render: (items, h) => (
      <View style={{ flexDirection: 'row', gap: GAP, height: U * 6 }}>
        {card(items[0], h, { flex: 2 })}
        <View style={{ flex: 4, gap: GAP }}>
          {card(items[1], h, { flex: 3 })}
          {card(items[2], h, { flex: 2 })}
        </View>
      </View>
    ),
  },
  // T8 ── 2 equal halves (3u + 3u), tall ── 2 movies
  {
    count: 2,
    render: (items, h) => (
      <View style={{ flexDirection: 'row', gap: GAP, height: U * 4.5 }}>
        {card(items[0], h, { flex: 3 })}
        {card(items[1], h, { flex: 3 })}
      </View>
    ),
  },
  // T9 ── 4 movies: top 4+2, bottom 3+3
  {
    count: 4,
    render: (items, h) => (
      <View style={{ gap: GAP }}>
        <View style={{ flexDirection: 'row', gap: GAP, height: U * 3.8 }}>
          {card(items[0], h, { flex: 4 })}
          {card(items[1], h, { flex: 2 })}
        </View>
        <View style={{ flexDirection: 'row', gap: GAP, height: U * 3 }}>
          {card(items[2], h, { flex: 3 })}
          {card(items[3], h, { flex: 3 })}
        </View>
      </View>
    ),
  },
];

const SEQUENCE = [0, 6, 1, 3, 7, 2, 4, 9, 5, 8, 0, 5, 6, 3, 1, 7, 4, 2, 8, 9];

function groupMovies(movies: Movie[]): { tIdx: number; items: MovieItem[] }[] {
  const groups: { tIdx: number; items: MovieItem[] }[] = [];
  let i = 0;
  let seqPos = 0;
  while (i < movies.length) {
    const tIdx = SEQUENCE[seqPos % SEQUENCE.length];
    const count = TEMPLATES[tIdx].count;
    if (i + count > movies.length) break;
    groups.push({
      tIdx,
      // Every 3rd movie globally (index 0, 3, 6, …) is a video tile
      items: movies.slice(i, i + count).map((movie, j) => ({
        movie,
        isVideo: (i + j) % 3 === 0,
      })),
    });
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

  const handlers: CardHandlers = { savedIds, onSave: handleSave };

  return (
    <View style={styles.container}>
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
            {TEMPLATES[g.tIdx].render(g.items, handlers)}
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
