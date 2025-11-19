import type {
  DeploymentNode,
  HealthStatus,
  InstanceMetadata,
  RegionMetadata,
} from "../nodes/types";

type GeneratorConfig = {
  regions: number;
  instancesPerRegion: { min: number; max: number };
  healthDistribution?: Partial<Record<HealthStatus, number>>;
  regionDirection?: "vertical" | "horizontal";
  instanceDirection?: "vertical" | "horizontal";
};

export function generateDeploymentTree(config: GeneratorConfig): DeploymentNode {
  const healthStatuses: HealthStatus[] = [
    "normal",
    "unstable",
    "degraded",
    "unhealthy",
    "recovering",
    "health_syncing",
    "unknown",
    "disabled",
  ];

  const regions = [
    "us-east-1",
    "eu-central-1",
    "ap-southeast-2",
    "ap-northeast-1",
    "ap-south-1",
    "sa-east-1",
  ] as const;

  const flags: Record<string, RegionMetadata["flagCode"]> = {
    "us-east-1": "us",
    "eu-central-1": "de",
    "ap-southeast-2": "au",
    "ap-northeast-1": "jp",
    "ap-south-1": "in",
    "sa-east-1": "br",
  };

  const getRandomHealth = (): HealthStatus => {
    if (config.healthDistribution) {
      const rand = Math.random() * 100;
      let cumulative = 0;
      for (const [status, percentage] of Object.entries(config.healthDistribution)) {
        cumulative += percentage;
        if (rand <= cumulative) {
          return status as HealthStatus;
        }
      }
    }
    return healthStatuses[Math.floor(Math.random() * healthStatuses.length)];
  };

  const getRandomInt = (min: number, max: number) =>
    Math.floor(Math.random() * (max - min + 1)) + min;

  const selectedRegions = regions.slice(0, config.regions);

  return {
    id: "internet",
    label: "INTERNET",
    metadata: { type: "origin" },
    children: selectedRegions.map((regionId) => {
      const instanceCount = getRandomInt(
        config.instancesPerRegion.min,
        config.instancesPerRegion.max,
      );
      const totalInstances = getRandomInt(20, 40);
      const regionHealth = getRandomHealth();

      const regionMetadata: RegionMetadata = {
        type: "region",
        flagCode: flags[regionId],
        zones: getRandomInt(1, 3),
        instances: totalInstances,
        replicas: 2,
        rps: getRandomInt(1000, 5000),
        cpu: getRandomInt(30, 80),
        memory: getRandomInt(40, 85),
        storage: getRandomInt(512, 1024),
        latency: `${(Math.random() * 5 + 1).toFixed(1)}ms`,
        health: regionHealth,
      };

      return {
        id: regionId,
        label: regionId,
        direction: config.instanceDirection ?? "vertical",
        metadata: regionMetadata,
        children: Array.from({ length: instanceCount }, (_, i) => {
          const instanceId = Math.random().toString(36).substring(2, 6);

          const instanceMetadata: InstanceMetadata = {
            type: "instance",
            description: "Instance replica",
            replicas: 2,
            rps: getRandomInt(100, 500),
            cpu: getRandomInt(20, 70),
            memory: getRandomInt(30, 75),
            latency: `${(Math.random() * 8 + 2).toFixed(1)}ms`,
            health: getRandomHealth(),
          };

          return {
            id: `${regionId}-gw-${instanceId}-${i + 1}`,
            label: `gw-${instanceId}`,
            metadata: instanceMetadata,
          };
        }),
      };
    }),
  };
}
