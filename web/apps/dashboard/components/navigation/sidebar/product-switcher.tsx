"use client";

import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuTrigger,
} from "@/components/ui/dropdown-menu";
import { useSidebar } from "@/components/ui/sidebar";
import type { Product } from "@/hooks/use-product-selection";
import { useProductSelection } from "@/hooks/use-product-selection";
import type { Workspace } from "@/lib/db";
import { cn } from "@/lib/utils";
import type { IconProps } from "@unkey/icons";
import { ChevronExpandY, CloudUp, Nodes } from "@unkey/icons";
import { Tooltip, TooltipContent, TooltipProvider, TooltipTrigger } from "@unkey/ui";
import type React from "react";
import { useState } from "react";

interface ProductConfig {
  id: Product;
  name: string;
  description: string;
  icon: React.ComponentType<IconProps>;
}

const PRODUCTS: ProductConfig[] = [
  {
    id: "api-management",
    name: "API Management",
    description: "Manage APIs and keys",
    icon: Nodes,
  },
  {
    id: "deploy",
    name: "Deploy",
    description: "Deploy applications",
    icon: CloudUp,
  },
];

interface ProductSwitcherProps {
  workspace: Workspace;
  currentProduct: Product;
}

export const ProductSwitcher: React.FC<ProductSwitcherProps> = ({
  workspace,
  currentProduct: product,
}) => {
  const { isMobile, state } = useSidebar();
  const { switchProduct } = useProductSelection();
  const [isDropdownOpen, setIsDropdownOpen] = useState(false);

  // Only collapsed in desktop mode, not in mobile mode
  const isCollapsed = state === "collapsed" && !isMobile;

  // Filter products based on workspace feature flags
  const availableProducts = PRODUCTS.filter((p) => {
    if (p.id === "deploy") {
      return workspace.betaFeatures?.deployments === true;
    }
    return true; // API Management is always available
  });

  const currentProduct = availableProducts.find((p) => p.id === product) || availableProducts[0];

  if (!currentProduct) {
    return null;
  }

  const Icon = currentProduct.icon;

  const trigger = (
    <button
      type="button"
      className={cn(
        "flex items-center rounded-lg bg-background border-gray-6 border hover:bg-background-subtle hover:cursor-pointer ring-0 focus:ring-0 focus:outline-none text-content transition-colors",
        isCollapsed
          ? "justify-center w-10 h-8 p-0"
          : "justify-between w-[200px] h-8 gap-2 px-2 flex-1",
      )}
    >
      <div
        className={cn(
          "flex items-center gap-2.5 min-w-0 flex-1",
          isCollapsed ? "justify-center" : "",
        )}
      >
        <Icon className="w-5 h-5 shrink-0 text-gray-11" />
        {!isCollapsed && (
          <span className="text-sm font-medium text-gray-12 truncate w-full text-left">
            {currentProduct.name}
          </span>
        )}
      </div>
      {!isCollapsed && (
        <ChevronExpandY className="w-4 h-4 shrink-0 [stroke-width:1px] text-gray-9" />
      )}
    </button>
  );

  // Shared dropdown content
  const dropdownContent = (
    <DropdownMenuContent
      className="w-72 bg-gray-1 dark:bg-black shadow-2xl border-gray-6 rounded-lg"
      align="start"
      side={isCollapsed ? "right" : undefined}
      sideOffset={isCollapsed ? 8 : undefined}
    >
      {availableProducts.map((prod) => {
        const ProdIcon = prod.icon;
        return (
          <DropdownMenuItem
            key={prod.id}
            className="flex items-start gap-3 p-3 cursor-pointer"
            onClick={() => {
              switchProduct(prod.id);
            }}
          >
            <ProdIcon className="w-5 h-5 shrink-0 text-gray-11 mt-0.5" />
            <div className="flex flex-col">
              <span
                className={cn(
                  "text-sm",
                  prod.id === product ? "font-medium text-gray-12" : "text-gray-11",
                )}
              >
                {prod.name}
              </span>
              <span className="text-xs text-gray-10">{prod.description}</span>
            </div>
          </DropdownMenuItem>
        );
      })}
    </DropdownMenuContent>
  );

  if (isCollapsed) {
    return (
      <TooltipProvider>
        <Tooltip open={!isDropdownOpen}>
          <DropdownMenu open={isDropdownOpen} onOpenChange={setIsDropdownOpen}>
            <TooltipTrigger asChild>
              <DropdownMenuTrigger asChild>{trigger}</DropdownMenuTrigger>
            </TooltipTrigger>
            {dropdownContent}
          </DropdownMenu>
          <TooltipContent side="right" sideOffset={8}>
            <p className="text-xs">{currentProduct.name}</p>
          </TooltipContent>
        </Tooltip>
      </TooltipProvider>
    );
  }

  return (
    <DropdownMenu>
      <DropdownMenuTrigger asChild>{trigger}</DropdownMenuTrigger>
      {dropdownContent}
    </DropdownMenu>
  );
};
