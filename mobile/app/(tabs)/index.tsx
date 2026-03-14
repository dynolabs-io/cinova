/**
 * Home screen — IMDB-style cinematic homepage
 *
 * Layout:
 *   HeroCarousel (65% screen height, auto-scrolling trending)
 *   ↓ Horizontal carousels:
 *       Trending Now · New on Netflix · Top Rated · Recommended for You
 */

import React, { useCallback } from 'react';
import {
  ScrollView,
  View,
  Text,
  FlatList,
  StyleSheet,
  RefreshControl,
  ActivityIndicator,
} from 'react-native';
import { useSafeAreaInsets } from 'react-native-safe-area-context';
import { useQuery, useQueryClient } from '@tanstack/react-query';
import HeroCarousel from '../../components/ui/HeroCarousel';
import MovieCard from '../../components/ui/MovieCard';
import {
  getTrending,
  getNewOnNetflix,
  getTopRated,
  getRecommendations,
  saveTitle,
} from '../../services/api';
import { useAppStore } from '../../store/useAppStore';
import { Colors, Typography, Spacing } from '../../constants/theme';
import type { Movie } from '../../types';

export default function HomeScreen() {
  const insets = useSafeAreaInsets();
  const country = useAppStore((s) => s.country);
  const queryClient = useQueryClient();

  const { data: trending, isLoading: trendingLoading } = useQuery({
    queryKey: ['trending', country],
    queryFn: () => getTrending(country),
  });

  const { data: netflix } = useQuery({
    queryKey: ['new-on-netflix', country],
    queryFn: () => getNewOnNetflix(country),
  });

  const { data: topRated } = useQuery({
    queryKey: ['top-rated', country],
    queryFn: () => getTopRated(country),
  });

  const { data: recommended } = useQuery({
    queryKey: ['recommendations', country],
    queryFn: () => getRecommendations(country),
  });

  const isRefreshing = false;

  const handleRefresh = useCallback(() => {
    queryClient.invalidateQueries({ queryKey: ['trending', country] });
    queryClient.invalidateQueries({ queryKey: ['new-on-netflix', country] });
    queryClient.invalidateQueries({ queryKey: ['top-rated', country] });
    queryClient.invalidateQueries({ queryKey: ['recommendations', country] });
  }, [country, queryClient]);

  const handleSave = useCallback(async (movie: Movie) => {
    try {
      await saveTitle(movie.tmdbId);
    } catch {
      // Silent fail — will retry when user opens watchlist
    }
  }, []);

  if (trendingLoading) {
    return (
      <View style={[styles.loadingContainer, { paddingTop: insets.top }]}>
        <ActivityIndicator color={Colors.primary} size="large" />
      </View>
    );
  }

  return (
    <ScrollView
      style={styles.container}
      contentContainerStyle={{ paddingBottom: insets.bottom + Spacing[4] }}
      showsVerticalScrollIndicator={false}
      refreshControl={
        <RefreshControl
          refreshing={isRefreshing}
          onRefresh={handleRefresh}
          tintColor={Colors.primary}
          colors={[Colors.primary]}
        />
      }
    >
      {/* Hero carousel — trending movies */}
      {trending && trending.length > 0 && (
        <HeroCarousel movies={trending.slice(0, 8)} onSave={handleSave} />
      )}

      {/* Content carousels */}
      <View style={[styles.carousels, { paddingTop: Spacing[5] }]}>
        {trending && trending.length > 0 && (
          <CarouselRow title="Trending Now" movies={trending} />
        )}

        {netflix && netflix.length > 0 && (
          <CarouselRow title="New on Netflix" movies={netflix} />
        )}

        {topRated && topRated.length > 0 && (
          <CarouselRow title="Top Rated" movies={topRated} />
        )}

        {recommended && recommended.length > 0 && (
          <CarouselRow title="Recommended for You" movies={recommended} />
        )}
      </View>
    </ScrollView>
  );
}

// ── Carousel row ──────────────────────────────────────────────────────────────

interface CarouselRowProps {
  title: string;
  movies: Movie[];
}

function CarouselRow({ title, movies }: CarouselRowProps) {
  return (
    <View style={styles.row}>
      <Text style={styles.rowTitle}>{title}</Text>
      <FlatList
        data={movies}
        keyExtractor={(item) => String(item.id)}
        renderItem={({ item }) => <MovieCard movie={item} size="md" />}
        horizontal
        showsHorizontalScrollIndicator={false}
        contentContainerStyle={styles.rowList}
        initialNumToRender={5}
        maxToRenderPerBatch={5}
        removeClippedSubviews
      />
    </View>
  );
}

const styles = StyleSheet.create({
  container: {
    flex: 1,
    backgroundColor: Colors.background,
  },
  loadingContainer: {
    flex: 1,
    backgroundColor: Colors.background,
    justifyContent: 'center',
    alignItems: 'center',
  },
  carousels: {
    gap: Spacing[6],
  },
  row: {
    gap: Spacing[3],
  },
  rowTitle: {
    color: Colors.textPrimary,
    fontSize: Typography.lg,
    fontWeight: Typography.bold,
    paddingHorizontal: Spacing[4],
  },
  rowList: {
    paddingHorizontal: Spacing[4],
    paddingBottom: Spacing[1],
  },
});
