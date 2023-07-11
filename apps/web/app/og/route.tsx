import { ImageResponse } from "next/server";

export const runtime = "edge";

export async function GET(request: Request) {
  try {
    const { searchParams } = new URL(request.url);
    const hasTitle = searchParams.has("title");
    const title = hasTitle ? searchParams.get("title")?.slice(0, 100) : "My default title";

    return new ImageResponse(
      <div
        style={{
          backgroundColor: "#000000",
          backgroundSize: "1200px 630px",
          height: "100%",
          width: "100%",
          display: "flex",
          textAlign: "center",
          alignItems: "center",
          justifyContent: "center",
          flexDirection: "column",
          flexWrap: "nowrap",
        }}
      >
        <div
          style={{
            display: "flex",
            alignItems: "center",
            justifyContent: "center",
            justifyItems: "center",
          }}
        >
          <img
            alt="Unkey"
            height={200}
            src={"https://unkey.dev/unkey.png"}
            style={{ margin: "0 30px" }}
            width={232}
          />
        </div>
        <div
          style={{
            fontSize: 60,
            fontStyle: "normal",
            letterSpacing: "-0.025em",
            color: "white",
            marginTop: 30,
            padding: "0 120px",
            lineHeight: 1.4,
            whiteSpace: "pre-wrap",
          }}
        >
          {title}
        </div>
      </div>,
      {
        width: 1200,
        height: 630,
      },
    );
  } catch (_e: any) {
    return new Response("Failed to generate the image", {
      status: 500,
    });
  }
}
