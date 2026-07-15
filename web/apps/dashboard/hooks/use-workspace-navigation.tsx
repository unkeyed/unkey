import { useWorkspace } from "@/providers/workspace-provider";
import { redirect } from "next/navigation";

/**
 * Custom React hook for handling workspace navigation and access control.
 *
 * This hook ensures that users have access to a workspace before rendering
 * protected components. It handles three key scenarios:
 * 1. Loading state - suspends rendering while workspace data is being fetched
 * 2. No workspace - redirects users to create a new workspace
 * 3. Valid workspace - returns the workspace data for use in components
 *
 * @returns {Workspace} The current workspace object
 * @throws {Promise} Throws a never-resolving promise during loading (React Suspense pattern)
 */
export const useWorkspaceNavigation = () => {
  // Get workspace data and loading state from the workspace provider context
  const { workspace, isLoading: isWorkspaceLoading, error } = useWorkspace();

  // Handle loading state by throwing a promise that never resolves
  // This triggers React Suspense boundaries to show loading UI
  if (isWorkspaceLoading) {
    throw new Promise(() => {});
  }

  if (!workspace) {
    // A failed lookup is not the same as "this user has no workspace":
    // redirecting an existing user to /new on a transient error would invite
    // them to create a duplicate workspace. Surface the failure to the error
    // boundary instead.
    if (error) {
      throw error;
    }

    // The workspace definitively does not exist (NOT_FOUND), so the user
    // still needs to complete onboarding.
    redirect("/new");
  }

  // Return the workspace object for use in components
  // At this point we're guaranteed to have a valid workspace
  return workspace;
};
