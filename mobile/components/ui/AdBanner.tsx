/**
 * AdBanner component
 *
 * Renders an AdMob banner ad for non-premium users.
 * Currently a graceful stub — replace the inner View with BannerAd
 * after installing react-native-google-mobile-ads.
 *
 * Install: npx expo install react-native-google-mobile-ads
 *
 * Then swap the stub block for:
 *   import { BannerAd, BannerAdSize } from 'react-native-google-mobile-ads';
 *   import { AD_UNITS } from '../../services/admob';
 *   <BannerAd unitId={AD_UNITS.banner} size={BannerAdSize.ANCHORED_ADAPTIVE_BANNER} />
 */

import React from 'react';
import { View, Text, StyleSheet } from 'react-native';
import { Colors, Typography, Spacing } from '../../constants/theme';

interface Props {
  isPremium?: boolean;
}

export function AdBanner({ isPremium = false }: Props) {
  if (isPremium) return null;

  return (
    <View style={styles.container}>
      {/* Placeholder until AdMob SDK installed */}
      <View style={styles.adPlaceholder}>
        <Text style={styles.adText}>Advertisement</Text>
      </View>
    </View>
  );
}

const styles = StyleSheet.create({
  container: {
    width: '100%',
    alignItems: 'center',
    backgroundColor: Colors.background,
  },
  adPlaceholder: {
    width: '100%',
    height: 50,
    backgroundColor: Colors.surface,
    borderTopWidth: 1,
    borderBottomWidth: 1,
    borderColor: Colors.border,
    alignItems: 'center',
    justifyContent: 'center',
    paddingVertical: Spacing[2],
  },
  adText: {
    color: Colors.textMuted,
    fontSize: Typography.xs,
    letterSpacing: 1,
  },
});
