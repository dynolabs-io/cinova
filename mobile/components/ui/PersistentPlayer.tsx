/**
 * PersistentPlayer — single long-lived WebView for the Reels screen.
 *
 * Loads the YouTube player ONCE via our embed endpoint. When the active
 * video key changes, injects player.loadVideoById() instead of creating
 * a new WebView. This eliminates the 3-4 second per-video overhead.
 *
 * No React Native overlay — the embed page handles spinner hiding internally.
 */

import React, { useRef, useEffect, useCallback } from 'react';
import { StyleSheet, Dimensions, View } from 'react-native';
import { WebView, WebViewMessageEvent } from 'react-native-webview';

const { width: SCREEN_WIDTH, height: SCREEN_HEIGHT } = Dimensions.get('window');
const EMBED_BASE = 'https://api.cinova.openova.io/api/v1/embed';

interface Props {
  initialVideoKey: string;
  videoKey: string | null;
  playing: boolean;
}

export default function PersistentPlayer({ initialVideoKey, videoKey, playing }: Props) {
  const webViewRef = useRef<WebView>(null);
  const playerReadyRef = useRef(false);
  const pendingKeyRef = useRef<string | null>(videoKey);
  const currentKeyRef = useRef<string | null>(initialVideoKey);
  const playingRef = useRef(playing);

  useEffect(() => { playingRef.current = playing; }, [playing]);

  const onMessage = useCallback((e: WebViewMessageEvent) => {
    try {
      const msg = JSON.parse(e.nativeEvent.data);
      if (msg.type === 'playerReady') {
        playerReadyRef.current = true;
        const key = pendingKeyRef.current;
        if (key && key !== currentKeyRef.current) {
          currentKeyRef.current = key;
          webViewRef.current?.injectJavaScript(
            `player.loadVideoById('${key}'); ${playingRef.current ? 'player.playVideo();' : ''} true;`
          );
        } else if (playingRef.current) {
          webViewRef.current?.injectJavaScript('player.playVideo(); true;');
        } else {
          // Pre-buffer without playing (tab not focused yet)
          const k = currentKeyRef.current;
          if (k) webViewRef.current?.injectJavaScript(`player.cueVideoById('${k}'); true;`);
        }
      }
    } catch {}
  }, []);

  useEffect(() => {
    pendingKeyRef.current = videoKey;
    if (!playerReadyRef.current || !videoKey) return;

    if (videoKey === currentKeyRef.current) {
      webViewRef.current?.injectJavaScript(
        playing ? 'player.playVideo(); true;' : 'player.pauseVideo(); true;'
      );
      return;
    }

    currentKeyRef.current = videoKey;
    webViewRef.current?.injectJavaScript(
      `player.loadVideoById('${videoKey}'); ${playing ? 'player.playVideo();' : 'player.pauseVideo();'} true;`
    );
  }, [videoKey, playing]);

  return (
    <View style={styles.container} pointerEvents="none">
      <WebView
        ref={webViewRef}
        source={{ uri: `${EMBED_BASE}/${initialVideoKey}?autoplay=0&controls=0&_t=${Date.now()}` }}
        style={styles.webView}
        allowsInlineMediaPlayback
        mediaPlaybackRequiresUserAction={false}
        scrollEnabled={false}
        bounces={false}
        startInLoadingState={false}
        renderLoading={() => <View style={styles.webView} />}
        onMessage={onMessage}
      />
    </View>
  );
}

const styles = StyleSheet.create({
  container: {
    position: 'absolute',
    top: 0,
    left: 0,
    width: SCREEN_WIDTH,
    height: SCREEN_HEIGHT,
  },
  webView: {
    flex: 1,
    backgroundColor: '#000',
  },
});
