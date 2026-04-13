CREATE TABLE `deployments` (
	`pk` bigint unsigned AUTO_INCREMENT NOT NULL,
	`id` varchar(128) NOT NULL,
	`k8s_name` varchar(255) NOT NULL,
	`workspace_id` varchar(256) NOT NULL,
	`project_id` varchar(256) NOT NULL,
	`environment_id` varchar(128) NOT NULL,
	`app_id` varchar(64) NOT NULL,
	`image` varchar(256),
	`build_id` varchar(128),
	`git_commit_sha` varchar(40),
	`git_branch` varchar(256),
	`git_commit_message` text,
	`git_commit_author_handle` varchar(256),
	`git_commit_author_avatar_url` varchar(512),
	`git_commit_timestamp` bigint,
	`sentinel_config` longblob NOT NULL,
	`cpu_millicores` int NOT NULL,
	`memory_mib` int NOT NULL,
	`storage_mib` int unsigned NOT NULL DEFAULT 0,
	`desired_state` enum('running','standby','archived') NOT NULL DEFAULT 'running',
	`encrypted_environment_variables` longblob NOT NULL,
	`command` json NOT NULL DEFAULT ('[]'),
	`port` int NOT NULL DEFAULT 8080,
	`shutdown_signal` enum('SIGTERM','SIGINT','SIGQUIT','SIGKILL') NOT NULL DEFAULT 'SIGTERM',
	`healthcheck` json,
	`pr_number` bigint,
	`fork_repository_full_name` varchar(256),
	`github_deployment_id` bigint,
	`status` enum('pending','starting','building','deploying','network','finalizing','ready','failed','skipped','awaiting_approval','stopped') NOT NULL DEFAULT 'pending',
	`created_at` bigint NOT NULL,
	`updated_at` bigint,
	CONSTRAINT `deployments_pk` PRIMARY KEY(`pk`),
	CONSTRAINT `deployments_id_unique` UNIQUE(`id`),
	CONSTRAINT `deployments_k8s_name_unique` UNIQUE(`k8s_name`),
	CONSTRAINT `deployments_build_id_unique` UNIQUE(`build_id`)
);

CREATE INDEX `workspace_idx` ON `deployments` (`workspace_id`);

CREATE INDEX `project_idx` ON `deployments` (`project_id`);

CREATE INDEX `status_idx` ON `deployments` (`status`);

