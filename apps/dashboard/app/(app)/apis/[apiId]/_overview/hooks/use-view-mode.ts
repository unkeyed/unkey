import { useState } from "react";

export type ViewMode = "verifications" | "credits";

export function useViewMode(defaultMode: ViewMode = "verifications") {
  const [viewMode, setViewMode] = useState<ViewMode>(defaultMode);

  const toggleViewMode = () => {
    setViewMode((prev) => (prev === "verifications" ? "credits" : "verifications"));
  };

  const setVerificationsMode = () => setViewMode("verifications");
  const setCreditsMode = () => setViewMode("credits");

  return {
    viewMode,
    setViewMode,
    toggleViewMode,
    setVerificationsMode,
    setCreditsMode,
    isVerificationsMode: viewMode === "verifications",
    isCreditsMode: viewMode === "credits",
  };
}
