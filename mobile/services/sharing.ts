/**
 * Social sharing helpers
 */

import { Share, Platform } from 'react-native';
import type { Movie, TVShow } from '../types';

const TMDB_IMAGE = 'https://image.tmdb.org/t/p';
const APP_LINK_BASE = 'https://cinova.app'; // universal link placeholder

export async function shareMovie(movie: Movie): Promise<void> {
  const posterUrl = movie.posterPath
    ? `${TMDB_IMAGE}/w500${movie.posterPath}`
    : undefined;

  const score = movie.cinovaScore != null
    ? ` — CinovaScore: ${movie.cinovaScore}/100`
    : '';

  const message = `${movie.title} (${movie.year})${score}\n\n${movie.overview?.slice(0, 120)}...`;
  const url = `${APP_LINK_BASE}/movie/${movie.id}`;

  try {
    await Share.share(
      {
        title: movie.title,
        message: Platform.OS === 'android' ? `${message}\n${url}` : message,
        url: Platform.OS === 'ios' ? url : undefined,
      },
      {
        dialogTitle: `Share "${movie.title}"`,
      }
    );
  } catch (_) {
    // Share cancelled — no-op
  }
}

export async function shareTV(show: TVShow): Promise<void> {
  const score = show.cinovaScore != null
    ? ` — CinovaScore: ${show.cinovaScore}/100`
    : '';

  const message = `${show.name} (${show.year})${score}\n\n${show.overview?.slice(0, 120)}...`;
  const url = `${APP_LINK_BASE}/tv/${show.id}`;

  try {
    await Share.share(
      {
        title: show.name,
        message: Platform.OS === 'android' ? `${message}\n${url}` : message,
        url: Platform.OS === 'ios' ? url : undefined,
      },
      {
        dialogTitle: `Share "${show.name}"`,
      }
    );
  } catch (_) {
    // Share cancelled — no-op
  }
}
