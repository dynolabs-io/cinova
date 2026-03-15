/**
 * Profile screen
 *
 * Anonymous: Sign In / Create Account CTA cards
 * Logged in: avatar, email, join date, stats row, region selector,
 *            "Remove Ads" premium button (RevenueCat placeholder)
 */

import React, { useState } from 'react';
import {
  View,
  Text,
  TouchableOpacity,
  StyleSheet,
  ScrollView,
  Modal,
  FlatList,
  Alert,
} from 'react-native';
import { useSafeAreaInsets } from 'react-native-safe-area-context';
import { useRouter } from 'expo-router';
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import { useAppStore } from '../../store/useAppStore';
import { clearSession } from '../../services/session';
import { getScoringProfile, setScoringProfile } from '../../services/api';
import {
  Colors,
  Typography,
  Spacing,
  Radius,
  Shadows,
} from '../../constants/theme';
import type { ScoringPresetDescription } from '../../types';

const DEFAULT_PRESETS: ScoringPresetDescription[] = [
  { id: 'mainstream', name: 'Mainstream', emoji: '🎯', description: 'Audience ratings drive recommendations', audience: 0.50, critic: 0.20, award: 0.15, prestige: 0.10, commercial: 0.05 },
  { id: 'cinephile', name: 'Cinephile', emoji: '🎬', description: 'Critics and awards matter most', audience: 0.25, critic: 0.35, award: 0.25, prestige: 0.15, commercial: 0.00 },
  { id: 'arthouse', name: 'Arthouse', emoji: '🎨', description: 'Director influence and auteur cinema', audience: 0.20, critic: 0.30, award: 0.20, prestige: 0.25, commercial: 0.05 },
  { id: 'blockbuster', name: 'Blockbuster', emoji: '💥', description: 'Commercial success and spectacle', audience: 0.45, critic: 0.15, award: 0.10, prestige: 0.05, commercial: 0.25 },
  { id: 'award_season', name: 'Award Season', emoji: '🏆', description: 'Oscars, BAFTAs, and film festivals', audience: 0.20, critic: 0.25, award: 0.45, prestige: 0.10, commercial: 0.00 },
];

const COUNTRIES = [
  { code: 'US', name: 'United States', flag: '🇺🇸' },
  { code: 'GB', name: 'United Kingdom', flag: '🇬🇧' },
  { code: 'CA', name: 'Canada', flag: '🇨🇦' },
  { code: 'AU', name: 'Australia', flag: '🇦🇺' },
  { code: 'DE', name: 'Germany', flag: '🇩🇪' },
  { code: 'FR', name: 'France', flag: '🇫🇷' },
  { code: 'JP', name: 'Japan', flag: '🇯🇵' },
  { code: 'IN', name: 'India', flag: '🇮🇳' },
  { code: 'BR', name: 'Brazil', flag: '🇧🇷' },
  { code: 'MX', name: 'Mexico', flag: '🇲🇽' },
  { code: 'ES', name: 'Spain', flag: '🇪🇸' },
  { code: 'IT', name: 'Italy', flag: '🇮🇹' },
  { code: 'NL', name: 'Netherlands', flag: '🇳🇱' },
  { code: 'SE', name: 'Sweden', flag: '🇸🇪' },
  { code: 'KR', name: 'South Korea', flag: '🇰🇷' },
];

export default function ProfileScreen() {
  const insets = useSafeAreaInsets();
  const router = useRouter();
  const queryClient = useQueryClient();
  const { user, isAnonymous, country, scoringPreset, setCountry, setScoringPreset, logout } = useAppStore();
  const [countryModalVisible, setCountryModalVisible] = useState(false);

  const { data: scoringData } = useQuery({
    queryKey: ['scoring-profile'],
    queryFn: getScoringProfile,
    enabled: !isAnonymous,
  });

  const presetMutation = useMutation({
    mutationFn: (preset: string) => setScoringProfile({
      preset,
      audience: 0, critic: 0, award: 0, prestige: 0, commercial: 0,
    }),
    onSuccess: (data) => {
      setScoringPreset(data.preset);
      queryClient.setQueryData(['scoring-profile'], data);
    },
  });

  const currentCountry =
    COUNTRIES.find((c) => c.code === country) ?? COUNTRIES[0];

  const handleLogout = () => {
    Alert.alert('Sign Out', 'Are you sure you want to sign out?', [
      { text: 'Cancel', style: 'cancel' },
      {
        text: 'Sign Out',
        style: 'destructive',
        onPress: async () => {
          await clearSession();
          logout();
        },
      },
    ]);
  };

  const handlePremium = () => {
    Alert.alert(
      'Cinova Premium',
      'Coming soon! Remove ads and unlock exclusive features.',
      [{ text: 'OK' }]
    );
  };

  return (
    <ScrollView
      style={[styles.container, { paddingTop: insets.top }]}
      contentContainerStyle={styles.content}
      showsVerticalScrollIndicator={false}
    >
      {/* Header */}
      <Text style={styles.screenTitle}>Profile</Text>

      {isAnonymous ? (
        // ── Anonymous state ───────────────────────────────────────────────
        <View style={styles.anonContainer}>
          <View style={styles.anonIcon}>
            <Text style={styles.anonIconText}>👤</Text>
          </View>
          <Text style={styles.anonTitle}>You're not signed in</Text>
          <Text style={styles.anonSubtitle}>
            Create an account to sync your watchlist and ratings across devices.
          </Text>

          <TouchableOpacity
            style={styles.signInCard}
            onPress={() => router.push('/auth/login' as never)}
            activeOpacity={0.85}
          >
            <Text style={styles.signInCardTitle}>Sign In</Text>
            <Text style={styles.signInCardSubtitle}>
              Continue with your existing account
            </Text>
          </TouchableOpacity>

          <TouchableOpacity
            style={styles.createAccountCard}
            onPress={() => router.push('/auth/signup' as never)}
            activeOpacity={0.85}
          >
            <Text style={styles.createAccountTitle}>Create Account</Text>
            <Text style={styles.createAccountSubtitle}>
              Free · Sync watchlist · Personalised picks
            </Text>
          </TouchableOpacity>
        </View>
      ) : (
        // ── Logged-in state ───────────────────────────────────────────────
        <View style={styles.profileContainer}>
          {/* Avatar + info */}
          <View style={styles.avatarRow}>
            <View style={styles.avatar}>
              <Text style={styles.avatarText}>
                {user?.email?.[0]?.toUpperCase() ?? '?'}
              </Text>
            </View>
            <View style={styles.userInfo}>
              {user?.displayName ? (
                <Text style={styles.displayName}>{user.displayName}</Text>
              ) : null}
              <Text style={styles.email}>{user?.email}</Text>
              <Text style={styles.joinDate}>
                Member since{' '}
                {user?.createdAt
                  ? new Date(user.createdAt).getFullYear()
                  : '—'}
              </Text>
            </View>
          </View>

          {/* Stats row */}
          <View style={styles.statsRow}>
            <StatCard label="Saved" value={user?.stats.saved ?? 0} />
            <View style={styles.statDivider} />
            <StatCard label="Rated" value={user?.stats.rated ?? 0} />
            <View style={styles.statDivider} />
            <StatCard label="Skipped" value={user?.stats.dismissed ?? 0} />
          </View>

          {/* Taste Profile — scoring preset selector */}
          <View style={styles.tasteSection}>
            <Text style={styles.tasteSectionTitle}>Taste Profile</Text>
            <Text style={styles.tasteSectionHint}>
              How should Cinova weigh scores?
            </Text>
            <View style={styles.presetGrid}>
              {(scoringData?.presets ?? DEFAULT_PRESETS).map((p) => {
                const active = (scoringData?.preset ?? scoringPreset) === p.id;
                return (
                  <TouchableOpacity
                    key={p.id}
                    style={[styles.presetCard, active && styles.presetCardActive]}
                    onPress={() => presetMutation.mutate(p.id)}
                    activeOpacity={0.8}
                  >
                    <Text style={styles.presetEmoji}>{p.emoji}</Text>
                    <Text style={[styles.presetName, active && styles.presetNameActive]}>
                      {p.name}
                    </Text>
                    <Text style={styles.presetDesc} numberOfLines={2}>
                      {p.description}
                    </Text>
                  </TouchableOpacity>
                );
              })}
            </View>
          </View>

          {/* Sign out */}
          <TouchableOpacity
            style={styles.signOutBtn}
            onPress={handleLogout}
            activeOpacity={0.75}
          >
            <Text style={styles.signOutText}>Sign Out</Text>
          </TouchableOpacity>
        </View>
      )}

      {/* Region selector */}
      <View style={styles.section}>
        <Text style={styles.sectionTitle}>Your Region</Text>
        <TouchableOpacity
          style={styles.regionRow}
          onPress={() => setCountryModalVisible(true)}
          activeOpacity={0.75}
        >
          <Text style={styles.regionFlag}>{currentCountry.flag}</Text>
          <Text style={styles.regionName}>{currentCountry.name}</Text>
          <Text style={styles.regionChevron}>›</Text>
        </TouchableOpacity>
        <Text style={styles.regionHint}>
          Used to show streaming availability in your country.
        </Text>
      </View>

      {/* Premium button */}
      <View style={styles.section}>
        <TouchableOpacity
          style={styles.premiumBtn}
          onPress={handlePremium}
          activeOpacity={0.85}
        >
          <View style={styles.premiumBadge}>
            <Text style={styles.premiumBadgeText}>PRO</Text>
          </View>
          <View style={styles.premiumText}>
            <Text style={styles.premiumTitle}>Remove Ads</Text>
            <Text style={styles.premiumSubtitle}>
              Unlimited access · No interruptions
            </Text>
          </View>
          <Text style={styles.premiumChevron}>›</Text>
        </TouchableOpacity>
      </View>

      {/* App info */}
      <View style={styles.appInfo}>
        <Text style={styles.appInfoText}>Cinova v1.0.0</Text>
        <Text style={styles.appInfoText}>
          Powered by TMDB · JustWatch data
        </Text>
      </View>

      {/* Country picker modal */}
      <Modal
        visible={countryModalVisible}
        animationType="slide"
        presentationStyle="pageSheet"
        onRequestClose={() => setCountryModalVisible(false)}
      >
        <View style={styles.modal}>
          <View style={styles.modalHeader}>
            <Text style={styles.modalTitle}>Select Region</Text>
            <TouchableOpacity
              onPress={() => setCountryModalVisible(false)}
              style={styles.modalClose}
            >
              <Text style={styles.modalCloseText}>Done</Text>
            </TouchableOpacity>
          </View>
          <FlatList
            data={COUNTRIES}
            keyExtractor={(item) => item.code}
            renderItem={({ item }) => (
              <TouchableOpacity
                style={[
                  styles.countryRow,
                  item.code === country && styles.countryRowActive,
                ]}
                onPress={() => {
                  setCountry(item.code);
                  setCountryModalVisible(false);
                }}
                activeOpacity={0.75}
              >
                <Text style={styles.countryFlag}>{item.flag}</Text>
                <Text style={styles.countryName}>{item.name}</Text>
                {item.code === country && (
                  <Text style={styles.countryCheck}>✓</Text>
                )}
              </TouchableOpacity>
            )}
            contentContainerStyle={{ paddingBottom: 40 }}
          />
        </View>
      </Modal>
    </ScrollView>
  );
}

function StatCard({ label, value }: { label: string; value: number }) {
  return (
    <View style={statStyles.card}>
      <Text style={statStyles.value}>{value}</Text>
      <Text style={statStyles.label}>{label}</Text>
    </View>
  );
}

const statStyles = StyleSheet.create({
  card: {
    flex: 1,
    alignItems: 'center',
    paddingVertical: Spacing[3],
  },
  value: {
    color: Colors.textPrimary,
    fontSize: Typography['2xl'],
    fontWeight: Typography.bold,
  },
  label: {
    color: Colors.textMuted,
    fontSize: Typography.xs,
    marginTop: 2,
    textTransform: 'uppercase',
    letterSpacing: Typography.wider,
  },
});

const styles = StyleSheet.create({
  container: {
    flex: 1,
    backgroundColor: Colors.background,
  },
  content: {
    padding: Spacing[4],
    paddingBottom: Spacing[12],
    gap: Spacing[6],
  },
  screenTitle: {
    color: Colors.textPrimary,
    fontSize: Typography['2xl'],
    fontWeight: Typography.bold,
    marginBottom: Spacing[2],
  },
  // Anonymous
  anonContainer: {
    alignItems: 'center',
    gap: Spacing[4],
  },
  anonIcon: {
    width: 80,
    height: 80,
    borderRadius: Radius.full,
    backgroundColor: Colors.surface,
    justifyContent: 'center',
    alignItems: 'center',
    borderWidth: 2,
    borderColor: Colors.border,
  },
  anonIconText: {
    fontSize: 36,
  },
  anonTitle: {
    color: Colors.textPrimary,
    fontSize: Typography.xl,
    fontWeight: Typography.bold,
    textAlign: 'center',
  },
  anonSubtitle: {
    color: Colors.textSecondary,
    fontSize: Typography.base,
    textAlign: 'center',
    lineHeight: Typography.base * 1.5,
  },
  signInCard: {
    width: '100%',
    backgroundColor: Colors.primary,
    borderRadius: Radius.lg,
    padding: Spacing[4],
    ...Shadows.glow,
  },
  signInCardTitle: {
    color: Colors.textPrimary,
    fontSize: Typography.lg,
    fontWeight: Typography.bold,
  },
  signInCardSubtitle: {
    color: 'rgba(255,255,255,0.8)',
    fontSize: Typography.sm,
    marginTop: 4,
  },
  createAccountCard: {
    width: '100%',
    backgroundColor: Colors.card,
    borderRadius: Radius.lg,
    padding: Spacing[4],
    borderWidth: 1.5,
    borderColor: Colors.border,
  },
  createAccountTitle: {
    color: Colors.textPrimary,
    fontSize: Typography.lg,
    fontWeight: Typography.bold,
  },
  createAccountSubtitle: {
    color: Colors.textSecondary,
    fontSize: Typography.sm,
    marginTop: 4,
  },
  // Logged in
  profileContainer: {
    gap: Spacing[4],
  },
  avatarRow: {
    flexDirection: 'row',
    alignItems: 'center',
    gap: Spacing[4],
  },
  avatar: {
    width: 64,
    height: 64,
    borderRadius: Radius.full,
    backgroundColor: Colors.primary,
    justifyContent: 'center',
    alignItems: 'center',
  },
  avatarText: {
    color: Colors.textPrimary,
    fontSize: Typography['2xl'],
    fontWeight: Typography.bold,
  },
  userInfo: {
    flex: 1,
    gap: 2,
  },
  displayName: {
    color: Colors.textPrimary,
    fontSize: Typography.lg,
    fontWeight: Typography.bold,
  },
  email: {
    color: Colors.textSecondary,
    fontSize: Typography.sm,
  },
  joinDate: {
    color: Colors.textMuted,
    fontSize: Typography.xs,
    marginTop: 2,
  },
  statsRow: {
    flexDirection: 'row',
    backgroundColor: Colors.card,
    borderRadius: Radius.lg,
    borderWidth: 1,
    borderColor: Colors.border,
    overflow: 'hidden',
  },
  statDivider: {
    width: 1,
    backgroundColor: Colors.border,
  },
  tasteSection: {
    gap: Spacing[2],
  },
  tasteSectionTitle: {
    color: Colors.textPrimary,
    fontSize: Typography.base,
    fontWeight: Typography.bold,
  },
  tasteSectionHint: {
    color: Colors.textMuted,
    fontSize: Typography.xs,
  },
  presetGrid: {
    flexDirection: 'row',
    flexWrap: 'wrap',
    gap: Spacing[2],
    marginTop: Spacing[1],
  },
  presetCard: {
    width: '47%',
    backgroundColor: Colors.card,
    borderRadius: Radius.md,
    padding: Spacing[3],
    borderWidth: 1,
    borderColor: Colors.border,
    gap: Spacing[1],
  },
  presetCardActive: {
    borderColor: Colors.primary,
    backgroundColor: Colors.elevated,
  },
  presetEmoji: {
    fontSize: 22,
  },
  presetName: {
    color: Colors.textSecondary,
    fontSize: Typography.sm,
    fontWeight: Typography.semibold,
  },
  presetNameActive: {
    color: Colors.primary,
  },
  presetDesc: {
    color: Colors.textMuted,
    fontSize: Typography.xs,
    lineHeight: Typography.xs * 1.4,
  },
  signOutBtn: {
    alignSelf: 'flex-start',
    paddingVertical: Spacing[2],
  },
  signOutText: {
    color: Colors.primary,
    fontSize: Typography.base,
    fontWeight: Typography.medium,
  },
  // Section
  section: {
    gap: Spacing[3],
  },
  sectionTitle: {
    color: Colors.textPrimary,
    fontSize: Typography.lg,
    fontWeight: Typography.bold,
  },
  // Region
  regionRow: {
    flexDirection: 'row',
    alignItems: 'center',
    backgroundColor: Colors.card,
    borderRadius: Radius.md,
    padding: Spacing[4],
    borderWidth: 1,
    borderColor: Colors.border,
    gap: Spacing[3],
  },
  regionFlag: {
    fontSize: 24,
  },
  regionName: {
    flex: 1,
    color: Colors.textPrimary,
    fontSize: Typography.base,
    fontWeight: Typography.medium,
  },
  regionChevron: {
    color: Colors.textMuted,
    fontSize: Typography.xl,
  },
  regionHint: {
    color: Colors.textMuted,
    fontSize: Typography.xs,
  },
  // Premium
  premiumBtn: {
    flexDirection: 'row',
    alignItems: 'center',
    backgroundColor: Colors.elevated,
    borderRadius: Radius.lg,
    padding: Spacing[4],
    borderWidth: 1.5,
    borderColor: Colors.scoreMid,
    gap: Spacing[3],
  },
  premiumBadge: {
    backgroundColor: Colors.scoreMid,
    borderRadius: Radius.sm,
    paddingHorizontal: Spacing[2],
    paddingVertical: Spacing[0.5],
  },
  premiumBadgeText: {
    color: Colors.textInverse,
    fontSize: Typography.xs,
    fontWeight: Typography.black,
    letterSpacing: Typography.wider,
  },
  premiumText: {
    flex: 1,
  },
  premiumTitle: {
    color: Colors.textPrimary,
    fontSize: Typography.base,
    fontWeight: Typography.bold,
  },
  premiumSubtitle: {
    color: Colors.textSecondary,
    fontSize: Typography.xs,
    marginTop: 2,
  },
  premiumChevron: {
    color: Colors.textMuted,
    fontSize: Typography.xl,
  },
  // App info
  appInfo: {
    alignItems: 'center',
    gap: Spacing[1],
    paddingTop: Spacing[4],
  },
  appInfoText: {
    color: Colors.textMuted,
    fontSize: Typography.xs,
  },
  // Modal
  modal: {
    flex: 1,
    backgroundColor: Colors.surface,
  },
  modalHeader: {
    flexDirection: 'row',
    justifyContent: 'space-between',
    alignItems: 'center',
    padding: Spacing[4],
    borderBottomWidth: 1,
    borderBottomColor: Colors.border,
  },
  modalTitle: {
    color: Colors.textPrimary,
    fontSize: Typography.lg,
    fontWeight: Typography.bold,
  },
  modalClose: {
    padding: Spacing[2],
  },
  modalCloseText: {
    color: Colors.primary,
    fontSize: Typography.base,
    fontWeight: Typography.semibold,
  },
  countryRow: {
    flexDirection: 'row',
    alignItems: 'center',
    padding: Spacing[4],
    gap: Spacing[3],
    borderBottomWidth: 1,
    borderBottomColor: Colors.borderFaint,
  },
  countryRowActive: {
    backgroundColor: Colors.card,
  },
  countryFlag: {
    fontSize: 24,
  },
  countryName: {
    flex: 1,
    color: Colors.textPrimary,
    fontSize: Typography.base,
  },
  countryCheck: {
    color: Colors.primary,
    fontSize: Typography.lg,
    fontWeight: Typography.bold,
  },
});
