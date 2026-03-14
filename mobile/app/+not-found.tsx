/**
 * 404 — Not Found screen
 *
 * Shown when expo-router cannot match a route.
 * Returns user to the home tab.
 */

import React from 'react';
import { View, Text, TouchableOpacity, StyleSheet } from 'react-native';
import { useRouter } from 'expo-router';
import { useSafeAreaInsets } from 'react-native-safe-area-context';
import { Feather } from '@expo/vector-icons';
import { Colors, Typography, Spacing, Radius } from '../constants/theme';

export default function NotFoundScreen() {
  const router = useRouter();
  const insets = useSafeAreaInsets();

  return (
    <View
      style={[
        styles.container,
        { paddingTop: insets.top, paddingBottom: insets.bottom },
      ]}
    >
      <Feather name="film" size={64} color={Colors.primary} style={styles.icon} />

      <Text style={styles.code}>404</Text>
      <Text style={styles.title}>Page Not Found</Text>
      <Text style={styles.subtitle}>
        The screen you're looking for doesn't exist or has been moved.
      </Text>

      <TouchableOpacity
        style={styles.homeBtn}
        onPress={() => router.replace('/(tabs)' as never)}
        activeOpacity={0.85}
      >
        <Feather name="home" size={18} color={Colors.textPrimary} />
        <Text style={styles.homeBtnText}>Go to Home</Text>
      </TouchableOpacity>
    </View>
  );
}

const styles = StyleSheet.create({
  container: {
    flex: 1,
    backgroundColor: Colors.background,
    justifyContent: 'center',
    alignItems: 'center',
    paddingHorizontal: Spacing[8],
    gap: Spacing[3],
  },
  icon: {
    marginBottom: Spacing[4],
    opacity: 0.8,
  },
  code: {
    color: Colors.primary,
    fontSize: Typography['5xl'],
    fontWeight: Typography.black,
    letterSpacing: -2,
  },
  title: {
    color: Colors.textPrimary,
    fontSize: Typography['2xl'],
    fontWeight: Typography.bold,
    textAlign: 'center',
  },
  subtitle: {
    color: Colors.textSecondary,
    fontSize: Typography.base,
    textAlign: 'center',
    lineHeight: Typography.base * 1.6,
  },
  homeBtn: {
    flexDirection: 'row',
    alignItems: 'center',
    gap: Spacing[2],
    backgroundColor: Colors.primary,
    borderRadius: Radius.md,
    paddingHorizontal: Spacing[8],
    paddingVertical: Spacing[3],
    marginTop: Spacing[4],
  },
  homeBtnText: {
    color: Colors.textPrimary,
    fontSize: Typography.base,
    fontWeight: Typography.bold,
  },
});
