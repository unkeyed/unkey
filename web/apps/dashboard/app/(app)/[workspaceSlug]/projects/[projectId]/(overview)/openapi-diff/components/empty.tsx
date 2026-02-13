import { Magnifier } from "@unkey/icons";
import { Card } from "@unkey/ui";

export const DiffViewerEmpty = () => (
  <Card
    className={
      "rounded-[14px] flex justify-center items-center overflow-hidden border-gray-4 border-dashed bg-gray-1/50 min-h-[200px] relative group hover:border-gray-5 transition-colors duration-200"
    }
  >
    <div className="flex flex-col items-center gap-4 px-8 py-12 text-center">
      {/* Icon with subtle animation */}
      <div className="relative">
        <div className="absolute inset-0 bg-linear-to-r from-accent-4 to-accent-3 rounded-full blur-xl opacity-20 group-hover:opacity-30 transition-opacity duration-300 animate-pulse" />
        <div className="relative bg-gray-3 rounded-full p-3 group-hover:bg-gray-4 transition-all duration-200">
          <Magnifier
            className="text-gray-9 size-6 group-hover:text-gray-11 transition-all duration-200 animate-pulse"
            style={{ animationDuration: "2s" }}
          />
        </div>
      </div>
      {/* Content */}
      <div className="space-y-2">
        <h3 className="text-gray-12 font-medium text-sm">No deployments selected</h3>
        <p className="text-gray-9 text-xs max-w-[280px] leading-relaxed">
          Select two deployments above to compare their OpenAPI specifications and see what changed
          between versions.
        </p>
      </div>
    </div>
  </Card>
);
