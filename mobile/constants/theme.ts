/**
 * Cinova Design System
 *
 * Cinematic dark theme. Deep blacks, rich contrast,
 * high-impact accent colors.
 */

export const Colors = {
  // Backgrounds — true black to near-black layers
  background: '#0a0a0a',
  surface: '#141414',
  card: '#1a1a1a',
  elevated: '#222222',
  overlay: 'rgba(0,0,0,0.7)',
  overlayLight: 'rgba(0,0,0,0.4)',

  // Brand
  primary: '#E50914',         // Cinematic red
  primaryDark: '#B20710',
  primaryLight: '#FF1F2C',

  // Score colors
  scoreHigh: '#21D07A',       // Green — 80+
  scoreMid: '#D2D531',        // Yellow — 60-79
  scoreLow: '#DB2360',        // Red — <60

  // Text
  textPrimary: '#FFFFFF',
  textSecondary: '#A3A3A3',
  textMuted: '#6B6B6B',
  textInverse: '#0a0a0a',

  // Borders / Dividers
  border: '#2A2A2A',
  borderFaint: '#1F1F1F',

  // Tab bar
  tabBarBackground: '#0f0f0f',
  tabBarActive: '#E50914',
  tabBarInactive: '#6B6B6B',

  // Gradient stops
  gradientBlack: '#000000',
  gradientTransparent: 'transparent',

  // Streaming provider accent colors (used for border/badge accents)
  netflix: '#E50914',
  prime: '#00A8E1',
  disney: '#113CCF',
  hulu: '#1CE783',
  hbo: '#5822B4',
  apple: '#FFFFFF',
  peacock: '#FF5F00',
  paramount: '#0064FF',
} as const;

export const Typography = {
  // Font families (system default — swap for custom via expo-font)
  fontSans: undefined,   // System default (SF Pro / Roboto)
  fontMono: undefined,

  // Font sizes
  xs: 11,
  sm: 13,
  base: 15,
  md: 17,
  lg: 20,
  xl: 24,
  '2xl': 28,
  '3xl': 34,
  '4xl': 40,
  '5xl': 48,

  // Aliases used by auth screens and UI components
  xxl: 24,
  xxxl: 32,
  display: 40,

  // Font weights (React Native uses string values)
  thin: '100' as const,
  light: '300' as const,
  regular: '400' as const,
  medium: '500' as const,
  semibold: '600' as const,
  bold: '700' as const,
  extrabold: '800' as const,
  black: '900' as const,

  // Line heights
  tight: 1.15,
  snug: 1.3,
  normal: 1.5,
  relaxed: 1.65,

  // Letter spacing
  tighter: -0.8,
  tight_ls: -0.4,
  normal_ls: 0,
  wide: 0.4,
  wider: 0.8,
  widest: 1.6,
} as const;

export const Spacing = {
  px: 1,
  0.5: 2,
  1: 4,
  1.5: 6,
  2: 8,
  2.5: 10,
  3: 12,
  3.5: 14,
  4: 16,
  5: 20,
  6: 24,
  7: 28,
  8: 32,
  9: 36,
  10: 40,
  12: 48,
  14: 56,
  16: 64,
  20: 80,
  24: 96,
} as const;

export const Radius = {
  none: 0,
  sm: 4,
  base: 8,
  md: 12,
  lg: 16,
  xl: 20,
  '2xl': 24,
  full: 9999,
} as const;

export const Shadows = {
  sm: {
    shadowColor: '#000',
    shadowOffset: { width: 0, height: 1 },
    shadowOpacity: 0.4,
    shadowRadius: 2,
    elevation: 2,
  },
  md: {
    shadowColor: '#000',
    shadowOffset: { width: 0, height: 4 },
    shadowOpacity: 0.5,
    shadowRadius: 8,
    elevation: 6,
  },
  lg: {
    shadowColor: '#000',
    shadowOffset: { width: 0, height: 8 },
    shadowOpacity: 0.6,
    shadowRadius: 16,
    elevation: 12,
  },
  glow: {
    shadowColor: '#E50914',
    shadowOffset: { width: 0, height: 0 },
    shadowOpacity: 0.6,
    shadowRadius: 12,
    elevation: 8,
  },
} as const;

export const Layout = {
  // Card sizes (width)
  cardSm: 100,
  cardMd: 130,
  cardLg: 160,
  cardXl: 200,

  // Hero heights
  heroPct: 0.62,        // % of screen height
  reelPct: 1.0,         // Full screen

  // Carousel
  carouselPadding: 16,
  carouselGap: 10,

  // Tab bar
  tabBarHeight: 60,
} as const;

export const Theme = {
  Colors,
  Typography,
  Spacing,
  Radius,
  Shadows,
  Layout,
} as const;

export default Theme;
