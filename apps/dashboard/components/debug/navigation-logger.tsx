"use client";

import { useWorkspace } from "@/providers/workspace-provider";
import { usePathname, useRouter } from "next/navigation";
import { useEffect } from "react";

/**
 * Debug component to log navigation events and workspace cache behavior
 * Add this to your layout or any page to monitor cache performance
 */
export function NavigationLogger() {
  const pathname = usePathname();
  const router = useRouter();
  const { workspace, isLoading, error } = useWorkspace();

  // Log navigation changes
  useEffect(() => {
    const timestamp = new Date().toISOString();
    console.log(`🧭 [${timestamp}] Navigation changed:`, {
      pathname,
      workspaceFromCache: workspace
        ? {
            id: workspace.id,
            name: workspace.name,
            loadedFromCache: !isLoading,
          }
        : null,
      isLoading,
      hasError: !!error,
    });
  }, [pathname, workspace, isLoading, error]);

  // Add test navigation functions to window for manual testing
  useEffect(() => {
    if (typeof window !== "undefined") {
      // Create test functions and attach to window
      const testFunctions = {
        // Navigate to different pages to test cache
        testNavigation: () => {
          console.log("🧪 Starting cache navigation test...");

          const originalPath = pathname || "/";

          // Navigate to home and back to test cache retention
          router.push("/");

          setTimeout(() => {
            console.log("🔄 Navigating back to test cache...");
            router.push(originalPath);
          }, 1000);
        },

        // Force a component re-render to test cache
        testRerender: () => {
          console.log("🧪 Testing cache during re-render...");
          // This will trigger useWorkspace hook again
          const event = new CustomEvent("test-rerender");
          window.dispatchEvent(event);
        },

        // Log current cache state
        logCacheState: () => {
          console.log("📊 Current workspace cache state:", {
            hasWorkspace: !!workspace,
            workspaceName: workspace?.name,
            isLoading,
            hasError: !!error,
            timestamp: new Date().toISOString(),
          });
        },
      };

      // Attach to window with any type to avoid TypeScript errors
      (window as any).testWorkspaceCache = testFunctions;

      // Log available test functions
      console.log(
        "🛠️ Cache testing functions available:",
        "Run window.testWorkspaceCache.testNavigation() to test navigation caching"
      );
      console.log(
        "🛠️ Available functions:",
        Object.keys(testFunctions).join(", ")
      );
    }
  }, [pathname, router, workspace, isLoading, error]);

  // This component renders nothing but provides logging
  return null;
}

/**
 * Performance timing logger for cache behavior
 * Use this to measure cache hit vs miss performance
 */
export function CachePerformanceLogger() {
  const { workspace, isLoading } = useWorkspace();

  useEffect(() => {
    let startTime = performance.now();

    if (isLoading) {
      startTime = performance.now();
      console.log("⏱️ Workspace query started");
    } else if (workspace) {
      const endTime = performance.now();
      const duration = endTime - startTime;

      console.log("⚡ Workspace query completed:", {
        duration: `${duration.toFixed(2)}ms`,
        likely: duration < 50 ? "FROM_CACHE" : "FROM_NETWORK",
        workspaceName: workspace.name,
        timestamp: new Date().toISOString(),
      });
    }
  }, [isLoading, workspace]);

  return null;
}
