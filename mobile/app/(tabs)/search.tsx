/**
 * Search screen — Natural language movie search
 *
 * Features:
 *  - Debounced search input (500ms)
 *  - Recent searches stored in AsyncStorage (shown as chips)
 *  - Results as vertical list of MovieCards (lg size)
 *  - Suggested searches in empty state
 *  - KeyboardAwareScrollView pattern
 */

import React, { useState, useCallback, useEffect, useRef } from 'react';
import {
  View,
  Text,
  TextInput,
  FlatList,
  TouchableOpacity,
  ScrollView,
  StyleSheet,
  ActivityIndicator,
  Keyboard,
  KeyboardAvoidingView,
  Platform,
} from 'react-native';
import { useSafeAreaInsets } from 'react-native-safe-area-context';
import AsyncStorage from '@react-native-async-storage/async-storage';
import { useQuery } from '@tanstack/react-query';
import MovieCard from '../../components/ui/MovieCard';
import { searchMovies } from '../../services/api';
import { useAppStore } from '../../store/useAppStore';
import {
  Colors,
  Typography,
  Spacing,
  Radius,
} from '../../constants/theme';
import type { Movie } from '../../types';

const RECENT_SEARCHES_KEY = 'cinova_recent_searches';
const MAX_RECENT = 8;

const SUGGESTED_SEARCHES = [
  'dark thriller like Parasite',
  'feel-good movies on Netflix',
  'action movies with a twist ending',
  'sci-fi classics from the 80s',
  'romantic comedies on Disney+',
  'horror films under 90 minutes',
  'award-winning dramas 2023',
  'animated films for adults',
];

export default function SearchScreen() {
  const insets = useSafeAreaInsets();
  const country = useAppStore((s) => s.country);

  const [inputValue, setInputValue] = useState('');
  const [debouncedQuery, setDebouncedQuery] = useState('');
  const [recentSearches, setRecentSearches] = useState<string[]>([]);
  const debounceTimer = useRef<ReturnType<typeof setTimeout> | null>(null);

  // Load recent searches on mount
  useEffect(() => {
    AsyncStorage.getItem(RECENT_SEARCHES_KEY).then((raw) => {
      if (raw) {
        try {
          setRecentSearches(JSON.parse(raw));
        } catch {
          // Ignore parse errors
        }
      }
    });
  }, []);

  const handleInputChange = useCallback((text: string) => {
    setInputValue(text);
    if (debounceTimer.current) clearTimeout(debounceTimer.current);
    debounceTimer.current = setTimeout(() => {
      setDebouncedQuery(text.trim());
    }, 500);
  }, []);

  const persistSearch = useCallback(async (query: string) => {
    if (!query) return;
    setRecentSearches((prev) => {
      const next = [query, ...prev.filter((q) => q !== query)].slice(
        0,
        MAX_RECENT
      );
      AsyncStorage.setItem(RECENT_SEARCHES_KEY, JSON.stringify(next));
      return next;
    });
  }, []);

  const handleSuggestionPress = useCallback(
    (suggestion: string) => {
      setInputValue(suggestion);
      setDebouncedQuery(suggestion);
      persistSearch(suggestion);
      Keyboard.dismiss();
    },
    [persistSearch]
  );

  const handleRecentPress = useCallback(
    (query: string) => {
      setInputValue(query);
      setDebouncedQuery(query);
      Keyboard.dismiss();
    },
    []
  );

  const handleRemoveRecent = useCallback((query: string) => {
    setRecentSearches((prev) => {
      const next = prev.filter((q) => q !== query);
      AsyncStorage.setItem(RECENT_SEARCHES_KEY, JSON.stringify(next));
      return next;
    });
  }, []);

  const handleClearInput = useCallback(() => {
    setInputValue('');
    setDebouncedQuery('');
  }, []);

  // Search query
  const { data: results, isLoading: searching } = useQuery({
    queryKey: ['search', debouncedQuery, country],
    queryFn: async () => {
      if (!debouncedQuery) return null;
      const result = await searchMovies(debouncedQuery, country);
      await persistSearch(debouncedQuery);
      return result;
    },
    enabled: debouncedQuery.length >= 2,
  });

  const movies = (results?.items ?? []) as Movie[];
  const hasQuery = debouncedQuery.length >= 2;
  const hasResults = movies.length > 0;

  return (
    <KeyboardAvoidingView
      style={[styles.container, { paddingTop: insets.top }]}
      behavior={Platform.OS === 'ios' ? 'padding' : 'height'}
      keyboardVerticalOffset={0}
    >
      {/* Search input */}
      <View style={styles.inputContainer}>
        <View style={styles.inputRow}>
          <Text style={styles.searchIcon}>⌕</Text>
          <TextInput
            style={styles.input}
            value={inputValue}
            onChangeText={handleInputChange}
            placeholder={'Ask anything… "dark thriller like Parasite"'}
            placeholderTextColor={Colors.textMuted}
            returnKeyType="search"
            autoCorrect={false}
            autoCapitalize="none"
            onSubmitEditing={() => {
              if (inputValue.trim()) {
                setDebouncedQuery(inputValue.trim());
                persistSearch(inputValue.trim());
                Keyboard.dismiss();
              }
            }}
          />
          {inputValue.length > 0 && (
            <TouchableOpacity onPress={handleClearInput} style={styles.clearBtn}>
              <Text style={styles.clearIcon}>✕</Text>
            </TouchableOpacity>
          )}
        </View>
      </View>

      {/* Content */}
      {hasQuery ? (
        // ── Search results ──────────────────────────────────────────────────
        <View style={styles.flex}>
          {searching ? (
            <View style={styles.centerContent}>
              <ActivityIndicator color={Colors.primary} size="large" />
              <Text style={styles.loadingText}>Searching…</Text>
            </View>
          ) : hasResults ? (
            <FlatList
              data={movies}
              keyExtractor={(item) => String(item.id)}
              renderItem={({ item }) => (
                <View style={styles.lgCard}>
                  <MovieCard movie={item} size="lg" />
                </View>
              )}
              contentContainerStyle={styles.resultsList}
              showsVerticalScrollIndicator={false}
              keyboardDismissMode="on-drag"
            />
          ) : (
            <View style={styles.centerContent}>
              <Text style={styles.emptyTitle}>No results found</Text>
              <Text style={styles.emptySubtitle}>
                Try rephrasing or use one of our suggestions
              </Text>
              <View style={styles.suggestionsGrid}>
                {SUGGESTED_SEARCHES.map((s) => (
                  <SuggestionChip
                    key={s}
                    label={s}
                    onPress={() => handleSuggestionPress(s)}
                  />
                ))}
              </View>
            </View>
          )}
        </View>
      ) : (
        // ── Empty state / suggestions ───────────────────────────────────────
        <ScrollView
          contentContainerStyle={styles.emptyState}
          keyboardDismissMode="on-drag"
          showsVerticalScrollIndicator={false}
        >
          {/* Recent searches */}
          {recentSearches.length > 0 && (
            <View style={styles.section}>
              <View style={styles.sectionHeader}>
                <Text style={styles.sectionTitle}>Recent</Text>
                <TouchableOpacity
                  onPress={() => {
                    setRecentSearches([]);
                    AsyncStorage.removeItem(RECENT_SEARCHES_KEY);
                  }}
                >
                  <Text style={styles.sectionAction}>Clear all</Text>
                </TouchableOpacity>
              </View>
              <View style={styles.chipsRow}>
                {recentSearches.map((q) => (
                  <TouchableOpacity
                    key={q}
                    style={styles.recentChip}
                    onPress={() => handleRecentPress(q)}
                    onLongPress={() => handleRemoveRecent(q)}
                    activeOpacity={0.75}
                  >
                    <Text style={styles.recentChipIcon}>↺</Text>
                    <Text style={styles.recentChipText} numberOfLines={1}>
                      {q}
                    </Text>
                  </TouchableOpacity>
                ))}
              </View>
            </View>
          )}

          {/* Suggested searches */}
          <View style={styles.section}>
            <Text style={styles.sectionTitle}>Try searching for…</Text>
            <View style={styles.suggestionsGrid}>
              {SUGGESTED_SEARCHES.map((s) => (
                <SuggestionChip
                  key={s}
                  label={s}
                  onPress={() => handleSuggestionPress(s)}
                />
              ))}
            </View>
          </View>
        </ScrollView>
      )}
    </KeyboardAvoidingView>
  );
}

function SuggestionChip({
  label,
  onPress,
}: {
  label: string;
  onPress: () => void;
}) {
  return (
    <TouchableOpacity
      style={styles.suggestionChip}
      onPress={onPress}
      activeOpacity={0.75}
    >
      <Text style={styles.suggestionChipText}>{label}</Text>
    </TouchableOpacity>
  );
}

const styles = StyleSheet.create({
  container: {
    flex: 1,
    backgroundColor: Colors.background,
  },
  flex: {
    flex: 1,
  },
  inputContainer: {
    paddingHorizontal: Spacing[4],
    paddingTop: Spacing[3],
    paddingBottom: Spacing[3],
  },
  inputRow: {
    flexDirection: 'row',
    alignItems: 'center',
    backgroundColor: Colors.surface,
    borderRadius: Radius.lg,
    paddingHorizontal: Spacing[3],
    borderWidth: 1,
    borderColor: Colors.border,
    height: 52,
  },
  searchIcon: {
    color: Colors.textMuted,
    fontSize: 22,
    marginRight: Spacing[2],
  },
  input: {
    flex: 1,
    color: Colors.textPrimary,
    fontSize: Typography.base,
    height: 52,
  },
  clearBtn: {
    padding: Spacing[2],
  },
  clearIcon: {
    color: Colors.textMuted,
    fontSize: Typography.sm,
  },
  resultsList: {
    padding: Spacing[4],
    gap: Spacing[3],
  },
  lgCard: {
    marginBottom: Spacing[3],
  },
  centerContent: {
    flex: 1,
    justifyContent: 'center',
    alignItems: 'center',
    padding: Spacing[6],
    gap: Spacing[3],
  },
  loadingText: {
    color: Colors.textSecondary,
    fontSize: Typography.base,
    marginTop: Spacing[2],
  },
  emptyTitle: {
    color: Colors.textPrimary,
    fontSize: Typography.xl,
    fontWeight: Typography.bold,
    textAlign: 'center',
  },
  emptySubtitle: {
    color: Colors.textSecondary,
    fontSize: Typography.base,
    textAlign: 'center',
  },
  emptyState: {
    padding: Spacing[4],
    gap: Spacing[6],
  },
  section: {
    gap: Spacing[3],
  },
  sectionHeader: {
    flexDirection: 'row',
    justifyContent: 'space-between',
    alignItems: 'center',
  },
  sectionTitle: {
    color: Colors.textPrimary,
    fontSize: Typography.lg,
    fontWeight: Typography.bold,
  },
  sectionAction: {
    color: Colors.primary,
    fontSize: Typography.sm,
    fontWeight: Typography.medium,
  },
  chipsRow: {
    flexDirection: 'row',
    flexWrap: 'wrap',
    gap: Spacing[2],
  },
  recentChip: {
    flexDirection: 'row',
    alignItems: 'center',
    backgroundColor: Colors.surface,
    borderRadius: Radius.full,
    paddingHorizontal: Spacing[3],
    paddingVertical: Spacing[1.5],
    borderWidth: 1,
    borderColor: Colors.border,
    gap: Spacing[1.5],
    maxWidth: 220,
  },
  recentChipIcon: {
    color: Colors.textMuted,
    fontSize: Typography.sm,
  },
  recentChipText: {
    color: Colors.textSecondary,
    fontSize: Typography.sm,
  },
  suggestionsGrid: {
    flexDirection: 'row',
    flexWrap: 'wrap',
    gap: Spacing[2],
  },
  suggestionChip: {
    backgroundColor: Colors.card,
    borderRadius: Radius.full,
    paddingHorizontal: Spacing[4],
    paddingVertical: Spacing[2],
    borderWidth: 1,
    borderColor: Colors.border,
  },
  suggestionChipText: {
    color: Colors.textSecondary,
    fontSize: Typography.sm,
    fontWeight: Typography.medium,
  },
});
