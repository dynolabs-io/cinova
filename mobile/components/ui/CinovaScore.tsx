/**
 * CinovaScore — Circular score badge
 *
 * Renders a thin progress ring using a View-based arc technique
 * (no external SVG dependency). The score number is centered,
 * with a "CINOVA" label beneath in small caps.
 */

import React, { useMemo } from 'react';
import { View, Text, StyleSheet } from 'react-native';
import { Colors, Typography } from '../../constants/theme';

interface CinovaScoreProps {
  score: number | null;
  size?: 'sm' | 'md' | 'lg';
}

const SIZE_MAP = {
  sm: { outer: 36, stroke: 3, font: Typography.xs, labelSize: 7 },
  md: { outer: 52, stroke: 4, font: Typography.sm, labelSize: 8 },
  lg: { outer: 72, stroke: 5, font: Typography.lg, labelSize: 10 },
};

function scoreColor(score: number): string {
  if (score >= 80) return Colors.scoreHigh;
  if (score >= 60) return Colors.scoreMid;
  return Colors.scoreLow;
}

/**
 * Pure View-based progress ring.
 *
 * Technique: two half-circles (left + right) clipped with overflow:hidden,
 * rotated to represent progress. Works without react-native-svg.
 */
export default function CinovaScore({ score, size = 'md' }: CinovaScoreProps) {
  const { outer, stroke, font, labelSize } = SIZE_MAP[size];

  const clampedScore = useMemo(
    () => (score != null ? Math.min(Math.max(score, 0), 100) : 0),
    [score]
  );

  const color = score != null ? scoreColor(score) : Colors.textMuted;
  const displayScore = score != null ? Math.round(score) : '–';

  // For simplicity use a solid ring indicator rather than a segmented arc —
  // a colored border on a circle with a cutout gap scaled by score.
  // Full ring = score 100, 3/4 ring = 75, etc.
  // We use a View border trick: border color split between colored and track.

  const inner = outer - stroke * 2;
  const pct = clampedScore / 100;

  // Rotate a half-circle overlay to mask progress
  const rightDeg = pct >= 0.5 ? 180 : pct * 360;
  const leftDeg = pct >= 0.5 ? (pct - 0.5) * 360 : 0;

  return (
    <View style={[styles.container, { width: outer, height: outer + labelSize + 4 }]}>
      {/* Background full ring */}
      <View
        style={[
          styles.ring,
          {
            width: outer,
            height: outer,
            borderRadius: outer / 2,
            borderWidth: stroke,
            borderColor: Colors.border,
          },
        ]}
      >
        {/* Right half fill */}
        <View
          style={[
            styles.halfClip,
            { width: outer / 2, height: outer, left: outer / 2 },
          ]}
        >
          <View
            style={[
              styles.halfCircle,
              {
                width: outer,
                height: outer,
                borderRadius: outer / 2,
                borderWidth: stroke,
                borderColor: pct > 0 ? color : 'transparent',
                transform: [{ rotate: `${rightDeg}deg` }],
              },
            ]}
          />
        </View>

        {/* Left half fill */}
        <View
          style={[
            styles.halfClip,
            { width: outer / 2, height: outer, left: 0 },
          ]}
        >
          <View
            style={[
              styles.halfCircle,
              {
                width: outer,
                height: outer,
                borderRadius: outer / 2,
                borderWidth: stroke,
                borderColor: pct > 0.5 ? color : 'transparent',
                transform: [{ rotate: `${leftDeg}deg` }],
              },
            ]}
          />
        </View>

        {/* Inner cutout */}
        <View
          style={[
            styles.inner,
            {
              width: inner,
              height: inner,
              borderRadius: inner / 2,
              backgroundColor: 'transparent',
            },
          ]}
        />
      </View>

      {/* Score text */}
      <View style={[styles.scoreOverlay, { width: outer, height: outer }]}>
        <Text style={[styles.score, { fontSize: font, color }]}>
          {displayScore}
        </Text>
      </View>

      {/* CINOVA label */}
      <Text style={[styles.label, { fontSize: labelSize, marginTop: 2 }]}>
        CINOVA
      </Text>
    </View>
  );
}

const styles = StyleSheet.create({
  container: {
    alignItems: 'center',
  },
  ring: {
    position: 'relative',
    justifyContent: 'center',
    alignItems: 'center',
    overflow: 'hidden',
    backgroundColor: 'transparent',
  },
  halfClip: {
    position: 'absolute',
    overflow: 'hidden',
    top: 0,
  },
  halfCircle: {
    position: 'absolute',
    top: 0,
    left: 0,
    backgroundColor: 'transparent',
  },
  inner: {
    position: 'absolute',
  },
  scoreOverlay: {
    position: 'absolute',
    top: 0,
    left: 0,
    justifyContent: 'center',
    alignItems: 'center',
  },
  score: {
    fontWeight: Typography.bold,
    textAlign: 'center',
    zIndex: 2,
  },
  label: {
    color: Colors.textMuted,
    fontWeight: Typography.semibold,
    letterSpacing: Typography.widest,
    textAlign: 'center',
  },
});
