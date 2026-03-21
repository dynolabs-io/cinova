/**
 * DiscoverGridCard — masonry grid cell for the Discover tab.
 *
 * Two visual variants driven by whether the movie has a vertical trailer:
 *   "video"  — tall portrait card (3:4 ratio) with a play-trailer CTA
 *   "poster" — shorter portrait card (2:3 ratio) with poster image
 *
 * Common elements: CinovaScore badge, title, year, primary streaming icon.
 * Tap → movie detail. Long-press → save quick action.
 */

import React, { useCallback } from 'react';
import {
  View,
  Text,
  StyleSheet,
  TouchableOpacity,
  Dimensions,
} from 'react-native';
import { Image } from 'expo-image';
import { LinearGradient } from 'expo-linear-gradient';
import { useRouter } from 'expo-router';
import CinovaScore from './CinovaScore';
import { Colors, Typography, Spacing, Radius } from '../../constants/theme';
import { hapticMedium, hapticSuccess } from '../../services/haptics';
import type { Movie } from '../../types';

const { width: SCREEN_WIDTH } = Dimensions.get('window');
const COL_GAP = 10;
const SIDE_PAD = 12;
// Each column is ~half screen width minus padding/gap
const COL_WIDTH = (SCREEN_WIDTH - SIDE_PAD * 2 - COL_GAP) / 2;

const TMDB_IMAGE = 'https://image.tmdb.org/t/p';

// Tall card: 3:4 ratio — matches portrait trailer frames
export const VIDEO_CARD_HEIGHT = Math.round(COL_WIDTH * (4 / 3));
// Normal card: 2:3 ratio — standard poster
export const POSTER_CARD_HEIGHT = Math.round(COL_WIDTH * (3 / 2));

interface DiscoverGridCardProps {
  movie: Movie;
  isSaved?: boolean;
  onSave?: (movie: Movie) => void;
  onPlayTrailer?: (movie: Movie) => void;
}

export default function DiscoverGridCard({
  movie,
  isSaved = false,
  onSave,
  onPlayTrailer,
}: DiscoverGridCardProps) {
  const router = useRouter();

  const hasVertical = !!movie.verticalTrailerYoutubeKey;
  const cardHeight = hasVertical ? VIDEO_CARD_HEIGHT : POSTER_CARD_HEIGHT;

  const imageUri = hasVertical
    ? (movie.posterPath ? `${TMDB_IMAGE}/w500${movie.posterPath}` : '')
    : (movie.posterPath ? `${TMDB_IMAGE}/w342${movie.posterPath}` : '');

  const primaryProvider = movie.providers?.[0] ?? null;

  const handleTap = useCallback(() => {
    router.push(`/movie/${movie.id}`);
  }, [movie.id, router]);

  const handleLongPress = useCallback(() => {
    hapticSuccess();
    onSave?.(movie);
  }, [movie, onSave]);

  const handleTrailer = useCallback(() => {
    hapticMedium();
    onPlayTrailer?.(movie);
  }, [movie, onPlayTrailer]);

  // Single root View — required by MasonryFlashList (no Fragment wrapper)
  return (
      <TouchableOpacity
        activeOpacity={0.92}
        onPress={handleTap}
        onLongPress={handleLongPress}
        style={[styles.card, { height: cardHeight }]}
        delayLongPress={400}
      >
        {/* Poster / backdrop image */}
        <Image
          source={imageUri ? { uri: imageUri } : undefined}
          style={StyleSheet.absoluteFill}
          contentFit="cover"
          transition={250}
          placeholder={{ blurhash: 'L00000fQfQfQfQfQfQfQfQfQfQfQ' }}
        />

        {/* Gradient overlay — bottom third */}
        <LinearGradient
          colors={['transparent', 'rgba(0,0,0,0.75)', Colors.background]}
          locations={[0.45, 0.78, 1]}
          style={StyleSheet.absoluteFill}
        />

        {/* CinovaScore — top-left */}
        {movie.cinovaScore != null && (
          <View style={styles.scoreBadge}>
            <CinovaScore score={movie.cinovaScore} size="sm" />
          </View>
        )}

        {/* Play button overlay — video cards only */}
        {hasVertical && (
          <TouchableOpacity style={styles.playBtn} onPress={handleTrailer} activeOpacity={0.8}>
            <Text style={styles.playIcon}>▶</Text>
          </TouchableOpacity>
        )}

        {/* Save indicator */}
        {isSaved && (
          <View style={styles.savedBadge}>
            <Text style={styles.savedIcon}>✓</Text>
          </View>
        )}

        {/* Bottom info */}
        <View style={styles.info}>
          {/* Primary streaming provider icon */}
          {primaryProvider?.logoPath ? (
            <Image
              source={{ uri: `https://image.tmdb.org/t/p/w92${primaryProvider.logoPath}` }}
              style={styles.providerIcon}
              contentFit="contain"
            />
          ) : null}
          <Text style={styles.title} numberOfLines={2}>
            {movie.title}
          </Text>
          <Text style={styles.year}>{movie.year}</Text>
        </View>
      </TouchableOpacity>
  );
}

const styles = StyleSheet.create({
  card: {
    borderRadius: Radius.md,
    overflow: 'hidden',
    backgroundColor: Colors.card,
  },
  scoreBadge: {
    position: 'absolute',
    top: Spacing[2],
    left: Spacing[2],
    backgroundColor: 'rgba(0,0,0,0.6)',
    borderRadius: Radius.full,
    padding: 3,
  },
  playBtn: {
    position: 'absolute',
    top: '38%',
    alignSelf: 'center',
    width: 44,
    height: 44,
    borderRadius: 22,
    backgroundColor: 'rgba(0,0,0,0.65)',
    borderWidth: 2,
    borderColor: 'rgba(255,255,255,0.7)',
    justifyContent: 'center',
    alignItems: 'center',
  },
  playIcon: {
    color: '#fff',
    fontSize: 18,
    marginLeft: 3,
  },
  savedBadge: {
    position: 'absolute',
    top: Spacing[2],
    right: Spacing[2],
    width: 22,
    height: 22,
    borderRadius: 11,
    backgroundColor: Colors.primary,
    justifyContent: 'center',
    alignItems: 'center',
  },
  savedIcon: {
    color: '#fff',
    fontSize: 11,
    fontWeight: '700',
  },
  info: {
    position: 'absolute',
    bottom: 0,
    left: 0,
    right: 0,
    paddingHorizontal: Spacing[2],
    paddingBottom: Spacing[2.5],
    gap: 2,
  },
  providerIcon: {
    width: 18,
    height: 18,
    borderRadius: 3,
    marginBottom: 2,
  },
  title: {
    color: Colors.textPrimary,
    fontSize: Typography.xs,
    fontWeight: Typography.bold,
    lineHeight: Typography.xs * 1.3,
  },
  year: {
    color: Colors.textMuted,
    fontSize: 10,
  },
});
