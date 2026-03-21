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
// Lens occupies top-left ~60% of box; handle extends diagonally bottom-right.
function DiscoverIcon({ color, size }: { color: string; size: number }) {
  const sw = Math.max(2.2, size * 0.115);   // stroke width
  const ld = size * 0.58;                    // lens outer diameter
  const hl = size * 0.32;                    // handle length
  // Handle tip offset so it visually connects to lens circle bottom-right
  const hOffset = ld * 0.62;
  return (
    <View style={{ width: size, height: size }}>
      {/* Lens ring */}
      <View style={{
        position: 'absolute',
        top: 0, left: 0,
        width: ld, height: ld,
        borderRadius: ld / 2,
        borderWidth: sw,
        borderColor: color,
      }} />
      {/* Handle — starts at lens bottom-right, 45° diagonal */}
      <View style={{
        position: 'absolute',
        top: hOffset,
        left: hOffset,
        width: sw * 1.1,
        height: hl,
        borderRadius: sw,
        backgroundColor: color,
        transform: [{ rotate: '45deg' }],
        transformOrigin: 'top center',
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

// ── Watchlist (heart) ─────────────────────────────────────────────────────────
// Two filled circles (lobes) + rotated square (bottom point) = classic heart.
function WatchlistIcon({ color, size }: { color: string; size: number }) {
  const lobeD = size * 0.46;    // diameter of each circle lobe
  const squareS = size * 0.54;  // rotated square size
  return (
    <View style={{ width: size, height: size, alignItems: 'center', justifyContent: 'center' }}>
      {/* Rotated square — heart bottom point */}
      <View style={{
        position: 'absolute',
        top: size * 0.23,
        width: squareS, height: squareS,
        backgroundColor: color,
        transform: [{ rotate: '45deg' }],
        borderRadius: 2,
      }} />
      {/* Left lobe */}
      <View style={{
        position: 'absolute',
        top: size * 0.08,
        left: size * 0.04,
        width: lobeD, height: lobeD,
        borderRadius: lobeD / 2,
        backgroundColor: color,
      }} />
      {/* Right lobe */}
      <View style={{
        position: 'absolute',
        top: size * 0.08,
        right: size * 0.04,
        width: lobeD, height: lobeD,
        borderRadius: lobeD / 2,
        backgroundColor: color,
      }} />
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
