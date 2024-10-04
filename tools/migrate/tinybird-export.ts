async function main() {
  const exportFile = Bun.file("./export_audit_log.json");
  const writer = exportFile.writer();
  let cursor = "";
  do {
    const res = await fetch(
      `https://api.tinybird.co/v0/pipes/export_audit_log.json?cursor=${cursor}`,
      {
        headers: {
          Authorization: `Bearer ${process.env.TINYBIRD_TOKEN}`,
        },
      },
    );
    const body = (await res.json()) as { data: { auditLogId: string }[] };
    for (const row of body.data) {
      writer.write(JSON.stringify(row));
      writer.write("\n");
    }

    cursor = body.data.at(-1)?.auditLogId ?? "";
  } while (cursor);
}

main();
