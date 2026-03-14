/**
 * Streaming provider configuration.
 *
 * Maps TMDB watch provider IDs to display metadata and
 * deep-link URL builders for each platform.
 *
 * TMDB provider IDs: https://developer.themoviedb.org/reference/watch-provider-movie-list
 */

export interface StreamingProvider {
  id: number;
  name: string;
  /** Key into local asset map or remote logo URL */
  logoKey: StreamingProviderKey;
  /** Brand color for badge borders / accents */
  color: string;
  /** Build a deep link to this title on the provider's app */
  buildDeepLink: (tmdbId: number, title?: string) => string;
  /** Build an App Store / Play Store fallback URL if deep link fails */
  storeUrl: { ios: string; android: string };
}

export type StreamingProviderKey =
  | 'netflix'
  | 'prime'
  | 'disney'
  | 'hulu'
  | 'hbo'
  | 'apple'
  | 'peacock'
  | 'paramount'
  | 'unknown';

/** Maps logo key → remote CDN URL (replace with local assets for production) */
export const PROVIDER_LOGOS: Record<StreamingProviderKey, string> = {
  netflix:   'https://assets.cinova.app/providers/netflix.png',
  prime:     'https://assets.cinova.app/providers/prime.png',
  disney:    'https://assets.cinova.app/providers/disney.png',
  hulu:      'https://assets.cinova.app/providers/hulu.png',
  hbo:       'https://assets.cinova.app/providers/hbo.png',
  apple:     'https://assets.cinova.app/providers/apple.png',
  peacock:   'https://assets.cinova.app/providers/peacock.png',
  paramount: 'https://assets.cinova.app/providers/paramount.png',
  unknown:   'https://assets.cinova.app/providers/unknown.png',
};

export const STREAMING_PROVIDERS: StreamingProvider[] = [
  {
    id: 8,
    name: 'Netflix',
    logoKey: 'netflix',
    color: '#E50914',
    buildDeepLink: (tmdbId) => `netflix://title/${tmdbId}`,
    storeUrl: {
      ios: 'https://apps.apple.com/app/netflix/id363590051',
      android: 'https://play.google.com/store/apps/details?id=com.netflix.mediaclient',
    },
  },
  {
    id: 9,
    name: 'Amazon Prime',
    logoKey: 'prime',
    color: '#00A8E1',
    buildDeepLink: (tmdbId) => `aiv://aiv/resume?asin=${tmdbId}`,
    storeUrl: {
      ios: 'https://apps.apple.com/app/amazon-prime-video/id545519333',
      android: 'https://play.google.com/store/apps/details?id=com.amazon.avod.thirdpartyclient',
    },
  },
  {
    id: 337,
    name: 'Disney+',
    logoKey: 'disney',
    color: '#113CCF',
    buildDeepLink: (tmdbId) => `disneyplus://deeplink?contentId=${tmdbId}`,
    storeUrl: {
      ios: 'https://apps.apple.com/app/disney/id1446075923',
      android: 'https://play.google.com/store/apps/details?id=com.disney.disneyplus',
    },
  },
  {
    id: 15,
    name: 'Hulu',
    logoKey: 'hulu',
    color: '#1CE783',
    buildDeepLink: (tmdbId) => `hulu://series/${tmdbId}`,
    storeUrl: {
      ios: 'https://apps.apple.com/app/hulu-watch-tv-shows-movies/id376510438',
      android: 'https://play.google.com/store/apps/details?id=com.hulu.plus',
    },
  },
  {
    id: 1899,
    name: 'Max',
    logoKey: 'hbo',
    color: '#5822B4',
    buildDeepLink: (tmdbId) => `max://title/${tmdbId}`,
    storeUrl: {
      ios: 'https://apps.apple.com/app/max-stream-hbo-tv-movies/id1666927579',
      android: 'https://play.google.com/store/apps/details?id=com.hbo.hbonow',
    },
  },
  {
    id: 350,
    name: 'Apple TV+',
    logoKey: 'apple',
    color: '#A2AAAD',
    buildDeepLink: (tmdbId) => `videos://`,
    storeUrl: {
      ios: 'https://apps.apple.com/app/apple-tv/id1146438582',
      android: 'https://play.google.com/store/apps/details?id=com.apple.atve.androidtv.appletv',
    },
  },
  {
    id: 386,
    name: 'Peacock',
    logoKey: 'peacock',
    color: '#FF5F00',
    buildDeepLink: (tmdbId) => `peacocktv://`,
    storeUrl: {
      ios: 'https://apps.apple.com/app/peacock-tv/id1508186374',
      android: 'https://play.google.com/store/apps/details?id=com.peacocktv.peacockandroid',
    },
  },
  {
    id: 531,
    name: 'Paramount+',
    logoKey: 'paramount',
    color: '#0064FF',
    buildDeepLink: (tmdbId) => `paramountplus://`,
    storeUrl: {
      ios: 'https://apps.apple.com/app/paramount/id1340650234',
      android: 'https://play.google.com/store/apps/details?id=com.cbs.app',
    },
  },
];

/** Look up a provider by TMDB ID. Returns undefined if unknown. */
export function getProviderById(id: number): StreamingProvider | undefined {
  return STREAMING_PROVIDERS.find((p) => p.id === id);
}

/** Map of provider ID → provider (for O(1) lookup) */
export const PROVIDER_MAP = new Map<number, StreamingProvider>(
  STREAMING_PROVIDERS.map((p) => [p.id, p])
);
