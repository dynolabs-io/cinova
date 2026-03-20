/**
 * TrailerPlayer — Full-screen YouTube trailer modal.
 *
 * Layout rules:
 *  - Landscape: player fills entire screen (no bars).
 *  - Portrait: player fills full width, height = width × 9/16, centered vertically
 *    so black bars above and below are equal.
 *  - Bottom 25% of player is fully passthrough → YouTube's native progress bar
 *    and controls are always accessible without interference.
 *  - Double-tap left third → rewind 10s (top 75% only)
 *  - Double-tap right third → forward 10s (top 75% only)
 *  - No custom title overlay — YouTube already shows it.
 *  - Shows "Watch on [Provider]" when video ends.
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
  useWindowDimensions,
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
  const { width, height } = useWindowDimensions(); // auto-updates on rotation
  const playerRef = useRef<YoutubeIframeRef>(null);
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
    return () => {
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

  const isLandscape = width > height;

  // Landscape: fill entire screen.
  // Portrait: fill width, height = width × 9/16 → equal black bars above and below.
  const playerW = isLandscape ? width : width;
  const playerH = isLandscape ? height : Math.round(width * 9 / 16);

  // Tap zones cover only the top 75% of the player.
  // The bottom 25% is fully passthrough so YouTube's progress bar is always usable.
  const tapZoneH = Math.round(playerH * 0.75);
  const tapZoneW = Math.round(playerW / 3);

  // Close button: top-left accounting for safe area
  const closeBtnTop = Math.max(insets.top, 12);
  const closeBtnLeft = Math.max(insets.left, 12);

  return (
    <Modal
      visible
      animationType="fade"
      statusBarTranslucent
      supportedOrientations={['portrait', 'landscape', 'landscape-left', 'landscape-right']}
      onRequestClose={handleClose}
    >
      <StatusBar hidden />

      {/* Full-screen black background, player centered */}
      <View style={styles.container}>

        {/* Player + all overlays in one positioned block */}
        <View style={{ width: playerW, height: playerH }}>

          {/* YouTube player */}
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

          {/* Tap zones — top 75% only, so bottom 25% passes through to YouTube controls */}
          <View
            style={[styles.tapOverlay, { width: playerW, height: tapZoneH }]}
            pointerEvents="box-none"
          >
            {/* Left: rewind */}
            <TouchableOpacity
              activeOpacity={1}
              style={{ width: tapZoneW, height: '100%' }}
              onPress={handleTapLeft}
            />
            {/* Center: fully passthrough to YouTube */}
            <View style={{ width: tapZoneW, height: '100%' }} pointerEvents="none" />
            {/* Right: forward */}
            <TouchableOpacity
              activeOpacity={1}
              style={{ width: tapZoneW, height: '100%' }}
              onPress={handleTapRight}
            />
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

          {/* Watch on Provider overlay — shown when video ends */}
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
                <Text style={styles.watchBtnText}>
                  Watch on {primaryProvider.providerName}
                </Text>
              </TouchableOpacity>
            </View>
          )}
        </View>

        {/* Close button — always on top, positioned in screen coordinates */}
        <TouchableOpacity
          style={[styles.closeBtn, { top: closeBtnTop, left: closeBtnLeft }]}
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
  container: {
    flex: 1,
    backgroundColor: '#000',
    justifyContent: 'center',
    alignItems: 'center',
  },
  tapOverlay: {
    position: 'absolute',
    top: 0,
    left: 0,
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
  seekIcon: {
    color: '#fff',
    fontSize: 28,
    fontWeight: '700',
  },
  seekLabel: {
    color: '#fff',
    fontSize: 12,
    fontWeight: '600',
    marginTop: 2,
  },
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
  closeBtnText: {
    color: '#fff',
    fontSize: 16,
    fontWeight: '700',
  },
  watchOverlay: {
    position: 'absolute',
    bottom: 80,
    left: 0,
    right: 0,
    alignItems: 'center',
    gap: 10,
    zIndex: 20,
  },
  watchLabel: {
    color: '#ccc',
    fontSize: 14,
  },
  watchBtn: {
    flexDirection: 'row',
    alignItems: 'center',
    backgroundColor: '#fff',
    borderRadius: 24,
    paddingHorizontal: 20,
    paddingVertical: 12,
    gap: 8,
  },
  providerLogo: {
    width: 24,
    height: 24,
    borderRadius: 4,
  },
  watchBtnText: {
    color: '#000',
    fontSize: 15,
    fontWeight: '700',
  },
});
