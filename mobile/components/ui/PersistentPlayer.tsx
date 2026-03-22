/**
 * PersistentPlayer — single long-lived WebView for the Reels screen.
 *
 * Mounted in root layout immediately on app start — video is buffered before
 * the user ever taps the Reels tab.
 *
 * - initialVideoKey: loaded via URL (YouTube buffers it immediately)
 * - videoKey: current active key — switches via player.loadVideoById()
 * - playing: play/pause control
 * - Black overlay hides YouTube's buffering spinner; fades when playerPlaying fires
 */

import React, { useRef, useEffect, useCallback, useState } from 'react';
import { StyleSheet, Dimensions, View } from 'react-native';
import { WebView, WebViewMessageEvent } from 'react-native-webview';

// Only show the black buffering overlay if the video hasn't started within this window.
// Cached / previously-seen videos typically play within 200ms so the overlay never appears.
const BUFFERING_OVERLAY_DELAY_MS = 300;

const { width: SCREEN_WIDTH, height: SCREEN_HEIGHT } = Dimensions.get('window');
const EMBED_BASE = 'https://api.cinova.openova.io/api/v1/embed';

interface Props {
  initialVideoKey: string;  // URL the WebView loads — set once, never changes
  videoKey: string | null;  // current video — changes via loadVideoById injection
  playing: boolean;         // play or pause
}

export default function PersistentPlayer({ initialVideoKey, videoKey, playing }: Props) {
  const webViewRef = useRef<WebView>(null);
  const playerReadyRef = useRef(false);
  const pendingKeyRef = useRef<string | null>(videoKey);
  // Start with initialVideoKey — YouTube player already has this loaded from URL
  const currentKeyRef = useRef<string | null>(initialVideoKey);
  const playingRef = useRef(playing);
  const bufferingTimerRef = useRef<ReturnType<typeof setTimeout> | null>(null);
  const [buffering, setBuffering] = useState(true);

  const showBufferingAfterDelay = useCallback(() => {
    if (bufferingTimerRef.current) clearTimeout(bufferingTimerRef.current);
    bufferingTimerRef.current = setTimeout(() => setBuffering(true), BUFFERING_OVERLAY_DELAY_MS);
  }, []);

  const cancelBuffering = useCallback(() => {
    if (bufferingTimerRef.current) clearTimeout(bufferingTimerRef.current);
    setBuffering(false);
  }, []);

  useEffect(() => { playingRef.current = playing; }, [playing]);

  useEffect(() => () => { if (bufferingTimerRef.current) clearTimeout(bufferingTimerRef.current); }, []);

  const onMessage = useCallback((e: WebViewMessageEvent) => {
    try {
      const msg = JSON.parse(e.nativeEvent.data);
      if (msg.type === 'playerReady') {
        playerReadyRef.current = true;
        const key = pendingKeyRef.current;
        if (key && key !== currentKeyRef.current) {
          // A different video was queued while the player was initialising
          currentKeyRef.current = key;
          showBufferingAfterDelay();
          webViewRef.current?.injectJavaScript(
            `player.loadVideoById('${key}'); ${playingRef.current ? 'player.playVideo();' : ''} true;`
          );
        } else if (playingRef.current) {
          // Same video as initialVideoKey — already buffered, just play
          webViewRef.current?.injectJavaScript('player.playVideo(); true;');
        } else {
          // Not playing yet (tab not focused) — cue to start buffering without playing
          const k = currentKeyRef.current;
          if (k) webViewRef.current?.injectJavaScript(`player.cueVideoById('${k}'); true;`);
        }
      } else if (msg.type === 'playerPlaying') {
        cancelBuffering();
      }
    } catch {}
  }, []);

  useEffect(() => {
    pendingKeyRef.current = videoKey;
    if (!playerReadyRef.current || !videoKey) return;

    if (videoKey === currentKeyRef.current) {
      // Same video — just play or pause
      webViewRef.current?.injectJavaScript(
        playing ? 'player.playVideo(); true;' : 'player.pauseVideo(); true;'
      );
      return;
    }

    // Different video — schedule overlay (only shows if buffering takes > 300ms)
    currentKeyRef.current = videoKey;
    showBufferingAfterDelay();
    webViewRef.current?.injectJavaScript(
      `player.loadVideoById('${videoKey}'); ${playing ? 'player.playVideo();' : 'player.pauseVideo();'} true;`
    );
  }, [videoKey, playing]);

  return (
    <View style={styles.container} pointerEvents="none">
      <WebView
        ref={webViewRef}
        source={{ uri: `${EMBED_BASE}/${initialVideoKey}?autoplay=0&controls=0` }}
        style={styles.webView}
        allowsInlineMediaPlayback
        mediaPlaybackRequiresUserAction={false}
        scrollEnabled={false}
        bounces={false}
        startInLoadingState={false}
        renderLoading={() => <View style={styles.webView} />}
        onMessage={onMessage}
      />
      {/* Hides YouTube's buffering spinner until video is actually playing */}
      {buffering && <View style={styles.bufferingOverlay} />}
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
  bufferingOverlay: {
    ...StyleSheet.absoluteFillObject,
    backgroundColor: '#000',
  },
});
