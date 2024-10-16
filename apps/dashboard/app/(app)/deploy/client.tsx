"use client";

import { useState } from "react";
import { Plus, ChevronRight, MoreHorizontal, ArrowUpCircle, Globe } from "lucide-react";
import { Button } from "@/components/ui/button";
import { Tooltip, TooltipContent, TooltipProvider, TooltipTrigger } from "@/components/ui/tooltip";
import { motion, AnimatePresence } from "framer-motion";
import Link from "next/link";
interface Branch {
  id: string;
  name: string;
  isDefault: boolean;
  domain: string;
  children?: Branch[];
}

export const BranchItem = ({ branch, isChild = false }: { branch: Branch; isChild?: boolean }) => {
  const [isOpen, setIsOpen] = useState(true);
  const hasChildren = branch.children && branch.children.length > 0;

  return (
    <motion.div
      initial={{ opacity: 0, y: -10 }}
      animate={{ opacity: 1, y: 0 }}
      exit={{ opacity: 0, y: -10 }}
      transition={{ duration: 0.2 }}
      className={`${isChild ? "ml-6" : ""}`}
    >
      <div className="relative grid grid-cols-[auto_1fr_auto_auto] items-center gap-4 py-3 hover:bg-gray-50 rounded-lg transition-colors duration-200">
        <div className="flex items-center space-x-2">
          {hasChildren && (
            <button
              onClick={() => setIsOpen(!isOpen)}
              className="text-gray-400 hover:text-gray-600 w-4 h-4 flex items-center justify-center"
              aria-label={isOpen ? "Collapse branch" : "Expand branch"}
            >
              <ChevronRight
                className={`h-4 w-4 transition-transform duration-200 ${isOpen ? "transform rotate-90" : ""}`}
              />
            </button>
          )}
          {!hasChildren && <div className="w-4" />}
          <div className={`size-1.5 rounded-full ${isChild ? "bg-gray-500" : "bg-blue-500"}`} />
          <p className="text-sm font-medium text-gray-800">{branch.name}</p>
          {branch.isDefault && (
            <span className="px-2 py-0.5 text-xs bg-blue-100 rounded-full text-blue-600">
              Production
            </span>
          )}
        </div>
        <div className="flex items-center space-x-1 text-xs text-gray-500">
          <Globe className="h-3 w-3 flex-shrink-0" />
          <span className="truncate">{branch.domain}</span>
        </div>
        <div>
          {isChild && (
            <TooltipProvider>
              <Tooltip>
                <TooltipTrigger asChild>
                  <Link href={`/deploy/promote/${branch.id}`}>
                    <Button
                      variant="ghost"
                      size="icon"
                      className="hover:bg-blue-50 hover:text-blue-600 transition-colors duration-200"
                    >
                      <ArrowUpCircle className="h-4 w-4" />
                      <span className="sr-only">Promote branch</span>
                    </Button>
                  </Link>
                </TooltipTrigger>
                <TooltipContent>
                  <p>Promote branch</p>
                </TooltipContent>
              </Tooltip>
            </TooltipProvider>
          )}
        </div>
        <div>
          <Button
            variant="ghost"
            size="icon"
            className="hover:bg-gray-100 transition-colors duration-200"
          >
            <MoreHorizontal className="h-4 w-4" />
            <span className="sr-only">More options</span>
          </Button>
        </div>
      </div>
      <AnimatePresence>
        {hasChildren && isOpen && (
          <motion.div
            initial={{ opacity: 0, height: 0 }}
            animate={{ opacity: 1, height: "auto" }}
            exit={{ opacity: 0, height: 0 }}
            transition={{ duration: 0.2 }}
            className="ml-6 mt-1 space-y-1"
          >
            {branch.children.map((child) => (
              <BranchItem key={child.name} branch={child} isChild={true} />
            ))}
          </motion.div>
        )}
      </AnimatePresence>
    </motion.div>
  );
};
