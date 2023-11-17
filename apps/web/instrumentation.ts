function checkEnvironmentVariables(envs: string[]) {
  envs.forEach((env) => {
    if (!process.env[env]) {
      throw new Error(`${env} is undefined`);
    }
  });
}

export function register() {
  const requiredEnvs = [
    // Clerk
    "NEXT_PUBLIC_CLERK_PUBLISHABLE_KEY",
    "CLERK_SECRET_KEY",
    // Database
    "DATABASE_HOST",
    "DATABASE_USERNAME",
    "DATABASE_PASSWORD",
    // Unkey
    "UNKEY_WORKSPACE_ID",
    "UNKEY_API_ID",
    "UNKEY_APP_AUTH_TOKEN",
  ];

  checkEnvironmentVariables(requiredEnvs);
}
