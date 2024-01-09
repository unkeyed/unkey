#!/usr/bin/env node

import { spawn } from "child_process";
import { promises as fs } from "fs";
import http from "http";
import { ParsedUrlQuery } from "node:querystring";
import os from "os";
import path from "path";
import url from "url";
import { listen } from "async-listen";
import { Command } from "commander";
import { customAlphabet } from "nanoid";
import pc from "picocolors";

const FILENAME = ".unkey";
// const CLIENT_URL = "http://localhost:3000";
const CLIENT_URL = "https://unkey-cli.vercel.app";

class UserCancellationError extends Error {
  constructor(message: string) {
    super(message);
    this.name = "UserCancellationError";
  }
}

async function writeToConfigFile(data: ParsedUrlQuery) {
  try {
    const homeDir = os.homedir();
    const filePath = path.join(homeDir, FILENAME);
    await fs.writeFile(filePath, JSON.stringify(data));
  } catch (error) {
    console.error("Error writing to local config file", error);
  }
}

const program = new Command();
const nanoid = customAlphabet("123456789QAZWSXEDCRFVTGBYHNUJMIKOLP", 8);

program.name("unkey-cli").description("Example CLI application with Unkey auth").version("0.0.1");

program
  .command("login")
  .description("Authenticate with your service via the CLI")
  .action(async (...args) => {
    if (args.length !== 2) {
      console.error("Usage: `unkey-cli login`.");
      process.exit(1);
    }

    // need to import ora dynamically since it's ESM-only
    const oraModule = await import("ora");
    const ora = oraModule.default;

    // create localhost server for our page to call back to
    const server = http.createServer();
    const { port } = await listen(server, 0, "127.0.0.1");

    // set up HTTP server that waits for a request containing an API key
    // as the only query parameter
    const authPromise = new Promise<ParsedUrlQuery>((resolve, reject) => {
      server.on("request", (req, res) => {
        // Set CORS headers for all responses
        res.setHeader("Access-Control-Allow-Origin", "*");
        res.setHeader("Access-Control-Allow-Methods", "GET, POST, OPTIONS");
        res.setHeader("Access-Control-Allow-Headers", "Content-Type, Authorization");

        if (req.method === "OPTIONS") {
          res.writeHead(200);
          res.end();
        } else if (req.method === "GET") {
          const parsedUrl = url.parse(req.url as string, true);
          const queryParams = parsedUrl.query;
          if (queryParams.cancelled) {
            res.writeHead(200);
            res.end();
            reject(new UserCancellationError("Login process cancelled by user."));
          } else {
            res.writeHead(200);
            res.end(JSON.stringify(queryParams));
            resolve(queryParams);
          }
        } else {
          res.writeHead(405);
          res.end();
        }
      });
    });

    const redirect = `http://127.0.0.1:${port}`;

    const code = nanoid();
    const confirmationUrl = new URL(`${CLIENT_URL}/auth/devices`);
    confirmationUrl.searchParams.append("code", code);
    confirmationUrl.searchParams.append("redirect", redirect);
    console.log(`Confirmation code: ${pc.bold(code)}\n`);
    console.log(
      `If something goes wrong, copy and paste this URL into your browser: ${pc.bold(
        confirmationUrl.toString(),
      )}\n`,
    );
    spawn("open", [confirmationUrl.toString()]);
    const spinner = ora("Waiting for authentication...\n\n");

    try {
      spinner.start();
      const authData = await authPromise;
      spinner.stop();
      writeToConfigFile(authData);
      console.log("Authentication successful.\n");
    } catch (error) {
      if (error instanceof UserCancellationError) {
        console.log("Authentication cancelled.\n");
        process.exit(0);
      } else {
        console.error("Authentication failed:", error);
        console.log("\n");
        process.exit(1);
      }
    } finally {
      server.close();
    }
  });

program.parse();
