import { useContext } from "react";
import { createContext } from "react";

// Create a simple context for just the workspace ID
const WorkspaceIdContext = createContext<string | null>(null);

/**
 * Custom hook to get the current workspace ID.
 *
 * This hook provides a convenient way to access the workspace ID throughout the application.
 * It can be used in any component that needs to reference the current workspace.
 *
 * @returns The current workspace ID or null if not available
 *
 * @example
 * ```tsx
 * function MyComponent() {
 *   const workspaceId = useWorkspaceId();
 *
 *   if (!workspaceId) {
 *     return <div>No workspace selected</div>;
 *   }
 *
 *   return <div>Current workspace: {workspaceId}</div>;
 * }
 * ```
 */
export function useWorkspaceId(): string | null {
  const context = useContext(WorkspaceIdContext);
  if (context === undefined) {
    throw new Error("useWorkspaceId must be used within a WorkspaceProvider");
  }
  return context;
}

/**
 * Custom hook to get the current workspace ID with error handling.
 *
 * This hook throws an error if no workspace ID is available, making it useful
 * for components that require a workspace ID to function.
 *
 * @returns The current workspace ID
 * @throws Error if no workspace ID is available
 *
 * @example
 * ```tsx
 * function MyComponent() {
 *   const workspaceId = useWorkspaceIdRequired();
 *
 *   // workspaceId is guaranteed to be a string here
 *   return <div>Current workspace: {workspaceId}</div>;
 * }
 * ```
 */
export function useWorkspaceIdRequired(): string {
  const workspaceId = useWorkspaceId();

  if (!workspaceId) {
    throw new Error(
      "Workspace ID is required but not available. Make sure this component is used within a workspace context.",
    );
  }

  return workspaceId;
}

// Export the context for use in the provider
export { WorkspaceIdContext };
