// The reads, polling and writes behind the app pages. Components import from
// here only; which store holds the data is this module's concern.
export { DeployDataProvider } from "./provider";
export {
  type DeployActions,
  type DeployScope,
  type DeploymentStatusTarget,
  type LiveDeploymentTarget,
  useAppScope,
  useDeployActions,
  useDeployScope,
  useDeploymentsPolling,
} from "./context";
export {
  type QueryResult,
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
  type DeploymentsPageResult,
  useCurrentDeployment,
  useDeployment,
  useDeploymentsPage,
  useProductionDeployments,
  useRecentDeployments,
  warmDeploymentsPage,
} from "./deployments";
export { useAppMutations, useCustomDomainMutations } from "./mutations";
export { warmAppPage, warmDeployment } from "./warm";

export {
  APP_SLUG_LENGTH,
  type App,
  type AppDeploymentRefs,
  createAppRequestSchema,
  latestDeploymentId,
  ociImageReferenceSchema,
} from "@/lib/collections/deploy/apps";
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
} from "@/lib/collections/deploy/deployments";
export type { Domain } from "@/lib/collections/deploy/domains";
export {
  ENVIRONMENT_KIND,
  ENVIRONMENT_KINDS,
  type Environment,
  type EnvironmentKind,
  productionFirst,
} from "@/lib/collections/deploy/environments";
