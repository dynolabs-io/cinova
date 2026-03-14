/**
 * ReelItem — Full-screen discover reel (Instagram Reels / TikTok style)
 *
 * Fills the full viewport. Backdrop image with gradient. Right-side
 * action column. Bottom content strip.
 */

import React, { useCallback } from 'react';
import {
  View,
  Text,
  StyleSheet,
  Dimensions,
  TouchableOpacity,
  Linking,
  Platform,
} from 'react-native';
import { Image } from 'expo-image';
import { LinearGradient } from 'expo-linear-gradient';
import { useRouter } from 'expo-router';
import CinovaScore from './CinovaScore';
import StreamingBadge from './StreamingBadge';
import { Colors, Typography, Spacing, Radius } from '../../constants/theme';
import { getProviderById } from '../../constants/providers';
import { hapticSuccess, hapticMedium, hapticLight } from '../../services/haptics';
import type { Movie, WatchProvider } from '../../types';

const { width: SCREEN_WIDTH, height: SCREEN_HEIGHT } = Dimensions.get('window');
const TMDB_IMAGE = 'https://image.tmdb.org/t/p/w1280';

interface ReelItemProps {
  movie: Movie;
  onSave?: (movie: Movie) => void;
  onRate?: (movie: Movie) => void;
  onDismiss?: (movie: Movie) => void;
  isSaved?: boolean;
}

function backdropUri(path: string | null): string {
  if (!path) return '';
  return `${TMDB_IMAGE}${path}`;
}

async function watchOnProvider(provider: WatchProvider, movieId: number): Promise<void> {
  const known = getProviderById(provider.providerId);
  if (known) {
    const deepLink = known.buildDeepLink(movieId);
    const canOpen = await Linking.canOpenURL(deepLink);
    if (canOpen) {
      await Linking.openURL(deepLink);
      return;
    }
    const store =
      Platform.OS === 'ios' ? known.storeUrl.ios : known.storeUrl.android;
    await Linking.openURL(store);
    return;
  }
  if (provider.link) await Linking.openURL(provider.link);
}

export default function ReelItem({
  movie,
  onSave,
  onRate,
  onDismiss,
  isSaved = false,
}: ReelItemProps) {
  const router = useRouter();
  const primaryProvider = movie.providers?.[0] ?? null;
  const genreLabel = movie.genres.slice(0, 2).map((g) => g.name).join(' · ');
  const runtimeLabel = movie.runtime ? `${movie.runtime}m` : '';

  const handleTap = useCallback(() => {
    router.push(`/movie/${movie.id}`);
  }, [movie.id, router]);

  const handleWatch = useCallback(async () => {
    if (primaryProvider) {
      await watchOnProvider(primaryProvider, movie.tmdbId);
    }
  }, [primaryProvider, movie.tmdbId]);

  return (
    <TouchableOpacity
      activeOpacity={1}
      onPress={handleTap}
      style={styles.container}
    >
      {/* Background backdrop */}
      <Image
        source={{ uri: backdropUri(movie.backdropPath) }}
        style={styles.backdrop}
        contentFit="cover"
        transition={300}
        placeholder={{ blurhash: 'L00000fQfQfQfQfQfQfQfQfQfQfQ' }}
      />

      {/* Gradient — sides transparent, bottom 50% black fade */}
      <LinearGradient
        colors={[
          'transparent',
          'transparent',
          'rgba(0,0,0,0.4)',
          'rgba(0,0,0,0.85)',
          Colors.background,
        ]}
        locations={[0, 0.3, 0.55, 0.78, 1]}
        style={styles.gradient}
      />

      {/* CinovaScore top-right */}
      {movie.cinovaScore != null && (
        <View style={styles.scoreContainer}>
          <CinovaScore score={movie.cinovaScore} size="md" />
        </View>
      )}

      {/* Right-side action column */}
      <View style={styles.actionColumn}>
        <ActionButton
          label={isSaved ? '✓' : '+'}
          sublabel="Save"
          color={isSaved ? Colors.primary : Colors.textPrimary}
          onPress={() => { hapticSuccess(); onSave?.(movie); }}
        />
        <ActionButton
          label="★"
          sublabel="Rate"
          color={Colors.scoreMid}
          onPress={() => { hapticMedium(); onRate?.(movie); }}
        />
        <ActionButton
          label="✕"
          sublabel="Skip"
          color={Colors.textSecondary}
          onPress={() => { hapticLight(); onDismiss?.(movie); }}
        />
        {primaryProvider && (
          <ActionButton
            label="▶"
            sublabel="Watch"
            color={Colors.scoreHigh}
            onPress={handleWatch}
          />
        )}
      </View>

      {/* Bottom content */}
      <View style={[styles.bottomContent, { pointerEvents: 'box-none' }]}>
        {/* Streaming badges row */}
        {movie.providers.length > 0 && (
          <View style={styles.providerRow}>
            {movie.providers.slice(0, 4).map((p) => (
              <StreamingBadge
                key={p.providerId}
                provider={p}
                variant="icon"
                size={32}
              />
            ))}
          </View>
        )}

        {/* Title */}
        <Text style={styles.title} numberOfLines={2}>
          {movie.title}
        </Text>

        {/* Meta */}
        <Text style={styles.meta}>
          {[movie.year, genreLabel, runtimeLabel].filter(Boolean).join(' · ')}
        </Text>

        {/* AI one-liner */}
        {movie.aiDescription ? (
          <Text style={styles.aiDescription} numberOfLines={2}>
            {movie.aiDescription}
          </Text>
        ) : movie.overview ? (
          <Text style={styles.aiDescription} numberOfLines={2}>
            {movie.overview}
          </Text>
        ) : null}
      </View>
    </TouchableOpacity>
  );
}

interface ActionButtonProps {
  label: string;
  sublabel: string;
  color: string;
  onPress: () => void;
}

function ActionButton({ label, sublabel, color, onPress }: ActionButtonProps) {
  return (
    <TouchableOpacity
      onPress={onPress}
      activeOpacity={0.75}
      style={styles.actionBtn}
    >
      <View style={[styles.actionIconCircle, { borderColor: color }]}>
        <Text style={[styles.actionIcon, { color }]}>{label}</Text>
      </View>
      <Text style={styles.actionLabel}>{sublabel}</Text>
    </TouchableOpacity>
  );
}

const styles = StyleSheet.create({
  container: {
    width: SCREEN_WIDTH,
    height: SCREEN_HEIGHT,
    backgroundColor: Colors.background,
  },
  backdrop: {
    position: 'absolute',
    width: SCREEN_WIDTH,
    height: SCREEN_HEIGHT,
  },
  gradient: {
    position: 'absolute',
    left: 0,
    right: 0,
    bottom: 0,
    height: SCREEN_HEIGHT * 0.7,
  },
  scoreContainer: {
    position: 'absolute',
    top: 60,
    right: Spacing[4],
    backgroundColor: 'rgba(0,0,0,0.6)',
    borderRadius: Radius.full,
    padding: Spacing[1.5],
  },
  actionColumn: {
    position: 'absolute',
    right: Spacing[3],
    bottom: 180,
    alignItems: 'center',
    gap: Spacing[5],
  },
  actionBtn: {
    alignItems: 'center',
    gap: Spacing[1],
  },
  actionIconCircle: {
    width: 48,
    height: 48,
    borderRadius: Radius.full,
    borderWidth: 2,
    backgroundColor: 'rgba(0,0,0,0.5)',
    justifyContent: 'center',
    alignItems: 'center',
  },
  actionIcon: {
    fontSize: Typography.lg,
    textAlign: 'center',
  },
  actionLabel: {
    color: Colors.textSecondary,
    fontSize: Typography.xs,
    fontWeight: Typography.medium,
  },
  bottomContent: {
    position: 'absolute',
    bottom: 0,
    left: 0,
    right: 72,
    paddingHorizontal: Spacing[4],
    paddingBottom: Spacing[10],
  },
  providerRow: {
    flexDirection: 'row',
    gap: Spacing[2],
    marginBottom: Spacing[3],
  },
  title: {
    color: Colors.textPrimary,
    fontSize: Typography['2xl'],
    fontWeight: Typography.black,
    letterSpacing: Typography.tighter,
    lineHeight: Typography['2xl'] * 1.15,
    marginBottom: Spacing[1.5],
  },
  meta: {
    color: Colors.textSecondary,
    fontSize: Typography.sm,
    fontWeight: Typography.medium,
    marginBottom: Spacing[2],
  },
  aiDescription: {
    color: Colors.textSecondary,
    fontSize: Typography.sm,
    fontStyle: 'italic',
    lineHeight: Typography.sm * 1.5,
  },
});
