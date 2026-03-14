/**
 * MovieCard — Reusable movie/show card
 *
 * Three sizes:
 *   sm  — 100 px wide poster only + CinovaScore badge
 *   md  — 140 px wide poster + title + streaming badge
 *   lg  — full-width card with backdrop + gradient + all info
 */

import React, { useCallback } from 'react';
import {
  TouchableOpacity,
  View,
  Text,
  StyleSheet,
  Dimensions,
} from 'react-native';
import { Image } from 'expo-image';
import { LinearGradient } from 'expo-linear-gradient';
import { useRouter } from 'expo-router';
import CinovaScore from './CinovaScore';
import StreamingBadge from './StreamingBadge';
import { Colors, Radius, Spacing, Typography, Shadows } from '../../constants/theme';
import type { Movie } from '../../types';

const { width: SCREEN_WIDTH } = Dimensions.get('window');
const TMDB_IMAGE = 'https://image.tmdb.org/t/p';

interface MovieCardProps {
  movie: Movie;
  size?: 'sm' | 'md' | 'lg';
  onPress?: (movie: Movie) => void;
}

const POSTER_SIZES = { sm: 100, md: 140, lg: SCREEN_WIDTH - 32 };
const POSTER_HEIGHTS = { sm: 150, md: 210, lg: 220 };

function posterUri(path: string | null, width: number): string {
  if (!path) return 'https://via.placeholder.com/300x450/141414/6B6B6B?text=No+Image';
  const w = width <= 100 ? 'w185' : width <= 200 ? 'w342' : 'w500';
  return `${TMDB_IMAGE}/${w}${path}`;
}

function backdropUri(path: string | null): string {
  if (!path) return 'https://via.placeholder.com/780x439/141414/6B6B6B?text=No+Image';
  return `${TMDB_IMAGE}/w780${path}`;
}

export default function MovieCard({ movie, size = 'md', onPress }: MovieCardProps) {
  const router = useRouter();

  const handlePress = useCallback(() => {
    if (onPress) {
      onPress(movie);
    } else {
      router.push(`/movie/${movie.id}`);
    }
  }, [movie, onPress, router]);

  const cardWidth = POSTER_SIZES[size];
  const posterHeight = POSTER_HEIGHTS[size];
  const primaryProvider = movie.providers?.[0] ?? null;

  // ── Small card ────────────────────────────────────────────────────────────
  if (size === 'sm') {
    return (
      <TouchableOpacity
        onPress={handlePress}
        activeOpacity={0.85}
        style={[styles.smCard, { width: cardWidth }]}
      >
        <Image
          source={{ uri: posterUri(movie.posterPath, cardWidth) }}
          style={[styles.smPoster, { height: posterHeight }]}
          contentFit="cover"
          transition={200}
          placeholder={{ blurhash: 'L00000fQfQfQfQfQfQfQfQfQfQfQ' }}
        />
        {movie.cinovaScore != null && (
          <View style={styles.smScoreBadge}>
            <CinovaScore score={movie.cinovaScore} size="sm" />
          </View>
        )}
      </TouchableOpacity>
    );
  }

  // ── Medium card ───────────────────────────────────────────────────────────
  if (size === 'md') {
    return (
      <TouchableOpacity
        onPress={handlePress}
        activeOpacity={0.85}
        style={[styles.mdCard, { width: cardWidth }]}
      >
        <View>
          <Image
            source={{ uri: posterUri(movie.posterPath, cardWidth) }}
            style={[styles.mdPoster, { height: posterHeight }]}
            contentFit="cover"
            transition={200}
            placeholder={{ blurhash: 'L00000fQfQfQfQfQfQfQfQfQfQfQ' }}
          />
          {movie.cinovaScore != null && (
            <View style={styles.mdScoreBadge}>
              <CinovaScore score={movie.cinovaScore} size="sm" />
            </View>
          )}
          {primaryProvider && (
            <View style={styles.mdProviderBadge}>
              <StreamingBadge provider={primaryProvider} variant="icon" size={26} />
            </View>
          )}
        </View>
        <Text style={styles.mdTitle} numberOfLines={2}>
          {movie.title}
        </Text>
        <Text style={styles.mdYear}>{movie.year}</Text>
      </TouchableOpacity>
    );
  }

  // ── Large card ────────────────────────────────────────────────────────────
  return (
    <TouchableOpacity
      onPress={handlePress}
      activeOpacity={0.85}
      style={[styles.lgCard, { width: cardWidth }]}
    >
      <Image
        source={{ uri: backdropUri(movie.backdropPath) }}
        style={[styles.lgBackdrop, { height: posterHeight }]}
        contentFit="cover"
        transition={200}
        placeholder={{ blurhash: 'L00000fQfQfQfQfQfQfQfQfQfQfQ' }}
      />
      <LinearGradient
        colors={['transparent', 'rgba(0,0,0,0.8)', Colors.background]}
        style={[styles.lgGradient, { height: posterHeight }]}
      />
      {/* Score top right */}
      {movie.cinovaScore != null && (
        <View style={styles.lgScoreBadge}>
          <CinovaScore score={movie.cinovaScore} size="sm" />
        </View>
      )}
      {/* Provider bottom left */}
      {primaryProvider && (
        <View style={styles.lgProviderBadge}>
          <StreamingBadge provider={primaryProvider} variant="icon" size={28} />
        </View>
      )}
      {/* Info overlay */}
      <View style={styles.lgInfo}>
        <Text style={styles.lgTitle} numberOfLines={2}>
          {movie.title}
        </Text>
        <View style={styles.lgMeta}>
          <Text style={styles.lgMetaText}>{movie.year}</Text>
          {movie.runtime && (
            <>
              <Text style={styles.lgDot}> · </Text>
              <Text style={styles.lgMetaText}>{movie.runtime}m</Text>
            </>
          )}
          {movie.genres.length > 0 && (
            <>
              <Text style={styles.lgDot}> · </Text>
              <Text style={styles.lgMetaText}>{movie.genres[0].name}</Text>
            </>
          )}
        </View>
      </View>
    </TouchableOpacity>
  );
}

const styles = StyleSheet.create({
  // Small
  smCard: {
    marginRight: Spacing[2],
  },
  smPoster: {
    borderRadius: Radius.base,
    backgroundColor: Colors.card,
  },
  smScoreBadge: {
    position: 'absolute',
    top: 6,
    right: 6,
    backgroundColor: 'rgba(0,0,0,0.75)',
    borderRadius: Radius.full,
    padding: 2,
  },

  // Medium
  mdCard: {
    marginRight: Spacing[2.5],
  },
  mdPoster: {
    borderRadius: Radius.md,
    backgroundColor: Colors.card,
  },
  mdScoreBadge: {
    position: 'absolute',
    top: 6,
    right: 6,
    backgroundColor: 'rgba(0,0,0,0.75)',
    borderRadius: Radius.full,
    padding: 2,
  },
  mdProviderBadge: {
    position: 'absolute',
    bottom: 6,
    left: 6,
  },
  mdTitle: {
    color: Colors.textPrimary,
    fontSize: Typography.sm,
    fontWeight: Typography.semibold,
    marginTop: Spacing[1.5],
    lineHeight: Typography.sm * Typography.snug,
  },
  mdYear: {
    color: Colors.textMuted,
    fontSize: Typography.xs,
    marginTop: 2,
  },

  // Large
  lgCard: {
    borderRadius: Radius.lg,
    overflow: 'hidden',
    backgroundColor: Colors.card,
    ...Shadows.md,
  },
  lgBackdrop: {
    width: '100%',
  },
  lgGradient: {
    position: 'absolute',
    left: 0,
    right: 0,
    bottom: 0,
  },
  lgScoreBadge: {
    position: 'absolute',
    top: 10,
    right: 10,
    backgroundColor: 'rgba(0,0,0,0.7)',
    borderRadius: Radius.full,
    padding: 4,
  },
  lgProviderBadge: {
    position: 'absolute',
    bottom: 48,
    left: 12,
  },
  lgInfo: {
    position: 'absolute',
    bottom: 0,
    left: 0,
    right: 0,
    padding: Spacing[3],
  },
  lgTitle: {
    color: Colors.textPrimary,
    fontSize: Typography.lg,
    fontWeight: Typography.bold,
    marginBottom: 4,
  },
  lgMeta: {
    flexDirection: 'row',
    alignItems: 'center',
  },
  lgMetaText: {
    color: Colors.textSecondary,
    fontSize: Typography.sm,
  },
  lgDot: {
    color: Colors.textMuted,
    fontSize: Typography.sm,
  },
});
