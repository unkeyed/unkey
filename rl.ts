const identifiers: string[] = [];

function newId(): string {
  if (identifiers.length === 0) {
    const id = crypto.randomUUID();
    identifiers.push(id);
    return id;
  }
  if (Math.random() < 0.95) {
    return identifiers[Math.floor(Math.random() * identifiers.length)];
  }
  const id = crypto.randomUUID();
  identifiers.push(id);
  return id;
}

const responses: boolean[] = [];
const start = Date.now();
for (let i = 0; i < 1000; i++) {
  const t1 = performance.now();
  //const res = await fetch("http://localhost:8787/v1/ratelimits.limit",{
  const res = await fetch("https://api.unkey.dev/v1/ratelimits.limit", {
    method: "POST",
    headers: {
      "Content-Type": "application/json",
      Authorization: "Bearer XXX",
    },
    body: JSON.stringify({
      namespace: "local",
      identifier: newId(),
      async: true,
      limit: 10,
      duration: 60000,
    }),
  }).then((res) => res.json());
  console.log(`i=${i}`, `latency=${Math.round(performance.now() - t1)} ms`, res);
  responses.push(res.success);
}

const success = responses.filter(Boolean).length;
console.log("passed", success, "/", responses.length);
console.log("time", Math.floor((Date.now() - start) / 1000), "s");
console.log("unique ids", identifiers.length);
