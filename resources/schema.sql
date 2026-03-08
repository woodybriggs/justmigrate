/* table: billing_plan_price */
CREATE TABLE `billing_plan_prices` (
    `id` text NOT NULL,
    `currency_code` varchar(4) NOT NULL,
    `price` integer NOT NULL,
    `start_date` integer,
    `end_date` integer,
    FOREIGN KEY (`id`) REFERENCES `billing_plan`(`id`) ON UPDATE no action ON DELETE no action,
    FOREIGN KEY (`currency_code`) REFERENCES `currencies`(`code`) ON UPDATE no action ON DELETE no action
);

/* table: billing_plan */
CREATE TABLE `billing_plan` (
    `id` text PRIMARY KEY NOT NULL,
    `name` text NOT NULL,
    `description` text NOT NULL,
    `created_at` integer,
    `modified_at` integer
);

/* table: currencies */
CREATE TABLE `currencies` (
    `code` text PRIMARY KEY NOT NULL,
    `decimal_places` integer NOT NULL,
    `name` text NOT NULL, 
    `type` text NOT NULL
);

/* table: exchange_rate_methods */
CREATE TABLE `exchange_rate_methods` (
    `code` text PRIMARY KEY NOT NULL,
    `name` text NOT NULL,
    `description` text NOT NULL,
    `created_at` integer,
    `updated_at` integer
);

/* table: exchange_rates */
CREATE TABLE `exchange_rates` (
    `base` text NOT NULL,
    `quote` text NOT NULL,
    `rate` real NOT NULL,
    `source_code` text NOT NULL,
    `method_code` text NOT NULL,
    `sourced_at` integer NOT NULL,
    PRIMARY KEY (`base`, `quote`, `sourced_at`),
    FOREIGN KEY (`base`) REFERENCES `currencies`(`code`) ON UPDATE no action ON DELETE no action,
    FOREIGN KEY (`quote`) REFERENCES `currencies`(`code`) ON UPDATE no action ON DELETE no action,
    FOREIGN KEY (`source_code`) REFERENCES `exchange_rate_sources`(`code`) ON UPDATE no action ON DELETE no action,
    FOREIGN KEY (`method_code`) REFERENCES `exchange_rate_methods`(`code`) ON UPDATE no action ON DELETE no action
);

/* table: exchange_rate_sources */
CREATE TABLE `exchange_rate_sources` (
    `code` text PRIMARY KEY NOT NULL,
    `name` text NOT NULL,
    `description` text NOT NULL,
    `created_at` integer,
    `updated_at` integer
);

/* table: exchange_rate_access */
CREATE TABLE `exchange_rate_access` (
    `id` text PRIMARY KEY NOT NULL,
    `source_code` text NOT NULL,
    `method_code` text NOT NULL,
    `billing_plan_id` text NOT NULL,
    FOREIGN KEY (`source_code`) REFERENCES `exchange_rate_sources`(`code`) ON UPDATE no action ON DELETE no action,
    FOREIGN KEY (`method_code`) REFERENCES `exchange_rate_methods`(`code`) ON UPDATE no action ON DELETE no action,
    FOREIGN KEY (`billing_plan_id`) REFERENCES `billing_plan`(`id`) ON UPDATE no action ON DELETE no action
);

/* table: parties */
CREATE TABLE `parties` (
    `id` text PRIMARY KEY NOT NULL,
    `name` text,
    `billing_plan_id` text NOT NULL,
    `billing_user_id` text NOT NULL,
    FOREIGN KEY (`billing_plan_id`) REFERENCES `billing_plan`(`id`) ON UPDATE no action ON DELETE no action,
    FOREIGN KEY (`billing_user_id`) REFERENCES `users`(`id`) ON UPDATE no action ON DELETE no action
);

/* table: party_users */
CREATE TABLE `party_users` (
    `party_id` text NOT NULL,
    `user_id` text NOT NULL,
    `role` text NOT NULL,
    FOREIGN KEY (`party_id`) REFERENCES `parties`(`id`) ON UPDATE no action ON DELETE no action,
    FOREIGN KEY (`user_id`) REFERENCES `users`(`id`) ON UPDATE no action ON DELETE no action
);

/* table: users */
CREATE TABLE `users` (
    `id` text PRIMARY KEY NOT NULL,
    `email` text NOT NULL,
    `name` text
);

/* index: billing_plan_price__idx__billing_plan_id_currency_code_start_date_end_date */
CREATE INDEX `billing_plan_price__idx__billing_plan_id_currency_code_start_date_end_date` ON `billing_plan_price` (`id`,`currency_code`,`start_date`,`end_date`);

/* index: exchange_rate_access__idx__billing_plan_id */
CREATE INDEX `exchange_rate_access__idx__billing_plan_id` ON `exchange_rate_access` (`billing_plan_id`);

/* index: exchange_rate_access__idx__source_method */
CREATE INDEX `exchange_rate_access__idx__source_method` ON `exchange_rate_access` (`source_code`,`method_code`);

/* index: exchange_rate_access_billing_plan_id_source_code_method_code_unique */
CREATE UNIQUE INDEX `exchange_rate_access_billing_plan_id_source_code_method_code_unique` ON `exchange_rate_access` (`billing_plan_id`,`source_code`,`method_code`);