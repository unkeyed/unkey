import { execSync } from "node:child_process";
import * as fs from "node:fs";
import * as path from "node:path";
import * as clack from "@clack/prompts";

async function ensureBuild(
  packageDirName: string,
  packageNameForFilter: string,
  artifactPath = "dist/index.js",
) {
  const monoRepoRoot = path.resolve(__dirname, "../../../");
  const packageDir = path.join(monoRepoRoot, "packages", packageDirName);
  const artifactFullPath = path.join(packageDir, artifactPath);

  clack.log.info(`Checking for artifact: ${artifactFullPath}`);

  if (fs.existsSync(artifactFullPath)) {
    clack.log.success(`Build artifacts for ${packageNameForFilter} found.`);
  } else {
    clack.log.warn(
      `Build artifacts for ${packageNameForFilter} not found at ${artifactFullPath}. Triggering build...`,
    );
    try {
      execSync(`pnpm turbo run build --filter=${packageNameForFilter}...`, {
        cwd: monoRepoRoot,
        stdio: "inherit",
      });
      clack.log.success(`${packageNameForFilter} built successfully.`);
    } catch {
      clack.log.error(`Failed to build ${packageNameForFilter}. Please check build logs.`);
      process.exit(1);
    }
  }
}

async function prepare() {
  clack.intro("Preparing local environment (checking builds)...");
  await ensureBuild("error", "@unkey/error");
  clack.log.success("Build checks complete.");
}

prepare().catch((err) => {
  clack.log.error(
    `An unexpected error occurred in prepare script: ${
      err instanceof Error ? err.message : String(err)
    }`,
  );
  if (err instanceof Error && err.stack) {
    console.error(err.stack);
  }
  process.exit(1);
});
