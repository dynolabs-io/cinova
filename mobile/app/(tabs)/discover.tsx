/**
 * Discover screen — Mosaic / Treemap Layout
 *
 * Poster-only movies cycle through template layouts (multi-tile rows).
 * Movies with a trailer get a dedicated video row sized to the video's
 * native aspect ratio — zero black bars anywhere.
 *
 *   Landscape trailer (16:9) → flex:4 tile,  height = tileWidth × (9/16)
 *   Portrait  trailer (9:16) → flex:2 tile,  height = tileWidth × (16/9)
 *
 * Each video row also has one companion poster tile at the same height.
 * A video row is inserted every VIDEO_EVERY poster template groups.
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
import WebView from 'react-native-webview';
import CinovaScore from '../../components/ui/CinovaScore';
import { getDiscoverMosaicFeed, saveTitle } from '../../services/api';
import { useAppStore } from '../../store/useAppStore';
import { hapticSuccess } from '../../services/haptics';
import { Colors, Typography, Spacing, Radius } from '../../constants/theme';
import type { Movie } from '../../types';

const { width: SCREEN_WIDTH } = Dimensions.get('window');
const GAP = 1;
const U = (SCREEN_WIDTH - GAP * 5) / 6; // 6-column grid unit
const TMDB = 'https://image.tmdb.org/t/p';

// Video tile pixel dimensions (2-tile row with 1 gap → each unit = (SCREEN_WIDTH-GAP)/6)
const TILE_UNIT = (SCREEN_WIDTH - GAP) / 6;
const LSCAPE_VIDEO_W = Math.round(4 * TILE_UNIT); // flex:4 tile in landscape row
const LSCAPE_VIDEO_H = Math.round(LSCAPE_VIDEO_W * (9 / 16)); // exact 16:9
const PORT_VIDEO_W   = Math.round(2 * TILE_UNIT); // flex:2 tile in portrait row
const PORT_VIDEO_H   = Math.round(PORT_VIDEO_W * (16 / 9));   // exact 9:16

// ── MosaicCard ────────────────────────────────────────────────────────────────

interface MosaicCardProps {
  movie: Movie;
  style?: object;
  videoKey?: string;  // if set, renders WebView iframe sized to videoW×videoH exactly
  videoW?: number;
  videoH?: number;
  savedIds: Set<number>;
  onSave: (m: Movie) => void;
}

function MosaicCard({ movie, style, videoKey, videoW, videoH, savedIds, onSave }: MosaicCardProps) {
  const router = useRouter();
  const [videoFailed, setVideoFailed] = useState(false);
  const showVideo = !!videoKey && !!videoW && !!videoH && !videoFailed;

  const handlePress = useCallback(() => {
    router.push(`/movie/${movie.id}`);
  }, [movie.id, router]);

  return (
    <View style={[cardStyles.card, style]}>
      {showVideo ? (
        <WebView
          style={{ width: videoW, height: videoH, backgroundColor: '#000' }}
          source={{ uri: `https://api.cinova.openova.io/api/v1/embed/${videoKey}?mute=1&controls=0` }}
          allowsInlineMediaPlayback
          mediaPlaybackRequiresUserAction={false}
          javaScriptEnabled
          scrollEnabled={false}
          bounces={false}
          onError={() => setVideoFailed(true)}
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

// ── Poster-only Layout Templates ──────────────────────────────────────────────

type CardHandlers = { savedIds: Set<number>; onSave: (m: Movie) => void };

type Template = {
  count: number;
  render: (movies: Movie[], h: CardHandlers) => React.ReactElement;
};

const c = (movie: Movie, h: CardHandlers, style: object) => (
  <MosaicCard key={movie.id} movie={movie} style={style} savedIds={h.savedIds} onSave={h.onSave} />
);

const TEMPLATES: Template[] = [
  // T0 — Large left (4u) + 2 stacked right (2u) — 3 movies
  {
    count: 3,
    render: (ms, h) => (
      <View style={{ flexDirection: 'row', gap: GAP, height: U * 5.5 }}>
        {c(ms[0], h, { flex: 4 })}
        <View style={{ flex: 2, gap: GAP }}>
          {c(ms[1], h, { flex: 1 })}
          {c(ms[2], h, { flex: 1 })}
        </View>
      </View>
    ),
  },
  // T1 — 3 equal thirds — 3 movies
  {
    count: 3,
    render: (ms, h) => (
      <View style={{ flexDirection: 'row', gap: GAP, height: U * 3.5 }}>
        {c(ms[0], h, { flex: 2 })}
        {c(ms[1], h, { flex: 2 })}
        {c(ms[2], h, { flex: 2 })}
      </View>
    ),
  },
  // T2 — Full-width banner — 1 movie
  {
    count: 1,
    render: (ms, h) => (
      <View style={{ height: U * 2.2 }}>
        {c(ms[0], h, { flex: 1 })}
      </View>
    ),
  },
  // T3 — Wide left (4u) + narrow right (2u) — 2 movies
  {
    count: 2,
    render: (ms, h) => (
      <View style={{ flexDirection: 'row', gap: GAP, height: U * 4 }}>
        {c(ms[0], h, { flex: 4 })}
        {c(ms[1], h, { flex: 2 })}
      </View>
    ),
  },
  // T4 — 2 stacked left (2u) + large right (4u) — 3 movies
  {
    count: 3,
    render: (ms, h) => (
      <View style={{ flexDirection: 'row', gap: GAP, height: U * 5 }}>
        <View style={{ flex: 2, gap: GAP }}>
          {c(ms[0], h, { flex: 1 })}
          {c(ms[1], h, { flex: 1 })}
        </View>
        {c(ms[2], h, { flex: 4 })}
      </View>
    ),
  },
  // T5 — Narrow left (2u) + wide right (4u) — 2 movies
  {
    count: 2,
    render: (ms, h) => (
      <View style={{ flexDirection: 'row', gap: GAP, height: U * 3.5 }}>
        {c(ms[0], h, { flex: 2 })}
        {c(ms[1], h, { flex: 4 })}
      </View>
    ),
  },
  // T6 — 4 movies: top row 3+3, bottom row 2+4
  {
    count: 4,
    render: (ms, h) => (
      <View style={{ gap: GAP }}>
        <View style={{ flexDirection: 'row', gap: GAP, height: U * 3 }}>
          {c(ms[0], h, { flex: 3 })}
          {c(ms[1], h, { flex: 3 })}
        </View>
        <View style={{ flexDirection: 'row', gap: GAP, height: U * 3.5 }}>
          {c(ms[2], h, { flex: 2 })}
          {c(ms[3], h, { flex: 4 })}
        </View>
      </View>
    ),
  },
  // T7 — Tall thin left (2u) + 2 stacked right (4u wide) — 3 movies
  {
    count: 3,
    render: (ms, h) => (
      <View style={{ flexDirection: 'row', gap: GAP, height: U * 6 }}>
        {c(ms[0], h, { flex: 2 })}
        <View style={{ flex: 4, gap: GAP }}>
          {c(ms[1], h, { flex: 3 })}
          {c(ms[2], h, { flex: 2 })}
        </View>
      </View>
    ),
  },
  // T8 — 2 equal halves — 2 movies
  {
    count: 2,
    render: (ms, h) => (
      <View style={{ flexDirection: 'row', gap: GAP, height: U * 4.5 }}>
        {c(ms[0], h, { flex: 3 })}
        {c(ms[1], h, { flex: 3 })}
      </View>
    ),
  },
  // T9 — 4 movies: top 4+2, bottom 3+3
  {
    count: 4,
    render: (ms, h) => (
      <View style={{ gap: GAP }}>
        <View style={{ flexDirection: 'row', gap: GAP, height: U * 3.8 }}>
          {c(ms[0], h, { flex: 4 })}
          {c(ms[1], h, { flex: 2 })}
        </View>
        <View style={{ flexDirection: 'row', gap: GAP, height: U * 3 }}>
          {c(ms[2], h, { flex: 3 })}
          {c(ms[3], h, { flex: 3 })}
        </View>
      </View>
    ),
  },
];

const SEQUENCE = [0, 6, 1, 3, 7, 2, 4, 9, 5, 8, 0, 5, 6, 3, 1, 7, 4, 2, 8, 9];

// ── Grouping ──────────────────────────────────────────────────────────────────

type Group =
  | { type: 'video_landscape'; videoMovie: Movie; companion: Movie }
  | { type: 'video_portrait';  videoMovie: Movie; companion: Movie }
  | { type: 'poster'; tIdx: number; movies: Movie[] };

function getVideoKey(movie: Movie): string | undefined {
  const vk = movie.verticalTrailerYoutubeKey;
  if (vk && vk !== 'NOT_FOUND') return vk;
  return movie.trailerYoutubeKey || undefined;
}

function isPortraitVideo(movie: Movie): boolean {
  const vk = movie.verticalTrailerYoutubeKey;
  return !!(vk && vk !== 'NOT_FOUND');
}

// After this many template groups without a video row, insert one if the
// current movie has a video key (and a next movie exists as companion).
const VIDEO_EVERY = 3;

function groupMovies(movies: Movie[]): Group[] {
  const groups: Group[] = [];
  let i = 0;
  let seqPos = 0;
  let tGroupsSinceVideo = 0;

  while (i < movies.length) {
    const movie = movies[i];
    const vk = getVideoKey(movie);

    // Time for a video row? Only if current movie has a key and a companion exists.
    if (tGroupsSinceVideo >= VIDEO_EVERY && vk && i + 1 < movies.length) {
      groups.push({
        type: isPortraitVideo(movie) ? 'video_portrait' : 'video_landscape',
        videoMovie: movie,
        companion: movies[i + 1],
      });
      i += 2;
      tGroupsSinceVideo = 0;
    } else {
      // Poster template group — all movies (including video-having ones)
      const tIdx = SEQUENCE[seqPos % SEQUENCE.length];
      const count = TEMPLATES[tIdx].count;
      if (i + count > movies.length) break;
      groups.push({ type: 'poster', tIdx, movies: movies.slice(i, i + count) });
      i += count;
      seqPos++;
      tGroupsSinceVideo++;
    }
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
      queryFn: ({ pageParam = 1 }) => getDiscoverMosaicFeed(country, pageParam as number),
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
          backgroundColor: '#000',
        }}
      >
        {groups.map((g, idx) => {
          if (g.type === 'video_landscape') {
            return (
              <View key={idx} style={{ flexDirection: 'row', gap: GAP, height: LSCAPE_VIDEO_H }}>
                <MosaicCard
                  movie={g.videoMovie}
                  style={{ flex: 4 }}
                  videoKey={getVideoKey(g.videoMovie)}
                  videoW={LSCAPE_VIDEO_W}
                  videoH={LSCAPE_VIDEO_H}
                  savedIds={handlers.savedIds}
                  onSave={handlers.onSave}
                />
                <MosaicCard
                  movie={g.companion}
                  style={{ flex: 2 }}
                  savedIds={handlers.savedIds}
                  onSave={handlers.onSave}
                />
              </View>
            );
          }

          if (g.type === 'video_portrait') {
            return (
              <View key={idx} style={{ flexDirection: 'row', gap: GAP, height: PORT_VIDEO_H }}>
                <MosaicCard
                  movie={g.videoMovie}
                  style={{ flex: 2 }}
                  videoKey={getVideoKey(g.videoMovie)}
                  videoW={PORT_VIDEO_W}
                  videoH={PORT_VIDEO_H}
                  savedIds={handlers.savedIds}
                  onSave={handlers.onSave}
                />
                <MosaicCard
                  movie={g.companion}
                  style={{ flex: 4 }}
                  savedIds={handlers.savedIds}
                  onSave={handlers.onSave}
                />
              </View>
            );
          }

          // poster template group
          return (
            <View key={idx} style={{ gap: GAP }}>
              {TEMPLATES[g.tIdx].render(g.movies, handlers)}
            </View>
          );
        })}

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
