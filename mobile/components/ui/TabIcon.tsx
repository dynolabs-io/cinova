/**
 * TabIcon — Font-free tab bar icons drawn with React Native Views.
 * No @expo/vector-icons, no font loading, zero dependencies.
 */

import React from 'react';
import { View } from 'react-native';

interface Props {
  name: 'home' | 'discover' | 'watchlist' | 'profile' | 'chat';
  color: string;
  size?: number;
}

export default function TabIcon({ name, color, size = 22 }: Props) {
  switch (name) {
    case 'home':      return <HomeIcon color={color} size={size} />;
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

// ── Discover (play circle) ────────────────────────────────────────────────────
function DiscoverIcon({ color, size }: { color: string; size: number }) {
  return (
    <View style={{ width: size, height: size, alignItems: 'center', justifyContent: 'center' }}>
      {/* Outer circle */}
      <View style={{
        width: size * 0.92,
        height: size * 0.92,
        borderRadius: size * 0.46,
        borderWidth: 2,
        borderColor: color,
        alignItems: 'center',
        justifyContent: 'center',
      }}>
        {/* Inner play triangle */}
        <View style={{
          width: 0, height: 0,
          marginLeft: size * 0.06,
          borderTopWidth: size * 0.22,
          borderBottomWidth: size * 0.22,
          borderLeftWidth: size * 0.34,
          borderTopColor: 'transparent',
          borderBottomColor: 'transparent',
          borderLeftColor: color,
        }} />
      </View>
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

// ── Watchlist (bookmark) ──────────────────────────────────────────────────────
function WatchlistIcon({ color, size }: { color: string; size: number }) {
  const w = size * 0.58;
  const h = size * 0.82;
  return (
    <View style={{ width: size, height: size, alignItems: 'center', justifyContent: 'center' }}>
      {/* Bookmark body */}
      <View style={{
        width: w,
        height: h,
        borderWidth: 2.5,
        borderColor: color,
        borderBottomWidth: 0,
      }}>
        {/* V-notch: two triangles */}
        <View style={{
          position: 'absolute',
          bottom: -(w * 0.5 - 1),
          left: -1,
          width: 0, height: 0,
          borderLeftWidth: w / 2,
          borderRightWidth: w / 2,
          borderTopWidth: w * 0.5,
          borderLeftColor: 'transparent',
          borderRightColor: color,
          borderTopColor: color,
        }} />
        <View style={{
          position: 'absolute',
          bottom: -(w * 0.5 - 1),
          left: -1,
          width: 0, height: 0,
          borderLeftWidth: w / 2,
          borderRightWidth: w / 2,
          borderTopWidth: w * 0.5,
          borderLeftColor: color,
          borderRightColor: 'transparent',
          borderTopColor: color,
        }} />
      </View>
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
