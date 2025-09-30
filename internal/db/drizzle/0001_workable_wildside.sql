ALTER TABLE `permissions` DROP INDEX `unique_name_per_workspace_idx`;--> statement-breakpoint
ALTER TABLE `workspaces` ADD `slug` varchar(64);
ALTER TABLE `workspaces` ADD CONSTRAINT `workspaces_slug_unique` UNIQUE(`slug`);--> statement-breakpoint
