/**
 * TabIcon — Font-free tab bar icons drawn with React Native Views.
 * No @expo/vector-icons, no font loading, zero dependencies.
 */

import React from 'react';
import { View, Text } from 'react-native';

interface Props {
  name: 'home' | 'reels' | 'discover' | 'watchlist' | 'profile' | 'chat';
  color: string;
  size?: number;
}

export default function TabIcon({ name, color, size = 22 }: Props) {
  switch (name) {
    case 'home':      return <HomeIcon color={color} size={size} />;
    case 'reels':     return <ReelsIcon color={color} size={size} />;
    case 'discover':  return <DiscoverIcon color={color} size={size} />;
    case 'chat':      return <ChatIcon color={color} size={size} />;
    case 'watchlist': return <WatchlistIcon color={color} size={size} />;
    case 'profile':   return <ProfileIcon color={color} size={size} />;
  }
}

// ── Home (house) ──────────────────────────────────────────────────────────────
function HomeIcon({ color, size }: { color: string; size: number }) {
  const r = size * 0.47;
  return (
    <View style={{ width: size, height: size, alignItems: 'center', justifyContent: 'flex-end' }}>
      {/* Roof */}
      <View style={{
        width: 0, height: 0,
        borderLeftWidth: r,
        borderRightWidth: r,
        borderBottomWidth: r * 0.85,
        borderLeftColor: 'transparent',
        borderRightColor: 'transparent',
        borderBottomColor: color,
        marginBottom: 1,
      }} />
      {/* Body */}
      <View style={{
        width: size * 0.62,
        height: size * 0.42,
        backgroundColor: color,
      }} />
    </View>
  );
}

// ── Reels — filled play circle ────────────────────────────────────────────────
function ReelsIcon({ color, size }: { color: string; size: number }) {
  const d = size * 0.86;
  return (
    <View style={{ width: size, height: size, alignItems: 'center', justifyContent: 'center' }}>
      <View style={{
        width: d, height: d,
        borderRadius: d / 2,
        backgroundColor: color,
        alignItems: 'center',
        justifyContent: 'center',
      }}>
        <View style={{
          width: 0, height: 0,
          marginLeft: d * 0.08,
          borderTopWidth: d * 0.22,
          borderBottomWidth: d * 0.22,
          borderLeftWidth: d * 0.35,
          borderTopColor: 'transparent',
          borderBottomColor: 'transparent',
          borderLeftColor: '#000',
        }} />
      </View>
    </View>
  );
}

// ── Discover — search magnifier ────────────────────────────────────────────────
// Matches catalog row-1 "Search": lens centered at 43% of box, long handle to corner.
function DiscoverIcon({ color, size }: { color: string; size: number }) {
  const sw = Math.max(2, size * 0.095);     // stroke width (≈2.5/44 scaled)
  const r  = size * 0.273;                  // lens radius  (12/44 scaled)
  const ld = r * 2;
  // Lens center at 43% of icon (19/44 scaled)
  const cx = size * 0.432;
  // Handle: line from (28/44, 28/44) → (38/44, 38/44) of icon box, 45° diagonal
  const hLen = size * 0.321;               // sqrt(10²+10²)/44 * size
  const hCX  = size * 0.75;               // handle center x (33/44)
  const hCY  = size * 0.75;               // handle center y
  return (
    <View style={{ width: size, height: size }}>
      {/* Lens ring */}
      <View style={{
        position: 'absolute',
        top:  cx - r,
        left: cx - r,
        width: ld, height: ld,
        borderRadius: r,
        borderWidth: sw,
        borderColor: color,
      }} />
      {/* Handle — rotated 45° around its own center → visual center stays at hCX, hCY */}
      <View style={{
        position: 'absolute',
        top:  hCY - hLen / 2,
        left: hCX - sw / 2,
        width: sw * 1.3,
        height: hLen,
        borderRadius: sw,
        backgroundColor: color,
        transform: [{ rotate: '45deg' }],
      }} />
    </View>
  );
}

// ── Chat (speech bubble) ──────────────────────────────────────────────────────
function ChatIcon({ color, size }: { color: string; size: number }) {
  return (
    <View style={{ width: size, height: size, alignItems: 'center', justifyContent: 'center' }}>
      {/* Bubble body */}
      <View style={{
        position: 'absolute',
        top: size * 0.04,
        left: 0,
        width: size * 0.94,
        height: size * 0.72,
        borderRadius: size * 0.18,
        borderWidth: 2.5,
        borderColor: color,
      }} />
      {/* Tail */}
      <View style={{
        position: 'absolute',
        bottom: size * 0.04,
        left: size * 0.12,
        width: 0,
        height: 0,
        borderTopWidth: size * 0.22,
        borderRightWidth: size * 0.16,
        borderTopColor: color,
        borderRightColor: 'transparent',
      }} />
    </View>
  );
}

// ── Watchlist (star) ──────────────────────────────────────────────────────────
function WatchlistIcon({ color, size }: { color: string; size: number }) {
  return (
    <View style={{ width: size, height: size, alignItems: 'center', justifyContent: 'center' }}>
      <Text style={{ color, fontSize: size * 0.92, lineHeight: size, textAlign: 'center', includeFontPadding: false }}>
        ★
      </Text>
    </View>
  );
}

// ── Profile (person) ──────────────────────────────────────────────────────────
function ProfileIcon({ color, size }: { color: string; size: number }) {
  const headSize = size * 0.38;
  return (
    <View style={{ width: size, height: size, alignItems: 'center', justifyContent: 'flex-end', paddingBottom: 1 }}>
      {/* Head */}
      <View style={{
        width: headSize,
        height: headSize,
        borderRadius: headSize / 2,
        borderWidth: 2.5,
        borderColor: color,
        marginBottom: 3,
      }} />
      {/* Shoulders: half-ellipse */}
      <View style={{
        width: size * 0.78,
        height: size * 0.34,
        borderTopLeftRadius: size * 0.39,
        borderTopRightRadius: size * 0.39,
        borderWidth: 2.5,
        borderBottomWidth: 0,
        borderColor: color,
      }} />
    </View>
  );
}
