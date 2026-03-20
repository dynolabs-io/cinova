/**
 * TrailerPlayer — Full-screen YouTube trailer modal.
 *
 * Layout rules:
 *  - Always opens in landscape (locked) — guarantees full-screen for all trailers
 *    regardless of whether the movie is 16:9, 1.85:1, or 2.35:1.
 *  - Portrait fallback still works if orientation lock fails (same 16:9 letterbox).
 *  - Uses onLayout on the container View (not Dimensions API) because:
 *    a) Dimensions.get('screen') returns portrait dims when the app is portrait-locked,
 *       even if the phone is physically in landscape when the modal opens.
 *    b) ScreenOrientation.unlockAsync() is async — initial render fires before unlock.
 *    c) onLayout measures what was actually rendered by the system, immune to all of this.
 *  - Player top offset is calculated explicitly → guaranteed equal bars.
 *  - Bottom 25% of player is fully passthrough → YouTube's progress bar and
 *    native controls are always accessible.
 *  - No custom title overlay — YouTube already shows it.
 */

import React, { useCallback, useEffect, useRef, useState } from 'react';
import {
  View,
  Text,
  Modal,
  TouchableOpacity,
  StyleSheet,
  StatusBar,
  Linking,
  Platform,
  Animated,
} from 'react-native';
import * as ScreenOrientation from 'expo-screen-orientation';
import YoutubePlayer, { type YoutubeIframeRef } from 'react-native-youtube-iframe';
import { useSafeAreaInsets } from 'react-native-safe-area-context';
import { Image } from 'expo-image';
import { getProviderById } from '../../constants/providers';
import type { WatchProvider } from '../../types';

interface TrailerPlayerProps {
  youtubeKey: string;
  title: string;
  primaryProvider?: WatchProvider | null;
  tmdbId: number;
  onClose: () => void;
}

const SEEK_SECONDS = 10;
const DOUBLE_TAP_DELAY = 300;

export default function TrailerPlayer({
  youtubeKey,
  title,
  primaryProvider,
  tmdbId,
  onClose,
}: TrailerPlayerProps) {
  const insets = useSafeAreaInsets();
  const playerRef = useRef<YoutubeIframeRef>(null);

  // onLayout-based dimensions — measured from actual rendered container.
  // Immune to portrait-lock, async unlock timing, and statusBarTranslucent chrome.
  const [containerSize, setContainerSize] = useState({ width: 0, height: 0 });
  const { width: CW, height: CH } = containerSize;

  const [playing, setPlaying] = useState(true);
  const [ended, setEnded] = useState(false);

  const tapTimerLeft = useRef<ReturnType<typeof setTimeout> | null>(null);
  const tapTimerRight = useRef<ReturnType<typeof setTimeout> | null>(null);
  const tapCountLeft = useRef(0);
  const tapCountRight = useRef(0);

  const seekLeftOpacity = useRef(new Animated.Value(0)).current;
  const seekRightOpacity = useRef(new Animated.Value(0)).current;

  useEffect(() => {
    // Force landscape immediately — trailers always look better full-width,
    // and this eliminates the double-letterboxing issue for widescreen trailers.
    ScreenOrientation.lockAsync(ScreenOrientation.OrientationLock.LANDSCAPE);
    return () => {
      ScreenOrientation.lockAsync(ScreenOrientation.OrientationLock.PORTRAIT_UP);
    };
  }, []);

  const handleClose = useCallback(() => {
    ScreenOrientation.lockAsync(ScreenOrientation.OrientationLock.PORTRAIT_UP)
      .catch(() => {})
      .finally(onClose);
    // Effect cleanup also fires, but lockAsync is idempotent — safe to call twice.
  }, [onClose]);

  const handleStateChange = useCallback((state: string) => {
    if (state === 'ended') { setPlaying(false); setEnded(true); }
    if (state === 'playing') setPlaying(true);
    if (state === 'paused') setPlaying(false);
  }, []);

  const handleWatch = useCallback(async () => {
    if (!primaryProvider) return;
    const known = getProviderById(primaryProvider.providerId);
    if (known) {
      const deepLink = known.buildDeepLink(tmdbId);
      const canOpen = await Linking.canOpenURL(deepLink);
      if (canOpen) { await Linking.openURL(deepLink); return; }
      const store = Platform.OS === 'ios' ? known.storeUrl.ios : known.storeUrl.android;
      await Linking.openURL(store);
      return;
    }
    if (primaryProvider.link) await Linking.openURL(primaryProvider.link);
  }, [primaryProvider, tmdbId]);

  function flashIndicator(anim: Animated.Value) {
    anim.setValue(1);
    Animated.timing(anim, { toValue: 0, duration: 600, useNativeDriver: true }).start();
  }

  async function seekBy(delta: number) {
    if (!playerRef.current) return;
    const current = await playerRef.current.getCurrentTime();
    playerRef.current.seekTo(Math.max(0, current + delta), true);
  }

  function handleTapLeft() {
    tapCountLeft.current += 1;
    if (tapCountLeft.current === 1) {
      tapTimerLeft.current = setTimeout(() => { tapCountLeft.current = 0; }, DOUBLE_TAP_DELAY);
    } else if (tapCountLeft.current >= 2) {
      if (tapTimerLeft.current) clearTimeout(tapTimerLeft.current);
      tapCountLeft.current = 0;
      seekBy(-SEEK_SECONDS);
      flashIndicator(seekLeftOpacity);
    }
  }

  function handleTapRight() {
    tapCountRight.current += 1;
    if (tapCountRight.current === 1) {
      tapTimerRight.current = setTimeout(() => { tapCountRight.current = 0; }, DOUBLE_TAP_DELAY);
    } else if (tapCountRight.current >= 2) {
      if (tapTimerRight.current) clearTimeout(tapTimerRight.current);
      tapCountRight.current = 0;
      seekBy(SEEK_SECONDS);
      flashIndicator(seekRightOpacity);
    }
  }

  // Compute player dimensions from measured container — not Dimensions API.
  // CW/CH = 0 until onLayout fires; player is hidden until then.
  const isLandscape = CW > 0 && CW > CH;
  const playerW = CW;
  const playerH = isLandscape ? CH : (CW > 0 ? Math.round(CW * 9 / 16) : 0);

  // Explicit top offset → equal black bars in portrait, zero bars in landscape
  const playerTop = CH > 0 ? Math.round((CH - playerH) / 2) : 0;

  // Tap zones: top 75% of player only — bottom 25% passthrough for YouTube progress bar
  const tapH = Math.round(playerH * 0.75);
  const tapW = Math.round(playerW / 3);

  return (
    <Modal
      visible
      animationType="fade"
      statusBarTranslucent
      supportedOrientations={['portrait', 'landscape', 'landscape-left', 'landscape-right']}
      onRequestClose={handleClose}
    >
      <StatusBar hidden />

      {/* Container fills the full modal area — onLayout gives us actual rendered dimensions */}
      <View
        style={StyleSheet.absoluteFillObject}
        onLayout={(e) => {
          const { width, height } = e.nativeEvent.layout;
          setContainerSize({ width, height });
        }}
      >
        {/* Black background */}
        <View style={[StyleSheet.absoluteFillObject, { backgroundColor: '#000' }]} />

        {/* Player block — only rendered once we have real dimensions */}
        {CW > 0 && (
          <View style={{ position: 'absolute', top: playerTop, left: 0, width: playerW, height: playerH }}>

            {/* key forces WebView remount when dimensions change (orientation change).
                Without this, the WebView keeps its original layout from mount time —
                causing tiny player when opened in portrait then rotated to landscape. */}
            <YoutubePlayer
              key={`${playerW}x${playerH}`}
              ref={playerRef}
              height={playerH}
              width={playerW}
              videoId={youtubeKey}
              play={playing}
              onChangeState={handleStateChange}
              webViewProps={{ allowsInlineMediaPlayback: true }}
              initialPlayerParams={{ controls: true, modestbranding: true, rel: false }}
            />

            {/* Tap zones — top 75% only; bottom 25% is passthrough for YouTube progress bar */}
            <View style={[styles.tapOverlay, { height: tapH }]} pointerEvents="box-none">
              <TouchableOpacity activeOpacity={1} style={{ width: tapW, height: '100%' }} onPress={handleTapLeft} />
              <View style={{ width: tapW, height: '100%' }} pointerEvents="none" />
              <TouchableOpacity activeOpacity={1} style={{ width: tapW, height: '100%' }} onPress={handleTapRight} />
            </View>

            {/* Seek flash indicators */}
            <Animated.View style={[styles.seekIndicator, styles.seekLeft, { opacity: seekLeftOpacity }]}>
              <Text style={styles.seekIcon}>«</Text>
              <Text style={styles.seekLabel}>{SEEK_SECONDS}s</Text>
            </Animated.View>
            <Animated.View style={[styles.seekIndicator, styles.seekRight, { opacity: seekRightOpacity }]}>
              <Text style={styles.seekIcon}>»</Text>
              <Text style={styles.seekLabel}>{SEEK_SECONDS}s</Text>
            </Animated.View>

            {/* Watch on Provider — shown when video ends */}
            {ended && primaryProvider && (
              <View style={styles.watchOverlay}>
                <Text style={styles.watchLabel}>Ready to watch?</Text>
                <TouchableOpacity style={styles.watchBtn} onPress={handleWatch}>
                  {primaryProvider.logoPath ? (
                    <Image
                      source={{ uri: `https://image.tmdb.org/t/p/w92${primaryProvider.logoPath}` }}
                      style={styles.providerLogo}
                      contentFit="contain"
                    />
                  ) : null}
                  <Text style={styles.watchBtnText}>Watch on {primaryProvider.providerName}</Text>
                </TouchableOpacity>
              </View>
            )}
          </View>
        )}

        {/* Close button — in container coordinates, always visible */}
        <TouchableOpacity
          style={[styles.closeBtn, { top: Math.max(insets.top, 12), left: Math.max(insets.left, 12) }]}
          onPress={handleClose}
          hitSlop={{ top: 16, bottom: 16, left: 16, right: 16 }}
        >
          <Text style={styles.closeBtnText}>✕</Text>
        </TouchableOpacity>
      </View>
    </Modal>
  );
}

const styles = StyleSheet.create({
  tapOverlay: {
    position: 'absolute',
    top: 0,
    left: 0,
    right: 0,
    flexDirection: 'row',
    zIndex: 5,
  },
  seekIndicator: {
    position: 'absolute',
    top: '35%',
    alignItems: 'center',
    backgroundColor: 'rgba(0,0,0,0.6)',
    borderRadius: 40,
    paddingHorizontal: 18,
    paddingVertical: 12,
    zIndex: 10,
  },
  seekLeft: { left: 24 },
  seekRight: { right: 24 },
  seekIcon: { color: '#fff', fontSize: 28, fontWeight: '700' },
  seekLabel: { color: '#fff', fontSize: 12, fontWeight: '600', marginTop: 2 },
  closeBtn: {
    position: 'absolute',
    width: 40,
    height: 40,
    borderRadius: 20,
    backgroundColor: 'rgba(0,0,0,0.7)',
    alignItems: 'center',
    justifyContent: 'center',
    zIndex: 20,
  },
  closeBtnText: { color: '#fff', fontSize: 16, fontWeight: '700' },
  watchOverlay: {
    position: 'absolute',
    bottom: 80,
    left: 0,
    right: 0,
    alignItems: 'center',
    gap: 10,
    zIndex: 20,
  },
  watchLabel: { color: '#ccc', fontSize: 14 },
  watchBtn: {
    flexDirection: 'row',
    alignItems: 'center',
    backgroundColor: '#fff',
    borderRadius: 24,
    paddingHorizontal: 20,
    paddingVertical: 12,
    gap: 8,
  },
  providerLogo: { width: 24, height: 24, borderRadius: 4 },
  watchBtnText: { color: '#000', fontSize: 15, fontWeight: '700' },
});
