import { pgTable, uniqueIndex, text, integer, boolean, timestamp, json, bigint, index, unique, pgEnum } from "drizzle-orm/pg-core"

export const organizationRole = pgEnum("organization_role", ['OWNER', 'ADMIN', 'MEMBER'])
export const projectRole = pgEnum("project_role", ['ADMIN', 'WRITE', 'MEMBER', 'READ'])
export const teamRole = pgEnum("team_role", ['MAINTAINER', 'MEMBER'])


export const accounts = pgTable("accounts", {
	id: text().primaryKey().notNull(),
	userId: text("user_id").notNull(),
	type: text().notNull(),
	provider: text().notNull(),
	providerAccountId: text("provider_account_id").notNull(),
	refreshToken: text("refresh_token"),
	accessToken: text("access_token"),
	expiresAt: integer("expires_at"),
	tokenType: text("token_type"),
	scope: text(),
	idToken: text("id_token"),
	sessionState: text("session_state"),
}, (table) => [
	uniqueIndex("provider_account_idx").using("btree", table.provider.asc().nullsLast().op("text_ops"), table.providerAccountId.asc().nullsLast().op("text_ops")),
]);

export const branches = pgTable("branches", {
	id: text().primaryKey().notNull(),
	name: text(),
	projectId: text("project_id"),
	isDefault: boolean("is_default").default(false),
	isProtected: boolean("is_protected").default(false),
	lastCommit: text("last_commit"),
	createdAt: timestamp("created_at", { mode: 'string' }).defaultNow(),
	updatedAt: timestamp("updated_at", { mode: 'string' }).defaultNow(),
}, (table) => [
	uniqueIndex("project_branch_name_idx").using("btree", table.projectId.asc().nullsLast().op("text_ops"), table.name.asc().nullsLast().op("text_ops")),
]);

export const commitTrees = pgTable("commit_trees", {
	id: text().primaryKey().notNull(),
	projectId: text("project_id").notNull(),
	commitId: text("commit_id").notNull(),
	files: json(),
	createdAt: timestamp("created_at", { mode: 'string' }).defaultNow(),
});

export const commits = pgTable("commits", {
	id: text().primaryKey().notNull(),
	projectId: text("project_id").notNull(),
	authorId: text("author_id").notNull(),
	message: text().notNull(),
	treeHash: text("tree_hash"),
	parentIds: json("parent_ids"),
	createdAt: timestamp("created_at", { mode: 'string' }).defaultNow(),
});

export const fileVersions = pgTable("file_versions", {
	id: text().primaryKey().notNull(),
	projectId: text("project_id").notNull(),
	path: text().notNull(),
	contentHash: text("content_hash").notNull(),
	commitId: text("commit_id").notNull(),
	// You can use { mode: "bigint" } if numbers are exceeding js number limitations
	size: bigint({ mode: "number" }),
	mimeType: text("mime_type"),
	createdAt: timestamp("created_at", { mode: 'string' }).defaultNow(),
});

export const files = pgTable("files", {
	id: text().primaryKey().notNull(),
	projectId: text("project_id"),
	path: text(),
	contentHash: text("content_hash"),
	// You can use { mode: "bigint" } if numbers are exceeding js number limitations
	size: bigint({ mode: "number" }),
	mimeType: text("mime_type"),
	branch: text().default('main'),
	isLocked: boolean("is_locked").default(false),
	lockedBy: text("locked_by"),
	lockedAt: timestamp("locked_at", { mode: 'string' }),
	lastModifiedBy: text("last_modified_by"),
	lastModifiedAt: timestamp("last_modified_at", { mode: 'string' }).defaultNow(),
}, (table) => [
	index("content_hash_idx").using("btree", table.contentHash.asc().nullsLast().op("text_ops")),
	uniqueIndex("project_path_branch_idx").using("btree", table.projectId.asc().nullsLast().op("text_ops"), table.path.asc().nullsLast().op("text_ops"), table.branch.asc().nullsLast().op("text_ops")),
	index("project_path_idx").using("btree", table.projectId.asc().nullsLast().op("text_ops"), table.path.asc().nullsLast().op("text_ops")),
]);

export const organizationMembers = pgTable("organization_members", {
	id: text().primaryKey().notNull(),
	organizationId: text("organization_id").notNull(),
	userId: text("user_id").notNull(),
	role: organizationRole().default('MEMBER'),
	joinedAt: timestamp("joined_at", { mode: 'string' }).defaultNow(),
}, (table) => [
	uniqueIndex("org_user_idx").using("btree", table.organizationId.asc().nullsLast().op("text_ops"), table.userId.asc().nullsLast().op("text_ops")),
]);

export const organizations = pgTable("organizations", {
	id: text().primaryKey().notNull(),
	name: text(),
	slug: text(),
	description: text(),
	avatarUrl: text("avatar_url"),
	website: text(),
	location: text(),
	settings: json(),
	ownerId: text("owner_id"),
	createdAt: timestamp("created_at", { mode: 'string' }).defaultNow(),
	updatedAt: timestamp("updated_at", { mode: 'string' }).defaultNow(),
}, (table) => [
	uniqueIndex("idx_organizations_slug").using("btree", table.slug.asc().nullsLast().op("text_ops")),
	unique("organizations_slug_unique").on(table.slug),
]);

export const projectMembers = pgTable("project_members", {
	id: text().primaryKey().notNull(),
	projectId: text("project_id"),
	userId: text("user_id"),
	role: text().default('member'),
	joinedAt: timestamp("joined_at", { mode: 'string' }).defaultNow(),
}, (table) => [
	uniqueIndex("project_user_idx").using("btree", table.projectId.asc().nullsLast().op("text_ops"), table.userId.asc().nullsLast().op("text_ops")),
]);

export const projects = pgTable("projects", {
	id: text().primaryKey().notNull(),
	name: text(),
	slug: text(),
	description: text(),
	isPrivate: boolean("is_private").default(true),
	defaultBranch: text("default_branch").default('main'),
	settings: json(),
	ownerId: text("owner_id"),
	organizationId: text("organization_id"),
	createdAt: timestamp("created_at", { mode: 'string' }).defaultNow(),
	updatedAt: timestamp("updated_at", { mode: 'string' }).defaultNow(),
}, (table) => [
	uniqueIndex("org_project_slug_idx").using("btree", table.organizationId.asc().nullsLast().op("text_ops"), table.slug.asc().nullsLast().op("text_ops")),
	uniqueIndex("owner_project_slug_idx").using("btree", table.ownerId.asc().nullsLast().op("text_ops"), table.slug.asc().nullsLast().op("text_ops")),
]);

export const refs = pgTable("refs", {
	id: text().primaryKey().notNull(),
	projectId: text("project_id").notNull(),
	name: text().notNull(),
	type: text().notNull(),
	commitId: text("commit_id").notNull(),
	createdAt: timestamp("created_at", { mode: 'string' }).defaultNow(),
	updatedAt: timestamp("updated_at", { mode: 'string' }).defaultNow(),
});

export const sessions = pgTable("sessions", {
	sessionToken: text("session_token").primaryKey().notNull(),
	userId: text("user_id").notNull(),
	expires: timestamp({ mode: 'string' }).notNull(),
});

export const tags = pgTable("tags", {
	id: text().primaryKey().notNull(),
	projectId: text("project_id").notNull(),
	name: text().notNull(),
	message: text(),
	commitId: text("commit_id").notNull(),
	taggerId: text("tagger_id").notNull(),
	isAnnotated: boolean("is_annotated").default(false),
	createdAt: timestamp("created_at", { mode: 'string' }).defaultNow(),
});

export const teamMembers = pgTable("team_members", {
	teamId: text("team_id").notNull(),
	userId: text("user_id").notNull(),
	role: teamRole().default('MEMBER'),
	joinedAt: timestamp("joined_at", { mode: 'string' }).defaultNow(),
}, (table) => [
	uniqueIndex("team_user_idx").using("btree", table.teamId.asc().nullsLast().op("text_ops"), table.userId.asc().nullsLast().op("text_ops")),
]);

export const teams = pgTable("teams", {
	id: text().primaryKey().notNull(),
	name: text().notNull(),
	slug: text().notNull(),
	description: text(),
	organizationId: text("organization_id").notNull(),
	createdAt: timestamp("created_at", { mode: 'string' }).defaultNow(),
	updatedAt: timestamp("updated_at", { mode: 'string' }).defaultNow(),
}, (table) => [
	uniqueIndex("org_slug_idx").using("btree", table.organizationId.asc().nullsLast().op("text_ops"), table.slug.asc().nullsLast().op("text_ops")),
]);

export const users = pgTable("users", {
	id: text().primaryKey().notNull(),
	name: text(),
	email: text(),
	emailVerified: timestamp("email_verified", { mode: 'string' }),
	image: text(),
	username: text(),
	bio: text(),
	location: text(),
	website: text(),
	company: text(),
	createdAt: timestamp("created_at", { mode: 'string' }).defaultNow(),
	updatedAt: timestamp("updated_at", { mode: 'string' }).defaultNow(),
	avatarUrl: text("avatar_url"),
	settings: json(),
}, (table) => [
	uniqueIndex("idx_users_email").using("btree", table.email.asc().nullsLast().op("text_ops")),
	uniqueIndex("idx_users_username").using("btree", table.username.asc().nullsLast().op("text_ops")),
	unique("users_email_unique").on(table.email),
	unique("users_username_unique").on(table.username),
]);

export const verificationtokens = pgTable("verificationtokens", {
	identifier: text().notNull(),
	token: text().notNull(),
	expires: timestamp({ mode: 'string' }).notNull(),
}, (table) => [
	uniqueIndex("identifier_token_idx").using("btree", table.identifier.asc().nullsLast().op("text_ops"), table.token.asc().nullsLast().op("text_ops")),
]);
