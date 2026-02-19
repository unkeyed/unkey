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
import { cn } from "@/lib/utils";
import type { IconProps } from "@unkey/icons";
import { ChevronExpandY, Cube, Layers3 } from "@unkey/icons";
import { Tooltip, TooltipContent, TooltipProvider, TooltipTrigger } from "@unkey/ui";
import type React from "react";

interface ProductConfig {
  id: Product;
  name: string;
  description: string;
  icon: React.ComponentType<IconProps>;
  enabled: boolean;
}

const PRODUCTS: ProductConfig[] = [
  {
    id: "api-management",
    name: "API Management",
    description: "Manage APIs and keys",
    icon: Layers3,
    enabled: true,
  },
  {
    id: "deploy",
    name: "Deploy",
    description: "Deploy applications",
    icon: Cube,
    enabled: true, // TODO: Check feature flag
  },
];

export const ProductSwitcher: React.FC = () => {
  const { isMobile, state } = useSidebar();
  const { product, switchProduct } = useProductSelection();

  // Only collapsed in desktop mode, not in mobile mode
  const isCollapsed = state === "collapsed" && !isMobile;

  const currentProduct = PRODUCTS.find((p) => p.id === product);
  const availableProducts = PRODUCTS.filter((p) => p.enabled);

  if (!currentProduct) {
    return null;
  }

  const Icon = currentProduct.icon;

  const trigger = (
    <button
      type="button"
      className={cn(
        "flex items-center overflow-hidden rounded-lg border border-gray-7 bg-gray-4 hover:bg-gray-5 hover:cursor-pointer whitespace-nowrap ring-0 focus:ring-0 focus:outline-none text-content transition-colors",
        isCollapsed ? "justify-center w-10 h-10 p-0" : "justify-between w-full h-12 gap-3 px-3",
      )}
    >
      <div
        className={cn(
          "flex items-center gap-3 overflow-hidden whitespace-nowrap",
          isCollapsed ? "justify-center" : "",
        )}
      >
        <Icon className="w-5 h-5 shrink-0 text-gray-11" iconSize="xl-medium" />
        {!isCollapsed && (
          <div className="flex flex-col items-start overflow-hidden">
            <span className="text-sm font-semibold text-gray-12 overflow-hidden text-ellipsis">
              {currentProduct.name}
            </span>
            <span className="text-xs text-gray-11 overflow-hidden text-ellipsis">
              {currentProduct.description}
            </span>
          </div>
        )}
      </div>
      {!isCollapsed && (
        <ChevronExpandY className="w-5 h-5 shrink-0 [stroke-width:1px] text-gray-9" />
      )}
    </button>
  );

  if (isCollapsed) {
    return (
      <TooltipProvider>
        <Tooltip>
          <TooltipTrigger asChild>
            <DropdownMenu>
              <DropdownMenuTrigger asChild>{trigger}</DropdownMenuTrigger>
              <DropdownMenuContent
                className="w-72 bg-gray-1 dark:bg-black shadow-2xl border-gray-6 rounded-lg"
                align="start"
                side="right"
                sideOffset={8}
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
                            prod.id === product ? "font-semibold text-gray-12" : "text-gray-11",
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
            </DropdownMenu>
          </TooltipTrigger>
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
      <DropdownMenuContent
        className="w-72 bg-gray-1 dark:bg-black shadow-2xl border-gray-6 rounded-lg"
        align="start"
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
                    prod.id === product ? "font-semibold text-gray-12" : "text-gray-11",
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
    </DropdownMenu>
  );
};
