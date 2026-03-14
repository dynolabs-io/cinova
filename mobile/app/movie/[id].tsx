/**
 * Movie detail screen
 *
 * Full-screen hero backdrop → gradient → scrollable content
 * Floating action bar: Save · Rate · Dismiss
 * Sections: title, metadata, CinovaScore, genres, themes, streaming,
 *           synopsis, cast, similar movies
 * Trailer plays fullscreen via expo-av
 */

import React, { useState, useCallback, useRef } from 'react';
import {
  View,
  Text,
  ScrollView,
  TouchableOpacity,
  StyleSheet,
  Dimensions,
  FlatList,
  ActivityIndicator,
  Alert,
} from 'react-native';
import { Image } from 'expo-image';
import { LinearGradient } from 'expo-linear-gradient';
import { BlurView } from 'expo-blur';
import { useLocalSearchParams, useRouter } from 'expo-router';
import { useSafeAreaInsets } from 'react-native-safe-area-context';
import { useQuery, useMutation } from '@tanstack/react-query';
import { Video, ResizeMode } from 'expo-av';
import CinovaScore from '../../components/ui/CinovaScore';
import MovieCard from '../../components/ui/MovieCard';
import StreamingBadge from '../../components/ui/StreamingBadge';
import { getMovie, saveTitle, rateTitle, dismissTitle, getRecommendations } from '../../services/api';
import { shareMovie } from '../../services/sharing';
import { useAppStore } from '../../store/useAppStore';
import {
  Colors,
  Typography,
  Spacing,
  Radius,
  Shadows,
} from '../../constants/theme';
import type { CastMember, Movie, WatchProvider } from '../../types';

const { width: SCREEN_WIDTH, height: SCREEN_HEIGHT } = Dimensions.get('window');
const HERO_HEIGHT = SCREEN_HEIGHT * 0.55;
const TMDB_IMAGE = 'https://image.tmdb.org/t/p';

function backdropUri(path: string | null): string {
  if (!path) return '';
  return `${TMDB_IMAGE}/w1280${path}`;
}

function posterUri(path: string | null): string {
  if (!path) return '';
  return `${TMDB_IMAGE}/w500${path}`;
}

function profileUri(path: string | null): string {
  if (!path) return '';
  return `${TMDB_IMAGE}/w185${path}`;
}

export default function MovieDetailScreen() {
  const { id } = useLocalSearchParams<{ id: string }>();
  const router = useRouter();
  const insets = useSafeAreaInsets();
  const country = useAppStore((s) => s.country);

  const [isSaved, setIsSaved] = useState(false);
  const [userRating, setUserRating] = useState<number | null>(null);
  const [isDismissed, setIsDismissed] = useState(false);
  const [showTrailer, setShowTrailer] = useState(false);
  const videoRef = useRef<Video>(null);

  const { data: movie, isLoading } = useQuery({
    queryKey: ['movie', id, country],
    queryFn: () => getMovie(Number(id), country),
    enabled: !!id,
  });

  const { data: similar } = useQuery({
    queryKey: ['recommendations', country],
    queryFn: () => getRecommendations(country),
    enabled: !!movie,
  });

  const saveMutation = useMutation({
    mutationFn: () => saveTitle(movie!.tmdbId),
    onSuccess: () => setIsSaved(true),
    onError: () => Alert.alert('Error', 'Could not save. Please try again.'),
  });

  const dismissMutation = useMutation({
    mutationFn: () => dismissTitle(movie!.tmdbId),
    onSuccess: () => {
      setIsDismissed(true);
      router.canGoBack() ? router.back() : router.replace('/(tabs)');
    },
  });

  const handleRate = useCallback(() => {
    if (!movie) return;
    Alert.alert(
      'Rate this movie',
      'How would you rate it?',
      [1, 2, 3, 4, 5, 6, 7, 8, 9, 10].map((score) => ({
        text: `${score}/10`,
        onPress: async () => {
          try {
            await rateTitle(movie.tmdbId, score);
            setUserRating(score);
          } catch {
            Alert.alert('Error', 'Could not save rating.');
          }
        },
      }))
    );
  }, [movie]);

  if (isLoading || !movie) {
    return (
      <View style={styles.loading}>
        <ActivityIndicator color={Colors.primary} size="large" />
      </View>
    );
  }

  const genreTags = movie.genres ?? [];
  const themeTags = movie.themes ?? [];
  const moodTags = movie.moods ?? [];

  const trailerKey = movie.trailer?.key;
  const trailerUrl = trailerKey
    ? `https://www.youtube.com/embed/${trailerKey}`
    : null;

  return (
    <View style={styles.container}>
      {/* Trailer overlay */}
      {showTrailer && trailerKey && (
        <View style={StyleSheet.absoluteFill}>
          <Video
            ref={videoRef}
            style={StyleSheet.absoluteFill}
            source={{
              uri: `https://www.youtube.com/watch?v=${trailerKey}`,
            }}
            resizeMode={ResizeMode.CONTAIN}
            shouldPlay
            useNativeControls
          />
          <TouchableOpacity
            style={[styles.trailerClose, { top: insets.top + 10 }]}
            onPress={() => setShowTrailer(false)}
          >
            <BlurView intensity={60} style={styles.backBlur}>
              <Text style={styles.backIcon}>✕</Text>
            </BlurView>
          </TouchableOpacity>
        </View>
      )}

      <ScrollView
        showsVerticalScrollIndicator={false}
        contentContainerStyle={{ paddingBottom: insets.bottom + 40 }}
      >
        {/* Hero section */}
        <View style={[styles.hero, { height: HERO_HEIGHT }]}>
          <Image
            source={{ uri: backdropUri(movie.backdropPath) }}
            style={StyleSheet.absoluteFill}
            contentFit="cover"
            transition={300}
          />
          <LinearGradient
            colors={['transparent', 'rgba(0,0,0,0.3)', Colors.background]}
            locations={[0, 0.55, 1]}
            style={StyleSheet.absoluteFill}
          />

          {/* Trailer button */}
          {trailerUrl && (
            <TouchableOpacity
              style={styles.trailerBtn}
              onPress={() => setShowTrailer(true)}
              activeOpacity={0.85}
            >
              <View style={styles.trailerPlay}>
                <Text style={styles.trailerPlayIcon}>▶</Text>
              </View>
              <Text style={styles.trailerLabel}>Watch Trailer</Text>
            </TouchableOpacity>
          )}
        </View>

        {/* Back button — positioned absolutely over hero */}
        <View
          style={[styles.backBtnContainer, { top: insets.top + 10 }]}
          pointerEvents="box-none"
        />

        {/* Floating action bar */}
        <View style={styles.actionBar}>
          <ActionBtn
            icon={isSaved ? '🔖' : '🔖'}
            label="Save"
            active={isSaved}
            onPress={() => saveMutation.mutate()}
          />
          <ActionBtn
            icon="★"
            label={userRating ? `${userRating}/10` : 'Rate'}
            active={userRating != null}
            onPress={handleRate}
          />
          <ActionBtn
            icon="↑"
            label="Share"
            active={false}
            onPress={() => movie && shareMovie(movie)}
          />
          <ActionBtn
            icon="✕"
            label="Dismiss"
            active={isDismissed}
            onPress={() => dismissMutation.mutate()}
            danger
          />
        </View>

        {/* Content */}
        <View style={styles.content}>
          {/* Title */}
          <Text style={styles.title}>{movie.title}</Text>

          {/* Metadata row */}
          <View style={styles.metaRow}>
            <MetaPill label={String(movie.year)} />
            {movie.runtime != null && (
              <MetaPill label={`${movie.runtime}m`} />
            )}
            <MetaPill label={movie.language.toUpperCase()} />
            {movie.voteAverage > 0 && (
              <MetaPill
                label={`★ ${movie.voteAverage.toFixed(1)}`}
                accent
              />
            )}
          </View>

          {/* CinovaScore */}
          {movie.cinovaScore != null && (
            <View style={styles.scoreSection}>
              <CinovaScore score={movie.cinovaScore} size="lg" />
              <View style={styles.scoreLabels}>
                <Text style={styles.scoreLabelTitle}>Cinova Score</Text>
                <Text style={styles.scoreLabelSubtitle}>
                  Based on quality, availability & relevance
                </Text>
              </View>
            </View>
          )}

          {/* Genre tags */}
          {genreTags.length > 0 && (
            <TagSection
              title="Genres"
              tags={genreTags.map((g) => g.name)}
              color={Colors.primary}
            />
          )}

          {/* Theme/Mood tags */}
          {themeTags.length > 0 && (
            <TagSection
              title="Themes"
              tags={themeTags.map((t) => t.name)}
              color={Colors.prime ?? '#00A8E1'}
            />
          )}

          {moodTags.length > 0 && (
            <TagSection
              title="Mood"
              tags={moodTags.map((m) => m.name)}
              color={Colors.hbo ?? '#5822B4'}
            />
          )}

          {/* Where to Watch */}
          {movie.providers.length > 0 && (
            <View style={styles.section}>
              <Text style={styles.sectionTitle}>Where to Watch</Text>
              <ScrollView horizontal showsHorizontalScrollIndicator={false}>
                <View style={styles.providersRow}>
                  {movie.providers.map((p) => (
                    <StreamingBadge
                      key={p.providerId}
                      provider={p}
                      variant="full"
                    />
                  ))}
                </View>
              </ScrollView>
            </View>
          )}

          {/* Synopsis */}
          {(movie.aiDescription ?? movie.overview) ? (
            <View style={styles.section}>
              <Text style={styles.sectionTitle}>Synopsis</Text>
              <Text style={styles.synopsis}>
                {movie.aiDescription ?? movie.overview}
              </Text>
            </View>
          ) : null}

          {/* Cast */}
          {movie.cast.length > 0 && (
            <View style={styles.section}>
              <Text style={styles.sectionTitle}>Cast</Text>
              <FlatList
                data={movie.cast.slice(0, 15)}
                keyExtractor={(item) => String(item.id)}
                renderItem={({ item }) => <CastCard member={item} />}
                horizontal
                showsHorizontalScrollIndicator={false}
                contentContainerStyle={{ gap: Spacing[3] }}
              />
            </View>
          )}

          {/* More like this */}
          {similar && similar.length > 0 && (
            <View style={styles.section}>
              <Text style={styles.sectionTitle}>More Like This</Text>
              <FlatList
                data={similar.slice(0, 10)}
                keyExtractor={(item) => String(item.id)}
                renderItem={({ item }) => <MovieCard movie={item} size="md" />}
                horizontal
                showsHorizontalScrollIndicator={false}
                contentContainerStyle={{ paddingRight: Spacing[4] }}
              />
            </View>
          )}
        </View>
      </ScrollView>

      {/* Back button — absolute, overlays hero */}
      <TouchableOpacity
        style={[styles.backBtn, { top: insets.top + 10 }]}
        onPress={() => router.canGoBack() ? router.back() : router.replace('/(tabs)')}
        activeOpacity={0.85}
      >
        <BlurView intensity={60} tint="dark" style={styles.backBlur}>
          <Text style={styles.backIcon}>‹</Text>
        </BlurView>
      </TouchableOpacity>
    </View>
  );
}

// ── Sub-components ────────────────────────────────────────────────────────────

function ActionBtn({
  icon,
  label,
  active,
  onPress,
  danger,
}: {
  icon: string;
  label: string;
  active: boolean;
  onPress: () => void;
  danger?: boolean;
}) {
  const activeColor = danger ? Colors.scoreLow : Colors.primary;
  return (
    <TouchableOpacity
      onPress={onPress}
      activeOpacity={0.75}
      style={[
        actionStyles.btn,
        active && { borderColor: activeColor },
      ]}
    >
      <Text style={[actionStyles.icon, active && { color: activeColor }]}>
        {icon}
      </Text>
      <Text style={[actionStyles.label, active && { color: activeColor }]}>
        {label}
      </Text>
    </TouchableOpacity>
  );
}

function MetaPill({ label, accent }: { label: string; accent?: boolean }) {
  return (
    <View
      style={[
        metaStyles.pill,
        accent && { borderColor: Colors.scoreMid },
      ]}
    >
      <Text
        style={[metaStyles.text, accent && { color: Colors.scoreMid }]}
      >
        {label}
      </Text>
    </View>
  );
}

function TagSection({
  title,
  tags,
  color,
}: {
  title: string;
  tags: string[];
  color: string;
}) {
  return (
    <View style={tagStyles.section}>
      <Text style={tagStyles.title}>{title}</Text>
      <ScrollView horizontal showsHorizontalScrollIndicator={false}>
        <View style={tagStyles.row}>
          {tags.map((tag) => (
            <View
              key={tag}
              style={[tagStyles.chip, { borderColor: color + '60' }]}
            >
              <Text style={[tagStyles.chipText, { color }]}>{tag}</Text>
            </View>
          ))}
        </View>
      </ScrollView>
    </View>
  );
}

function CastCard({ member }: { member: CastMember }) {
  const router = useRouter();
  return (
    <TouchableOpacity
      onPress={() => router.push(`/person/${member.id}`)}
      activeOpacity={0.85}
      style={castStyles.card}
    >
      <Image
        source={
          member.profilePath
            ? { uri: profileUri(member.profilePath) }
            : { uri: 'https://via.placeholder.com/185x278/1a1a1a/6B6B6B?text=?' }
        }
        style={castStyles.photo}
        contentFit="cover"
        transition={200}
      />
      <Text style={castStyles.name} numberOfLines={2}>
        {member.name}
      </Text>
      <Text style={castStyles.character} numberOfLines={2}>
        {member.character}
      </Text>
    </TouchableOpacity>
  );
}

// ── Styles ────────────────────────────────────────────────────────────────────

const styles = StyleSheet.create({
  container: {
    flex: 1,
    backgroundColor: Colors.background,
  },
  loading: {
    flex: 1,
    backgroundColor: Colors.background,
    justifyContent: 'center',
    alignItems: 'center',
  },
  hero: {
    width: SCREEN_WIDTH,
    justifyContent: 'flex-end',
    alignItems: 'center',
    paddingBottom: Spacing[8],
  },
  trailerBtn: {
    flexDirection: 'row',
    alignItems: 'center',
    gap: Spacing[2],
    backgroundColor: 'rgba(255,255,255,0.15)',
    borderRadius: Radius.full,
    paddingHorizontal: Spacing[5],
    paddingVertical: Spacing[2.5],
    borderWidth: 1,
    borderColor: 'rgba(255,255,255,0.3)',
  },
  trailerPlay: {
    width: 28,
    height: 28,
    borderRadius: Radius.full,
    backgroundColor: Colors.primary,
    justifyContent: 'center',
    alignItems: 'center',
  },
  trailerPlayIcon: {
    color: Colors.textPrimary,
    fontSize: Typography.xs,
    marginLeft: 2,
  },
  trailerLabel: {
    color: Colors.textPrimary,
    fontSize: Typography.base,
    fontWeight: Typography.semibold,
  },
  trailerClose: {
    position: 'absolute',
    right: 16,
    zIndex: 10,
  },
  backBtnContainer: {
    position: 'absolute',
    left: 0,
    right: 0,
    pointerEvents: 'box-none',
  },
  backBtn: {
    position: 'absolute',
    left: Spacing[4],
    zIndex: 10,
  },
  backBlur: {
    width: 40,
    height: 40,
    borderRadius: Radius.full,
    justifyContent: 'center',
    alignItems: 'center',
    overflow: 'hidden',
    backgroundColor: 'rgba(0,0,0,0.4)',
  },
  backIcon: {
    color: Colors.textPrimary,
    fontSize: Typography.xl,
    fontWeight: Typography.bold,
    lineHeight: Typography.xl * 1.2,
  },
  actionBar: {
    flexDirection: 'row',
    justifyContent: 'center',
    gap: Spacing[4],
    paddingHorizontal: Spacing[4],
    paddingVertical: Spacing[3],
    backgroundColor: Colors.background,
    borderBottomWidth: 1,
    borderBottomColor: Colors.borderFaint,
  },
  content: {
    padding: Spacing[4],
    gap: Spacing[5],
  },
  title: {
    color: Colors.textPrimary,
    fontSize: Typography['3xl'],
    fontWeight: Typography.black,
    letterSpacing: Typography.tighter,
    lineHeight: Typography['3xl'] * Typography.tight,
  },
  metaRow: {
    flexDirection: 'row',
    flexWrap: 'wrap',
    gap: Spacing[2],
  },
  scoreSection: {
    flexDirection: 'row',
    alignItems: 'center',
    gap: Spacing[4],
    backgroundColor: Colors.card,
    borderRadius: Radius.lg,
    padding: Spacing[4],
    borderWidth: 1,
    borderColor: Colors.border,
  },
  scoreLabels: {
    flex: 1,
    gap: 4,
  },
  scoreLabelTitle: {
    color: Colors.textPrimary,
    fontSize: Typography.base,
    fontWeight: Typography.bold,
  },
  scoreLabelSubtitle: {
    color: Colors.textSecondary,
    fontSize: Typography.xs,
  },
  section: {
    gap: Spacing[3],
  },
  sectionTitle: {
    color: Colors.textPrimary,
    fontSize: Typography.lg,
    fontWeight: Typography.bold,
  },
  providersRow: {
    flexDirection: 'row',
    alignItems: 'center',
  },
  synopsis: {
    color: Colors.textSecondary,
    fontSize: Typography.base,
    lineHeight: Typography.base * Typography.relaxed,
  },
});

const actionStyles = StyleSheet.create({
  btn: {
    flex: 1,
    alignItems: 'center',
    gap: Spacing[1],
    paddingVertical: Spacing[2],
    borderRadius: Radius.md,
    borderWidth: 1,
    borderColor: Colors.border,
    backgroundColor: Colors.card,
  },
  icon: {
    color: Colors.textSecondary,
    fontSize: Typography.lg,
  },
  label: {
    color: Colors.textMuted,
    fontSize: Typography.xs,
    fontWeight: Typography.medium,
  },
});

const metaStyles = StyleSheet.create({
  pill: {
    paddingHorizontal: Spacing[3],
    paddingVertical: Spacing[1],
    borderRadius: Radius.full,
    borderWidth: 1,
    borderColor: Colors.border,
    backgroundColor: Colors.card,
  },
  text: {
    color: Colors.textSecondary,
    fontSize: Typography.xs,
    fontWeight: Typography.medium,
  },
});

const tagStyles = StyleSheet.create({
  section: {
    gap: Spacing[2],
  },
  title: {
    color: Colors.textPrimary,
    fontSize: Typography.base,
    fontWeight: Typography.semibold,
  },
  row: {
    flexDirection: 'row',
    gap: Spacing[2],
  },
  chip: {
    paddingHorizontal: Spacing[3],
    paddingVertical: Spacing[1.5],
    borderRadius: Radius.full,
    borderWidth: 1,
    backgroundColor: Colors.card,
  },
  chipText: {
    fontSize: Typography.xs,
    fontWeight: Typography.medium,
  },
});

const castStyles = StyleSheet.create({
  card: {
    width: 90,
    alignItems: 'center',
    gap: Spacing[1.5],
  },
  photo: {
    width: 70,
    height: 70,
    borderRadius: Radius.full,
    backgroundColor: Colors.card,
  },
  name: {
    color: Colors.textPrimary,
    fontSize: Typography.xs,
    fontWeight: Typography.semibold,
    textAlign: 'center',
  },
  character: {
    color: Colors.textMuted,
    fontSize: Typography.xs,
    textAlign: 'center',
  },
});
