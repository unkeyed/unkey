"use client";
import { cn } from "@/lib/utils";
import { Check } from "lucide-react";
import { Button } from "@/components/ui/button";
import { Particles } from "@/components/particles";
import { Input } from "@/components/ui/input";
import { useToast } from "@/hooks/use-toast";
import { useState } from "react";
import { trpc } from "@/lib/trpc/client";
import { CardDescription, CardHeader, CardTitle, Card, CardContent } from "@/components/ui/card";

const steps = [
  {
    id: "1",
    name: "Create your team",
    description: "Before we can get started, we need to create your team.",
  },
  {
    id: "2",
    name: "Add a slug",
    description: "Where should your team belong?",
  },
  {
    id: "3",
    name: "Review your team",
    description: "Let's make sure everything is correct before we create your team",
  },
];

type Props = {
  tenantId: string;
};
export const Onboarding: React.FC<Props> = ({ tenantId }) => {
  const [currentStep, setCurrentStep] = useState(0);
  const [teamName, setTeamName] = useState("");
  const [teamSlug, setTeamSlug] = useState("");
  const { toast } = useToast();
  const tenant = trpc.tenant.create.useMutation({
    onSuccess() {
      toast({
        title: "Team Created",
        description: "Your team has been created",
      });
    },
    onError(err) {
      console.error(err);
      toast({ title: "Error", description: err.message, variant: "destructive" });
    },
  });

  return (
    <div>
      <ol
        role="list"
        className="overflow-hidden border-b rounded-md lg:flex lg:rounded-none  divide-x divide-white/10 border-white/10 bg-primary-900"
      >
        {steps.map((step, stepIdx) => (
          <li key={step.id} className="relative overflow-hidden lg:flex-1">
            <Step
              key={step.id}
              id={step.id}
              title={step.name}
              description={step.description}
              state={
                stepIdx < currentStep
                  ? "completed"
                  : stepIdx === currentStep
                  ? "current"
                  : "upcoming"
              }
            />
          </li>
        ))}
      </ol>

      <main className="p-4 md:p-6 lg:p-8">
        {currentStep === 0 && (
          <div className="flex flex-col gap-4 md:gap-6 lg:gap-8">
            <Card>
              <CardHeader>
                <CardTitle>Choose your team Name</CardTitle>
                <CardDescription>Let's choose a name for your team</CardDescription>
              </CardHeader>
              <CardContent>
                <div>
                  <div className="flex items-center justify-center px-2 py-1 mt-8 rounded gap-4 ">
                    <Input
                      name="teamName"
                      type="text"
                      placeholder="Team Name"
                      value={teamName}
                      onChange={(e) =>
                        setTeamName(e.target.value.replace(/[^a-zA-Z0-9_-\s]+/g, ""))
                      }
                    />
                  </div>
                </div>
                <div className="flex justify-end ">
                  <Button
                    disabled={teamName.length < 1}
                    className="my-4 w-1/3"
                    onClick={() => {
                      setCurrentStep(1);
                      setTeamSlug(teamName.replace(/\s+/g, "-").toLowerCase());
                    }}
                  >
                    Next
                  </Button>
                </div>
              </CardContent>
            </Card>
          </div>
        )}
        {teamName && currentStep === 1 && (
          <div className="flex flex-col gap-4 md:gap-6 lg:gap-8">
            <Card>
              <CardHeader>
                <CardTitle>Choose your team slug</CardTitle>
                <CardDescription>
                  Let's choose a slug for your team. by default we used your team name as a slug.
                  You can change it if you want.'
                </CardDescription>
              </CardHeader>
              <CardContent>
                <div>
                  <div className="flex items-start justify-between px-2 py-1 mt-8 rounded gap-4 ">
                    <Input
                      name="teamSlug"
                      type="text"
                      defaultValue={teamSlug}
                      onChange={(e) =>
                        setTeamSlug(e.target.value.replace(/^[a-zA-Z0-9-_\.]+$/, ""))
                      }
                      placeholder="Team Slug"
                    />
                  </div>
                </div>
                <div className="flex justify-end items- gap-4 px-2">
                  <Button
                    variant="secondary"
                    className="my-4 w-1/3"
                    onClick={() => setCurrentStep(0)}
                  >
                    Previous
                  </Button>
                  <Button
                    disabled={teamSlug.length < 1}
                    className="my-4 w-1/3"
                    onClick={() => setCurrentStep(2)}
                  >
                    Next
                  </Button>
                </div>
              </CardContent>
            </Card>
          </div>
        )}
        {teamSlug && currentStep === 2 && (
          <div className="flex flex-col gap-4 md:gap-6 lg:gap-8">
            <Card>
              <CardHeader>
                <CardTitle>Review Your Team </CardTitle>
                <CardDescription>
                  Let's make sure everything is correct before we create your team.
                </CardDescription>
              </CardHeader>
              <CardContent>
                <div>
                  <div className="flex flex-col items-center justify-center px-2 py-1 mt-8 rounded gap-4">
                    <p>
                      {" "}
                      <span className="font-bold">Team Name: </span> {teamName}
                    </p>
                    <p>
                      {" "}
                      <span className="font-bold">Team Slug: </span> {teamSlug}
                    </p>
                  </div>
                </div>
                <div className="flex justify-end gap-4 px-2">
                  <Button
                    variant="secondary"
                    className="my-4 w-1/3"
                    onClick={() => setCurrentStep(1)}
                  >
                    Previous
                  </Button>
                  <Button
                    disabled={!(teamName && teamSlug)}
                    className=" my-4 w-1/3"
                    onClick={() =>
                      tenant.mutate({
                        tenantId: tenantId,
                        name: teamName,
                        slug: teamSlug,
                      })
                    }
                  >
                    Submit
                  </Button>
                </div>
              </CardContent>
            </Card>
          </div>
        )}
      </main>
    </div>
  );
};

type StepProps = {
  id: string;
  title: string;
  description: string;
  state: "current" | "upcoming" | "completed";
};

const Step: React.FC<StepProps> = ({ id, title, description, state }) => {
  return (
    <div
      className={cn("group h-full flex items-start px-6 py-5 text-sm font-medium duration-1000", {
        "bg-gray-400/5 hover:bg-gray-600/10  ": state === "current",
      })}
    >
      <Particles
        className="absolute inset-0"
        vy={-1}
        quantity={50}
        staticity={200}
        color="#7c3aed"
      />

      <span
        className={cn(
          "flex justify-center items-center rounded w-6 h-6 text-xs font-medium ring-1 ring-inset",
          {
            "bg-gray-100/10 text-gray-900 ring-gray-100/10 shadow-xl shadow-gray-100/10 group-hover:shadow-gray-200/70 duration-1000":
              state === "current",
            "text-gray-600 ring-gray-400/30": state === "upcoming",
            "text-gray-700 ring-gray-200/70": state === "completed",
          },
        )}
      >
        {state === "completed" ? <Check className="w-3 h-3" /> : id}
      </span>
      <span className="ml-4 mt-0.5 flex min-w-0 flex-col">
        <span
          className={cn("text-sm font-medium", {
            "text-zinc-600": state === "current",
            "text-zinc-800": state !== "current",
          })}
        >
          {title}
        </span>
        <span
          className={cn("text-sm font-medium", {
            "text-zinc-600": state === "current",
            "text-zinc-800": state !== "current",
          })}
        >
          {description}
        </span>
      </span>
    </div>
  );
};
