/**
 * TODO: With WorkOS, orgs don't have an image.
 * We will need to implement this ourselves at a future date.
 * Currently unsupported and ununsed until re-implemented.
 */

"use client";
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card";
import { toast } from "@/components/ui/toaster";
import { useOrganization } from "@clerk/nextjs";
import { UploadCloud } from "lucide-react";
import Link from "next/link";
import type React from "react";
import { type ChangeEvent, useCallback, useEffect, useState } from "react";

export const UpdateWorkspaceImage: React.FC = () => {
  const { organization } = useOrganization();

  const [image, setImage] = useState<string | null>(organization?.imageUrl ?? null);

  const [dragActive, setDragActive] = useState(false);

  useEffect(() => {
    if (organization?.imageUrl) {
      setImage(organization.imageUrl);
    }
  }, [organization?.imageUrl]);

  const onChangePicture = useCallback(
    (e: ChangeEvent<HTMLInputElement>) => {
      toast("Uploading image...");
      const file = e.target.files?.[0];

      if (!file) {
        toast.error("No image selected");
        return;
      }
      if (file.size / 1024 / 1024 > 2) {
        toast.error("File size too big (max 2MB)");
        return;
      }
      if (file.type !== "image/png" && file.type !== "image/jpeg") {
        toast.error("File type not supported (.png or .jpg only)");
        return;
      }

      const reader = new FileReader();
      reader.onload = (e) => {
        setImage(e.target?.result as string);
      };
      reader.readAsDataURL(file);

      if (!organization) {
        toast.error("Only allowed for orgs");
        return;
      }
      organization
        .setLogo({ file })
        .then(() => {
          toast.success("Image uploaded");
        })
        .catch(() => {
          toast.error("Error uploading image");
        });
    },
    [organization],
  );

  if (!organization) {
    return (
      <Card>
        <CardHeader>
          <CardTitle>Workspace Avatar</CardTitle>
        </CardHeader>
        <CardContent>
          <p className="text-sm text-content-subtle">
            This is a personal workspace. Change your personal avatar{" "}
            <Link href="/settings/user" className="underline hover:text-content">
              here
            </Link>
            .
          </p>
        </CardContent>
      </Card>
    );
  }

  return (
    <form
      onSubmit={async (e) => {
        e.preventDefault();
        if (!image) {
          toast.error("No image selected");
          return;
        }

        await organization?.setLogo({ file: image });
        await organization?.reload();
        toast.success("Image uploaded");
      }}
    >
      <Card className="flex items-start justify-between">
        <CardHeader>
          <CardTitle>Workspace Avatar</CardTitle>
          <CardDescription>
            Click on the avatar to upload a custom one from your files.
            <br />
            Square image recommended. Accepted file types: .png, .jpg. Max file size: 2MB.
          </CardDescription>
        </CardHeader>
        <CardContent>
          <label
            htmlFor="image"
            className="relative flex flex-col items-center justify-center w-24 h-24 mt-1 transition-all border rounded-full shadow-sm cursor-pointer bg-background border-border group hover:bg-background-subtle"
          >
            <div
              className="absolute z-[5] h-full w-full rounded-full"
              onDragOver={(e) => {
                e.preventDefault();
                e.stopPropagation();
                setDragActive(true);
              }}
              onDragEnter={(e) => {
                e.preventDefault();
                e.stopPropagation();
                setDragActive(true);
              }}
              onDragLeave={(e) => {
                e.preventDefault();
                e.stopPropagation();
                setDragActive(false);
              }}
              onDrop={(e) => {
                e.preventDefault();
                e.stopPropagation();
                setDragActive(false);
                const file = e.dataTransfer.files?.[0];
                if (file) {
                  if (file.size / 1024 / 1024 > 2) {
                    toast.error("File size too big (max 2MB)");
                  } else if (file.type !== "image/png" && file.type !== "image/jpeg") {
                    toast.error("File type not supported (.png or .jpg only)");
                  } else {
                    const reader = new FileReader();
                    reader.onload = (e) => {
                      setImage(e.target?.result as string);
                    };
                    reader.readAsDataURL(file);
                  }
                }
              }}
            />
            <div
              className={`${
                dragActive ? "cursor-copy border-2 border-black bg-gray-50 opacity-100" : ""
              } absolute z-[3] flex h-full w-full flex-col items-center justify-center rounded-full bg-white transition-all ${
                image ? "opacity-0 group-hover:opacity-100" : "group-hover:bg-gray-50"
              }`}
            >
              <UploadCloud
                className={`${
                  dragActive ? "scale-110" : "scale-100"
                } h-5 w-5 text-gray-500 transition-all duration-75 group-hover:scale-110 group-active:scale-95`}
              />
            </div>
            {image && (
              <img src={image} alt="Preview" className="object-cover w-full h-full rounded-full" />
            )}
          </label>
          <div className="flex mt-1 rounded-full shadow-sm">
            <input
              id="image"
              name="image"
              type="file"
              accept="image/*"
              className="sr-only"
              onChange={onChangePicture}
            />
          </div>
        </CardContent>
      </Card>
    </form>
  );
};
