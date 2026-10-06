type InstallationToken = { token: string; expires_at: string };

const REFRESH_MARGIN_MS = 5 * 60 * 1000;

export function createInstallationTokenCache(
  mint: (installationId: number) => Promise<InstallationToken>,
  now: () => number = Date.now,
): (installationId: number) => Promise<InstallationToken> {
  const tokens = new Map<number, Promise<InstallationToken>>();

  return async (installationId) => {
    const cached = tokens.get(installationId);
    if (cached) {
      const token = await cached.catch(() => null);
      if (token && Date.parse(token.expires_at) - REFRESH_MARGIN_MS > now()) {
        return token;
      }
      if (tokens.get(installationId) === cached) {
        tokens.delete(installationId);
      }
    }
    const minted = mint(installationId);
    tokens.set(installationId, minted);
    minted.catch(() => {
      if (tokens.get(installationId) === minted) {
        tokens.delete(installationId);
      }
    });
    return minted;
  };
}
