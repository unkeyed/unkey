async function main() {
  const exportFile = Bun.file("./export_ratelimits.json");
  const writer = exportFile.writer();
  let cursor = 0;
  do {
    const res = await fetch(
      `https://api.tinybird.co/v0/pipes/ratelimits_export_endpoint.json?cursor=${cursor}`,
      {
        headers: {
          Authorization: `Bearer ${process.env.TINYBIRD_TOKEN}`,
        },
      },
    );
    const body = await res.text();
    //  console.log(body);
    const { data } = JSON.parse(body) as { data: { time: number }[] };
    for (const row of data) {
      writer.write(JSON.stringify(row));
      writer.write("\n");
    }

    cursor = data.at(-1)?.time ?? 0;
  } while (cursor);
}

main();
