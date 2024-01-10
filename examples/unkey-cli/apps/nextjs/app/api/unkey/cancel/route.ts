export async function POST(req: Request) {
  const { localServer } = await req.json();

  try {
    await fetch(localServer, {
      method: "POST",
      headers: {
        "Content-Type": "application/json",
      },
      body: JSON.stringify({ cancel: true }),
    });
  } catch (error) {
    return new Response(`Error cancelling login process: ${error}`, { status: 500 });
  }

  return new Response("ok");
}
