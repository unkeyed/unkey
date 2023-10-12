import fs from "fs";
import path from "path";
import { format } from "prettier";
import { getConfigPath } from "@redwoodjs/project-config";
import { writeFile, getPaths } from "@redwoodjs/cli-helpers";
import * as dotenv from "dotenv";

export interface EnvVar {
  key: string;
  value: string;
  overwrite?: boolean;
}

export const addEnvironmentVariablesToEnvFile = (variables: EnvVar[]): void => {
  // Load the .env file
  const envFilePath = path.join(getPaths().base, ".env");
  const envFileContents = fs.readFileSync(envFilePath, "utf-8");
  const envConfig = dotenv.parse(envFileContents);

  // Preserve existing comments by extracting them
  const existingComments: string[] = [];
  const envLines = envFileContents.split("\n");
  let insideCommentBlock = false;

  for (const line of envLines) {
    if (line.startsWith("#")) {
      existingComments.push(line);
    } else if (line.trim() === "" && existingComments.length > 0) {
      // Detect empty lines within comment blocks
      existingComments.push(line);
    } else {
      insideCommentBlock = false;
    }

    if (line.trim() === "") {
      // Detect empty lines between comment blocks
      insideCommentBlock = true;
    }
  }

  for (const { key, value, overwrite = false } of variables) {
    if (envConfig[key] !== undefined) {
      if (overwrite) {
        envConfig[key] = value;
      } else {
        console.log(`Skipping "${key}" as it already exists in .env`);
        continue;
      }
    } else {
      envConfig[key] = value;
    }
  }

  // Serialize the envConfig back to a string
  const updatedEnvContents =
    existingComments.join("\n") +
    "\n" +
    Object.entries(envConfig)
      .map(([key, value]) => `${key}=${value}`)
      .join("\n");

  // Write the updated contents back to the .env file
  fs.writeFileSync(envFilePath, updatedEnvContents);

  console.log("Environment variables added to .env file.");
};

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
export const updateTomlConfig = (packageName: string) => {
  const redwoodTomlPath = getConfigPath();
  const configContent = fs.readFileSync(redwoodTomlPath, "utf-8");

  if (!configContent.includes(packageName)) {
    if (configContent.includes("[experimental.cli]")) {
      if (configContent.includes("  [[experimental.cli.plugins]]")) {
        writeFile(
          redwoodTomlPath,
          configContent.replace(
            "  [[experimental.cli.plugins]]",
            `
    [[experimental.cli.plugins]]
      package = "${packageName}"
  
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
      package = "${packageName}"`
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
      package = "${packageName}"`
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
    package = "${packageName}"`),
        {
          existingFiles: "OVERWRITE",
        }
      );
    }
  }
};
