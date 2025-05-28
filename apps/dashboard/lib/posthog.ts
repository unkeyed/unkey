import { PostHog } from "posthog-node";

class PostHogClientWrapper {
  private static instance: PostHog | null = null;

  private constructor() {}

  public static getInstance(): PostHog {
    if (!PostHogClientWrapper.instance) {
      if (!(process.env.NEXT_PUBLIC_POSTHOG_KEY && process.env.NEXT_PUBLIC_POSTHOG_HOST)) {
        console.warn("PostHog key is missing. Analytics data will not be sent.");
        // Return a mock client when the key is not present
        PostHogClientWrapper.instance = {
          capture: () => {},
          // Add other methods from PostHog, implementing them as no-ops
        } as unknown as PostHog;
      } else {
        PostHogClientWrapper.instance = new PostHog(process.env.NEXT_PUBLIC_POSTHOG_KEY, {
          host: process.env.NEXT_PUBLIC_POSTHOG_HOST,
          flushAt: 1,
          flushInterval: 0,
        });
      }
    }
    return PostHogClientWrapper.instance;
  }
}

export const PostHogClient = PostHogClientWrapper.getInstance();
