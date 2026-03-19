/**
 * CinovaScore — Circular score badge
 *
 * Clean colored ring with score number centered. Color encodes quality:
 * green (80+), yellow (60-79), red (<60). Same visual language as TMDB/Letterboxd.
 */

import React from 'react';
import { View, Text, StyleSheet } from 'react-native';
import { Colors, Typography } from '../../constants/theme';

interface CinovaScoreProps {
  score: number | null;
  size?: 'sm' | 'md' | 'lg';
}

const SIZE_MAP = {
  sm: { outer: 36, stroke: 3, font: 11, labelSize: 7 },
  md: { outer: 52, stroke: 4, font: 14, labelSize: 8 },
  lg: { outer: 72, stroke: 5, font: 20, labelSize: 10 },
};

function scoreColor(score: number): string {
  if (score >= 80) return Colors.scoreHigh;
  if (score >= 60) return Colors.scoreMid;
  return Colors.scoreLow;
}

export default function CinovaScore({ score, size = 'md' }: CinovaScoreProps) {
  const { outer, stroke, font, labelSize } = SIZE_MAP[size];

  const color = score != null ? scoreColor(score) : Colors.textMuted;
  const displayScore = score != null ? Math.round(score) : '–';

  return (
    <View style={styles.container}>
      <View
        style={[
          styles.ring,
          {
            width: outer,
            height: outer,
            borderRadius: outer / 2,
            borderWidth: stroke,
            borderColor: color,
          },
        ]}
      >
        <Text style={[styles.score, { fontSize: font, color }]}>
          {displayScore}
        </Text>
      </View>
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
    justifyContent: 'center',
    alignItems: 'center',
    backgroundColor: 'rgba(0,0,0,0.55)',
  },
  score: {
    fontWeight: '700',
    textAlign: 'center',
  },
  label: {
    color: Colors.textMuted,
    fontWeight: '600',
    letterSpacing: 1.6,
    textAlign: 'center',
  },
});
