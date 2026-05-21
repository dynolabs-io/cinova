import { ExpoConfig, ConfigContext } from 'expo/config';

export default ({ config }: ConfigContext): ExpoConfig => ({
  ...config,
  name: 'Cinova',
  slug: 'cinova',
  version: '1.0.0',
  orientation: 'portrait',
  icon: './assets/icon.png',
  userInterfaceStyle: 'dark',
  splash: {
    image: './assets/splash.png',
    resizeMode: 'contain',
    backgroundColor: '#0A0A0F',
  },
  ios: {
    supportsTablet: false,
    bundleIdentifier: 'io.dynolabs.cinova',
    buildNumber: '1',
    infoPlist: {
      NSCameraUsageDescription: 'Used for profile photo.',
      NSPhotoLibraryUsageDescription: 'Used to select profile photo.',
      UIBackgroundModes: ['remote-notification'],
    },
  },
  android: {
    adaptiveIcon: {
      foregroundImage: './assets/adaptive-icon.png',
      backgroundColor: '#0A0A0F',
    },
    package: 'io.dynolabs.cinova',
    versionCode: 1,
    permissions: [
      'RECEIVE_BOOT_COMPLETED',
      'VIBRATE',
      'POST_NOTIFICATIONS',
    ],
  },
  owner: 'emrahbaysal',
  extra: {
    eas: {
      projectId: '0463a8c8-cd5d-4234-9418-caa2468a3a8e',
    },
  },
  plugins: [
    'expo-router',
    'expo-secure-store',
    [
      'expo-notifications',
      {
        icon: './assets/notification-icon.png',
        color: '#6C5CE7',
      },
    ],
  ],
  scheme: 'cinova',
  experiments: {
    typedRoutes: true,
  },
});
