/**
 * Service for managing refresh token ownership verification
 * TODO: Replace with Redis or other global implementation (Planetscale?)
 */
export class TokenManager {
  private static instance: TokenManager;
  private tokenOwners: Map<string, string> = new Map();

  private constructor() {
    // Private constructor to enforce singleton
  }

  /**
   * Get the singleton instance
   */
  public static getInstance(): TokenManager {
    if (!TokenManager.instance) {
      TokenManager.instance = new TokenManager();
    }
    return TokenManager.instance;
  }

  /**
   * Check if a refresh token is owned by the given user identity
   * If no ownership is recorded, associate the token with this user
   */
  public verifyTokenOwnership({
    refreshToken,
    userIdentity,
  }: {
    refreshToken: string;
    userIdentity: string;
  }): boolean {
    const owner = this.tokenOwners.get(refreshToken);

    // If we don't have ownership data, create it now
    if (!owner) {
      this.tokenOwners.set(refreshToken, userIdentity);
      return true;
    }

    return owner === userIdentity;
  }

  /**
   * Update token ownership mapping after a successful refresh
   */
  public updateTokenOwnership({
    oldToken,
    newToken,
    userIdentity,
  }: {
    oldToken: string;
    newToken: string;
    userIdentity: string;
  }): void {
    // Remove the old mapping
    this.tokenOwners.delete(oldToken);

    // Add the new mapping
    this.tokenOwners.set(newToken, userIdentity);
  }

  /**
   * Remove a token from the ownership mapping
   */
  public removeToken(token: string): boolean {
    return this.tokenOwners.delete(token);
  }

  /**
   * Get the count of tracked tokens (for debugging)
   */
  public getTokenCount(): number {
    return this.tokenOwners.size;
  }
}

// Export a singleton instance
export const tokenManager = TokenManager.getInstance();
