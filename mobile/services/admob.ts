/**
 * AdMob service
 *
 * Uses react-native-google-mobile-ads.
 * Install: npx expo install react-native-google-mobile-ads
 *
 * Until installed, this file exports constants and stubs only.
 * Once the SDK is added, uncomment the imports below.
 */

// import {
//   InterstitialAd,
//   AdEventType,
//   BannerAd,
//   BannerAdSize,
//   TestIds,
// } from 'react-native-google-mobile-ads';

// Inline stubs so the file compiles without the SDK installed
const TestIds = {
  BANNER: 'ca-app-pub-3940256099942544/6300978111',
  INTERSTITIAL: 'ca-app-pub-3940256099942544/1033173712',
  REWARDED: 'ca-app-pub-3940256099942544/5224354917',
};

const IS_PROD = !__DEV__;

export const AD_UNITS = {
  banner: IS_PROD ? 'ca-app-pub-PLACEHOLDER/banner' : TestIds.BANNER,
  interstitial: IS_PROD
    ? 'ca-app-pub-PLACEHOLDER/interstitial'
    : TestIds.INTERSTITIAL,
  rewarded: IS_PROD ? 'ca-app-pub-PLACEHOLDER/rewarded' : TestIds.REWARDED,
};

/**
 * Creates a preloaded interstitial ad.
 *
 * TODO: Uncomment after installing react-native-google-mobile-ads:
 *
 *   const ad = InterstitialAd.createForAdRequest(AD_UNITS.interstitial, {
 *     requestNonPersonalizedAdsOnly: true,
 *   });
 *   return ad;
 */
export function createInterstitial(): null {
  // Stub — replace return type and body after SDK install
  return null;
}
