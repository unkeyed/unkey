"use client";

/**
 * Role Recipes: curated permission presets with typed holes. The gallery
 * teaches the URN grammar by example; picking a recipe opens a dialog where
 * one Select per hole resolves the templates into real grants.
 */

import {
  Badge,
  type BadgeProps,
  Button,
  Empty,
  PageBody,
  PageContainer,
  PageHeader,
  PageHeaderContent,
  PageHeaderDescription,
  PageHeaderTitle,
  cn,
} from "@unkey/ui";
import { useState } from "react";
import { ApplyRecipeDialog } from "./apply-recipe-dialog";
import { RECIPES, type Recipe, type RecipeCategory, distinctHoles } from "./recipes";
import { HolePill, TemplateLine } from "./template-line";

const CATEGORY_BADGE: Record<RecipeCategory, BadgeProps["variant"]> = {
  Observability: "secondary",
  Operations: "primary",
  Deploy: "success",
  Admin: "warning",
};

export default function RoleRecipesPage() {
  const [activeRecipe, setActiveRecipe] = useState<Recipe | null>(null);

  return (
    <PageContainer width="full">
      <PageHeader>
        <PageHeaderContent>
          <PageHeaderTitle>Role Recipes</PageHeaderTitle>
          <PageHeaderDescription>
            Curated permission presets for common jobs. Placeholders like{" "}
            <HolePill kind="keyspace" /> are typed holes: pick a resource and the whole recipe
            resolves into real grants.
          </PageHeaderDescription>
        </PageHeaderContent>
      </PageHeader>
      <PageBody>
        {RECIPES.length === 0 ? (
          <Empty>
            <Empty.Icon />
            <Empty.Title>No recipes yet</Empty.Title>
            <Empty.Description>
              Recipes are curated permission presets. None are defined for this workspace.
            </Empty.Description>
          </Empty>
        ) : (
          <div className="grid grid-cols-1 gap-4 md:grid-cols-2 2xl:grid-cols-3">
            {RECIPES.map((recipe) => (
              <RecipeCard key={recipe.id} recipe={recipe} onUse={() => setActiveRecipe(recipe)} />
            ))}
          </div>
        )}
      </PageBody>
      {activeRecipe && (
        <ApplyRecipeDialog
          key={activeRecipe.id}
          recipe={activeRecipe}
          onClose={() => setActiveRecipe(null)}
        />
      )}
    </PageContainer>
  );
}

function RecipeCard({ recipe, onUse }: { recipe: Recipe; onUse: () => void }) {
  const holes = distinctHoles(recipe);
  const isCaution = recipe.caution !== undefined;

  return (
    <div
      className={cn(
        "flex flex-col gap-3 rounded-lg border p-5",
        isCaution ? "border-warningA-5 bg-warningA-2" : "border-grayA-4",
      )}
    >
      <div className="flex items-start justify-between gap-3">
        <span className={cn("font-medium", isCaution ? "text-warning-11" : "text-gray-12")}>
          {recipe.name}
        </span>
        <Badge variant={CATEGORY_BADGE[recipe.category]} className="shrink-0">
          {recipe.category}
        </Badge>
      </div>

      <p className="text-sm leading-5 text-gray-11">{recipe.tagline}</p>

      <div
        className={cn(
          "flex flex-col gap-1.5 overflow-x-auto rounded-md border p-3",
          isCaution ? "border-warningA-4" : "border-grayA-3 bg-grayA-2",
        )}
      >
        {recipe.templates.map((template, i) => (
          // biome-ignore lint/suspicious/noArrayIndexKey: templates are static per recipe
          <TemplateLine key={i} template={template} />
        ))}
      </div>

      {isCaution && (
        <p className="text-xs leading-5 text-warning-11">
          Grants every action on every resource, including future ones. Applying requires typed
          confirmation.
        </p>
      )}

      <div className="mt-auto flex items-center justify-between gap-3 pt-1">
        <span className="flex items-center gap-1.5 text-[11px] text-gray-9">
          {recipe.templates.length} grant{recipe.templates.length === 1 ? "" : "s"}
          {holes.length > 0 && (
            <>
              <span>&middot; fill in</span>
              {holes.map((kind) => (
                <HolePill key={kind} kind={kind} />
              ))}
            </>
          )}
        </span>
        <Button
          variant="outline"
          color={isCaution ? "warning" : "default"}
          onClick={onUse}
          className="shrink-0"
        >
          Use recipe
        </Button>
      </div>
    </div>
  );
}
