export const NETWORK_ERROR = "Network error!";
export const RATE_LIMIT_ERROR = "Rate limit exceeded!";

export class NetworkError extends Error {
  constructor(message?: string) {
    super([NETWORK_ERROR, message].filter(Boolean).join(": "));
  }
}
export class RateLimitError extends Error {
  constructor(message?: string) {
    super([RATE_LIMIT_ERROR, message].filter(Boolean).join(": "));
  }
}
