/**
 * PersistentPlayer — WebView-based YouTube player for the Reels screen.
 *
 * When the video key changes, the WebView navigates to a new embed URL.
 * No JS injection — each video is a fresh page load via our embed endpoint.
 */

import React, { useState, useEffect } from 'react';
import { StyleSheet, Dimensions, View } from 'react-native';
import { WebView } from 'react-native-webview';

const { width: SCREEN_WIDTH, height: SCREEN_HEIGHT } = Dimensions.get('window');
const EMBED_BASE = 'https://api.cinova.openova.io/api/v1/embed';

interface Props {
  initialVideoKey: string;
  videoKey: string | null;
  playing: boolean;
}

export default function PersistentPlayer({ initialVideoKey, videoKey, playing }: Props) {
  // Build the embed URL — changes whenever videoKey changes
  const activeKey = videoKey || initialVideoKey;
  const autoplay = playing ? '1' : '0';
  const uri = `${EMBED_BASE}/${activeKey}?autoplay=${autoplay}&controls=0&mute=0`;

  return (
    <View style={styles.container} pointerEvents="none">
      <WebView
        key={activeKey}
        source={{ uri }}
        style={styles.webView}
        allowsInlineMediaPlayback
        mediaPlaybackRequiresUserAction={false}
        scrollEnabled={false}
        bounces={false}
        startInLoadingState={false}
        renderLoading={() => <View style={styles.webView} />}
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
