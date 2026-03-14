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
    bundleIdentifier: 'io.openova.cinova',
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
    package: 'io.openova.cinova',
    versionCode: 1,
    googleServicesFile: './google-services.json',
    permissions: [
      'RECEIVE_BOOT_COMPLETED',
      'VIBRATE',
      'POST_NOTIFICATIONS',
    ],
  },
  extra: {
    eas: {
      projectId: '',
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
        sounds: ['./assets/notification.wav'],
      },
    ],
  ],
  scheme: 'cinova',
  experiments: {
    typedRoutes: true,
  },
});
