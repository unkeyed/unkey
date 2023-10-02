"use client";
import {
  Accordion,
  AccordionContent,
  AccordionItem,
  AccordionTrigger,
} from "@/components/ui/accordion";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { Card, CardContent, CardFooter, CardHeader, CardTitle } from "@/components/ui/card";
import { Checkbox } from "@/components/ui/checkbox";
import { Label } from "@/components/ui/label";
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select";
import { toast } from "@/components/ui/use-toast";
import { type VercelBinding } from "@unkey/db";
import { Api } from "@unkey/db";
import { ArrowLeft, Check } from "lucide-react";
import Link from "next/link";
import React, { useState } from "react";
import { experimental_useFormStatus as useFormStatus } from "react-dom";
import { updateBindings } from "./actions";

type Props = {
  projects: {
    id: string;
    name: string;
  }[];
  apis: Api[];
  integrationId: string;
  returnUrl: string;
};

type Step =
  | {
      id: "selectProject";
    }
  | {
      id: "addApi";
      projectId: string;
    };

export const Client: React.FC<Props> = ({ projects, apis, returnUrl, integrationId}) => {
  if (projects.length === 0) {
    return <div>no projects</div>;
  }
  const [step, setStep] = useState<Step>({ id: "selectProject" });

  const environments: Record<VercelBinding["apiId"], string> = {
    production: "Production",
    preview: "Preview",
    development: "Development",
  };
  return (
    <div className="flex items-center justify-center p-8 h-min-screen">
      <div>
        <svg
          width="76"
          height="65"
          viewBox="0 0 76 65"
          fill="none"
          xmlns="http://www.w3.org/2000/svg"
        >
          <path d="M37.5274 0L75.0548 65H0L37.5274 0Z" fill="#000000" />
        </svg>
        <Card>
          <CardHeader>
            <CardTitle>Connect a Vercel Project</CardTitle>
            {step.id === "selectProject" ? (
              <Select onValueChange={(projectId) => setStep({ id: "addApi", projectId })}>
                <SelectTrigger>
                  <SelectValue placeholder="Select a project" />
                </SelectTrigger>
                <SelectContent>
                  {projects.map((project) => (
                    <SelectItem key={project.id} value={project.id}>
                      {project.name}
                    </SelectItem>
                  ))}
                </SelectContent>
              </Select>
            ) : null}
          </CardHeader>
          <CardContent>
            {step.id === "addApi" ? (
              <form
                action={async (formData: FormData) => {
                  formData.forEach((value, key) => {
                    console.log({ value, key });
                  });
                  const res = await updateBindings(formData);
                  if (res.error) {
                    toast({
                      title: "Error",
                      description: res.error,
                      variant: "alert",
                    });
                    setStep({ id: "selectProject" });
                    return;
                  }
                  toast({
                    title: "Success",
                    description: "Updated bindings",
                  });
                }}
                className="flex flex-col gap-4"
              >
                <input
                  type="hidden"
                  name="integrationId"
                  value={integrationId}
                  className="sr-only"
                />
                 <input
                  type="hidden"
                  name="projectId"
                  value={step.projectId}
                  className="sr-only"
                />
                {Object.entries(environments).map(([env, label]) => (
                  <div className="flex items-center gap-2">
                    <Label>{label}</Label>{" "}
                    <Select name={env}>
                      <SelectTrigger>
                        <SelectValue defaultValue="" placeholder="None" />
                      </SelectTrigger>
                      <SelectContent>
                        <SelectItem value="">None</SelectItem>
                        {apis.map((api) => (
                          <SelectItem key={api.id} value={api.id}>
                            {api.name}
                          </SelectItem>
                        ))}
                      </SelectContent>
                    </Select>
                  </div>
                ))}

                <Button variant="primary" type="submit">
                  Add
                </Button>
              </form>
            ) : null}
          </CardContent>
          <CardFooter className="flex items-center justify-between">
            <Button variant="secondary" onClick={() => {}}>
              <ArrowLeft className="w-4 h-4 mr-2" />
              Back
            </Button>
            <Link href={returnUrl}>
              <Button variant="primary">
                <Check className="w-4 h-4 mr-2" />
                Finish
              </Button>
            </Link>
          </CardFooter>
        </Card>
      </div>
    </div>
    // <form
    //   action={async (_formData: FormData) => {
    //     //   const res = await updateApiName(formData);
    //     //   if (res.error) {
    //     //     toast({
    //     //       title: "Error",
    //     //       description: res.error,
    //     //       variant: "alert",
    //     //     });
    //     //     return;
    //     //   }
    //     //   toast({
    //     //     title: "Success",
    //     //     description: "Api name updated",
    //     //   });
    //   }}
    // >
    //   <Select>
    //     <SelectTrigger>
    //       <SelectValue placeholder="Select a project" />
    //     </SelectTrigger>
    //     <SelectContent>
    //       {props.projects.map((project) => (
    //         <SelectItem key={project.id} value={project.id}>
    //           {project.name}
    //         </SelectItem>
    //       ))}
    //     </SelectContent>
    //   </Select>
    // </form>
  );
};
