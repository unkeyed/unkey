// The reads, polling and writes behind the app pages.
//
// Two QueryClients hold this data until the collections move to React Query.
// Projects, apps and deployments live on the app's React client
// (useQueryClient). Environments, domains, custom domains and policies live on
// the collections client in lib/collections/client.ts. Pass the React client to
// the query options, refetch and warm functions. Any other client gives a
// second, unpolled cache.
export { DeployDataProvider } from "./provider";
export {
  type DeployActions,
  type DeployScope,
  type DeploymentStatusTarget,
  type LiveDeploymentTarget,
  useAppScope,
  useDeployActions,
  useDeployScope,
  useAwaitedStatus,
  useDeploymentsPolling,
  useLiveTarget,
} from "./context";
export {
  type DeployQuery,
  policiesQueryFor,
  useApp,
  useCustomDomains,
  useDomains,
  useEnvironments,
  useProjectApps,
} from "./queries";
export {
  DEPLOYMENT_PICKER_LIMIT,
  RECENT_DEPLOYMENTS_LIMIT,
  type DeploymentListFilter,
  useCurrentDeployment,
  useDeployment,
  useDeploymentsPage,
  useProductionDeployments,
  useRecentDeployments,
  warmDeploymentsPage,
} from "./deployments";
export { useCustomDomainMutations } from "./queries";
export { toastWhenSettled } from "./toast-when-settled";
export { warmAppPage, warmDeployment } from "./warm";

export {
  APP_SLUG_LENGTH,
  type App,
  type AppDeploymentRefs,
  createAppRequestSchema,
  latestDeploymentId,
  ociImageReferenceSchema,
} from "@/lib/queries/apps";
export {
  type CustomDomain,
  type CustomDomainDnsRecord,
  type VerificationStatus,
  customDomainErrorKind,
} from "@/lib/collections/deploy/custom-domains";
export {
  type Deployment,
  type DeploymentSummary,
  hasBuildLogs,
  isPrebuilt,
} from "@/lib/deploy-data/deployment";
export type { Domain } from "@/lib/collections/deploy/domains";
export {
  ENVIRONMENT_KIND,
  ENVIRONMENT_KINDS,
  type Environment,
  type EnvironmentKind,
  productionFirst,
} from "@/lib/collections/deploy/environments";
