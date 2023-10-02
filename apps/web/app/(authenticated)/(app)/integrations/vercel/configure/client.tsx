"use client";
import {
  Accordion,
  AccordionContent,
  AccordionItem,
  AccordionTrigger,
} from "@/components/ui/accordion";
import { Avatar, AvatarImage } from "@/components/ui/avatar";
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
import { useClerk } from "@clerk/nextjs";
import { AvatarFallback } from "@radix-ui/react-avatar";
import { type VercelBinding } from "@unkey/db";
import { Api } from "@unkey/db";
import { ArrowLeft, Check, Code, MoreHorizontal } from "lucide-react";
import ms from "ms";
import Link from "next/link";
import React, { useState } from "react";
import { experimental_useFormStatus as useFormStatus } from "react-dom";

type Props = {
  bindings: VercelBinding[];
  users: Record<string, { id: string, name: string, image: string }>
  apis: Api[];
};

export const Client: React.FC<Props> = ({ bindings, users,apis }) => {
  return (
    <div className="flex items-center justify-center p-8 h-min-screen">
      <ul className="bg-white border divide-y rounded">
        {bindings.map((binding, i) => {
          const user = users[binding.lastEditedBy]
          return (
            <li key={`${binding.projectId}-${binding.environment}`} className="flex items-center justify-between gap-8 p-4 hover:bg-background">
              <div className="flex flex-col items-start">
                <span className="text-sm text-content">{binding.projectId}</span>
                <span className="text-xs text-content-subtle">{binding.environment}</span>
              </div>

              <div className="flex items-center gap-2">
                <Code className="w-4 h-4"/>
                {apis.find((api) => api.id === binding.apiId)?.name ?? "No API"}
              </div>

              <div className="flex items-center gap-4">
                <span className="text-xs">
                  {ms(Date.now() - binding.updatedAt.getTime(), { long: true })} ago by {user.name}
                </span>
                <Avatar className="w-6 h-6">
                  <AvatarImage
                    src={user.image}
                    alt={user.name}
                  />

                </Avatar>
              </div>

              <MoreHorizontal className="w-4 h-4"/>

            </li>
          )
        })}

      </ul>
    </div>
  );
};
