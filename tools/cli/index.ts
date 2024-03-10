#!/usr/bin/env node
import { spawn } from "child_process";
import { promises as fs } from "fs";
import { readFileSync, writeFileSync } from "fs";
import http from "http";
import { ParsedUrlQuery } from "node:querystring";
import os from "os";
import path from "path";
import url from "url";
import * as clack from "@clack/prompts";
import { listen } from "async-listen";
import { Command } from "commander";
import "dotenv/config";
import { customAlphabet } from "nanoid";
import pc from "picocolors";

const FILENAME = ".unkey";
const API_BASE_URL = process.env.API_BASE_URL || "https://planetfall-two.vercel.app";

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
    writeFileSync(filePath, JSON.stringify(data));
  } catch (error) {
    console.error("Error writing to local config file", error);
  }
}

function getVersion() {
  const packageJson = readFileSync(path.join(__dirname, "package.json"), "utf-8");
  const { version } = JSON.parse(packageJson);
  return version;
}
// need to import ora dynamically since it's ESM-only
const oraModule = await import("ora");
const ora = oraModule.default;
const programm = new Command();
const nanoid = customAlphabet("123456789QAZWSXEDCRFVTGBYHNUJMIKOLP", 8);
const version = getVersion();

programm.name("globalstat").description("httpstat, but globally").version(version);

programm
  .command("check")
  .argument("<url>", "the url to check")
  .option<number>(
    "-n, --runs <RUNS>",
    "the number of times to run the check",
    (s) => parseInt(s),
    1,
  )
  .option<string>("-m, --method <METHOD>", "the http method to use", (s) => s, "GET")
  .option<string>("-d, --data <DATA>", "the body to send", (s) => s, "")
  .option<Record<string, string>>(
    "-H, --header <HEADER>",
    "the http headers to use",
    (s) => JSON.parse(s),
    {},
  )
  .description("Check the status of a url")
  .action(async (url, options) => {
    if (!url) {
      console.error("Usage: `globalstat check {URL}`.");
      process.exit(1);
    }

    const dotUnkey = readFileSync(path.join(os.homedir(), FILENAME));
    const { key } = JSON.parse(dotUnkey.toString());
    if (typeof key !== "string") {
      console.error("Error: API key not found. Please run `globalstat login`.");
      process.exit(1);
    }

    const spinner = ora("Checking...\n\n");
    spinner.start();

    const init = {
      method: "POST",
      headers: {
        "Content-Type": "application/json",
        Authorization: `Bearer ${key}`,
      },
      body: JSON.stringify({
        method: options.method ?? "GET",
        url,
        headers: options.header,
        body: options.data,
        n: options.runs ?? 1,
      }),
    };
    const res = await fetch(`${API_BASE_URL}/api/check`, init);
    spinner.stop();

    if (!res.ok) {
      console.error(`Error: ${res.status} ${res.statusText}\n\n${await res.text()}`);
      process.exit(1);
    }

    const json = await res.json();

    console.table(json.regions);

    process.exit(0);
  });
programm
  .command("login")
  .description("Login to your account and save your API key locally")
  .action(async (...args) => {
    if (args.length !== 2) {
      console.error("Usage: `globalstat login`.");
      process.exit(1);
    }

    // create localhost server for our page to call back to
    const server = http.createServer();
    const { port } = await listen(server, 0, "127.0.0.1");

    // set up HTTP server that waits for a request containing an API key
    // as the only query parameter
    const authPromise = new Promise<ParsedUrlQuery>((resolve, reject) => {
      server.on("request", (req, res) => {
        // Set CORS headers for all responses
        res.setHeader("Access-Control-Allow-Origin", "*");
        res.setHeader("Access-Control-Allow-Methods", "GET, OPTIONS");
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
            res.end();
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
    const confirmationUrl = new URL(`${API_BASE_URL}/auth/devices`);
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
      console.log(
        `Authentication successful: wrote key to config file. To view it, type 'cat ~/${FILENAME}'.\n`,
      );
      server.close();
      process.exit(0);
    } catch (error) {
      if (error instanceof UserCancellationError) {
        console.log("Authentication cancelled.\n");
        server.close();
        process.exit(0);
      } else {
        console.error("Authentication failed:", error);
        console.log("\n");
        server.close();
        process.exit(1);
      }
    } finally {
      server.close();
      process.exit(0);
    }
  });

programm.parse();
