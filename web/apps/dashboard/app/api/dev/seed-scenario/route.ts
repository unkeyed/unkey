import { readFile, writeFile } from "node:fs/promises";
import { NextResponse } from "next/server";

const MARKER = "/tmp/unkey-seed-scenario";
const REQUEST = "/tmp/unkey-seed-request";

const SCENARIOS = [
  "no-plan",
  "starter-over",
  "pro-healthy",
  "business-high",
  "suspended",
  "budget-no-stop",
  "budget-stop",
  "api-over-quota",
  "both-over",
  "zero-usage",
  "under-credit",
  "unattributed",
] as const;

export const dynamic = "force-dynamic";

function devOnly(): NextResponse | null {
  if (process.env.NODE_ENV === "production") {
    return NextResponse.json({ error: "not available" }, { status: 404 });
  }
  return null;
}

/** Which scenario seed-scenario.sh last applied, for the on-screen banner. */
export async function GET() {
  const blocked = devOnly();
  if (blocked) {
    return blocked;
  }

  try {
    const scenario = (await readFile(MARKER, "utf8")).trim();
    return NextResponse.json({ scenario: scenario === "" ? null : scenario, known: SCENARIOS });
  } catch {
    return NextResponse.json({ scenario: null, known: SCENARIOS });
  }
}

/**
 * Asks for a scenario by writing its name to a request file that
 * `seed-scenario.sh --watch` picks up and applies.
 *
 * Deliberately a file write and not a command: the seeding is a shell script,
 * and running one in response to a web request is not a thing to build even in
 * development. The watcher owns execution; this only ever leaves a name behind.
 */
export async function POST(request: Request) {
  const blocked = devOnly();
  if (blocked) {
    return blocked;
  }

  let scenario: unknown;
  try {
    ({ scenario } = await request.json());
  } catch {
    return NextResponse.json({ error: "expected a JSON body" }, { status: 400 });
  }

  const match = SCENARIOS.find((known) => known === scenario);
  if (!match) {
    return NextResponse.json(
      { error: `unknown scenario, expected one of: ${SCENARIOS.join(", ")}` },
      { status: 400 },
    );
  }

  await writeFile(REQUEST, match, "utf8");
  return NextResponse.json({ requested: match });
}
