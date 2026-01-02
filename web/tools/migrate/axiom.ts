import { Axiom } from "@axiomhq/js";
async function main() {
  const axiom = new Axiom({
    token: process.env.AXIOM_TOKEN!,
    orgId: process.env.AXIOM_ORG_ID,
  });

  const interval = 60 * 60 * 1000;
  let t = new Date("2024-06-02T15:58:32.681Z");

  t.setUTCMinutes(0, 0, 0);
  const cutoff = new Date(t.getTime() - 100 * 24 * 60 * 60 * 1000).getTime();

  while (t.getTime() > cutoff) {
    console.info(t);
    const end = t.getTime();
    const start = end - interval;

    const res = await axiom.query("cf_api_metrics_production | order by _time desc", {
      startTime: new Date(start).toISOString(),
      endTime: new Date(end).toISOString(),
    });

    const matches = res.matches ?? [];
    console.info(matches.length);

    if (matches.length > 0) {
      t = new Date(Math.min(...res.matches!.map((m) => new Date(m._time).getTime())));
      axiom.ingest(
        "metrics",
        matches.map((m) => ({
          _time: m._time,
          application: "api",
          environment: "production",
          ...m.data,
        })),
      );
    } else {
      t = new Date(t.getTime() - interval);
    }
  }
  await axiom.flush();
}

main();
