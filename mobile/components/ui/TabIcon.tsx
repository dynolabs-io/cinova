/**
 * TabIcon — Tab bar icons rendered as Text glyphs via the CinovaIcons font.
 *
 * Why not <Ionicons> from @expo/vector-icons?
 *  Ionicons checks Font.isLoaded('Ionicons') before rendering. In Expo Go the host binary
 *  pre-registers its own Ionicons version under that name, so loading our version fails and
 *  Font.isLoaded() either returns false (blank) or true-but-wrong-version ("?" glyphs).
 *
 * This component bypasses that entirely:
 *  - _layout.tsx loads the correct Ionicons.ttf under the name "CinovaIcons" (no conflict)
 *  - We render Text with fontFamily="CinovaIcons" + the exact Unicode code point from the
 *    installed @expo/vector-icons glyph map → always the right glyph, no "?", no blanks.
 */

import React from 'react';
import { Text, StyleSheet } from 'react-native';

interface Props {
  name: 'home' | 'discover' | 'watchlist' | 'profile' | 'chat';
  color: string;
  size?: number;
}

// Unicode code points from @expo/vector-icons Ionicons glyph map (installed version).
// These match the Ionicons.ttf bundled at:
//   @expo/vector-icons/build/vendor/react-native-vector-icons/Fonts/Ionicons.ttf
const GLYPH_MAP: Record<Props['name'], number> = {
  home:      62206, // film           (0xF2FE)
  discover:  62084, // compass-outline (0xF284)
  chat:      61971, // chatbubble-ellipses-outline (0xF213)
  watchlist: 62314, // heart          (0xF36A)
  profile:   62634, // person-circle-outline (0xF49A)
};

export default function TabIcon({ name, color, size = 25 }: Props) {
  const glyph = String.fromCodePoint(GLYPH_MAP[name]);
  return (
    <Text style={[styles.icon, { color, fontSize: size }]}>
      {glyph}
    </Text>
  );
}

const styles = StyleSheet.create({
  icon: {
    fontFamily: 'CinovaIcons',
  },
});
