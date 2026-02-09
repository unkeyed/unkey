import { db, eq, schema } from "@/lib/db";
import { env } from "@/lib/env";
import { Resend } from "@unkey/resend";
import { betterAuth } from "better-auth";
import { drizzleAdapter } from "better-auth/adapters/drizzle";
import { nextCookies } from "better-auth/next-js";
import { admin, emailOTP, organization } from "better-auth/plugins";

/**
 * Better Auth server instance configuration.
 *
 * This instance is created lazily and should only be used when AUTH_PROVIDER=better-auth.
 * It configures:
 * - Drizzle adapter with MySQL provider using ba_ prefixed tables
 * - Email OTP plugin for passwordless authentication
 * - Organization plugin for multi-tenancy with admin as creator role
 * - Admin plugin for user impersonation
 * - nextCookies plugin for server action cookie handling
 * - GitHub and Google OAuth providers
 * - 7-day session expiry with cookie caching for performance
 * - Database hooks to auto-set activeOrganizationId on session creation
 */
export function createBetterAuthInstance() {
  const config = env();

  // Validate required env vars for Better Auth
  if (!config.BETTER_AUTH_SECRET) {
    throw new Error("BETTER_AUTH_SECRET is required when AUTH_PROVIDER=better-auth");
  }
  if (!config.BETTER_AUTH_URL) {
    throw new Error("BETTER_AUTH_URL is required when AUTH_PROVIDER=better-auth");
  }

  // Initialize Resend client if API key is available
  const resend = config.RESEND_API_KEY ? new Resend({ apiKey: config.RESEND_API_KEY }) : null;

  return betterAuth({
    database: drizzleAdapter(db, {
      provider: "mysql",
      schema: {
        user: schema.baUser,
        session: schema.baSession,
        account: schema.baAccount,
        verification: schema.baVerification,
        organization: schema.baOrganization,
        member: schema.baMember,
        invitation: schema.baInvitation,
      },
    }),
    secret: config.BETTER_AUTH_SECRET,
    baseURL: config.BETTER_AUTH_URL,
    socialProviders: {
      github:
        config.GITHUB_CLIENT_ID && config.GITHUB_CLIENT_SECRET
          ? {
              clientId: config.GITHUB_CLIENT_ID,
              clientSecret: config.GITHUB_CLIENT_SECRET,
            }
          : undefined,
      google:
        config.GOOGLE_CLIENT_ID && config.GOOGLE_CLIENT_SECRET
          ? {
              clientId: config.GOOGLE_CLIENT_ID,
              clientSecret: config.GOOGLE_CLIENT_SECRET,
            }
          : undefined,
    },
    session: {
      expiresIn: 60 * 60 * 24 * 7, // 7 days in seconds
      // Cookie caching is disabled to ensure activeOrganizationId is always fresh
      // after setActiveOrganization() calls. The performance impact is minimal
      // since session validation only happens on page loads, not every request.
      cookieCache: {
        enabled: false,
      },
    },
    databaseHooks: {
      session: {
        create: {
          before: async (session) => {
            // Auto-set activeOrganizationId when session is created
            // Only auto-select if user has exactly 1 org
            // If user has multiple orgs, leave activeOrganizationId null
            // so they can be prompted to select one
            try {
              // Find ALL organization memberships for this user
              const memberships = await db.query.baMember.findMany({
                where: eq(schema.baMember.userId, session.userId),
              });

              // Only auto-select if exactly 1 org
              if (memberships.length === 1 && memberships[0].organizationId) {
                console.log("[Better Auth] Session hook: Auto-selecting single org:", {
                  userId: session.userId,
                  orgId: memberships[0].organizationId,
                });
                return {
                  data: {
                    ...session,
                    activeOrganizationId: memberships[0].organizationId,
                  },
                };
              }

              if (memberships.length > 1) {
                console.log("[Better Auth] Session hook: Multiple orgs found, user must select:", {
                  userId: session.userId,
                  orgCount: memberships.length,
                });
              } else {
                console.log("[Better Auth] Session hook: No org membership found for user:", session.userId);
              }
            } catch (error) {
              console.error("[Better Auth] Session hook error:", error);
            }

            // Return unchanged session - user will need to select org
            return { data: session };
          },
        },
      },
    },
    plugins: [
      emailOTP({
        sendVerificationOTP: async ({ email, otp, type }) => {
          if (resend) {
            await resend.sendOtpVerificationEmail({ email, otp, type });
          } else {
            // Fallback: log in development when Resend is not configured
            console.warn(`[Better Auth] OTP for ${email} (${type}): ${otp}`);
          }
        },
      }),
      organization({
        creatorRole: "admin",
        sendInvitationEmail: async ({ invitation, organization: org, inviter }) => {
          const inviterEmail = inviter.user.email;
          const invitationUrl = `${config.BETTER_AUTH_URL}/auth/accept-invitation?token=${invitation.id}`;

          if (resend) {
            await resend.sendOrgInvitationEmail({
              email: invitation.email,
              inviterEmail,
              organizationName: org.name,
              invitationUrl,
            });
          } else {
            // Fallback: log in development when Resend is not configured
            console.warn(
              `[Better Auth] Invitation to ${invitation.email} for org ${org.name} from ${inviterEmail}`,
            );
          }
        },
      }),
      admin({
        impersonationSessionDuration: 60 * 60, // 1 hour in seconds
      }),
      nextCookies(),
    ],
  });
}

// Use globalThis to persist singleton across HMR in development
// This prevents memory leaks from recreating Better Auth instances on every hot reload
const globalForBetterAuth = globalThis as unknown as {
  betterAuthInstance: ReturnType<typeof createBetterAuthInstance> | undefined;
};

/**
 * Gets the Better Auth instance, creating it lazily on first access.
 * This ensures the instance is only created when AUTH_PROVIDER=better-auth.
 * Uses globalThis to survive HMR in development mode.
 */
export function getBetterAuthInstance() {
  if (!globalForBetterAuth.betterAuthInstance) {
    globalForBetterAuth.betterAuthInstance = createBetterAuthInstance();
  }
  return globalForBetterAuth.betterAuthInstance;
}

// Export type for use in provider
export type BetterAuthInstance = ReturnType<typeof createBetterAuthInstance>;
