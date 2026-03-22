/**
 * ReelItem — Full-screen vertical reel (TikTok / Instagram Reels style)
 *
 * When isActive=true and the movie has an embeddable verticalTrailerYoutubeKey,
 * plays it full-screen using react-native-youtube-iframe (handles embed auth).
 * Center-crop: scales video to fill SCREEN_HEIGHT, clips overflowing width.
 * Otherwise shows the backdrop image. UI overlay is always on top.
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
import YoutubeIframe from 'react-native-youtube-iframe';
import CinovaScore from './CinovaScore';
import StreamingBadge from './StreamingBadge';
import { Colors, Typography, Spacing, Radius } from '../../constants/theme';
import { getProviderById } from '../../constants/providers';
import { hapticSuccess, hapticMedium } from '../../services/haptics';
import type { Movie, WatchProvider } from '../../types';

const { width: SCREEN_WIDTH, height: SCREEN_HEIGHT } = Dimensions.get('window');
const TMDB_IMAGE = 'https://image.tmdb.org/t/p/w1280';

// Vertical trailer is 9:16 — scale to fill SCREEN_HEIGHT, crop sides
const VIDEO_W = Math.ceil(SCREEN_HEIGHT * (9 / 16));
const VIDEO_LEFT = -Math.floor(Math.max(0, VIDEO_W - SCREEN_WIDTH) / 2);

interface ReelItemProps {
  movie: Movie;
  isActive?: boolean;
  onSave?: (movie: Movie) => void;
  onRate?: (movie: Movie) => void;
  onDismiss?: (movie: Movie) => void;
  isSaved?: boolean;
  userRating?: number;
}

async function watchOnProvider(provider: WatchProvider, movieId: number): Promise<void> {
  const known = getProviderById(provider.providerId);
  if (known) {
    const deepLink = known.buildDeepLink(movieId);
    const canOpen = await Linking.canOpenURL(deepLink);
    if (canOpen) { await Linking.openURL(deepLink); return; }
    const store = Platform.OS === 'ios' ? known.storeUrl.ios : known.storeUrl.android;
    await Linking.openURL(store);
    return;
  }
  if (provider.link) await Linking.openURL(provider.link);
}

export default function ReelItem({
  movie,
  isActive = false,
  onSave,
  onRate,
  onDismiss,
  isSaved = false,
  userRating,
}: ReelItemProps) {
  const router = useRouter();
  const primaryProvider = movie.providers?.[0] ?? null;
  const genreLabel = movie.genres.slice(0, 2).map((g) => g.name).join(' · ');
  const runtimeLabel = movie.runtime ? `${movie.runtime}m` : '';

  // Only use verified-embeddable vertical trailers
  const videoKey = movie.verticalTrailerYoutubeKey || null;
  const showVideo = isActive && !!videoKey;

  const handleTap = useCallback(() => {
    router.push(`/movie/${movie.id}`);
  }, [movie.id, router]);

  const handleWatch = useCallback(async () => {
    if (primaryProvider) await watchOnProvider(primaryProvider, movie.tmdbId);
  }, [primaryProvider, movie.tmdbId]);

  return (
    <View style={styles.container}>
      {/* Background: center-cropped YouTube video or backdrop image */}
      {showVideo ? (
        // Clip container to screen width — the YoutubeIframe is wider (VIDEO_W)
        // and offset left so the video is centered. overflow:hidden crops the sides.
        <View style={styles.videoClip}>
          <View style={[styles.videoInner, { left: VIDEO_LEFT }]}>
            <YoutubeIframe
              videoId={videoKey!}
              height={SCREEN_HEIGHT}
              width={VIDEO_W}
              play={isActive}
              mute={false}
              initialPlayerParams={{
                controls: 1,
                rel: 0,
                modestbranding: 1,
                loop: 1,
                playlist: videoKey!,
              }}
              webViewStyle={{ backgroundColor: '#000' }}
              webViewProps={{
                userAgent: 'Mozilla/5.0 (iPhone; CPU iPhone OS 17_0 like Mac OS X) AppleWebKit/605.1.15 (KHTML, like Gecko) Version/17.0 Mobile/15E148 Safari/604.1',
                allowsInlineMediaPlayback: true,
                mediaPlaybackRequiresUserAction: false,
              }}
            />
          </View>
        </View>
      ) : (
        <Image
          source={movie.backdropPath ? { uri: `${TMDB_IMAGE}${movie.backdropPath}` } : undefined}
          style={styles.backdrop}
          contentFit="cover"
          transition={300}
          placeholder={{ blurhash: 'L00000fQfQfQfQfQfQfQfQfQfQfQ' }}
        />
      )}

      {/* Gradient overlay */}
      <LinearGradient
        colors={['transparent', 'transparent', 'rgba(0,0,0,0.5)', 'rgba(0,0,0,0.88)', Colors.background]}
        locations={[0, 0.35, 0.6, 0.8, 1]}
        style={styles.gradient}
        pointerEvents="none"
      />

      {/* Tap target — whole screen navigates to detail */}
      <TouchableOpacity
        activeOpacity={1}
        onPress={handleTap}
        style={StyleSheet.absoluteFill}
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
          label={userRating ? `${userRating}★` : '★'}
          sublabel={userRating ? 'Rated' : 'Rate'}
          color={userRating ? Colors.scoreMid : Colors.textPrimary}
          onPress={() => { hapticMedium(); onRate?.(movie); }}
        />
        {primaryProvider && (
          <ActionButton
            label="↗"
            sublabel={primaryProvider.providerName.split(' ')[0]}
            color={Colors.scoreHigh}
            onPress={handleWatch}
          />
        )}
      </View>

      {/* Bottom content */}
      <View style={styles.bottomContent} pointerEvents="box-none">
        {movie.providers.length > 0 && (
          <View style={styles.providerRow}>
            {movie.providers.slice(0, 4).map((p, i) => (
              <StreamingBadge key={`${p.providerId}-${i}`} provider={p} variant="icon" size={32} />
            ))}
          </View>
        )}
        <Text style={styles.title} numberOfLines={2}>{movie.title}</Text>
        <Text style={styles.meta}>
          {[movie.year, genreLabel, runtimeLabel].filter(Boolean).join(' · ')}
        </Text>
        {(movie.cinovaSynopsis ?? movie.aiDescription ?? movie.overview) ? (
          <Text style={styles.synopsis} numberOfLines={2}>
            {movie.cinovaSynopsis ?? movie.aiDescription ?? movie.overview}
          </Text>
        ) : null}
        {movie.awards && movie.awards.filter((a) => !a.isNomination).length > 0 && (
          <View style={styles.awardChip}>
            <Text style={styles.awardChipText}>
              🏆 {movie.awards.filter((a) => !a.isNomination).length} Award Win{movie.awards.filter((a) => !a.isNomination).length !== 1 ? 's' : ''}
            </Text>
          </View>
        )}
      </View>
    </View>
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
    <TouchableOpacity onPress={onPress} activeOpacity={0.75} style={styles.actionBtn}>
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
    backgroundColor: '#000',
  },
  // Clips the wider-than-screen YoutubeIframe to screen width
  videoClip: {
    position: 'absolute',
    top: 0, left: 0,
    width: SCREEN_WIDTH,
    height: SCREEN_HEIGHT,
    overflow: 'hidden',
    backgroundColor: '#000',
  },
  // Inner view is VIDEO_W wide, offset left so video is centred
  videoInner: {
    position: 'absolute',
    top: 0,
    width: VIDEO_W,
    height: SCREEN_HEIGHT,
  },
  backdrop: {
    position: 'absolute',
    width: SCREEN_WIDTH,
    height: SCREEN_HEIGHT,
  },
  gradient: {
    position: 'absolute',
    left: 0, right: 0, bottom: 0,
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
  synopsis: {
    color: Colors.textSecondary,
    fontSize: Typography.sm,
    fontStyle: 'italic',
    lineHeight: Typography.sm * 1.5,
  },
  awardChip: {
    alignSelf: 'flex-start',
    marginTop: Spacing[2],
    backgroundColor: 'rgba(255, 200, 0, 0.15)',
    borderRadius: Radius.full,
    paddingHorizontal: Spacing[3],
    paddingVertical: Spacing[1],
    borderWidth: 1,
    borderColor: 'rgba(255, 200, 0, 0.4)',
  },
  awardChipText: {
    color: '#FFD700',
    fontSize: Typography.xs,
    fontWeight: Typography.semibold,
  },
});
