/**
 * Chat screen — AI film concierge powered by Cinova's graph-based RAG.
 * Two-pass backend: intent extraction → Neo4j candidates → Claude recommendation.
 */

import React, { useState, useRef, useCallback, useEffect } from 'react';
import {
  View,
  Text,
  StyleSheet,
  TextInput,
  TouchableOpacity,
  FlatList,
  KeyboardAvoidingView,
  Platform,
  Animated,
  SafeAreaView,
} from 'react-native';
import { useSafeAreaInsets } from 'react-native-safe-area-context';
import { streamChatMessage } from '../../services/api';
import type { ChatSuggestion } from '../../services/api';
import ChatBubble from '../../components/ui/ChatBubble';
import { Colors } from '../../constants/theme';

// ── Types ─────────────────────────────────────────────────────────────────────

interface Message {
  id: string;
  role: 'user' | 'assistant';
  content: string;
  suggestions?: ChatSuggestion[];
}

// ── Quick-reply chips shown before first message ───────────────────────────────

const QUICK_REPLIES = [
  "What should I watch tonight?",
  "Something like Parasite but lighter",
  "Best thrillers on Netflix",
  "I want something that'll make me cry",
];

// ── Component ─────────────────────────────────────────────────────────────────

export default function ChatScreen() {
  const insets = useSafeAreaInsets();
  const [messages, setMessages] = useState<Message[]>([]);
  const [input, setInput] = useState('');
  const [loading, setLoading] = useState(false);
  const [streamingMsgId, setStreamingMsgId] = useState<string | null>(null);
  const [isStatusPhase, setIsStatusPhase] = useState(false);
  const [convId, setConvId] = useState<string | undefined>();
  const listRef = useRef<FlatList>(null);

  const scrollToBottom = useCallback(() => {
    setTimeout(() => listRef.current?.scrollToEnd({ animated: true }), 100);
  }, []);

  const sendMessage = useCallback((text: string) => {
    const trimmed = text.trim();
    if (!trimmed || loading) return;

    const userMsg: Message = { id: `u-${Date.now()}`, role: 'user', content: trimmed };
    const assistantId = `a-${Date.now()}`;
    const assistantMsg: Message = { id: assistantId, role: 'assistant', content: '' };

    const initialStatus = 'Analyzing your request…';
    setMessages((prev) => [...prev, userMsg, { ...assistantMsg, content: initialStatus }]);
    setInput('');
    setLoading(true);
    setStreamingMsgId(assistantId);
    setIsStatusPhase(true);
    scrollToBottom();

    let firstDelta = true;
    let cycleInterval: ReturnType<typeof setInterval> | null = null;

    const WRITING_CYCLE = [
      'Writing recommendations…',
      'Thinking about what you\'d enjoy…',
      'Crafting personalised picks…',
      'Almost there…',
    ];

    const clearCycle = () => {
      if (cycleInterval !== null) {
        clearInterval(cycleInterval);
        cycleInterval = null;
      }
    };

    streamChatMessage(
      trimmed,
      convId,
      'US',
      // onDelta — first delta switches out of status phase, then appends real content
      (chunk) => {
        clearCycle();
        const wasFirst = firstDelta;
        if (firstDelta) {
          setIsStatusPhase(false);
          firstDelta = false;
        }
        setMessages((prev) =>
          prev.map((m) => {
            if (m.id !== assistantId) return m;
            return { ...m, content: wasFirst ? chunk : m.content + chunk };
          })
        );
      },
      // onSuggestions — attach movie cards to the assistant bubble
      (suggestions, newConvId) => {
        if (newConvId) setConvId(newConvId);
        setMessages((prev) =>
          prev.map((m) => m.id === assistantId ? { ...m, suggestions: suggestions as ChatSuggestion[] } : m)
        );
        scrollToBottom();
      },
      // onDone
      () => {
        clearCycle();
        setLoading(false);
        setStreamingMsgId(null);
        setIsStatusPhase(false);
        scrollToBottom();
      },
      // onError
      () => {
        clearCycle();
        setMessages((prev) =>
          prev.map((m) =>
            m.id === assistantId
              ? { ...m, content: "Sorry, I'm having trouble connecting right now. Please try again." }
              : m
          )
        );
        setLoading(false);
        setStreamingMsgId(null);
        setIsStatusPhase(false);
      },
      // onStatus — update the bubble content while still in status phase; start
      // cycling messages when we reach the slow "Writing recommendations…" phase
      (statusText) => {
        clearCycle();
        setMessages((prev) =>
          prev.map((m) => m.id === assistantId ? { ...m, content: statusText } : m)
        );
        if (statusText === 'Writing recommendations…') {
          let idx = 1;
          cycleInterval = setInterval(() => {
            const next = WRITING_CYCLE[idx % WRITING_CYCLE.length];
            idx++;
            setMessages((prev) =>
              prev.map((m) => m.id === assistantId ? { ...m, content: next } : m)
            );
          }, 4500);
        }
      },
    );
  }, [loading, convId, scrollToBottom]);

  const isEmpty = messages.length === 0;

  return (
    <SafeAreaView style={styles.safe} edges={['top']}>
      <KeyboardAvoidingView
        style={styles.flex}
        behavior={Platform.OS === 'ios' ? 'padding' : 'height'}
        keyboardVerticalOffset={Platform.OS === 'ios' ? 0 : 0}
      >
        {/* Header */}
        <View style={styles.header}>
          <Text style={styles.headerTitle}>Cinova</Text>
          <Text style={styles.headerSub}>Your AI film concierge</Text>
        </View>

        {/* Message list */}
        <FlatList
          ref={listRef}
          data={messages}
          keyExtractor={(m) => m.id}
          renderItem={({ item }) => {
            if (item.id === streamingMsgId && isStatusPhase) {
              return <StatusBubble text={item.content} />;
            }
            return (
              <ChatBubble
                role={item.role}
                content={item.content}
                suggestions={item.suggestions}
              />
            );
          }}
          contentContainerStyle={[
            styles.listContent,
            isEmpty && styles.listContentEmpty,
          ]}
          ListEmptyComponent={<EmptyState />}
          onContentSizeChange={scrollToBottom}
        />

        {/* Quick replies — only before first message */}
        {isEmpty && !loading && (
          <View style={styles.quickReplies}>
            {QUICK_REPLIES.map((q) => (
              <TouchableOpacity
                key={q}
                style={styles.chip}
                activeOpacity={0.75}
                onPress={() => sendMessage(q)}
              >
                <Text style={styles.chipText}>{q}</Text>
              </TouchableOpacity>
            ))}
          </View>
        )}

        {/* Typing indicator */}

        {/* Input bar */}
        <View style={[styles.inputBar, { paddingBottom: Math.max(insets.bottom, 12) }]}>
          <TextInput
            style={styles.input}
            value={input}
            onChangeText={setInput}
            placeholder="Ask me anything about films…"
            placeholderTextColor="#555"
            multiline
            maxLength={500}
            returnKeyType="send"
            blurOnSubmit={false}
            onSubmitEditing={() => sendMessage(input)}
          />
          <TouchableOpacity
            style={[styles.sendBtn, (!input.trim() || loading) && styles.sendBtnDisabled]}
            activeOpacity={0.8}
            onPress={() => sendMessage(input)}
            disabled={!input.trim() || loading}
          >
            <SendArrow />
          </TouchableOpacity>
        </View>
      </KeyboardAvoidingView>
    </SafeAreaView>
  );
}

// ── Animated status bubble ────────────────────────────────────────────────────

function AnimatedDots() {
  const d1 = useRef(new Animated.Value(0)).current;
  const d2 = useRef(new Animated.Value(0)).current;
  const d3 = useRef(new Animated.Value(0)).current;

  useEffect(() => {
    // Each dot: delay → bounce up → bounce down → pause. All 1500ms cycles.
    const bounce = (d: Animated.Value, delay: number) =>
      Animated.loop(
        Animated.sequence([
          Animated.delay(delay),
          Animated.timing(d, { toValue: -5, duration: 280, useNativeDriver: true }),
          Animated.timing(d, { toValue: 0, duration: 280, useNativeDriver: true }),
          Animated.delay(940 - delay),
        ])
      );

    const a1 = bounce(d1, 0);
    const a2 = bounce(d2, 200);
    const a3 = bounce(d3, 400);
    a1.start(); a2.start(); a3.start();
    return () => { a1.stop(); a2.stop(); a3.stop(); };
  }, []);

  return (
    <View style={{ flexDirection: 'row', alignItems: 'center', marginTop: 8, gap: 5 }}>
      {[d1, d2, d3].map((d, i) => (
        <Animated.View
          key={i}
          style={{
            width: 6, height: 6, borderRadius: 3,
            backgroundColor: '#3b82f6',
            transform: [{ translateY: d }],
          }}
        />
      ))}
    </View>
  );
}

function StatusBubble({ text }: { text: string }) {
  return (
    <View style={{ flexDirection: 'row', marginVertical: 6, paddingHorizontal: 12, alignItems: 'flex-start' }}>
      <View style={{
        width: 28, height: 28, borderRadius: 14, backgroundColor: '#2563EB',
        alignItems: 'center', justifyContent: 'center',
        marginRight: 8, marginTop: 2, flexShrink: 0,
      }}>
        <Text style={{ color: '#fff', fontSize: 13, fontWeight: '700' }}>C</Text>
      </View>
      <View style={{
        backgroundColor: '#1a1a1a', borderRadius: 18, borderBottomLeftRadius: 4,
        paddingHorizontal: 14, paddingVertical: 12,
        borderWidth: 1, borderColor: 'rgba(255,255,255,0.08)',
        maxWidth: '80%',
      }}>
        <Text style={{ color: '#888', fontSize: 14 }}>{text}</Text>
        <AnimatedDots />
      </View>
    </View>
  );
}

// ── Empty state ───────────────────────────────────────────────────────────────

function EmptyState() {
  return (
    <View style={styles.emptyState}>
      <View style={styles.emptyAvatar}>
        <Text style={styles.emptyAvatarText}>C</Text>
      </View>
      <Text style={styles.emptyTitle}>What are you in the mood for?</Text>
      <Text style={styles.emptyBody}>
        Tell me a genre, a feeling, a movie you loved — I'll find the perfect watch for you.
      </Text>
    </View>
  );
}

// ── Send arrow icon ───────────────────────────────────────────────────────────

function SendArrow() {
  return (
    <View style={{ width: 18, height: 18, alignItems: 'center', justifyContent: 'center' }}>
      {/* Arrow body */}
      <View style={{
        position: 'absolute',
        width: 13,
        height: 2.5,
        backgroundColor: '#fff',
        borderRadius: 2,
        transform: [{ rotate: '-45deg' }, { translateX: -1 }],
        top: 5,
      }} />
      <View style={{
        position: 'absolute',
        width: 13,
        height: 2.5,
        backgroundColor: '#fff',
        borderRadius: 2,
        transform: [{ rotate: '45deg' }, { translateX: -1 }],
        bottom: 5,
      }} />
      <View style={{
        position: 'absolute',
        right: 1,
        width: 2.5,
        height: 14,
        backgroundColor: '#fff',
        borderRadius: 2,
      }} />
    </View>
  );
}

// ── Styles ────────────────────────────────────────────────────────────────────

const styles = StyleSheet.create({
  safe: { flex: 1, backgroundColor: '#0a0a0a' },
  flex: { flex: 1 },

  header: {
    paddingHorizontal: 16,
    paddingTop: 12,
    paddingBottom: 10,
    borderBottomWidth: 1,
    borderBottomColor: 'rgba(255,255,255,0.06)',
  },
  headerTitle: { color: '#fff', fontSize: 17, fontWeight: '700' },
  headerSub: { color: '#555', fontSize: 12, marginTop: 1 },

  listContent: { paddingVertical: 12 },
  listContentEmpty: { flex: 1, justifyContent: 'center' },

  emptyState: {
    alignItems: 'center',
    paddingHorizontal: 32,
    paddingBottom: 24,
  },
  emptyAvatar: {
    width: 56,
    height: 56,
    borderRadius: 28,
    backgroundColor: '#2563EB',
    alignItems: 'center',
    justifyContent: 'center',
    marginBottom: 16,
  },
  emptyAvatarText: { color: '#fff', fontSize: 22, fontWeight: '700' },
  emptyTitle: { color: '#e5e5e5', fontSize: 18, fontWeight: '600', textAlign: 'center', marginBottom: 8 },
  emptyBody: { color: '#666', fontSize: 14, lineHeight: 21, textAlign: 'center' },

  quickReplies: {
    flexDirection: 'row',
    flexWrap: 'wrap',
    paddingHorizontal: 12,
    paddingBottom: 8,
    gap: 8,
  },
  chip: {
    backgroundColor: '#1a1a1a',
    borderRadius: 20,
    paddingHorizontal: 14,
    paddingVertical: 8,
    borderWidth: 1,
    borderColor: 'rgba(255,255,255,0.1)',
  },
  chipText: { color: '#ccc', fontSize: 13 },

  inputBar: {
    flexDirection: 'row',
    alignItems: 'flex-end',
    paddingHorizontal: 12,
    paddingTop: 10,
    borderTopWidth: 1,
    borderTopColor: 'rgba(255,255,255,0.06)',
    backgroundColor: '#0a0a0a',
    gap: 10,
  },
  input: {
    flex: 1,
    backgroundColor: '#1a1a1a',
    borderRadius: 22,
    paddingHorizontal: 16,
    paddingVertical: 10,
    color: '#e5e5e5',
    fontSize: 15,
    maxHeight: 120,
    borderWidth: 1,
    borderColor: 'rgba(255,255,255,0.08)',
  },
  sendBtn: {
    width: 42,
    height: 42,
    borderRadius: 21,
    backgroundColor: '#2563EB',
    alignItems: 'center',
    justifyContent: 'center',
    flexShrink: 0,
  },
  sendBtnDisabled: { backgroundColor: '#1a2a4a', opacity: 0.5 },
});
