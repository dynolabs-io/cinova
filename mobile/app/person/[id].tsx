/**
 * Person detail screen
 *
 * Header: large photo, name, nationality, birth year
 * "Known For" horizontal carousel
 * "Filmography" list sorted by year (descending)
 */

import React from 'react';
import {
  View,
  Text,
  ScrollView,
  FlatList,
  TouchableOpacity,
  StyleSheet,
  ActivityIndicator,
} from 'react-native';
import { Image } from 'expo-image';
import { LinearGradient } from 'expo-linear-gradient';
import { BlurView } from 'expo-blur';
import { useLocalSearchParams, useRouter } from 'expo-router';
import { useSafeAreaInsets } from 'react-native-safe-area-context';
import { useQuery } from '@tanstack/react-query';
import { getPerson } from '../../services/api';
import MovieCard from '../../components/ui/MovieCard';
import {
  Colors,
  Typography,
  Spacing,
  Radius,
} from '../../constants/theme';
import type { FilmographyEntry } from '../../types';

const HEADER_HEIGHT = 420;
const TMDB_IMAGE = 'https://image.tmdb.org/t/p';

function profileUri(path: string | null): string {
  if (!path) return '';
  return `${TMDB_IMAGE}/h632${path}`;
}

function posterUri(path: string | null): string {
  if (!path) return '';
  return `${TMDB_IMAGE}/w342${path}`;
}

export default function PersonDetailScreen() {
  const { id } = useLocalSearchParams<{ id: string }>();
  const router = useRouter();
  const insets = useSafeAreaInsets();

  const { data: person, isLoading, isError, refetch } = useQuery({
    queryKey: ['person', id],
    queryFn: () => getPerson(Number(id)),
    enabled: !!id,
  });

  if (isLoading) {
    return (
      <View style={styles.loading}>
        <ActivityIndicator color={Colors.primary} size="large" />
      </View>
    );
  }

  if (isError || !person) {
    return (
      <View style={styles.loading}>
        <Text style={{ color: Colors.textSecondary, fontSize: Typography.base, marginBottom: Spacing[4] }}>
          Could not load person
        </Text>
        <TouchableOpacity
          onPress={() => refetch()}
          style={{ backgroundColor: Colors.primary, borderRadius: Radius.md, paddingHorizontal: Spacing[6], paddingVertical: Spacing[3] }}
        >
          <Text style={{ color: Colors.textPrimary, fontWeight: Typography.semibold }}>Retry</Text>
        </TouchableOpacity>
      </View>
    );
  }

  const birthYear = person.birthday
    ? new Date(person.birthday).getFullYear()
    : null;

  const filmographySorted = [...(person.filmography ?? [])].sort(
    (a, b) => b.year - a.year
  );

  return (
    <View style={styles.container}>
      <ScrollView
        showsVerticalScrollIndicator={false}
        contentContainerStyle={{ paddingBottom: insets.bottom + 40 }}
      >
        {/* Hero header */}
        <View style={{ height: HEADER_HEIGHT }}>
          {person.profilePath ? (
            <Image
              source={{ uri: profileUri(person.profilePath) }}
              style={StyleSheet.absoluteFill}
              contentFit="cover"
              transition={300}
            />
          ) : (
            <View style={[StyleSheet.absoluteFill, styles.noPhoto]}>
              <Text style={styles.noPhotoText}>👤</Text>
            </View>
          )}

          <LinearGradient
            colors={['transparent', 'rgba(0,0,0,0.5)', Colors.background]}
            locations={[0.3, 0.65, 1]}
            style={StyleSheet.absoluteFill}
          />

          {/* Person info */}
          <View style={styles.headerContent}>
            <Text style={styles.name}>{person.name}</Text>
            <View style={styles.headerMeta}>
              {person.nationality && (
                <Text style={styles.headerMetaText}>{person.nationality}</Text>
              )}
              {birthYear && (
                <>
                  {person.nationality && (
                    <Text style={styles.dot}> · </Text>
                  )}
                  <Text style={styles.headerMetaText}>b. {birthYear}</Text>
                </>
              )}
              {person.knownForDepartment && (
                <>
                  <Text style={styles.dot}> · </Text>
                  <Text style={styles.headerMetaText}>
                    {person.knownForDepartment}
                  </Text>
                </>
              )}
            </View>
          </View>
        </View>

        {/* Content */}
        <View style={styles.content}>
          {/* Biography */}
          {person.biography ? (
            <View style={styles.section}>
              <Text style={styles.sectionTitle}>Biography</Text>
              <Text style={styles.biography} numberOfLines={6}>
                {person.biography}
              </Text>
            </View>
          ) : null}

          {/* Known For */}
          {(person.knownFor ?? []).length > 0 && (
            <View style={styles.section}>
              <Text style={styles.sectionTitle}>Known For</Text>
              <FlatList
                data={person.knownFor ?? []}
                keyExtractor={(item, index) => String(item.tmdbId ?? item.id ?? index)}
                renderItem={({ item }) => <MovieCard movie={item} size="md" />}
                horizontal
                showsHorizontalScrollIndicator={false}
                contentContainerStyle={{ gap: 0 }}
              />
            </View>
          )}

          {/* Filmography */}
          {filmographySorted.length > 0 && (
            <View style={styles.section}>
              <Text style={styles.sectionTitle}>Filmography</Text>
              {filmographySorted.map((entry) => (
                <FilmographyRow
                  key={`${entry.id}-${entry.mediaType}`}
                  entry={entry}
                  onPress={() => router.push(`/movie/${entry.id}`)}
                />
              ))}
            </View>
          )}
        </View>
      </ScrollView>

      {/* Back button */}
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

function FilmographyRow({
  entry,
  onPress,
}: {
  entry: FilmographyEntry;
  onPress: () => void;
}) {
  return (
    <TouchableOpacity
      onPress={onPress}
      activeOpacity={0.75}
      style={filmStyles.row}
    >
      <Image
        source={
          entry.posterPath
            ? { uri: posterUri(entry.posterPath) }
            : undefined
        }
        style={filmStyles.poster}
        contentFit="cover"
        transition={200}
      />

      <View style={filmStyles.info}>
        <Text style={filmStyles.title} numberOfLines={1}>
          {entry.title}
        </Text>
        <View style={filmStyles.meta}>
          <Text style={filmStyles.year}>{entry.year || '—'}</Text>
          {(entry.character ?? entry.job) ? (
            <>
              <Text style={filmStyles.dot}> · </Text>
              <Text style={filmStyles.role} numberOfLines={1}>
                {entry.character ?? entry.job}
              </Text>
            </>
          ) : null}
        </View>
        <View
          style={[
            filmStyles.badge,
            entry.mediaType === 'tv' && filmStyles.badgeTv,
          ]}
        >
          <Text style={filmStyles.badgeText}>
            {entry.mediaType === 'tv' ? 'TV' : 'Film'}
          </Text>
        </View>
      </View>

      <Text style={filmStyles.chevron}>›</Text>
    </TouchableOpacity>
  );
}

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
  noPhoto: {
    backgroundColor: Colors.card,
    justifyContent: 'center',
    alignItems: 'center',
  },
  noPhotoText: {
    fontSize: 80,
  },
  headerContent: {
    position: 'absolute',
    bottom: 0,
    left: 0,
    right: 0,
    padding: Spacing[4],
    gap: Spacing[2],
  },
  name: {
    color: Colors.textPrimary,
    fontSize: Typography['3xl'],
    fontWeight: Typography.black,
    letterSpacing: Typography.tighter,
  },
  headerMeta: {
    flexDirection: 'row',
    alignItems: 'center',
    flexWrap: 'wrap',
  },
  headerMetaText: {
    color: Colors.textSecondary,
    fontSize: Typography.sm,
    fontWeight: Typography.medium,
  },
  dot: {
    color: Colors.textMuted,
    fontSize: Typography.sm,
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
  content: {
    padding: Spacing[4],
    gap: Spacing[6],
  },
  section: {
    gap: Spacing[3],
  },
  sectionTitle: {
    color: Colors.textPrimary,
    fontSize: Typography.lg,
    fontWeight: Typography.bold,
  },
  biography: {
    color: Colors.textSecondary,
    fontSize: Typography.base,
    lineHeight: Typography.base * 1.6,
  },
});

const filmStyles = StyleSheet.create({
  row: {
    flexDirection: 'row',
    alignItems: 'center',
    gap: Spacing[3],
    paddingVertical: Spacing[2.5],
    borderBottomWidth: 1,
    borderBottomColor: Colors.borderFaint,
  },
  poster: {
    width: 44,
    height: 64,
    borderRadius: Radius.sm,
    backgroundColor: Colors.card,
  },
  info: {
    flex: 1,
    gap: Spacing[1],
  },
  title: {
    color: Colors.textPrimary,
    fontSize: Typography.base,
    fontWeight: Typography.semibold,
  },
  meta: {
    flexDirection: 'row',
    alignItems: 'center',
  },
  year: {
    color: Colors.textMuted,
    fontSize: Typography.sm,
  },
  dot: {
    color: Colors.textMuted,
    fontSize: Typography.sm,
  },
  role: {
    color: Colors.textSecondary,
    fontSize: Typography.sm,
    flex: 1,
  },
  badge: {
    alignSelf: 'flex-start',
    backgroundColor: Colors.card,
    borderRadius: Radius.sm,
    paddingHorizontal: Spacing[1.5],
    paddingVertical: 2,
    borderWidth: 1,
    borderColor: Colors.border,
  },
  badgeTv: {
    borderColor: '#00A8E1',
  },
  badgeText: {
    color: Colors.textMuted,
    fontSize: 10,
    fontWeight: Typography.semibold,
    letterSpacing: 0.5,
  },
  chevron: {
    color: Colors.textMuted,
    fontSize: Typography.xl,
  },
});
