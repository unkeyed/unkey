"use client";
import { trpc } from "@/lib/trpc/client";
import { Layers3, XMark } from "@unkey/icons";
import { Button } from "@unkey/ui";
import { useState } from "react";
import type { DeploymentNode, HealthStatus } from "../nodes/types";

type GeneratorConfig = {
  sentinels: number;
  instancesPerSentinel: { min: number; max: number };
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
    label: "Small (3 sentinels, 1-3 instances)",
    config: {
      sentinels: 3,
      instancesPerSentinel: { min: 1, max: 3 },
      regionDirection: "horizontal" as const,
      instanceDirection: "horizontal" as const,
      healthDistribution: {
        normal: 80,
        unhealthy: 5,
        health_syncing: 5,
        unknown: 5,
        disabled: 5,
      },
    },
  },
  medium: {
    label: "Medium (5 sentinels, 2-5 instances)",
    config: {
      sentinels: 5,
      instancesPerSentinel: { min: 2, max: 5 },
      regionDirection: "horizontal" as const,
      instanceDirection: "horizontal" as const,
      healthDistribution: {
        normal: 70,
        unhealthy: 10,
        health_syncing: 10,
        unknown: 5,
        disabled: 5,
      },
    },
  },
  large: {
    label: "Large (7 sentinels, 5-10 instances)",
    config: {
      sentinels: 7,
      instancesPerSentinel: { min: 5, max: 10 },
      regionDirection: "horizontal" as const,
      instanceDirection: "horizontal" as const,
      healthDistribution: {
        normal: 60,
        unhealthy: 15,
        health_syncing: 10,
        unknown: 10,
        disabled: 5,
      },
    },
  },
  stress: {
    label: "Stress Test (7 sentinels, 15-20 instances)",
    config: {
      sentinels: 7,
      instancesPerSentinel: { min: 15, max: 20 },
      regionDirection: "horizontal" as const,
      instanceDirection: "horizontal" as const,
      healthDistribution: {
        normal: 50,
        unhealthy: 20,
        health_syncing: 15,
        unknown: 10,
        disabled: 5,
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
                className="text-left px-3 py-2 rounded-sm border border-grayA-4 text-xs transition-colors"
              >
                {preset.label}
              </Button>
            ))}
          </div>
        </div>

        {/* Custom Configuration */}
        <div className="space-y-3 pt-3 border-t border-grayA-4">
          <div className="text-xs font-medium text-gray-11">Custom</div>

          {/* Sentinels*/}
          <div className="space-y-1">
            <div className="text-xs text-gray-11">Sentinels: {customConfig.sentinels}</div>
            <input
              type="range"
              min="1"
              max="7"
              value={customConfig.sentinels}
              onChange={(e) =>
                setCustomConfig((c) => ({
                  ...c,
                  sentinels: Number(e.target.value),
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
                <span className="text-xs text-gray-11 w-20">Sentinels:</span>
                <select
                  value={customConfig.regionDirection}
                  onChange={(e) =>
                    setCustomConfig((c) => ({
                      ...c,
                      regionDirection: e.target.value as "vertical" | "horizontal",
                    }))
                  }
                  disabled={generateMutation.isLoading}
                  className="flex-1 px-2 py-1 text-xs rounded-sm border border-grayA-4 bg-gray-1"
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
                  className="flex-1 px-2 py-1 text-xs rounded-sm border border-grayA-4 bg-gray-1"
                >
                  <option value="horizontal">Horizontal (side-by-side)</option>
                  <option value="vertical">Vertical (stacked)</option>
                </select>
              </div>
            </div>
          </div>

          {/* Instances per sentinel */}
          <div className="space-y-1">
            <div className="text-xs text-gray-11">
              Instances per sentinel: {customConfig.instancesPerSentinel.min}-
              {customConfig.instancesPerSentinel.max}
            </div>
            <div className="flex gap-2">
              <input
                type="range"
                min="0"
                max="20"
                value={customConfig.instancesPerSentinel.min}
                onChange={(e) =>
                  setCustomConfig((c) => ({
                    ...c,
                    instancesPerSentinel: {
                      ...c.instancesPerSentinel,
                      min: Number(e.target.value),
                    },
                  }))
                }
                disabled={generateMutation.isLoading}
                className="flex-1"
              />
              <input
                type="range"
                min="0"
                max="20"
                value={customConfig.instancesPerSentinel.max}
                onChange={(e) =>
                  setCustomConfig((c) => ({
                    ...c,
                    instancesPerSentinel: {
                      ...c.instancesPerSentinel,
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
            className="w-full px-3 py-2 rounded-sm text-xs font-medium transition-colors"
          >
            Generate Custom Tree
          </Button>
        </div>

        {/* Reset */}
        <Button
          onClick={onReset}
          disabled={generateMutation.isLoading}
          className="w-full px-3 py-2 rounded-sm text-xs font-medium transition-colors border border-grayA-4"
        >
          Reset to Original
        </Button>
      </div>
    </div>
  );
}
