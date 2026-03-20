/**
 * TrailerPlayer — Full-screen YouTube trailer modal.
 *
 * Layout rules:
 *  - No auto-rotation. User decides orientation.
 *  - useWindowDimensions() is the single source of truth for screen size.
 *    It updates reactively on rotation. With StatusBar hidden + statusBarTranslucent,
 *    window === screen — no offset issues.
 *  - Fetches the video's native aspect ratio from the backend (YouTube Data API v3)
 *    so the player container exactly matches the video — zero internal letterboxing.
 *  - Player is centered with explicit top/left offsets — equal bars on all sides.
 *  - Bottom 25% of player passthrough → YouTube progress bar always accessible.
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
  useWindowDimensions,
} from 'react-native';
import * as ScreenOrientation from 'expo-screen-orientation';
import YoutubePlayer, { type YoutubeIframeRef } from 'react-native-youtube-iframe';
import { useSafeAreaInsets } from 'react-native-safe-area-context';
import { Image } from 'expo-image';
import { getProviderById } from '../../constants/providers';
import { getVideoAspectRatio } from '../../services/api';
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
const DEFAULT_ASPECT_RATIO = 16 / 9;

// react-native-youtube-iframe hardcodes `padding-bottom: 56.25%` (16:9) in its
// HTML template CSS, which overrides any height prop we pass. This injection
// removes that constraint so the player fills the WebView at whatever aspect
// ratio we set — allowing our YouTube Data API sizing to actually take effect.
const FILL_CONTAINER_CSS = `
  (function(){
    var s = document.createElement('style');
    s.textContent = 'html,body{margin:0;padding:0;height:100%;overflow:hidden}.container{padding-bottom:0!important;height:100%!important}';
    document.head.appendChild(s);
  })();
  true;
`;

export default function TrailerPlayer({
  youtubeKey,
  title,
  primaryProvider,
  tmdbId,
  onClose,
}: TrailerPlayerProps) {
  const insets = useSafeAreaInsets();
  const playerRef = useRef<YoutubeIframeRef>(null);

  // useWindowDimensions updates reactively on every orientation change.
  // With StatusBar hidden + statusBarTranslucent, window === physical screen.
  const { width: SW, height: SH } = useWindowDimensions();

  // Native aspect ratio of this specific YouTube video (width / height).
  // Fetched once on mount — sizes the player to eliminate YouTube internal letterboxing.
  const [videoAspectRatio, setVideoAspectRatio] = useState(DEFAULT_ASPECT_RATIO);

  const [playing, setPlaying] = useState(true);
  const [ended, setEnded] = useState(false);

  const tapTimerLeft = useRef<ReturnType<typeof setTimeout> | null>(null);
  const tapTimerRight = useRef<ReturnType<typeof setTimeout> | null>(null);
  const tapCountLeft = useRef(0);
  const tapCountRight = useRef(0);

  const seekLeftOpacity = useRef(new Animated.Value(0)).current;
  const seekRightOpacity = useRef(new Animated.Value(0)).current;

  useEffect(() => {
    // Unlock rotation so user can choose orientation
    ScreenOrientation.unlockAsync();

    // Fetch native aspect ratio — eliminates YouTube internal letterboxing
    getVideoAspectRatio(youtubeKey).then(setVideoAspectRatio);

    return () => {
      ScreenOrientation.lockAsync(ScreenOrientation.OrientationLock.PORTRAIT_UP);
    };
  }, [youtubeKey]);

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

  // Fit the video within the screen using its native aspect ratio.
  // Width-constrained:  full width, letterbox top/bottom
  // Height-constrained: full height, pillarbox left/right
  // Pick whichever fits within both SW and SH.
  const hFromW = Math.round(SW / videoAspectRatio);
  const playerW = hFromW <= SH ? SW : Math.round(SH * videoAspectRatio);
  const playerH = hFromW <= SH ? hFromW : SH;

  const playerTop  = Math.round((SH - playerH) / 2);
  const playerLeft = Math.round((SW - playerW) / 2);

  // Tap zones: top 75% of player — bottom 25% passthrough for YouTube progress bar
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

      {/* Full screen black background */}
      <View style={[StyleSheet.absoluteFillObject, { backgroundColor: '#000' }]} />

      {/* Player — centered using explicit top/left */}
      <View style={{ position: 'absolute', top: playerTop, left: playerLeft, width: playerW, height: playerH }}>

        {/* key forces WebView remount when dimensions change */}
        <YoutubePlayer
          key={`${playerW}x${playerH}`}
          ref={playerRef}
          height={playerH}
          width={playerW}
          videoId={youtubeKey}
          play={playing}
          onChangeState={handleStateChange}
          webViewProps={{ allowsInlineMediaPlayback: true, injectedJavaScript: FILL_CONTAINER_CSS }}
          initialPlayerParams={{ controls: true, modestbranding: true, rel: false }}
        />

        {/* Tap zones — top 75% only */}
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

      {/* Close button — screen coordinates, always visible */}
      <TouchableOpacity
        style={[styles.closeBtn, { top: playerTop + Math.max(insets.top, 12), left: playerLeft + Math.max(insets.left, 12) }]}
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
