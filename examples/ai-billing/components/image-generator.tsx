"use client";
import { Button, Card, TextInput } from "@tremor/react";
import { useState } from "react";
import { toast } from "sonner";
import { revalidate } from "../app/revalidate";

export function ImageGenerator({ credits }: { credits: number }) {
  const [prompt, setPrompt] = useState("");
  const [loading, setLoading] = useState(false);
  const [images, setImages] = useState([] as string[]);

  async function generate() {
    setLoading(true);
    try {
      const res = await fetch("/api/openai", {
        method: "POST",
        headers: {
          "Content-Type": "application/json",
        },
        body: JSON.stringify({
          prompt: prompt,
        }),
      });
      if (!res.ok) {
        throw new Error(await res.text());
      }
      const { images } = await res.json();
      setImages(images);
      revalidate("/");
      setLoading(false);
    } catch (error) {
      toast.error((error as Error).message);
      setLoading(false);
    }
  }

  return (
    <div className="pb-10 mt-10 max-w-[340px] md:max-w-full">
      <Card className="py-8 px-12">
        <h1 className="text-3xl font-medium">Generate images</h1>
        <div className="flex flex-col">
          <div className="flex mt-5">
            <TextInput
              value={prompt}
              placeholder="A slow loris riding a unicycle"
              onChange={(e) => setPrompt(e.target.value)}
              className=""
            />
            <Button loading={loading} disabled={credits === 0} className="ml-2" onClick={generate}>
              Generate
            </Button>
          </div>
        </div>
      </Card>
      <div className="mt-10">
        <div className="grid grid-cols-2 gap-4 w-full max-w-2xl mx-auto mt-8">
          <img
            src={images.length > 0 ? images[0] : "placeholder.svg"}
            alt={prompt}
            className="shadow-sm aspect-square object-cover border border-gray-200 w-full rounded-lg overflow-hidden dark:border-gray-800"
          />
          <img
            src={images.length > 0 ? images[1] : "placeholder.svg"}
            alt={prompt}
            className="shadow-sm aspect-square object-cover border border-gray-200 w-full rounded-lg overflow-hidden dark:border-gray-800"
          />
        </div>
      </div>
    </div>
  );
}
