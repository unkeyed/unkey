"use client";
import { useState } from "react";
import { GitMerge, ChevronRight } from "lucide-react";
import { Card, CardContent } from "@/components/ui/card";
import { Button } from "@/components/ui/button";
import { Badge } from "@/components/ui/badge";

interface MergeConfirmationCardProps {
  sourceBranch: string;
  targetBranch: string;
}

export default function MergeConfirmationCard({
  sourceBranch,
  targetBranch,
}: MergeConfirmationCardProps) {
  const [isLoading, setIsLoading] = useState(false);

  const handleConfirm = async () => {
    setIsLoading(true);
    // Simulating an API call
    await new Promise((resolve) => setTimeout(resolve, 1000));
    onConfirm();
    setIsLoading(false);
  };

  return (
    <Card className="w-full max-w-md mx-auto h-min">
      <CardContent className="pt-6">
        <div className="flex items-center justify-center mb-6">
          <Badge variant="secondary" className="text-sm font-medium">
            {sourceBranch}
          </Badge>
          <ChevronRight className="mx-2 h-4 w-4 text-gray-400" />
          <Badge variant="default" className="text-sm font-medium">
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
              <GitMerge className="mr-2 h-4 w-4 animate-spin" />
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
