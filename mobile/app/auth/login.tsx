/**
 * Login screen
 *
 * - Email + password inputs
 * - Calls POST /api/v1/auth/login via the existing api.ts client
 * - Passes anonymous session UUID for server-side merge
 * - On success: persists JWT to SecureStore, updates Zustand store, navigates to tabs
 */

import React, { useState } from 'react';
import {
  View,
  Text,
  TextInput,
  TouchableOpacity,
  StyleSheet,
  ActivityIndicator,
  KeyboardAvoidingView,
  Platform,
  ScrollView,
  Alert,
} from 'react-native';
import { useRouter, Link } from 'expo-router';
import { useSafeAreaInsets } from 'react-native-safe-area-context';
import { Colors, Typography, Spacing, Radius } from '../../constants/theme';
import { login } from '../../services/api';
import { saveToken, getSessionId } from '../../services/session';
import { useAppStore } from '../../store/useAppStore';

export default function LoginScreen() {
  const router = useRouter();
  const insets = useSafeAreaInsets();
  const setUser = useAppStore((s) => s.setUser);
  const setSessionId = useAppStore((s) => s.setSessionId);

  const [email, setEmail] = useState('');
  const [password, setPassword] = useState('');
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);

  async function handleLogin() {
    if (!email.trim() || !password) {
      setError('Please enter your email and password.');
      return;
    }

    setError(null);
    setLoading(true);

    try {
      const sessionId = await getSessionId();
      const response = await login(email.trim().toLowerCase(), password, sessionId ?? '');

      // Persist token securely
      await saveToken(response.access_token);

      // Update store
      setUser({ id: response.user_id, email: email.trim().toLowerCase(), country: 'US', isPremium: false, createdAt: '', stats: { saved: 0, rated: 0, dismissed: 0 } });
      setSessionId(sessionId ?? '');

      router.replace('/(tabs)');
    } catch (err: unknown) {
      const msg =
        err instanceof Error ? err.message : 'Login failed. Please try again.';
      setError(msg);
    } finally {
      setLoading(false);
    }
  }

  return (
    <KeyboardAvoidingView
      style={styles.flex}
      behavior={Platform.OS === 'ios' ? 'padding' : 'height'}
    >
      {/* Back button */}
      <TouchableOpacity
        style={[styles.backBtn, { top: insets.top + 10 }]}
        onPress={() => router.canGoBack() ? router.back() : router.replace('/(tabs)')}
        hitSlop={{ top: 16, bottom: 16, left: 16, right: 16 }}
        activeOpacity={0.7}
      >
        <Text style={{ color: Colors.textPrimary, fontSize: 24, lineHeight: 28 }}>‹</Text>
      </TouchableOpacity>

      <ScrollView
        contentContainerStyle={styles.container}
        keyboardShouldPersistTaps="handled"
      >
        {/* Logo / wordmark */}
        <View style={styles.header}>
          <Text style={styles.wordmark}>CINOVA</Text>
          <Text style={styles.tagline}>Your cinematic universe</Text>
        </View>

        {/* SSO Options */}
        <View style={styles.ssoContainer}>
          {Platform.OS === 'ios' && (
            <TouchableOpacity
              style={styles.ssoButton}
              activeOpacity={0.85}
              onPress={() => Alert.alert('Apple Sign In', 'Apple Sign In requires an EAS production build. Coming soon!')}
            >
              <Text style={{ color: Colors.textPrimary, fontSize: 18, fontWeight: '600' }}></Text>
              <Text style={styles.ssoButtonText}>Continue with Apple</Text>
            </TouchableOpacity>
          )}
          <TouchableOpacity
            style={[styles.ssoButton, styles.ssoButtonGoogle]}
            activeOpacity={0.85}
            onPress={() => Alert.alert('Google Sign In', 'Google Sign In requires an EAS production build. Coming soon!')}
          >
            <Text style={{ color: '#fff', fontSize: 16, fontWeight: '700' }}>G</Text>
            <Text style={styles.ssoButtonText}>Continue with Google</Text>
          </TouchableOpacity>
        </View>

        {/* Divider */}
        <View style={styles.divider}>
          <View style={styles.dividerLine} />
          <Text style={styles.dividerText}>or sign in with email</Text>
          <View style={styles.dividerLine} />
        </View>

        {/* Form */}
        <View style={styles.form}>
          <Text style={styles.label}>Email</Text>
          <TextInput
            style={styles.input}
            placeholder="you@example.com"
            placeholderTextColor={Colors.textMuted}
            value={email}
            onChangeText={setEmail}
            keyboardType="email-address"
            autoCapitalize="none"
            autoCorrect={false}
            returnKeyType="next"
            editable={!loading}
          />

          <Text style={[styles.label, styles.labelSpaced]}>Password</Text>
          <TextInput
            style={styles.input}
            placeholder="••••••••"
            placeholderTextColor={Colors.textMuted}
            value={password}
            onChangeText={setPassword}
            secureTextEntry
            returnKeyType="done"
            onSubmitEditing={handleLogin}
            editable={!loading}
          />

          {error ? <Text style={styles.error}>{error}</Text> : null}

          <TouchableOpacity
            style={[styles.button, loading && styles.buttonDisabled]}
            onPress={handleLogin}
            disabled={loading}
            activeOpacity={0.8}
          >
            {loading ? (
              <ActivityIndicator color={Colors.textPrimary} />
            ) : (
              <Text style={styles.buttonText}>Log In</Text>
            )}
          </TouchableOpacity>
        </View>

        {/* Sign-up link */}
        <View style={styles.footer}>
          <Text style={styles.footerText}>Don&apos;t have an account? </Text>
          <Link href="/auth/signup" asChild>
            <TouchableOpacity>
              <Text style={styles.footerLink}>Sign up</Text>
            </TouchableOpacity>
          </Link>
        </View>
      </ScrollView>
    </KeyboardAvoidingView>
  );
}

const styles = StyleSheet.create({
  flex: {
    flex: 1,
    backgroundColor: Colors.background,
  },
  backBtn: {
    position: 'absolute',
    left: 16,
    zIndex: 10,
    width: 40,
    height: 40,
    borderRadius: 20,
    backgroundColor: 'rgba(255,255,255,0.08)',
    alignItems: 'center',
    justifyContent: 'center',
  },
  container: {
    flexGrow: 1,
    justifyContent: 'center',
    paddingHorizontal: Spacing[4],
    paddingVertical: Spacing[12],
  },
  header: {
    alignItems: 'center',
    marginBottom: Spacing[10],
  },
  wordmark: {
    fontSize: Typography['4xl'],
    fontWeight: '900',
    color: Colors.primary,
    letterSpacing: 6,
  },
  tagline: {
    fontSize: Typography.sm,
    color: Colors.textMuted,
    marginTop: Spacing[1],
    letterSpacing: 1,
  },
  form: {
    gap: 0,
  },
  label: {
    fontSize: Typography.sm,
    color: Colors.textSecondary,
    marginBottom: Spacing[1],
    fontWeight: '500',
  },
  labelSpaced: {
    marginTop: Spacing[4],
  },
  input: {
    backgroundColor: Colors.surface,
    borderWidth: 1,
    borderColor: Colors.border,
    borderRadius: Radius.md,
    paddingHorizontal: Spacing[4],
    paddingVertical: Spacing[3],
    fontSize: Typography.base,
    color: Colors.textPrimary,
  },
  error: {
    color: Colors.primary,
    fontSize: Typography.sm,
    marginTop: Spacing[2],
    textAlign: 'center',
  },
  button: {
    backgroundColor: Colors.primary,
    borderRadius: Radius.md,
    paddingVertical: Spacing[4],
    alignItems: 'center',
    marginTop: Spacing[6],
  },
  buttonDisabled: {
    opacity: 0.6,
  },
  buttonText: {
    color: Colors.textPrimary,
    fontSize: Typography.md,
    fontWeight: '700',
  },
  ssoContainer: {
    gap: Spacing[3],
    marginBottom: Spacing[6],
  },
  ssoButton: {
    flexDirection: 'row',
    alignItems: 'center',
    justifyContent: 'center',
    gap: Spacing[3],
    backgroundColor: '#1a1a1a',
    borderWidth: 1,
    borderColor: Colors.border,
    borderRadius: Radius.md,
    paddingVertical: Spacing[4],
  },
  ssoButtonGoogle: {
    backgroundColor: '#4285F4',
    borderColor: '#4285F4',
  },
  ssoButtonText: {
    color: Colors.textPrimary,
    fontSize: Typography.base,
    fontWeight: '600',
  },
  divider: {
    flexDirection: 'row',
    alignItems: 'center',
    gap: Spacing[3],
    marginBottom: Spacing[6],
  },
  dividerLine: {
    flex: 1,
    height: 1,
    backgroundColor: Colors.border,
  },
  dividerText: {
    color: Colors.textMuted,
    fontSize: Typography.xs,
    fontWeight: '500',
  },
  footer: {
    flexDirection: 'row',
    justifyContent: 'center',
    marginTop: Spacing[8],
  },
  footerText: {
    color: Colors.textSecondary,
    fontSize: Typography.sm,
  },
  footerLink: {
    color: Colors.primary,
    fontSize: Typography.sm,
    fontWeight: '600',
  },
});
