/**
 * TabIcon — Tab bar icons using Ionicons for consistent, professional look.
 */

import React from 'react';
import { Ionicons } from '@expo/vector-icons';

interface Props {
  name: 'home' | 'discover' | 'watchlist' | 'profile' | 'chat';
  color: string;
  size?: number;
}

const ICON_MAP: Record<Props['name'], keyof typeof Ionicons.glyphMap> = {
  home:      'film',
  discover:  'compass-outline',
  chat:      'chatbubble-ellipses-outline',
  watchlist: 'heart',
  profile:   'person-circle-outline',
};

export default function TabIcon({ name, color, size = 25 }: Props) {
  return <Ionicons name={ICON_MAP[name]} size={size} color={color} />;
}
