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

// ── Reels — large filled play circle (TikTok-style center tab) ────────────────
function ReelsIcon({ color, size }: { color: string; size: number }) {
  return (
    <View style={{ width: size, height: size, alignItems: 'center', justifyContent: 'center' }}>
      {/* Filled circle */}
      <View style={{
        width: size * 0.94,
        height: size * 0.94,
        borderRadius: size * 0.47,
        backgroundColor: color,
        alignItems: 'center',
        justifyContent: 'center',
      }}>
        {/* Play triangle — slightly right-offset for visual balance */}
        <View style={{
          width: 0, height: 0,
          marginLeft: size * 0.06,
          borderTopWidth: size * 0.2,
          borderBottomWidth: size * 0.2,
          borderLeftWidth: size * 0.32,
          borderTopColor: 'transparent',
          borderBottomColor: 'transparent',
          borderLeftColor: '#000',
        }} />
      </View>
    </View>
  );
}

// ── Discover — search magnifier ────────────────────────────────────────────────
function DiscoverIcon({ color, size }: { color: string; size: number }) {
  const lens = size * 0.6;
  const strokeW = size * 0.12;
  return (
    <View style={{ width: size, height: size, alignItems: 'center', justifyContent: 'center' }}>
      {/* Lens circle — ring only */}
      <View style={{
        position: 'absolute',
        top: 0,
        left: 0,
        width: lens,
        height: lens,
        borderRadius: lens / 2,
        borderWidth: strokeW,
        borderColor: color,
      }} />
      {/* Handle — rotated bar at bottom-right */}
      <View style={{
        position: 'absolute',
        bottom: size * 0.02,
        right: size * 0.02,
        width: strokeW,
        height: size * 0.38,
        borderRadius: strokeW / 2,
        backgroundColor: color,
        transform: [{ rotate: '-45deg' }],
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

// ── Watchlist (heart) ────────────────────────────────────────────────────────
// Unicode ♥ renders via the system font — no custom font loading needed.
function WatchlistIcon({ color, size }: { color: string; size: number }) {
  return (
    <View style={{ width: size, height: size, alignItems: 'center', justifyContent: 'center' }}>
      <Text style={{ color, fontSize: size * 0.9, lineHeight: size }}>♥</Text>
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
