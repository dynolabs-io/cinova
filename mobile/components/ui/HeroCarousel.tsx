/**
 * HeroCarousel — Auto-scrolling full-width hero banner
 *
 * Displays trending movies with backdrop images, gradient overlay,
 * movie info, and two CTA buttons. Auto-advances every 5 seconds.
 */

import React, { useRef, useEffect, useState, useCallback } from 'react';
import {
  View,
  Text,
  StyleSheet,
  Dimensions,
  FlatList,
  TouchableOpacity,
  ViewToken,
} from 'react-native';
import { Image } from 'expo-image';
import { LinearGradient } from 'expo-linear-gradient';
import { useRouter } from 'expo-router';
import { Colors, Typography, Spacing, Radius, Shadows } from '../../constants/theme';
import type { Movie } from '../../types';

const { width: SCREEN_WIDTH, height: SCREEN_HEIGHT } = Dimensions.get('window');
const HERO_HEIGHT = SCREEN_HEIGHT * 0.62;
const TMDB_IMAGE = 'https://image.tmdb.org/t/p/w1280';
const AUTO_SCROLL_INTERVAL = 5000;

interface HeroCarouselProps {
  movies: Movie[];
  onSave?: (movie: Movie) => void;
}

function backdropUri(path: string | null): string {
  if (!path) return '';
  return `${TMDB_IMAGE}${path}`;
}

export default function HeroCarousel({ movies, onSave }: HeroCarouselProps) {
  const router = useRouter();
  const flatListRef = useRef<FlatList<Movie>>(null);
  const [currentIndex, setCurrentIndex] = useState(0);
  const timerRef = useRef<ReturnType<typeof setInterval> | null>(null);

  const startAutoScroll = useCallback(() => {
    if (timerRef.current) clearInterval(timerRef.current);
    timerRef.current = setInterval(() => {
      setCurrentIndex((prev) => {
        const next = (prev + 1) % movies.length;
        flatListRef.current?.scrollToIndex({ index: next, animated: true });
        return next;
      });
    }, AUTO_SCROLL_INTERVAL);
  }, [movies.length]);

  useEffect(() => {
    if (movies.length > 1) startAutoScroll();
    return () => {
      if (timerRef.current) clearInterval(timerRef.current);
    };
  }, [movies.length, startAutoScroll]);

  const onViewableItemsChanged = useRef(
    ({ viewableItems }: { viewableItems: ViewToken[] }) => {
      if (viewableItems.length > 0 && viewableItems[0].index != null) {
        setCurrentIndex(viewableItems[0].index);
      }
    }
  ).current;

  const viewabilityConfig = useRef({ itemVisiblePercentThreshold: 60 }).current;

  const handleScrollBeginDrag = () => {
    if (timerRef.current) clearInterval(timerRef.current);
  };

  const handleScrollEndDrag = () => {
    if (movies.length > 1) startAutoScroll();
  };

  const renderItem = ({ item }: { item: Movie }) => (
    <HeroItem
      movie={item}
      onWatchNow={() => router.push(`/movie/${item.id}`)}
      onSave={() => onSave?.(item)}
    />
  );

  const getItemLayout = (_: unknown, index: number) => ({
    length: SCREEN_WIDTH,
    offset: SCREEN_WIDTH * index,
    index,
  });

  if (!movies.length) return null;

  return (
    <View style={styles.container}>
      <FlatList
        ref={flatListRef}
        data={movies}
        renderItem={renderItem}
        keyExtractor={(item) => String(item.id)}
        horizontal
        pagingEnabled
        showsHorizontalScrollIndicator={false}
        onViewableItemsChanged={onViewableItemsChanged}
        viewabilityConfig={viewabilityConfig}
        onScrollBeginDrag={handleScrollBeginDrag}
        onScrollEndDrag={handleScrollEndDrag}
        getItemLayout={getItemLayout}
        initialNumToRender={2}
        maxToRenderPerBatch={3}
        removeClippedSubviews
      />

      {/* Dot pagination */}
      <View style={styles.dots}>
        {movies.map((_, i) => (
          <View
            key={i}
            style={[
              styles.dot,
              i === currentIndex ? styles.dotActive : styles.dotInactive,
            ]}
          />
        ))}
      </View>
    </View>
  );
}

// ── Individual hero item ──────────────────────────────────────────────────────

interface HeroItemProps {
  movie: Movie;
  onWatchNow: () => void;
  onSave: () => void;
}

function HeroItem({ movie, onWatchNow, onSave }: HeroItemProps) {
  const genreLabel = movie.genres.slice(0, 2).map((g) => g.name).join(' · ');

  return (
    <View style={styles.item}>
      <Image
        source={{ uri: backdropUri(movie.backdropPath) }}
        style={styles.backdrop}
        contentFit="cover"
        transition={{ duration: 400, effect: 'cross-dissolve' }}
        placeholder={{ blurhash: 'L00000fQfQfQfQfQfQfQfQfQfQfQ' }}
      />

      {/* Gradient overlay */}
      <LinearGradient
        colors={['transparent', 'rgba(0,0,0,0.3)', 'rgba(0,0,0,0.9)', Colors.background]}
        locations={[0, 0.4, 0.75, 1]}
        style={styles.gradient}
      />

      {/* Content */}
      <View style={styles.content}>
        {movie.tagline ? (
          <Text style={styles.tagline} numberOfLines={1}>
            {movie.tagline}
          </Text>
        ) : null}

        <Text style={styles.title} numberOfLines={2}>
          {movie.title}
        </Text>

        <View style={styles.pills}>
          <View style={styles.pill}>
            <Text style={styles.pillText}>{movie.year}</Text>
          </View>
          {movie.runtime != null && (
            <View style={styles.pill}>
              <Text style={styles.pillText}>{movie.runtime}m</Text>
            </View>
          )}
          {genreLabel ? (
            <View style={styles.pill}>
              <Text style={styles.pillText}>{genreLabel}</Text>
            </View>
          ) : null}
        </View>

        <View style={styles.buttons}>
          <TouchableOpacity
            style={styles.btnWatchNow}
            onPress={onWatchNow}
            activeOpacity={0.85}
          >
            <Text style={styles.btnWatchNowText}>▶  Watch Now</Text>
          </TouchableOpacity>

          <TouchableOpacity
            style={styles.btnSave}
            onPress={onSave}
            activeOpacity={0.85}
          >
            <Text style={styles.btnSaveText}>＋ Save</Text>
          </TouchableOpacity>
        </View>
      </View>
    </View>
  );
}

const styles = StyleSheet.create({
  container: {
    width: SCREEN_WIDTH,
    height: HERO_HEIGHT,
  },
  item: {
    width: SCREEN_WIDTH,
    height: HERO_HEIGHT,
  },
  backdrop: {
    width: SCREEN_WIDTH,
    height: HERO_HEIGHT,
    position: 'absolute',
    top: 0,
    left: 0,
  },
  gradient: {
    position: 'absolute',
    left: 0,
    right: 0,
    bottom: 0,
    height: HERO_HEIGHT * 0.75,
  },
  content: {
    position: 'absolute',
    bottom: 0,
    left: 0,
    right: 0,
    paddingHorizontal: Spacing[4],
    paddingBottom: Spacing[5],
  },
  tagline: {
    color: Colors.textSecondary,
    fontSize: Typography.sm,
    fontWeight: Typography.medium,
    letterSpacing: Typography.wider,
    textTransform: 'uppercase',
    marginBottom: Spacing[1],
  },
  title: {
    color: Colors.textPrimary,
    fontSize: Typography['3xl'],
    fontWeight: Typography.black,
    letterSpacing: Typography.tighter,
    lineHeight: Typography['3xl'] * Typography.tight,
    marginBottom: Spacing[3],
  },
  pills: {
    flexDirection: 'row',
    flexWrap: 'wrap',
    gap: Spacing[1.5],
    marginBottom: Spacing[4],
  },
  pill: {
    backgroundColor: Colors.overlayLight,
    borderRadius: Radius.full,
    paddingHorizontal: Spacing[3],
    paddingVertical: Spacing[1],
    borderWidth: 1,
    borderColor: Colors.border,
  },
  pillText: {
    color: Colors.textSecondary,
    fontSize: Typography.xs,
    fontWeight: Typography.medium,
  },
  buttons: {
    flexDirection: 'row',
    gap: Spacing[3],
  },
  btnWatchNow: {
    flex: 1,
    backgroundColor: Colors.primary,
    borderRadius: Radius.md,
    paddingVertical: Spacing[3],
    alignItems: 'center',
    ...Shadows.glow,
  },
  btnWatchNowText: {
    color: Colors.textPrimary,
    fontSize: Typography.base,
    fontWeight: Typography.bold,
  },
  btnSave: {
    flex: 1,
    backgroundColor: 'transparent',
    borderRadius: Radius.md,
    paddingVertical: Spacing[3],
    alignItems: 'center',
    borderWidth: 1.5,
    borderColor: Colors.textPrimary,
  },
  btnSaveText: {
    color: Colors.textPrimary,
    fontSize: Typography.base,
    fontWeight: Typography.semibold,
  },
  dots: {
    position: 'absolute',
    bottom: Spacing[3],
    alignSelf: 'center',
    flexDirection: 'row',
    gap: Spacing[1],
  },
  dot: {
    height: 3,
    borderRadius: Radius.full,
  },
  dotActive: {
    width: 20,
    backgroundColor: Colors.primary,
  },
  dotInactive: {
    width: 6,
    backgroundColor: Colors.textMuted,
  },
});
