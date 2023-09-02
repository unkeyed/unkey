"use client";
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card";
import { useToast } from "@/components/ui/use-toast";
import { useOrganization } from "@clerk/nextjs";
import { UploadCloud } from "lucide-react";
import Link from "next/link";
import React, { ChangeEvent, useCallback, useEffect, useState } from "react";

export const UpdateWorkspaceImage: React.FC = () => {
  const { toast } = useToast();
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
      toast({ description: "Uploading image..." });
      const file = e.target.files?.[0];

      if (!file) {
        toast({ description: "No image selected", variant: "alert" });
        return;
      }
      if (file.size / 1024 / 1024 > 2) {
        toast({ description: "File size too big (max 2MB)", variant: "alert"});
        return;
      }
      if (file.type !== "image/png" && file.type !== "image/jpeg") {
        toast({ description: "File type not supported (.png or .jpg only)", variant: "alert"});
        return;
      }

      const reader = new FileReader();
      reader.onload = (e) => {
        setImage(e.target?.result as string);
      };
      reader.readAsDataURL(file);

      if (!organization) {
        toast({ description: "Only allowed for orgs", variant: "alert" });
        return;
      }
      organization
        .setLogo({ file })
        .then(() => {
          toast({ description: "Image uploaded" });
        })
        .catch(() => {
          toast({ description: "Error uploading image", variant: "alert" });
        });
    },
    [setImage, organization],
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
            <Link href="/app/settings/user" className="underline hover:text-content">
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
          toast({ variant: "alert", description: "No image selected" });
          return;
        }

        await organization?.setLogo({ file: image });
        await organization?.reload();
        toast({ description: "Image uploaded" });
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
                // rome-ignore lint/complexity/useOptionalChain: <explanation>
                const file = e.dataTransfer.files && e.dataTransfer.files[0];
                if (file) {
                  if (file.size / 1024 / 1024 > 2) {
                    toast({ description: "File size too big (max 2MB)" });
                  } else if (file.type !== "image/png" && file.type !== "image/jpeg") {
                    toast({ description: "File type not supported (.png or .jpg only)" });
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
    // <form
    //   action={async (formData: FormData) => {
    //     const res = await updateWorkspaceName(formData);
    //     if (res.error) {
    //       toast({
    //         title: "Error",
    //         description: res.error,
    //         variant: "alert",
    //       });
    //       return;
    //     }
    //     toast({
    //       title: "Success",
    //       description: "Workspace name updated",
    //     });

    //     user?.reload();
    //   }}
    // >
    //   <Card>

    //     <CardContent>

    //       <div className="flex flex-col space-y-2">
    //         <input type="hidden" name="workspaceId" value={workspace.id} />
    //         <Label>Name</Label>
    //         <Input name="name" className="max-w-sm" defaultValue={workspace.name} />
    //         <p className="text-xs text-content-subtle">What should your workspace be called?</p>
    //       </div>
    //     </CardContent>
    //     <CardFooter className="justify-end">

    //       <Button variant={pending ? "disabled" : "primary"} type="submit" disabled={pending}>
    //         {pending ? <Loading /> : "Save"}
    //       </Button>
    //     </CardFooter>
    //   </Card>
    // </form>
  );
};
