// Global debug functions for workspace cache testing
declare global {
  interface Window {
    testWorkspaceCache: {
      testNavigation: () => void;
      testRerender: () => void;
      logCacheState: () => void;
    };
  }
}

export {};
