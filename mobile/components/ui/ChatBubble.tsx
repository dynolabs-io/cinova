/**
 * ChatBubble — renders a single chat message turn.
 * User messages appear right-aligned; assistant messages left-aligned.
 * Assistant bubbles include an optional list of MovieCard suggestions below the text.
 */

import React from 'react';
import { View, Text, StyleSheet, TouchableOpacity, Image } from 'react-native';
import { useRouter } from 'expo-router';
import type { ChatSuggestion } from '../../services/api';
import { Colors } from '../../constants/theme';

const TMDB_IMAGE = 'https://image.tmdb.org/t/p/w185';

interface Props {
  role: 'user' | 'assistant';
  content: string;
  suggestions?: ChatSuggestion[];
}

export default function ChatBubble({ role, content, suggestions }: Props) {
  const isUser = role === 'user';
  return (
    <View style={[styles.row, isUser ? styles.rowUser : styles.rowAssistant]}>
      {!isUser && <AssistantAvatar />}
      <View style={styles.bubbleCol}>
        <View style={[styles.bubble, isUser ? styles.bubbleUser : styles.bubbleAssistant]}>
          <Text style={[styles.text, isUser ? styles.textUser : styles.textAssistant]}>
            {content}
          </Text>
        </View>
        {!isUser && suggestions && suggestions.length > 0 && (
          <SuggestionRow suggestions={suggestions} />
        )}
      </View>
    </View>
  );
}

// ── Avatar ────────────────────────────────────────────────────────────────────

function AssistantAvatar() {
  return (
    <View style={styles.avatar}>
      <Text style={styles.avatarText}>C</Text>
    </View>
  );
}

// ── Suggestion row ────────────────────────────────────────────────────────────

function SuggestionRow({ suggestions }: { suggestions: ChatSuggestion[] }) {
  return (
    <View style={styles.suggestions}>
      {suggestions.map((s) => (
        <SuggestionCard key={s.tmdbId} suggestion={s} />
      ))}
    </View>
  );
}

function SuggestionCard({ suggestion: s }: { suggestion: ChatSuggestion }) {
  const router = useRouter();
  const poster = s.posterPath ? `${TMDB_IMAGE}${s.posterPath}` : null;
  const streamers = (s.providers ?? [])
    .filter((p) => p.type === 'flatrate')
    .map((p) => p.providerName)
    .filter((v, i, a) => a.indexOf(v) === i)
    .slice(0, 2)
    .join(' · ');

  return (
    <TouchableOpacity
      style={styles.card}
      activeOpacity={0.85}
      onPress={() => router.push(`/movie/${s.tmdbId}`)}
    >
      {poster ? (
        <Image source={{ uri: poster }} style={styles.poster} />
      ) : (
        <View style={[styles.poster, styles.posterFallback]}>
          <Text style={styles.posterFallbackText}>🎬</Text>
        </View>
      )}
      <View style={styles.cardInfo}>
        <Text style={styles.cardTitle} numberOfLines={2}>{s.title}</Text>
        {s.releaseYear && (
          <Text style={styles.cardMeta}>{s.releaseYear}{streamers ? ` · ${streamers}` : ''}</Text>
        )}
        {s.reason ? (
          <Text style={styles.cardReason} numberOfLines={3}>{s.reason}</Text>
        ) : s.cinovaSynopsis ? (
          <Text style={styles.cardReason} numberOfLines={3}>{s.cinovaSynopsis}</Text>
        ) : null}
        {s.cinovaScore != null && s.cinovaScore > 0 && (
          <View style={styles.scoreRow}>
            <View style={styles.scoreBadge}>
              <Text style={styles.scoreText}>{Math.round(s.cinovaScore)}</Text>
            </View>
          </View>
        )}
      </View>
    </TouchableOpacity>
  );
}

// ── Styles ────────────────────────────────────────────────────────────────────

const styles = StyleSheet.create({
  row: {
    flexDirection: 'row',
    marginVertical: 6,
    paddingHorizontal: 12,
    alignItems: 'flex-start',
  },
  rowUser: { justifyContent: 'flex-end' },
  rowAssistant: { justifyContent: 'flex-start' },
  bubbleCol: { flex: 1, maxWidth: '88%' },
  bubble: {
    borderRadius: 18,
    paddingHorizontal: 14,
    paddingVertical: 10,
  },
  bubbleUser: {
    alignSelf: 'flex-end',
    backgroundColor: '#2563EB',
    borderBottomRightRadius: 4,
  },
  bubbleAssistant: {
    alignSelf: 'flex-start',
    backgroundColor: '#1a1a1a',
    borderBottomLeftRadius: 4,
    borderWidth: 1,
    borderColor: 'rgba(255,255,255,0.08)',
  },
  text: { fontSize: 15, lineHeight: 22 },
  textUser: { color: '#fff' },
  textAssistant: { color: '#e5e5e5' },

  // Avatar
  avatar: {
    width: 28,
    height: 28,
    borderRadius: 14,
    backgroundColor: '#2563EB',
    alignItems: 'center',
    justifyContent: 'center',
    marginRight: 8,
    marginTop: 2,
    flexShrink: 0,
  },
  avatarText: { color: '#fff', fontSize: 13, fontWeight: '700' },

  // Suggestions
  suggestions: { marginTop: 8, gap: 8 },
  card: {
    flexDirection: 'row',
    backgroundColor: '#141414',
    borderRadius: 12,
    overflow: 'hidden',
    borderWidth: 1,
    borderColor: 'rgba(255,255,255,0.08)',
  },
  poster: { width: 70, height: 105 },
  posterFallback: {
    backgroundColor: '#222',
    alignItems: 'center',
    justifyContent: 'center',
  },
  posterFallbackText: { fontSize: 24 },
  cardInfo: { flex: 1, padding: 10, justifyContent: 'flex-start' },
  cardTitle: { color: '#fff', fontSize: 14, fontWeight: '600', marginBottom: 2 },
  cardMeta: { color: '#888', fontSize: 12, marginBottom: 4 },
  cardReason: { color: '#bbb', fontSize: 12, lineHeight: 17 },
  scoreRow: { marginTop: 6 },
  scoreBadge: {
    alignSelf: 'flex-start',
    backgroundColor: 'rgba(37,99,235,0.25)',
    borderRadius: 6,
    paddingHorizontal: 6,
    paddingVertical: 2,
  },
  scoreText: { color: '#60a5fa', fontSize: 11, fontWeight: '700' },
});
