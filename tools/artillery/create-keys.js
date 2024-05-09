const fs = require("node:fs");

async function main() {
  const rootKey = process.env.UNKEY_ROOT_KEY;
  if (!rootKey) {
    throw new Error("UNKEY_ROOT_KEY not set");
  }

  const apiId = process.env.UNKEY_API_ID;
  if (!apiId) {
    throw new Error("UNKEY_API_ID not set");
  }

  const keys = [];

  for (let i = 0; i < 100; i++) {
    const res = await fetch("https://api.unkey.dev/v1/keys.createKey", {
      method: "POST",
      headers: { Authorization: `Bearer ${rootKey}`, "Content-Type": "application/json" },
      body: JSON.stringify({
        apiId,
        prefix: "art",
        name: "artillery",
        expires: Date.now() + 60 * 60 * 1000, // 1 hour
      }),
    });
    const { key, keyId } = await res.json();
    console.info(i, "created", keyId);
    keys.push(key);
  }

  fs.writeFileSync(".keys.csv", keys.join("\n"));
}

main();
