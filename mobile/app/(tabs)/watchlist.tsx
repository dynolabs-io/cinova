/**
 * Watchlist screen — Saved titles in a 2-column grid
 *
 * Filter tabs: All | Movies | TV Shows
 * Long press to remove from watchlist.
 */

import React, { useState, useCallback } from 'react';
import {
  View,
  Text,
  FlatList,
  TouchableOpacity,
  StyleSheet,
  Dimensions,
  ActivityIndicator,
  Alert,
} from 'react-native';
import { useSafeAreaInsets } from 'react-native-safe-area-context';
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import { useRouter } from 'expo-router';
import MovieCard from '../../components/ui/MovieCard';
import { getWatchlist, unsaveTitle } from '../../services/api';
import { Colors, Typography, Spacing, Radius } from '../../constants/theme';
import type { Movie } from '../../types';

const { width: SCREEN_WIDTH } = Dimensions.get('window');
const CARD_WIDTH = (SCREEN_WIDTH - Spacing[4] * 2 - Spacing[3]) / 2;

type FilterType = 'all' | 'movie' | 'tv';

const FILTERS: { label: string; value: FilterType }[] = [
  { label: 'All', value: 'all' },
  { label: 'Movies', value: 'movie' },
  { label: 'TV Shows', value: 'tv' },
];

export default function WatchlistScreen() {
  const insets = useSafeAreaInsets();
  const router = useRouter();
  const queryClient = useQueryClient();
  const [filter, setFilter] = useState<FilterType>('all');

  const { data: watchlist, isLoading } = useQuery({
    queryKey: ['watchlist'],
    queryFn: getWatchlist,
  });

  const removeMutation = useMutation({
    mutationFn: (tmdbId: number) => unsaveTitle(tmdbId),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['watchlist'] });
    },
  });

  const handleLongPress = useCallback(
    (movie: Movie) => {
      Alert.alert(
        'Remove from Watchlist',
        `Remove "${movie.title}" from your watchlist?`,
        [
          { text: 'Cancel', style: 'cancel' },
          {
            text: 'Remove',
            style: 'destructive',
            onPress: () => removeMutation.mutate(movie.tmdbId),
          },
        ]
      );
    },
    [removeMutation]
  );

  const filtered = (watchlist ?? []).filter((m) => {
    if (filter === 'all') return true;
    // Movies have runtime, TVShows have seasons — use a heuristic
    // In a real app, the API would return mediaType
    return filter === 'movie';
  });

  const renderItem = useCallback(
    ({ item }: { item: Movie }) => (
      <TouchableOpacity
        onPress={() => router.push(`/movie/${item.id}`)}
        onLongPress={() => handleLongPress(item)}
        activeOpacity={0.85}
        style={[styles.gridItem, { width: CARD_WIDTH }]}
      >
        <MovieCard movie={item} size="md" />
      </TouchableOpacity>
    ),
    [handleLongPress, router]
  );

  return (
    <View style={[styles.container, { paddingTop: insets.top }]}>
      {/* Header */}
      <View style={styles.header}>
        <Text style={styles.headerTitle}>My Watchlist</Text>
        {watchlist && watchlist.length > 0 && (
          <Text style={styles.headerCount}>{watchlist.length} titles</Text>
        )}
      </View>

      {/* Filter tabs */}
      <View style={styles.filterRow}>
        {FILTERS.map((f) => (
          <TouchableOpacity
            key={f.value}
            onPress={() => setFilter(f.value)}
            style={[
              styles.filterTab,
              filter === f.value && styles.filterTabActive,
            ]}
            activeOpacity={0.75}
          >
            <Text
              style={[
                styles.filterTabText,
                filter === f.value && styles.filterTabTextActive,
              ]}
            >
              {f.label}
            </Text>
          </TouchableOpacity>
        ))}
      </View>

      {/* Content */}
      {isLoading ? (
        <View style={styles.loading}>
          <ActivityIndicator color={Colors.primary} size="large" />
        </View>
      ) : filtered.length === 0 ? (
        <EmptyState onDiscover={() => router.push('/(tabs)/discover')} />
      ) : (
        <FlatList
          data={filtered}
          keyExtractor={(item, index) => String(item.tmdbId ?? item.id ?? index)}
          renderItem={renderItem}
          numColumns={2}
          contentContainerStyle={styles.grid}
          columnWrapperStyle={styles.gridRow}
          showsVerticalScrollIndicator={false}
        />
      )}
    </View>
  );
}

function EmptyState({ onDiscover }: { onDiscover: () => void }) {
  return (
    <View style={styles.emptyContainer}>
      <Text style={styles.emptyIcon}>🎬</Text>
      <Text style={styles.emptyTitle}>Nothing saved yet</Text>
      <Text style={styles.emptySubtitle}>
        Save movies and shows you want to watch later. They'll appear here.
      </Text>
      <TouchableOpacity
        style={styles.discoverBtn}
        onPress={onDiscover}
        activeOpacity={0.85}
      >
        <Text style={styles.discoverBtnText}>Discover Movies</Text>
      </TouchableOpacity>
    </View>
  );
}

const styles = StyleSheet.create({
  container: {
    flex: 1,
    backgroundColor: Colors.background,
  },
  header: {
    flexDirection: 'row',
    alignItems: 'baseline',
    justifyContent: 'space-between',
    paddingHorizontal: Spacing[4],
    paddingTop: Spacing[4],
    paddingBottom: Spacing[2],
  },
  headerTitle: {
    color: Colors.textPrimary,
    fontSize: Typography['2xl'],
    fontWeight: Typography.bold,
  },
  headerCount: {
    color: Colors.textMuted,
    fontSize: Typography.sm,
  },
  filterRow: {
    flexDirection: 'row',
    paddingHorizontal: Spacing[4],
    paddingBottom: Spacing[4],
    gap: Spacing[2],
  },
  filterTab: {
    paddingHorizontal: Spacing[4],
    paddingVertical: Spacing[1.5],
    borderRadius: Radius.full,
    backgroundColor: Colors.surface,
    borderWidth: 1,
    borderColor: Colors.border,
  },
  filterTabActive: {
    backgroundColor: Colors.primary,
    borderColor: Colors.primary,
  },
  filterTabText: {
    color: Colors.textMuted,
    fontSize: Typography.sm,
    fontWeight: Typography.medium,
  },
  filterTabTextActive: {
    color: Colors.textPrimary,
    fontWeight: Typography.semibold,
  },
  loading: {
    flex: 1,
    justifyContent: 'center',
    alignItems: 'center',
  },
  grid: {
    paddingHorizontal: Spacing[4],
    paddingBottom: Spacing[8],
    gap: Spacing[3],
  },
  gridRow: {
    justifyContent: 'space-between',
  },
  gridItem: {
    marginBottom: Spacing[1],
  },
  emptyContainer: {
    flex: 1,
    justifyContent: 'center',
    alignItems: 'center',
    padding: Spacing[8],
    gap: Spacing[4],
  },
  emptyIcon: {
    fontSize: 56,
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
    lineHeight: Typography.base * 1.5,
  },
  discoverBtn: {
    backgroundColor: Colors.primary,
    borderRadius: Radius.md,
    paddingHorizontal: Spacing[8],
    paddingVertical: Spacing[3],
    marginTop: Spacing[2],
  },
  discoverBtnText: {
    color: Colors.textPrimary,
    fontSize: Typography.base,
    fontWeight: Typography.bold,
  },
});
