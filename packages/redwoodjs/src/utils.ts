import fs from "fs";
import { format } from "prettier";
import { getConfigPath } from "@redwoodjs/project-config";
import { writeFile, getPaths } from "@redwoodjs/cli-helpers";

/**
 *  Use prettier to format the template according to the RedwoodJS style
 * */
export const prettifyTemplate = async (template: string) => {
  return await format(template, {
    trailingComma: "es5",
    bracketSpacing: true,
    tabWidth: 2,
    semi: false,
    singleQuote: true,
    arrowParens: "always",
    parser: "typescript",
  });
};

/**
 * Updates the project's redwood.toml file to include the @unkey/redwoodjs plugin
 */
export const updateTomlConfig = () => {
  const redwoodTomlPath = getConfigPath();
  const configContent = fs.readFileSync(redwoodTomlPath, "utf-8");

  if (!configContent.includes("@unkey/redwoodjs")) {
    if (configContent.includes("[experimental.cli]")) {
      if (configContent.includes("  [[experimental.cli.plugins]]")) {
        writeFile(
          redwoodTomlPath,
          configContent.replace(
            "  [[experimental.cli.plugins]]",
            `
    [[experimental.cli.plugins]]
      package = "@unkey/redwoodjs"
  
    [[experimental.cli.plugins]]`
          ),
          {
            existingFiles: "OVERWRITE",
          }
        );
      } else {
        if (
          configContent.match(`[experimental.cli]
    autoInstall = true`)
        ) {
          writeFile(
            redwoodTomlPath,
            configContent.replace(
              `[experimental.cli]
    autoInstall = true`,
              `
  [experimental.cli]
    autoInstall = true
  
    [[experimental.cli.plugins]]
      package = "@unkey/redwoodjs"`
            ),
            {
              existingFiles: "OVERWRITE",
            }
          );
        } else {
          writeFile(
            redwoodTomlPath,
            configContent.replace(
              `[experimental.cli]
    autoInstall = false`,
              `
  [experimental.cli]
    autoInstall = false
  
    [[experimental.cli.plugins]]
      package = "@unkey/redwoodjs"`
            ),
            {
              existingFiles: "OVERWRITE",
            }
          );
        }
      }
    } else {
      writeFile(
        redwoodTomlPath,
        configContent.concat(`
[experimental.cli]
  autoInstall = true

  [[experimental.cli.plugins]]
    package = "@unkey/redwoodjs"`),
        {
          existingFiles: "OVERWRITE",
        }
      );
    }
  }
};
