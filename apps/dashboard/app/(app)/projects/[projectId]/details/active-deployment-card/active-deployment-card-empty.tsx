import { Cloud, CodeBranch, Plus } from "@unkey/icons";
import { Button, Card } from "@unkey/ui";
import { cn } from "@unkey/ui/src/lib/utils";

type Props = {
  onCreateDeployment?: () => void;
  className?: string;
};

export const ActiveDeploymentCardEmpty = ({ onCreateDeployment, className }: Props) => (
  <Card
    className={cn(
      "rounded-[14px] flex justify-center items-center overflow-hidden border-gray-4 border-dashed bg-gray-1/50",
      "min-h-[200px] relative group hover:border-gray-5 transition-colors duration-200",
      className,
    )}
  >
    <div className="flex flex-col items-center gap-4 px-8 py-12 text-center">
      {/* Icon with subtle animation */}
      <div className="relative">
        <div className="absolute inset-0 bg-gradient-to-r from-accent-4 to-accent-3 rounded-full blur-xl opacity-20 group-hover:opacity-30 transition-opacity duration-300 animate-pulse" />
        <div className="relative bg-gray-3 rounded-full p-3 group-hover:bg-gray-4 transition-all duration-200">
          <Cloud
            className="text-gray-9 size-6 group-hover:text-gray-11 transition-all duration-200 animate-pulse"
            style={{ animationDuration: "2s" }}
          />
        </div>
      </div>

      {/* Content */}
      <div className="space-y-2">
        <h3 className="text-gray-12 font-medium text-sm">No active deployments</h3>
        <p className="text-gray-9 text-xs max-w-[280px] leading-relaxed">
          Your deployments will appear here once you push code to your connected repository or
          trigger a manual deployment.
        </p>
      </div>

      {/* Action button */}
      {onCreateDeployment && (
        <Button onClick={onCreateDeployment} size="sm" className="mt-2 group/button">
          <Plus className="size-4 mr-2 group-hover/button:rotate-90 transition-transform duration-200" />
          Create deployment
        </Button>
      )}

      {/* Subtle decorative elements */}
      <div className="absolute top-4 left-4 opacity-20 animate-bounce">
        <CodeBranch className="text-gray-7 size-3" />
      </div>
      <div className="absolute bottom-4 right-4 opacity-80">
        <div className="flex gap-1">
          <div className="w-1 h-1 bg-gray-6 rounded-full animate-pulse" />
          <div className="w-1 h-1 bg-gray-5 rounded-full animate-pulse animation-delay-200" />
          <div className="w-1 h-1 bg-gray-4 rounded-full animate-pulse animation-delay-400" />
        </div>
      </div>
    </div>
  </Card>
);
