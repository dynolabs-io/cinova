/**
 * TrailerPlayer — full-screen YouTube trailer modal.
 *
 * Opens in portrait, auto-rotates to landscape for 16:9 playback.
 * Offers "Watch on [Provider]" deep-link on close.
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
} from 'react-native';
import * as ScreenOrientation from 'expo-screen-orientation';
import YoutubePlayer from 'react-native-youtube-iframe';
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

export default function TrailerPlayer({
  youtubeKey,
  title,
  primaryProvider,
  tmdbId,
  onClose,
}: TrailerPlayerProps) {
  const insets = useSafeAreaInsets();
  const [dims, setDims] = useState(Dimensions.get('window'));
  const [playing, setPlaying] = useState(true);
  const [ended, setEnded] = useState(false);

  // Unlock rotation and go landscape on mount
  useEffect(() => {
    ScreenOrientation.unlockAsync().then(() => {
      ScreenOrientation.lockAsync(ScreenOrientation.OrientationLock.LANDSCAPE);
    });

    const sub = Dimensions.addEventListener('change', ({ window }) => {
      setDims(window);
    });

    return () => {
      sub.remove();
      // Re-lock portrait when unmounting
      ScreenOrientation.lockAsync(ScreenOrientation.OrientationLock.PORTRAIT_UP);
    };
  }, []);

  const handleClose = useCallback(() => {
    ScreenOrientation.lockAsync(ScreenOrientation.OrientationLock.PORTRAIT_UP).then(onClose);
  }, [onClose]);

  const handleStateChange = useCallback((state: string) => {
    if (state === 'ended') {
      setPlaying(false);
      setEnded(true);
    }
  }, []);

  const handleWatch = useCallback(async () => {
    if (!primaryProvider) return;
    const known = getProviderById(primaryProvider.providerId);
    if (known) {
      const deepLink = known.buildDeepLink(tmdbId);
      const canOpen = await Linking.canOpenURL(deepLink);
      if (canOpen) {
        await Linking.openURL(deepLink);
        return;
      }
      const store = Platform.OS === 'ios' ? known.storeUrl.ios : known.storeUrl.android;
      await Linking.openURL(store);
      return;
    }
    if (primaryProvider.link) await Linking.openURL(primaryProvider.link);
  }, [primaryProvider, tmdbId]);

  const isLandscape = dims.width > dims.height;
  const playerWidth = dims.width;
  const playerHeight = isLandscape ? dims.height : dims.width * (9 / 16);

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
            height={playerHeight}
            width={playerWidth}
            videoId={youtubeKey}
            play={playing}
            onChangeState={handleStateChange}
            webViewProps={{ allowsInlineMediaPlayback: true }}
            initialPlayerParams={{ controls: true, modestbranding: true, rel: false }}
          />
        </View>

        {/* Close button */}
        <TouchableOpacity
          style={[styles.closeBtn, { top: Math.max(insets.top, 12), left: Math.max(insets.left, 12) }]}
          onPress={handleClose}
          hitSlop={{ top: 12, bottom: 12, left: 12, right: 12 }}
        >
          <Text style={styles.closeBtnText}>✕</Text>
        </TouchableOpacity>

        {/* Title bar */}
        <View style={[styles.titleBar, { top: Math.max(insets.top, 12), right: Math.max(insets.right, 12) }]}>
          <Text style={styles.titleText} numberOfLines={1}>{title}</Text>
        </View>

        {/* "Watch on Provider" overlay — shown when video ends */}
        {ended && primaryProvider && (
          <View style={[styles.watchOverlay, { bottom: Math.max(insets.bottom, 24) }]}>
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
  closeBtn: {
    position: 'absolute',
    width: 36,
    height: 36,
    borderRadius: 18,
    backgroundColor: 'rgba(0,0,0,0.7)',
    alignItems: 'center',
    justifyContent: 'center',
    zIndex: 10,
  },
  closeBtnText: {
    color: '#fff',
    fontSize: 15,
    fontWeight: '700',
  },
  titleBar: {
    position: 'absolute',
    maxWidth: '60%',
    backgroundColor: 'rgba(0,0,0,0.5)',
    borderRadius: 8,
    paddingHorizontal: 10,
    paddingVertical: 5,
    zIndex: 10,
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
