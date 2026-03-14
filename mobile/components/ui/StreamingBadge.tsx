/**
 * StreamingBadge — Tappable streaming provider badge
 *
 * Shows the provider logo. On tap, attempts the provider's deep link
 * and falls back to the appropriate app store URL.
 */

import React, { useCallback } from 'react';
import {
  TouchableOpacity,
  Text,
  View,
  StyleSheet,
  Platform,
  Linking,
} from 'react-native';
import { Image } from 'expo-image';
import {
  getProviderById,
  PROVIDER_LOGOS,
  StreamingProvider,
} from '../../constants/providers';
import { Colors, Radius, Spacing, Typography } from '../../constants/theme';
import type { WatchProvider } from '../../types';

interface StreamingBadgeProps {
  /** TMDB watch provider object from API */
  provider: WatchProvider;
  /** 'icon' — logo only (for cards), 'full' — logo + name (for detail page) */
  variant?: 'icon' | 'full';
  /** Override size in pixels (icon variant) */
  size?: number;
  onPress?: (provider: WatchProvider) => void;
}

async function openProvider(
  knownProvider: StreamingProvider | undefined,
  fallbackLink: string,
  tmdbId: number
): Promise<void> {
  if (knownProvider) {
    const deepLink = knownProvider.buildDeepLink(tmdbId);
    const canOpen = await Linking.canOpenURL(deepLink);
    if (canOpen) {
      await Linking.openURL(deepLink);
      return;
    }
    // Fall back to app store
    const storeUrl =
      Platform.OS === 'ios'
        ? knownProvider.storeUrl.ios
        : knownProvider.storeUrl.android;
    await Linking.openURL(storeUrl);
    return;
  }

  // Unknown provider — use JustWatch link from API
  if (fallbackLink) {
    await Linking.openURL(fallbackLink);
  }
}

export default function StreamingBadge({
  provider,
  variant = 'icon',
  size = 36,
  onPress,
}: StreamingBadgeProps) {
  const known = getProviderById(provider.providerId);
  const logoUri = known
    ? PROVIDER_LOGOS[known.logoKey]
    : provider.logoPath
    ? `https://image.tmdb.org/t/p/original${provider.logoPath}`
    : PROVIDER_LOGOS.unknown;

  const borderColor = known?.color ?? Colors.border;

  const handlePress = useCallback(async () => {
    if (onPress) {
      onPress(provider);
      return;
    }
    await openProvider(known, provider.link, provider.providerId);
  }, [known, onPress, provider]);

  if (variant === 'full') {
    return (
      <TouchableOpacity
        onPress={handlePress}
        activeOpacity={0.75}
        style={[styles.fullBadge, { borderColor }]}
      >
        <Image
          source={{ uri: logoUri }}
          style={styles.fullLogo}
          contentFit="contain"
          transition={150}
        />
        <Text style={styles.fullName} numberOfLines={1}>
          {known?.name ?? provider.providerName}
        </Text>
      </TouchableOpacity>
    );
  }

  return (
    <TouchableOpacity
      onPress={handlePress}
      activeOpacity={0.75}
      style={[
        styles.iconBadge,
        {
          width: size,
          height: size,
          borderRadius: size * 0.2,
          borderColor,
        },
      ]}
    >
      <Image
        source={{ uri: logoUri }}
        style={{ width: size - 8, height: size - 8 }}
        contentFit="contain"
        transition={150}
      />
    </TouchableOpacity>
  );
}

const styles = StyleSheet.create({
  iconBadge: {
    borderWidth: 1.5,
    backgroundColor: Colors.card,
    justifyContent: 'center',
    alignItems: 'center',
    overflow: 'hidden',
  },
  fullBadge: {
    flexDirection: 'row',
    alignItems: 'center',
    backgroundColor: Colors.card,
    borderWidth: 1.5,
    borderRadius: Radius.md,
    paddingHorizontal: Spacing[3],
    paddingVertical: Spacing[2],
    gap: Spacing[2],
    marginRight: Spacing[2],
  },
  fullLogo: {
    width: 28,
    height: 28,
  },
  fullName: {
    color: Colors.textPrimary,
    fontSize: Typography.sm,
    fontWeight: Typography.medium,
    maxWidth: 110,
  },
});
