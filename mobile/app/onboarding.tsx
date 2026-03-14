/**
 * Onboarding screen — shown once on first launch
 * Animated logo + tagline → fade out → Home
 */

import React, { useEffect, useRef } from 'react';
import {
  View,
  Text,
  StyleSheet,
  Animated,
  Dimensions,
  StatusBar,
  Platform,
} from 'react-native';
import { useRouter } from 'expo-router';
import { useAppStore } from '../store/useAppStore';
import { Colors, Typography } from '../constants/theme';
import { initSession } from '../services/session';

const { width: W, height: H } = Dimensions.get('window');

export default function OnboardingScreen() {
  const router = useRouter();
  const setHasOnboarded = useAppStore((s) => s.setHasOnboarded);
  const setSessionId = useAppStore((s) => s.setSessionId);

  const logoScale = useRef(new Animated.Value(0.6)).current;
  const logoOpacity = useRef(new Animated.Value(0)).current;
  const taglineOpacity = useRef(new Animated.Value(0)).current;
  const screenOpacity = useRef(new Animated.Value(1)).current;

  useEffect(() => {
    // Init anonymous session in background
    initSession().then((sid) => setSessionId(sid)).catch(() => {});

    // Entrance animation
    Animated.sequence([
      Animated.parallel([
        Animated.spring(logoScale, {
          toValue: 1,
          tension: 50,
          friction: 7,
          useNativeDriver: Platform.OS !== 'web',
        }),
        Animated.timing(logoOpacity, {
          toValue: 1,
          duration: 600,
          useNativeDriver: Platform.OS !== 'web',
        }),
      ]),
      Animated.timing(taglineOpacity, {
        toValue: 1,
        duration: 400,
        delay: 200,
        useNativeDriver: true,
      }),
      // Hold
      Animated.delay(1200),
      // Fade out and navigate
      Animated.timing(screenOpacity, {
        toValue: 0,
        duration: 500,
        useNativeDriver: true,
      }),
    ]).start(() => {
      setHasOnboarded(true);
      router.replace('/(tabs)');
    });
  }, []);

  return (
    <Animated.View style={[styles.container, { opacity: screenOpacity }]}>
      <StatusBar barStyle="light-content" />

      <Animated.View
        style={[
          styles.logoContainer,
          { opacity: logoOpacity, transform: [{ scale: logoScale }] },
        ]}
      >
        {/* Cinova "C" mark */}
        <View style={styles.logoMark}>
          <Text style={styles.logoLetter}>C</Text>
        </View>
        <Text style={styles.logoText}>cinova</Text>
      </Animated.View>

      <Animated.Text style={[styles.tagline, { opacity: taglineOpacity }]}>
        Discover what to watch next
      </Animated.Text>
    </Animated.View>
  );
}

const styles = StyleSheet.create({
  container: {
    flex: 1,
    backgroundColor: Colors.background,
    alignItems: 'center',
    justifyContent: 'center',
    gap: 24,
  },
  logoContainer: {
    alignItems: 'center',
    gap: 16,
  },
  logoMark: {
    width: 96,
    height: 96,
    borderRadius: 28,
    backgroundColor: Colors.primary,
    alignItems: 'center',
    justifyContent: 'center',
  },
  logoLetter: {
    fontSize: 56,
    fontWeight: '800',
    color: '#FFFFFF',
    letterSpacing: -2,
  },
  logoText: {
    fontSize: 40,
    fontWeight: '700',
    color: Colors.textPrimary,
    letterSpacing: -1.5,
  },
  tagline: {
    fontSize: 16,
    color: Colors.textSecondary,
    letterSpacing: 0.2,
  },
});
