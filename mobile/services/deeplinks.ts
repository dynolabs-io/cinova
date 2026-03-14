/**
 * Deep link handler for streaming providers and in-app navigation.
 *
 * Maps TMDB provider IDs to platform-native deep links.
 * Falls back to JustWatch search if the app is not installed.
 */

import { Linking } from 'react-native';

// TMDB provider ID → deep link URL template
const PROVIDER_DEEP_LINKS: Record<
  number,
  { ios: string; android: string; web: string }
> = {
  8: {
    ios: 'netflix://',
    android:
      'intent://netflix.com#Intent;scheme=https;package=com.netflix.mediaclient;end',
    web: 'https://netflix.com',
  },
  9: {
    ios: 'aiv://',
    android:
      'intent://amazon.com#Intent;scheme=https;package=com.amazon.avod.thirdpartyclient;end',
    web: 'https://primevideo.com',
  },
  337: {
    ios: 'disneyplus://',
    android:
      'intent://disneyplus.com#Intent;scheme=https;package=com.disney.disneyplus;end',
    web: 'https://disneyplus.com',
  },
  350: {
    ios: 'videos://home',
    android:
      'intent://tv.apple.com#Intent;scheme=https;package=com.apple.atve.androidtv.appletv;end',
    web: 'https://tv.apple.com',
  },
  1899: {
    ios: 'hbomax://deeplink',
    android:
      'intent://max.com#Intent;scheme=https;package=com.hbo.hbonow;end',
    web: 'https://max.com',
  },
  15: {
    ios: 'hulu://',
    android:
      'intent://hulu.com#Intent;scheme=https;package=com.hulu.plus;end',
    web: 'https://hulu.com',
  },
  531: {
    ios: 'paramountplus://',
    android:
      'intent://paramountplus.com#Intent;scheme=https;package=com.cbs.ott;end',
    web: 'https://paramountplus.com',
  },
  386: {
    ios: 'nbcsports://',
    android:
      'intent://peacocktv.com#Intent;scheme=https;package=com.peacocktv.peacockandroid;end',
    web: 'https://peacocktv.com',
  },
};

/**
 * Opens the streaming provider app for a given title.
 *
 * 1. Looks up the provider's platform deep link.
 * 2. If the app is installed and the deep link can be opened, launches it.
 * 3. Otherwise falls back to a JustWatch search URL in the browser.
 *
 * @param providerId  TMDB watch provider ID
 * @param tmdbId      TMDB title ID (reserved for future direct-title deep links)
 * @param title       Title name used for JustWatch fallback search
 * @param platform    'ios' | 'android'
 */
export async function openStreamingProvider(
  providerId: number,
  tmdbId: number,
  title: string,
  platform: 'ios' | 'android'
): Promise<void> {
  const links = PROVIDER_DEEP_LINKS[providerId];

  if (links) {
    const deepLink = links[platform];
    const canOpen = await Linking.canOpenURL(deepLink);
    if (canOpen) {
      await Linking.openURL(deepLink);
      return;
    }
    // Try web fallback for the provider before JustWatch
    const canOpenWeb = await Linking.canOpenURL(links.web);
    if (canOpenWeb) {
      await Linking.openURL(links.web);
      return;
    }
  }

  // Final fallback: JustWatch search
  const justWatchUrl = `https://www.justwatch.com/us/search?q=${encodeURIComponent(title)}`;
  await Linking.openURL(justWatchUrl);
}
