import { useApp, useCurrentDeployment } from "@/lib/deploy-data";

// Legacy pages only. Everything else reads through @/lib/deploy-data.
export function useAppCurrentDeployment() {
  const app = useApp();
  const currentDeployment = useCurrentDeployment();
  return {
    app: app.data,
    currentDeployment: currentDeployment.data ?? undefined,
    isRolledBack: app.data?.isRolledBack ?? false,
    isLoading: currentDeployment.isLoading,
  };
}
