/**
 * Signup screen
 *
 * - Email, username, password inputs
 * - Calls POST /api/v1/auth/signup via the existing api.ts client
 * - Passes anonymous session UUID so the server can merge watchlist/ratings
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
import { Ionicons } from '@expo/vector-icons';
import { Colors, Typography, Spacing, Radius } from '../../constants/theme';
import { signUp } from '../../services/api';
import { saveToken, getSessionId } from '../../services/session';
import { useAppStore } from '../../store/useAppStore';

export default function SignupScreen() {
  const router = useRouter();
  const setUser = useAppStore((s) => s.setUser);
  const setSessionId = useAppStore((s) => s.setSessionId);

  const [email, setEmail] = useState('');
  const [username, setUsername] = useState('');
  const [password, setPassword] = useState('');
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);

  async function handleSignup() {
    if (!email.trim() || !password || !username.trim()) {
      setError('Please fill in all fields.');
      return;
    }
    if (password.length < 8) {
      setError('Password must be at least 8 characters.');
      return;
    }

    setError(null);
    setLoading(true);

    try {
      const sessionId = await getSessionId();
      // signUp accepts email, password, sessionId — username is sent via email field convention.
      // The API accepts a `displayName` field; pass username as display name.
      const response = await signUp(
        email.trim().toLowerCase(),
        password,
        username.trim(),
        sessionId ?? ''
      );

      // Persist token securely
      await saveToken(response.access_token);

      // Update store
      setUser({ id: response.user_id, email: email.trim().toLowerCase(), country: 'US', isPremium: false, createdAt: '', stats: { saved: 0, rated: 0, dismissed: 0 } });
      setSessionId(sessionId ?? '');

      router.replace('/(tabs)');
    } catch (err: unknown) {
      const msg =
        err instanceof Error
          ? err.message
          : 'Sign up failed. Please try again.';
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
      <ScrollView
        contentContainerStyle={styles.container}
        keyboardShouldPersistTaps="handled"
      >
        {/* Logo / wordmark */}
        <View style={styles.header}>
          <Text style={styles.wordmark}>CINOVA</Text>
          <Text style={styles.tagline}>Create your account</Text>
        </View>

        {/* SSO Options */}
        <View style={styles.ssoContainer}>
          {Platform.OS === 'ios' && (
            <TouchableOpacity
              style={styles.ssoButton}
              activeOpacity={0.85}
              onPress={() => Alert.alert('Apple Sign In', 'Apple Sign In requires an EAS production build. Coming soon!')}
            >
              <Ionicons name="logo-apple" size={20} color={Colors.textPrimary} />
              <Text style={styles.ssoButtonText}>Continue with Apple</Text>
            </TouchableOpacity>
          )}
          <TouchableOpacity
            style={[styles.ssoButton, styles.ssoButtonGoogle]}
            activeOpacity={0.85}
            onPress={() => Alert.alert('Google Sign In', 'Google Sign In requires an EAS production build. Coming soon!')}
          >
            <Ionicons name="logo-google" size={18} color="#fff" />
            <Text style={styles.ssoButtonText}>Continue with Google</Text>
          </TouchableOpacity>
        </View>

        {/* Divider */}
        <View style={styles.divider}>
          <View style={styles.dividerLine} />
          <Text style={styles.dividerText}>or sign up with email</Text>
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

          <Text style={[styles.label, styles.labelSpaced]}>Username</Text>
          <TextInput
            style={styles.input}
            placeholder="cinephile42"
            placeholderTextColor={Colors.textMuted}
            value={username}
            onChangeText={setUsername}
            autoCapitalize="none"
            autoCorrect={false}
            returnKeyType="next"
            editable={!loading}
          />

          <Text style={[styles.label, styles.labelSpaced]}>Password</Text>
          <TextInput
            style={styles.input}
            placeholder="At least 8 characters"
            placeholderTextColor={Colors.textMuted}
            value={password}
            onChangeText={setPassword}
            secureTextEntry
            returnKeyType="done"
            onSubmitEditing={handleSignup}
            editable={!loading}
          />

          {error ? <Text style={styles.error}>{error}</Text> : null}

          <TouchableOpacity
            style={[styles.button, loading && styles.buttonDisabled]}
            onPress={handleSignup}
            disabled={loading}
            activeOpacity={0.8}
          >
            {loading ? (
              <ActivityIndicator color={Colors.textPrimary} />
            ) : (
              <Text style={styles.buttonText}>Create Account</Text>
            )}
          </TouchableOpacity>
        </View>

        {/* Login link */}
        <View style={styles.footer}>
          <Text style={styles.footerText}>Already have an account? </Text>
          <Link href="/auth/login" asChild>
            <TouchableOpacity>
              <Text style={styles.footerLink}>Log in</Text>
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
