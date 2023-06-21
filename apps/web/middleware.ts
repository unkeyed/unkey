import { authMiddleware } from "@clerk/nextjs";

const DEBUG_ON = process.env.CLERK_DEBUG === "true";

export default authMiddleware({
  publicRoutes: ["/", "/auth(.*)"],
  signInUrl: "/auth/sign-in",
  debug: DEBUG_ON,
});

export const config = {
  matcher: ["/((?!.*\\..*|_next).*)", "/", "/(api|trpc)(.*)"],
};
