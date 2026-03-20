/**
 * TrailerPlayer — Full-screen YouTube trailer modal.
 *
 * Layout rules:
 *  - Landscape: player fills entire screen (no bars).
 *  - Portrait: player fills full width, height = width × 9/16, centered vertically
 *    so black bars above and below are exactly equal.
 *  - Uses Dimensions.get('screen') (physical screen) not 'window', because
 *    statusBarTranslucent makes the Modal content area larger than 'window',
 *    causing flex centering to shift the player down.
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
  Dimensions,
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

function getScreen() {
  // 'screen' = physical screen size — always correct regardless of status bar / modal chrome
  return Dimensions.get('screen');
}

export default function TrailerPlayer({
  youtubeKey,
  title,
  primaryProvider,
  tmdbId,
  onClose,
}: TrailerPlayerProps) {
  const insets = useSafeAreaInsets();
  const playerRef = useRef<YoutubeIframeRef>(null);
  const [screen, setScreen] = useState(getScreen);
  const [playing, setPlaying] = useState(true);
  const [ended, setEnded] = useState(false);

  const tapTimerLeft = useRef<ReturnType<typeof setTimeout> | null>(null);
  const tapTimerRight = useRef<ReturnType<typeof setTimeout> | null>(null);
  const tapCountLeft = useRef(0);
  const tapCountRight = useRef(0);

  const seekLeftOpacity = useRef(new Animated.Value(0)).current;
  const seekRightOpacity = useRef(new Animated.Value(0)).current;

  useEffect(() => {
    ScreenOrientation.unlockAsync();
    const sub = Dimensions.addEventListener('change', () => setScreen(getScreen()));
    return () => {
      sub.remove();
      ScreenOrientation.lockAsync(ScreenOrientation.OrientationLock.PORTRAIT_UP);
    };
  }, []);

  const handleClose = useCallback(() => {
    ScreenOrientation.lockAsync(ScreenOrientation.OrientationLock.PORTRAIT_UP)
      .catch(() => {})
      .finally(onClose);
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

  const SW = screen.width;
  const SH = screen.height;
  const isLandscape = SW > SH;

  // Landscape: player fills full screen.
  // Portrait: player fills width, height = width × 9/16 (16:9 letterbox).
  const playerW = SW;
  const playerH = isLandscape ? SH : Math.round(SW * 9 / 16);

  // Explicit top/left offset → guaranteed equal bars, immune to Modal chrome issues
  const playerTop = Math.round((SH - playerH) / 2);
  const playerLeft = Math.round((SW - playerW) / 2);

  // Tap zones: top 75% of player only — bottom 25% passthrough for YouTube controls
  const tapH = Math.round(playerH * 0.75);
  const tapW = Math.round(playerW / 3);

  const closeBtnTop = playerTop + Math.max(insets.top, 12);
  const closeBtnLeft = playerLeft + Math.max(insets.left, 12);

  return (
    <Modal
      visible
      animationType="fade"
      statusBarTranslucent
      supportedOrientations={['portrait', 'landscape', 'landscape-left', 'landscape-right']}
      onRequestClose={handleClose}
    >
      <StatusBar hidden />

      {/* Full physical screen black background */}
      <View style={[StyleSheet.absoluteFillObject, { backgroundColor: '#000' }]} />

      {/* Player block — explicitly positioned for pixel-perfect centering */}
      <View style={{ position: 'absolute', top: playerTop, left: playerLeft, width: playerW, height: playerH }}>

        <YoutubePlayer
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

      {/* Close button — positioned in screen coordinates, always visible */}
      <TouchableOpacity
        style={[styles.closeBtn, { top: closeBtnTop, left: closeBtnLeft }]}
        onPress={handleClose}
        hitSlop={{ top: 16, bottom: 16, left: 16, right: 16 }}
      >
        <Text style={styles.closeBtnText}>✕</Text>
      </TouchableOpacity>
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
