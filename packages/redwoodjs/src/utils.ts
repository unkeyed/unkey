import fs from "fs";
import path from "path";
import * as toml from "toml";
import { getConfigPath } from "@redwoodjs/project-config";
import { writeFile, getPaths } from "@redwoodjs/cli-helpers";

/**
 * Updates the project's redwood.toml file to include the @unkey/redwoodjs plugin
 * 
 * Uses toml parsing to determine if the plugin is already included in the file and
 * only adds it if it is not.
 * 
 * Writes the updated config to the file system by appending strings, not stringify-ing the toml.
 */
export const updateTomlConfig = (packageName: string) => {
  const redwoodTomlPath = getConfigPath();

  const configLines = []

  const configContent = fs.readFileSync(redwoodTomlPath, 'utf-8');
  const config = toml.parse(configContent);

  if (!config || !config.experimental || !config.experimental.cli) {
    configLines.push('[experimental.cli]');
  }

  if (!config?.experimental?.cli?.autoInstall) {
    configLines.push('  autoInstall = true');
  }

  if (!config?.experimental?.cli?.plugins) {
    // If plugins array is missing, create it.
    configLines.push('  [[experimental.cli.plugins]]');
  }

  // Check if the package is not already in the plugins array
  if (!config?.experimental?.cli?.plugins?.some((plugin: any) => plugin.package === packageName)) {
    configLines.push(`    package = "${packageName}"`);
  }

  const newConfig = configContent + (configLines.join('\n'));

  fs.writeFileSync(redwoodTomlPath, newConfig, 'utf-8');
};
