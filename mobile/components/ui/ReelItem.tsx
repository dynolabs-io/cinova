/**
 * ReelItem — single reel with embedded video player.
 *
 * Each ReelItem owns its own WebView. The parent only mounts WebViews for
 * items within ±1 of the active index (`shouldLoad` prop), keeping memory low.
 * Because the WebView is INSIDE the FlatList item, it scrolls naturally with
 * the list — giving the Instagram-style half-drag two-video-visible effect.
 */

import React, { useCallback, useRef, useEffect, useState } from 'react';
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
import { WebView } from 'react-native-webview';
import { useRouter } from 'expo-router';
import CinovaScore from './CinovaScore';
import StreamingBadge from './StreamingBadge';
import { Colors, Typography, Spacing, Radius } from '../../constants/theme';
import { getProviderById } from '../../constants/providers';
import { hapticSuccess, hapticMedium } from '../../services/haptics';
import type { Movie, WatchProvider } from '../../types';

const { width: SCREEN_WIDTH, height: SCREEN_HEIGHT } = Dimensions.get('window');
const EMBED_BASE = 'https://api.cinova.openova.io/api/v1/embed';

interface ReelItemProps {
  movie: Movie;
  isActive?: boolean;
  shouldLoad?: boolean;
  onSave?: (movie: Movie) => void;
  onRate?: (movie: Movie) => void;
  onDismiss?: (movie: Movie) => void;
  isSaved?: boolean;
  userRating?: number;
}

function getVideoKey(movie: Movie): string | null {
  const k = movie.verticalTrailerYoutubeKey;
  return k && k !== 'NOT_FOUND' ? k : null;
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

export default React.memo(function ReelItem({
  movie,
  isActive = false,
  shouldLoad = false,
  onSave,
  onRate,
  onDismiss,
  isSaved = false,
  userRating,
}: ReelItemProps) {
  const router = useRouter();
  const webViewRef = useRef<WebView>(null);
  const readyRef = useRef(false);
  const [debugState, setDebugState] = useState('INIT');
  const primaryProvider = movie.providers?.[0] ?? null;
  const genreLabel = movie.genres.slice(0, 2).map((g) => g.name).join(' · ');
  const runtimeLabel = movie.runtime ? `${movie.runtime}m` : '';
  const videoKey = getVideoKey(movie);

  const handleTap = useCallback(() => {
    router.push(`/movie/${movie.id}`);
  }, [movie.id, router]);

  const handleWatch = useCallback(async () => {
    if (primaryProvider) await watchOnProvider(primaryProvider, movie.tmdbId);
  }, [primaryProvider, movie.tmdbId]);

  // Play/pause + mute/unmute when isActive changes
  useEffect(() => {
    if (!readyRef.current) return;
    if (isActive) {
      setDebugState('UNMUTE+PLAY');
      webViewRef.current?.injectJavaScript('player.unMute(); playAll(); true;');
    } else {
      setDebugState('MUTE+PAUSE');
      webViewRef.current?.injectJavaScript('player.mute(); pauseAll(); true;');
    }
  }, [isActive]);

  const isActiveRef = useRef(isActive);
  useEffect(() => { isActiveRef.current = isActive; }, [isActive]);

  const onMessage = useCallback((e: any) => {
    try {
      const msg = JSON.parse(e.nativeEvent.data);
      if (msg.type === 'playerReady') {
        readyRef.current = true;
        setDebugState(isActiveRef.current ? 'READY+ACTIVE' : 'READY+PRELOAD');
        if (isActiveRef.current) {
          webViewRef.current?.injectJavaScript('player.unMute(); true;');
        }
      }
      if (msg.type === 'playerPlaying') {
        if (!isActiveRef.current) {
          setDebugState('PAUSING_AT_FRAME');
          webViewRef.current?.injectJavaScript('pauseAll(); true;');
        } else {
          setDebugState('PLAYING');
        }
      }
    } catch {}
  }, []);

  const showVideo = shouldLoad && videoKey;
  // Show YouTube thumbnail for videos that haven't played yet
  const thumbUri = videoKey ? `https://img.youtube.com/vi/${videoKey}/maxresdefault.jpg` : null;
  // Track if this video has ever reached PLAYING state
  const hasPlayedRef = useRef(false);
  if (debugState === 'PLAYING' || debugState === 'MUTE+PAUSE' || debugState === 'PAUSING_AT_FRAME') {
    hasPlayedRef.current = true;
  }

  return (
    <View style={styles.container}>

      {/* YouTube thumbnail — shows for videos that haven't played yet */}
      {thumbUri && !hasPlayedRef.current && (
        <Image
          source={{ uri: thumbUri }}
          style={StyleSheet.absoluteFill}
          contentFit="cover"
          priority="high"
        />
      )}

      {/* WebView video player — on top of thumbnail */}
      {showVideo && (
        <WebView
          ref={webViewRef}
          source={{ uri: `${EMBED_BASE}/${videoKey}?autoplay=1&controls=0&mute=1` }}
          style={[StyleSheet.absoluteFill, { backgroundColor: 'transparent' }]}
          allowsInlineMediaPlayback
          mediaPlaybackRequiresUserAction={false}
          scrollEnabled={false}
          bounces={false}
          startInLoadingState={false}
          onMessage={onMessage}
          pointerEvents="none"
          transparent
        />
      )}

      {/* DEBUG: visible state indicator */}
      <View style={{ position: 'absolute', top: 100, left: 12, backgroundColor: 'rgba(255,0,0,0.8)', paddingHorizontal: 8, paddingVertical: 4, borderRadius: 4, zIndex: 999 }}>
        <Text style={{ color: '#fff', fontSize: 11, fontWeight: '700' }}>
          {debugState} | {shouldLoad ? 'LOAD' : 'UNLOAD'} | {isActive ? 'ACTIVE' : 'IDLE'} | {videoKey ? videoKey.slice(0, 6) : 'NOKEY'}
        </Text>
      </View>

      {/* Gradient overlay */}
      <LinearGradient
        colors={['transparent', 'transparent', 'rgba(0,0,0,0.5)', 'rgba(0,0,0,0.88)', Colors.background]}
        locations={[0, 0.35, 0.6, 0.8, 1]}
        style={styles.gradient}
        pointerEvents="none"
      />

      {/* Tap target — navigates to detail */}
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
});

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
