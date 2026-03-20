/**
 * Cinova — Core TypeScript types
 */

export interface Award {
  wikidataId?: string;
  awardName: string;
  ceremonyName?: string;
  year?: number;
  recipientName?: string;
  category?: string;
  isNomination: boolean;
}

export interface Movie {
  id: number;
  tmdbId: number;
  title: string;
  originalTitle: string;
  overview: string;
  tagline?: string;
  backdropPath: string | null;
  posterPath: string | null;
  releaseDate: string;
  year: number;
  runtime: number | null;
  language: string;
  genres: Genre[];
  themes: Theme[];
  moods: Mood[];
  awards?: Award[];
  keywords?: string[];
  cinovaScore: number | null;
  voteAverage: number;
  voteCount: number;
  popularity: number;
  providers: WatchProvider[];
  cast: CastMember[];
  trailer?: Trailer;
  trailerYoutubeKey?: string;
  plotSummary?: string;
  cinovaSynopsis?: string;
  /** @deprecated use cinovaSynopsis */
  aiDescription?: string;
}

export interface TVShow {
  id: number;
  tmdbId: number;
  name: string;
  originalName: string;
  overview: string;
  tagline?: string;
  backdropPath: string | null;
  posterPath: string | null;
  firstAirDate: string;
  year: number;
  episodeRuntime: number[];
  language: string;
  genres: Genre[];
  themes: Theme[];
  moods: Mood[];
  awards?: Award[];
  keywords?: string[];
  cinovaScore: number | null;
  voteAverage: number;
  voteCount: number;
  popularity: number;
  providers: WatchProvider[];
  seasons: number;
  trailerYoutubeKey?: string;
  plotSummary?: string;
  cinovaSynopsis?: string;
  /** @deprecated use cinovaSynopsis */
  aiDescription?: string;
}

export interface Genre {
  id: number;
  name: string;
}

export interface Theme {
  id: number;
  name: string;
}

export interface Mood {
  id: number;
  name: string;
}

export interface WatchProvider {
  providerId: number;
  providerName: string;
  logoPath: string | null;
  displayPriority: number;
  link: string;
}

export interface CastMember {
  id?: number;
  tmdbId?: number;
  name: string;
  character?: string;
  role?: string;
  profilePath: string | null;
  order: number;
}

export interface Person {
  id: number;
  tmdbId: number;
  name: string;
  biography: string;
  birthday: string | null;
  deathday: string | null;
  placeOfBirth: string | null;
  profilePath: string | null;
  knownForDepartment: string;
  nationality: string | null;
  popularity: number;
  knownFor: Movie[];
  filmography: FilmographyEntry[];
}

export interface FilmographyEntry {
  id: number;
  tmdbId: number;
  title: string;
  posterPath: string | null;
  releaseDate: string;
  year: number;
  character?: string;
  job?: string;
  mediaType: 'movie' | 'tv';
}

export interface Provider {
  id: number;
  name: string;
  logoPath: string;
  color: string;
}

export interface SearchResult {
  items: (Movie | TVShow)[];
  total: number;
  page: number;
  hasMore: boolean;
  query: string;
}

export interface User {
  id: string;
  email: string;
  displayName?: string;
  avatarUrl?: string;
  createdAt: string;
  country: string;
  isPremium: boolean;
  stats: UserStats;
}

export interface UserStats {
  saved: number;
  rated: number;
  dismissed: number;
}

/** Matches Go backend AuthResponse (snake_case) */
export interface AuthResponse {
  access_token: string;
  refresh_token: string;
  token_type: string;
  expires_in: number;
  user_id: string;
  anonymous: boolean;
}

export interface CinovaScoreData {
  score: number;
  breakdown: {
    quality: number;
    relevance: number;
    availability: number;
  };
  confidence: 'high' | 'medium' | 'low';
}

export interface Trailer {
  key: string;
  site: 'YouTube' | 'Vimeo';
  name: string;
  type: string;
  official: boolean;
}

export type MediaType = 'movie' | 'tv';

export interface ScoringProfile {
  preset: string;
  audience: number;
  critic: number;
  award: number;
  prestige: number;
  commercial: number;
  presets?: ScoringPresetDescription[];
}

export interface ScoringPresetDescription {
  id: string;
  name: string;
  emoji: string;
  description: string;
  audience: number;
  critic: number;
  award: number;
  prestige: number;
  commercial: number;
}
