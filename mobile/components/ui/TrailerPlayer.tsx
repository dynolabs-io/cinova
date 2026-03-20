/**
 * TrailerPlayer — Full-screen YouTube trailer modal.
 *
 * Controls:
 *  - Double-tap left third  → rewind 10s
 *  - Double-tap right third → forward 10s
 *  - Single tap center      → show/hide native controls overlay
 *  - Supports free rotation (portrait + landscape)
 *  - Fills screen in any orientation
 *  - Shows "Watch on [Provider]" when video ends
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

export default function TrailerPlayer({
  youtubeKey,
  title,
  primaryProvider,
  tmdbId,
  onClose,
}: TrailerPlayerProps) {
  const insets = useSafeAreaInsets();
  const playerRef = useRef<YoutubeIframeRef>(null);
  const [dims, setDims] = useState(Dimensions.get('window'));
  const [playing, setPlaying] = useState(true);
  const [ended, setEnded] = useState(false);

  // Double-tap tracking
  const tapTimerLeft = useRef<ReturnType<typeof setTimeout> | null>(null);
  const tapTimerRight = useRef<ReturnType<typeof setTimeout> | null>(null);
  const tapCountLeft = useRef(0);
  const tapCountRight = useRef(0);

  // Seek indicator animations
  const seekLeftOpacity = useRef(new Animated.Value(0)).current;
  const seekRightOpacity = useRef(new Animated.Value(0)).current;

  useEffect(() => {
    ScreenOrientation.unlockAsync();
    const sub = Dimensions.addEventListener('change', ({ window }) => setDims(window));
    return () => {
      sub.remove();
      ScreenOrientation.lockAsync(ScreenOrientation.OrientationLock.PORTRAIT_UP);
    };
  }, []);

  const handleClose = useCallback(() => {
    ScreenOrientation.lockAsync(ScreenOrientation.OrientationLock.PORTRAIT_UP).then(onClose);
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
      tapTimerLeft.current = setTimeout(() => {
        tapCountLeft.current = 0;
      }, DOUBLE_TAP_DELAY);
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
      tapTimerRight.current = setTimeout(() => {
        tapCountRight.current = 0;
      }, DOUBLE_TAP_DELAY);
    } else if (tapCountRight.current >= 2) {
      if (tapTimerRight.current) clearTimeout(tapTimerRight.current);
      tapCountRight.current = 0;
      seekBy(SEEK_SECONDS);
      flashIndicator(seekRightOpacity);
    }
  }

  const isLandscape = dims.width > dims.height;
  // In landscape: fill full screen. In portrait: fill full height (pillarbox).
  const playerWidth = isLandscape ? dims.width : dims.height * (16 / 9);
  const playerHeight = isLandscape ? dims.height : dims.height;
  const tapZoneWidth = isLandscape ? playerWidth / 3 : dims.width / 3;

  return (
    <Modal
      visible
      animationType="fade"
      statusBarTranslucent
      supportedOrientations={['portrait', 'landscape', 'landscape-left', 'landscape-right']}
      onRequestClose={handleClose}
    >
      <StatusBar hidden />
      <View style={[styles.container, { width: dims.width, height: dims.height }]}>

        {/* YouTube player */}
        <View style={[styles.playerWrapper, { width: playerWidth, height: playerHeight }]}>
          <YoutubePlayer
            ref={playerRef}
            height={playerHeight}
            width={playerWidth}
            videoId={youtubeKey}
            play={playing}
            onChangeState={handleStateChange}
            webViewProps={{ allowsInlineMediaPlayback: true }}
            initialPlayerParams={{ controls: true, modestbranding: true, rel: false }}
          />
        </View>

        {/* Double-tap zones — left / right only; center passes touches to YouTube */}
        <View style={[styles.tapOverlay, { width: playerWidth, height: playerHeight }]} pointerEvents="box-none">
          {/* Left zone: rewind */}
          <TouchableOpacity
            activeOpacity={1}
            style={[styles.tapZone, { width: tapZoneWidth }]}
            onPress={handleTapLeft}
          />
          {/* Center zone: transparent — YouTube controls accessible here */}
          <View style={[styles.tapZone, { width: tapZoneWidth }]} pointerEvents="none" />
          {/* Right zone: forward */}
          <TouchableOpacity
            activeOpacity={1}
            style={[styles.tapZone, { width: tapZoneWidth }]}
            onPress={handleTapRight}
          />
        </View>

        {/* Seek indicators */}
        <Animated.View style={[styles.seekIndicator, styles.seekLeft, { opacity: seekLeftOpacity }]}>
          <Text style={styles.seekIcon}>«</Text>
          <Text style={styles.seekLabel}>{SEEK_SECONDS}s</Text>
        </Animated.View>
        <Animated.View style={[styles.seekIndicator, styles.seekRight, { opacity: seekRightOpacity }]}>
          <Text style={styles.seekIcon}>»</Text>
          <Text style={styles.seekLabel}>{SEEK_SECONDS}s</Text>
        </Animated.View>

        {/* Close button */}
        <TouchableOpacity
          style={[styles.closeBtn, { top: Math.max(insets.top, 12), left: Math.max(insets.left, 12) }]}
          onPress={handleClose}
          hitSlop={{ top: 16, bottom: 16, left: 16, right: 16 }}
        >
          <Text style={styles.closeBtnText}>✕</Text>
        </TouchableOpacity>

        {/* Title */}
        <View style={[styles.titleBar, { top: Math.max(insets.top, 12), right: Math.max(insets.right, 12) }]}>
          <Text style={styles.titleText} numberOfLines={1}>{title}</Text>
        </View>

        {/* Watch on Provider overlay */}
        {ended && primaryProvider && (
          <View style={[styles.watchOverlay, { bottom: Math.max(insets.bottom, 32) }]}>
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
  playerWrapper: {
    backgroundColor: '#000',
  },
  tapOverlay: {
    position: 'absolute',
    top: 0,
    left: 0,
    flexDirection: 'row',
    zIndex: 5,
  },
  tapZone: {
    height: '100%',
  },
  seekIndicator: {
    position: 'absolute',
    top: '40%',
    alignItems: 'center',
    backgroundColor: 'rgba(0,0,0,0.6)',
    borderRadius: 40,
    paddingHorizontal: 18,
    paddingVertical: 12,
    zIndex: 10,
  },
  seekLeft: {
    left: 24,
  },
  seekRight: {
    right: 24,
  },
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
  titleBar: {
    position: 'absolute',
    maxWidth: '60%',
    backgroundColor: 'rgba(0,0,0,0.5)',
    borderRadius: 8,
    paddingHorizontal: 10,
    paddingVertical: 5,
    zIndex: 20,
  },
  titleText: {
    color: '#fff',
    fontSize: 13,
    fontWeight: '600',
  },
  watchOverlay: {
    position: 'absolute',
    alignSelf: 'center',
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
