"use client";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { Card, CardContent } from "@/components/ui/card";
import { ChevronRight, GitMerge } from "lucide-react";
import { useRouter } from "next/navigation";
import { useState } from "react";

interface MergeConfirmationCardProps {
  sourceBranch: string;
  targetBranch: string;
  targetBranchId: string;
}

export default function MergeConfirmationCard({
  sourceBranch,
  targetBranch,
  targetBranchId,
}: MergeConfirmationCardProps) {
  const [isLoading, setIsLoading] = useState(false);
  const router = useRouter();
  const handleConfirm = async () => {
    setIsLoading(true);

    await fetch("/api/deploy");
    setIsLoading(false);

    router.push(`/deploy/promote/${targetBranchId}/success`);
  };

  return (
    <Card className="w-full max-w-md mx-auto h-min">
      <CardContent className="pt-6">
        <div className="flex items-center justify-center mb-6">
          <Badge variant="secondary" className="text-sm font-medium">
            {sourceBranch}
          </Badge>
          <ChevronRight className="mx-2 h-4 w-4 text-gray-400" />
          <Badge variant="primary" className="text-sm font-medium">
            {targetBranch}
          </Badge>
        </div>
        <h2 className="text-lg font-semibold text-center mb-2">Confirm Changes</h2>
        <p className="text-sm text-gray-600 text-center mb-6">
          You are about to deploy changes from <strong>{sourceBranch}</strong> into{" "}
          <strong>{targetBranch}</strong>.
        </p>
        <div className="flex justify-center w-full">
          <Button onClick={handleConfirm} disabled={isLoading} className="w-full ">
            {isLoading ? (
              <GitMerge className="mr-2 h-4 w-4 animate-pulse" />
            ) : (
              <GitMerge className="mr-2 h-4 w-4" />
            )}
            {isLoading ? "Deploying..." : "Deploy changes"}
          </Button>
        </div>
      </CardContent>
    </Card>
  );
}
