"use client";
import { trpc } from "@/lib/trpc/client";
import { Layers3, XMark } from "@unkey/icons";
import { Button } from "@unkey/ui";
import { useState } from "react";
import type { DeploymentNode, HealthStatus } from "../nodes/types";

type GeneratorConfig = {
  regions: number;
  instancesPerRegion: { min: number; max: number };
  healthDistribution: Record<HealthStatus, number>;
  regionDirection: "vertical" | "horizontal";
  instanceDirection: "vertical" | "horizontal";
};

type DevTreeGeneratorProps = {
  deploymentId: string;
  onGenerate: (tree: DeploymentNode) => void;
  onReset: () => void;
};

const PRESETS = {
  small: {
    label: "Small (3 regions, 1-3 instances)",
    config: {
      regions: 3,
      instancesPerRegion: { min: 1, max: 3 },
      regionDirection: "horizontal" as const,
      instanceDirection: "horizontal" as const,
      healthDistribution: {
        normal: 80,
        unstable: 10,
        degraded: 5,
        unhealthy: 5,
        recovering: 0,
        health_syncing: 0,
        unknown: 0,
        disabled: 0,
      },
    },
  },
  medium: {
    label: "Medium (5 regions, 2-5 instances)",
    config: {
      regions: 5,
      instancesPerRegion: { min: 2, max: 5 },
      regionDirection: "horizontal" as const,
      instanceDirection: "horizontal" as const,
      healthDistribution: {
        normal: 70,
        unstable: 15,
        degraded: 10,
        unhealthy: 5,
        recovering: 0,
        health_syncing: 0,
        unknown: 0,
        disabled: 0,
      },
    },
  },
  large: {
    label: "Large (7 regions, 5-10 instances)",
    config: {
      regions: 7,
      instancesPerRegion: { min: 5, max: 10 },
      regionDirection: "horizontal" as const,
      instanceDirection: "horizontal" as const,
      healthDistribution: {
        normal: 60,
        unstable: 20,
        degraded: 10,
        unhealthy: 5,
        recovering: 3,
        health_syncing: 2,
        unknown: 0,
        disabled: 0,
      },
    },
  },
  stress: {
    label: "Stress Test (7 regions, 15-20 instances)",
    config: {
      regions: 7,
      instancesPerRegion: { min: 15, max: 20 },
      regionDirection: "horizontal" as const,
      instanceDirection: "horizontal" as const,
      healthDistribution: {
        normal: 50,
        unstable: 20,
        degraded: 15,
        unhealthy: 10,
        recovering: 3,
        health_syncing: 1,
        unknown: 1,
        disabled: 0,
      },
    },
  },
} as const;

export function InternalDevTreeGenerator({ onGenerate, onReset }: DevTreeGeneratorProps) {
  const [isOpen, setIsOpen] = useState(false);
  const [customConfig, setCustomConfig] = useState<GeneratorConfig>(PRESETS.medium.config);

  const generateMutation = trpc.deploy.network.generate.useMutation({
    onSuccess: (tree) => {
      onGenerate(tree);
    },
    onError: (error) => {
      console.error("Failed to generate tree:", error);
    },
  });

  const handleGenerate = (config: GeneratorConfig) => {
    generateMutation.mutate({
      ...config,
    });
  };

  if (!isOpen) {
    return (
      <Button
        variant="outline"
        onClick={() => setIsOpen(true)}
        className="pointer-events-auto fixed bottom-4 right-4 rounded-full shadow-lg transition-colors"
        title="Tree Generator"
      >
        <Layers3 iconSize="sm-medium" />
      </Button>
    );
  }

  return (
    <div className="pointer-events-auto fixed bottom-4 right-4 z-50 w-80 bg-gray-1 border border-grayA-6 rounded-lg shadow-xl">
      <div className="flex items-center justify-between p-3 border-b border-grayA-4">
        <div className="flex items-center gap-2">
          <Layers3 iconSize="sm-medium" className="text-accent-9" />
          <span className="font-medium text-sm">Tree Generator</span>
        </div>
        <Button onClick={() => setIsOpen(false)}>
          <XMark iconSize="sm-medium" />
        </Button>
      </div>
      <div className="p-3 space-y-3 max-h-[600px] overflow-y-auto">
        {/* Loading State */}
        {generateMutation.isLoading && (
          <div className="absolute inset-0 bg-black/50 flex items-center justify-center rounded-lg">
            <div className="text-white text-sm">Generating...</div>
          </div>
        )}

        {/* Presets */}
        <div className="space-y-2">
          <div className="text-xs font-medium text-gray-11">Presets</div>
          <div className="grid grid-cols-1 gap-2">
            {Object.entries(PRESETS).map(([key, preset]) => (
              <Button
                key={key}
                onClick={() => {
                  handleGenerate(preset.config);
                  setCustomConfig(preset.config);
                }}
                disabled={generateMutation.isLoading}
                className="text-left px-3 py-2 rounded border border-grayA-4 text-xs transition-colors"
              >
                {preset.label}
              </Button>
            ))}
          </div>
        </div>

        {/* Custom Configuration */}
        <div className="space-y-3 pt-3 border-t border-grayA-4">
          <div className="text-xs font-medium text-gray-11">Custom</div>

          {/* Regions */}
          <div className="space-y-1">
            <div className="text-xs text-gray-11">Regions: {customConfig.regions}</div>
            <input
              type="range"
              min="1"
              max="7"
              value={customConfig.regions}
              onChange={(e) =>
                setCustomConfig((c) => ({
                  ...c,
                  regions: Number(e.target.value),
                }))
              }
              disabled={generateMutation.isLoading}
              className="w-full"
            />
          </div>

          {/* Layout Direction Controls */}
          <div className="space-y-2">
            <div className="text-xs text-gray-11">Layout Direction</div>
            <div className="space-y-1.5">
              <div className="flex items-center gap-2">
                <span className="text-xs text-gray-11 w-20">Regions:</span>
                <select
                  value={customConfig.regionDirection}
                  onChange={(e) =>
                    setCustomConfig((c) => ({
                      ...c,
                      regionDirection: e.target.value as "vertical" | "horizontal",
                    }))
                  }
                  disabled={generateMutation.isLoading}
                  className="flex-1 px-2 py-1 text-xs rounded border border-grayA-4 bg-gray-1"
                >
                  <option value="horizontal">Horizontal (side-by-side)</option>
                  <option value="vertical">Vertical (stacked)</option>
                </select>
              </div>
              <div className="flex items-center gap-2">
                <span className="text-xs text-gray-11 w-20">Instances:</span>
                <select
                  value={customConfig.instanceDirection}
                  onChange={(e) =>
                    setCustomConfig((c) => ({
                      ...c,
                      instanceDirection: e.target.value as "vertical" | "horizontal",
                    }))
                  }
                  disabled={generateMutation.isLoading}
                  className="flex-1 px-2 py-1 text-xs rounded border border-grayA-4 bg-gray-1"
                >
                  <option value="horizontal">Horizontal (side-by-side)</option>
                  <option value="vertical">Vertical (stacked)</option>
                </select>
              </div>
            </div>
          </div>

          {/* Instances per region */}
          <div className="space-y-1">
            <div className="text-xs text-gray-11">
              Instances per region: {customConfig.instancesPerRegion.min}-
              {customConfig.instancesPerRegion.max}
            </div>
            <div className="flex gap-2">
              <input
                type="range"
                min="1"
                max="20"
                value={customConfig.instancesPerRegion.min}
                onChange={(e) =>
                  setCustomConfig((c) => ({
                    ...c,
                    instancesPerRegion: {
                      ...c.instancesPerRegion,
                      min: Number(e.target.value),
                    },
                  }))
                }
                disabled={generateMutation.isLoading}
                className="flex-1"
              />
              <input
                type="range"
                min="1"
                max="20"
                value={customConfig.instancesPerRegion.max}
                onChange={(e) =>
                  setCustomConfig((c) => ({
                    ...c,
                    instancesPerRegion: {
                      ...c.instancesPerRegion,
                      max: Number(e.target.value),
                    },
                  }))
                }
                disabled={generateMutation.isLoading}
                className="flex-1"
              />
            </div>
          </div>

          <Button
            onClick={() => handleGenerate(customConfig)}
            disabled={generateMutation.isLoading}
            className="w-full px-3 py-2 rounded text-xs font-medium transition-colors"
          >
            Generate Custom Tree
          </Button>
        </div>

        {/* Reset */}
        <Button
          onClick={onReset}
          disabled={generateMutation.isLoading}
          className="w-full px-3 py-2 rounded text-xs font-medium transition-colors border border-grayA-4"
        >
          Reset to Original
        </Button>
      </div>
    </div>
  );
}
