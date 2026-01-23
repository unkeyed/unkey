import type { NextApiRequest, NextApiResponse } from "next";

export async function main() {
  const WEBHOOK_PAYLOAD = [
    {
      token: "unkey_3Zji53gSycvTL7vJGiCoh7es",
      type: "unkey_root_key",
      url: "https://github.com/octocat/Hello-World/blob/12345600b9cbe38a219f39a9941c9319b600c002/foo/bar.txt",
      source: "content", // where it was found on Github: code, PR title, etc
    },
  ];
  const request = await fetch("http://localhost:3000/api/v1/github/verify", {
    method: "POST",
    body: JSON.stringify(WEBHOOK_PAYLOAD),
    headers: {
      "content-type": "application/json",
      "Github-Public-Key-Identifier":
        "bcb53661c06b4728e59d897fb6165d5c9cda0fd9cdf9d09ead458168deb7518c",
      "Github-Public-Key-Signature":
        "MEQCIQDaMKqrGnE27S0kgMrEK0eYBmyG0LeZismAEz/BgZyt7AIfXt9fErtRS4XaeSt/AO1RtBY66YcAdjxji410VQV4xg==",
    },
  });
  const response = await request.json();
  // console.log(response);
  return response;
}

// biome-ignore lint/style/noDefaultExport: Required by next.js
export default async function handler(request: NextApiRequest, response: NextApiResponse) {
  try {
    const result = await main();
    return response.status(200).json(result);
  } catch (error) {
    return response.status(500).json({ error: "Internal Server Error" });
  }
}
