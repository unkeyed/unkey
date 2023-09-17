import { ImageResponse } from "next/server";
import { templates } from "../data";

// Route segment config
export const runtime = "edge";

// Image metadata
export const alt = "Unkey Templates";
export const size = {
  width: 1200,
  height: 630,
};

export const contentType = "image/png";

// Image generation
export default async function Image(props: { params: { slug: string } }) {
  // Font
  const satoshiSemiBold = fetch(new URL("@/styles/Satoshi-Bold.ttf", import.meta.url)).then(
    (res) => res.arrayBuffer(),
  );

  const template = templates[props.params.slug];

  return new ImageResponse(
    // ImageResponse JSX element
    <div
      style={{
        fontSize: 128,
        background: "white",
        width: "100%",
        height: "100%",
        display: "flex",
        alignItems: "center",
        justifyContent: "center",
      }}
    >
      {template.title ?? "Unkey Template"}
    </div>,
    // ImageResponse options
    {
      // For convenience, we can re-use the exported opengraph-image
      // size config to also set the ImageResponse's width and height.
      ...size,
      fonts: [
        {
          name: "Satoshi Bold",
          data: await satoshiSemiBold,
        },
      ],
    },
  );
}
