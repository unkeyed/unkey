/**
 * Cache reads within this time since writing will be considered fresh and can be used as is
 */
export const CACHE_FRESHNESS_TIME_MS = 1 * 60 * 1000; // 1 minute
/**
 * Cache reads within this time sicne writing can be used but will run a revalidation in the background
 */
export const CACHE_STALENESS_TIME_MS = 24 * 60 * 60 * 1000; // 24 hours
