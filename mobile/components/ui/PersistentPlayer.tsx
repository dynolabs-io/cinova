/**
 * PersistentPlayer — single long-lived WebView for the Reels screen.
 *
 * Loads the YouTube player ONCE via our embed endpoint. When the active
 * video key changes, calls switchVideo() defined in the embed page.
 * All YouTube API calls happen in the embed page's JS context.
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
      console.log('[PersistentPlayer] onMessage:', msg.type, JSON.stringify(msg));
      if (msg.type === 'playerReady') {
        playerReadyRef.current = true;
        const key = pendingKeyRef.current;
        if (key && key !== currentKeyRef.current) {
          console.log('[PersistentPlayer] playerReady: switching to pending key', key);
          currentKeyRef.current = key;
          webViewRef.current?.injectJavaScript(`switchVideo('${key}'); true;`);
        } else if (playingRef.current) {
          console.log('[PersistentPlayer] playerReady: playing initial video');
          webViewRef.current?.injectJavaScript('playAll(); true;');
        }
      }
    } catch {}
  }, []);

  useEffect(() => {
    console.log('[PersistentPlayer] useEffect videoKey=', videoKey, 'playing=', playing, 'ready=', playerReadyRef.current, 'current=', currentKeyRef.current);
    pendingKeyRef.current = videoKey;
    if (!playerReadyRef.current || !videoKey) {
      console.log('[PersistentPlayer] early return: ready=', playerReadyRef.current, 'videoKey=', videoKey);
      return;
    }

    if (videoKey === currentKeyRef.current) {
      // Same video — just play or pause
      console.log('[PersistentPlayer] same video, playing=', playing);
      webViewRef.current?.injectJavaScript(
        playing ? 'playAll(); true;' : 'pauseAll(); true;'
      );
      return;
    }

    // Different video — switch via embed page function
    console.log('[PersistentPlayer] SWITCHING from', currentKeyRef.current, 'to', videoKey);
    currentKeyRef.current = videoKey;
    webViewRef.current?.injectJavaScript(`switchVideo('${videoKey}'); true;`);
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
