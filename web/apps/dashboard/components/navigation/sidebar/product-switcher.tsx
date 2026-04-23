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
import { Check, ChevronExpandY, CloudUp, Nodes } from "@unkey/icons";
import { Tooltip, TooltipContent, TooltipProvider, TooltipTrigger } from "@unkey/ui";
import type React from "react";
import { useState } from "react";

interface ProductConfig {
  id: Product;
  name: string;
  description: string;
  icon: React.ComponentType<IconProps>;
}

interface ProductSwitcherProps {
  currentProduct: Product;
}

export const ProductSwitcher: React.FC<ProductSwitcherProps> = ({ currentProduct: product }) => {
  const { isMobile, state } = useSidebar();
  const { switchProduct } = useProductSelection();
  const [isDropdownOpen, setIsDropdownOpen] = useState(false);

  // Only collapsed in desktop mode, not in mobile mode
  const isCollapsed = state === "collapsed" && !isMobile;

  const products: ProductConfig[] = [
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

  const currentProduct = products.find((p) => p.id === product) || products[0];

  if (!currentProduct) {
    return null;
  }

  const Icon = currentProduct.icon;

  const trigger = (
    <button
      type="button"
      className={cn(
        "flex items-center rounded-md bg-background border-gray-6 border hover:bg-grayA-3 cursor-pointer ring-0 focus:ring-0 focus:outline-none text-content transition-colors",
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
        <Icon className="w-5 h-5 shrink-0 text-accent-12" iconSize="sm-medium" />
        {!isCollapsed && (
          <span className="text-[13px] font-medium text-accent-12 truncate w-full text-left">
            {currentProduct.name}
          </span>
        )}
      </div>
      {!isCollapsed && <ChevronExpandY className="w-4 h-4 shrink-0 stroke-[1px] text-gray-9" />}
    </button>
  );

  // Shared dropdown content
  const dropdownContent = (
    <DropdownMenuContent
      className="min-w-64"
      align="start"
      side={isCollapsed ? "right" : undefined}
      sideOffset={isCollapsed ? 8 : undefined}
    >
      {products.map((prod) => {
        const ProdIcon = prod.icon;
        const isSelected = prod.id === product;
        return (
          <DropdownMenuItem
            key={prod.id}
            className={cn(
              "flex items-center justify-between gap-2.5 p-2.5 cursor-pointer hover:bg-grayA-3",
              isSelected && "bg-grayA-2",
            )}
            onClick={() => {
              switchProduct(prod.id);
            }}
          >
            <div className="flex items-center gap-2.5">
              <ProdIcon
                className={cn("size-5 shrink-0", isSelected ? "text-accent-12" : "text-gray-11")}
                iconSize="sm-medium"
              />
              <div className="flex flex-col">
                <span
                  className={cn(
                    "text-[13px] font-medium",
                    isSelected ? "text-accent-12" : "text-gray-12",
                  )}
                >
                  {prod.name}
                </span>
                <span className="text-xs text-gray-11">{prod.description}</span>
              </div>
            </div>
            {isSelected && <Check className="w-4 h-4 text-accent-11" iconSize="sm-medium" />}
          </DropdownMenuItem>
        );
      })}
    </DropdownMenuContent>
  );

  if (isCollapsed) {
    return (
      <TooltipProvider>
        <Tooltip>
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
    <DropdownMenu open={isDropdownOpen} onOpenChange={setIsDropdownOpen}>
      <DropdownMenuTrigger asChild>{trigger}</DropdownMenuTrigger>
      {dropdownContent}
    </DropdownMenu>
  );
};
