export async function POST(request: Request) {
  const data = await request.json();
  if (data.type === "checkout.session.completed") {
    console.log(data);
    return new Response("OK", { status: 200 });
  }

  return new Response("Bad Request", { status: 400 });
}
