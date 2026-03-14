/**
 * Offline banner — shows when network is unavailable
 */

import React, { useEffect, useRef } from 'react';
import {
  View,
  Text,
  StyleSheet,
  Animated,
} from 'react-native';
import NetInfo from '@react-native-community/netinfo';
import { Colors, Typography, Spacing } from '../../constants/theme';

export default function OfflineBanner() {
  const [isOffline, setIsOffline] = React.useState(false);
  const slideAnim = useRef(new Animated.Value(-60)).current;

  useEffect(() => {
    const unsub = NetInfo.addEventListener((state) => {
      const offline = !state.isConnected;
      setIsOffline(offline);
      Animated.spring(slideAnim, {
        toValue: offline ? 0 : -60,
        tension: 60,
        friction: 9,
        useNativeDriver: true,
      }).start();
    });
    return unsub;
  }, []);

  if (!isOffline) return null;

  return (
    <Animated.View
      style={[styles.banner, { transform: [{ translateY: slideAnim }] }]}
    >
      <Text style={styles.text}>No internet connection</Text>
    </Animated.View>
  );
}

const styles = StyleSheet.create({
  banner: {
    position: 'absolute',
    top: 0,
    left: 0,
    right: 0,
    backgroundColor: '#EF4444',
    paddingVertical: Spacing[2],
    alignItems: 'center',
    zIndex: 9999,
  },
  text: {
    fontSize: Typography.sm,
    color: '#FFFFFF',
    fontWeight: '600',
  },
});
