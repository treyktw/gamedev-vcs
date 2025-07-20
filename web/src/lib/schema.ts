import { pgTable, text, timestamp, boolean, integer, bigint, json, pgEnum, uniqueIndex, index, uuid } from 'drizzle-orm/pg-core';
import { relations } from 'drizzle-orm';

// Enums
export const organizationRoleEnum = pgEnum('organization_role', ['OWNER', 'ADMIN', 'MEMBER']);
export const teamRoleEnum = pgEnum('team_role', ['MAINTAINER', 'MEMBER']);
export const projectRoleEnum = pgEnum('project_role', ['ADMIN', 'WRITE', 'MEMBER', 'READ']);

// Users table
export const users = pgTable('users', {
  id: text('id').primaryKey(),
  name: text('name'),
  email: text('email').unique(),
  emailVerified: timestamp('email_verified'),
  image: text('image'),
  username: text('username').unique(),
  bio: text('bio'),
  location: text('location'),
  website: text('website'),
  company: text('company'),
  createdAt: timestamp('created_at').defaultNow(),
  updatedAt: timestamp('updated_at').defaultNow(),
  avatarUrl: text('avatar_url'),
  settings: json('settings'),
}, (table) => ({
  emailIdx: uniqueIndex('idx_users_email').on(table.email),
  usernameIdx: uniqueIndex('idx_users_username').on(table.username),
}));

// Accounts table (for OAuth) - NextAuth format
export const accounts = pgTable('accounts', {
  id: text('id').primaryKey().$defaultFn(() => crypto.randomUUID()),
  userId: text('user_id').notNull(),
  type: text('type').notNull(),
  provider: text('provider').notNull(),
  providerAccountId: text('provider_account_id').notNull(),
  refresh_token: text('refresh_token'),
  access_token: text('access_token'),
  expires_at: integer('expires_at'),
  token_type: text('token_type'),
  scope: text('scope'),
  id_token: text('id_token'),
  session_state: text('session_state'),
}, (table) => ({
  providerAccountIdx: uniqueIndex('provider_account_idx').on(table.provider, table.providerAccountId),
}));

// Sessions table - NextAuth format
export const sessions = pgTable('sessions', {
  sessionToken: text('session_token').primaryKey(),
  userId: text('user_id').notNull(),
  expires: timestamp('expires').notNull(),
});

// Verification tokens - NextAuth format
export const verificationTokens = pgTable('verificationtokens', {
  identifier: text('identifier').notNull(),
  token: text('token').notNull(),
  expires: timestamp('expires').notNull(),
}, (table) => ({
  identifierTokenIdx: uniqueIndex('identifier_token_idx').on(table.identifier, table.token),
}));

// Organizations table
export const organizations = pgTable('organizations', {
  id: text('id').primaryKey().$defaultFn(() => crypto.randomUUID()),
  name: text('name'),
  slug: text('slug').unique(),
  description: text('description'),
  avatarUrl: text('avatar_url'),
  website: text('website'),
  location: text('location'),
  settings: json('settings'),
  ownerId: text('owner_id'),
  createdAt: timestamp('created_at').defaultNow(),
  updatedAt: timestamp('updated_at').defaultNow(),
}, (table) => ({
  slugIdx: uniqueIndex('idx_organizations_slug').on(table.slug),
}));

// Organization members
export const organizationMembers = pgTable('organization_members', {
  id: text('id').primaryKey().$defaultFn(() => crypto.randomUUID()),
  organizationId: text('organization_id').notNull(),
  userId: text('user_id').notNull(),
  role: organizationRoleEnum('role').default('MEMBER'),
  joinedAt: timestamp('joined_at').defaultNow(),
}, (table) => ({
  orgUserIdx: uniqueIndex('org_user_idx').on(table.organizationId, table.userId),
}));

// Teams table
export const teams = pgTable('teams', {
  id: text('id').primaryKey().$defaultFn(() => crypto.randomUUID()),
  name: text('name').notNull(),
  slug: text('slug').notNull(),
  description: text('description'),
  organizationId: text('organization_id').notNull(),
  createdAt: timestamp('created_at').defaultNow(),
  updatedAt: timestamp('updated_at').defaultNow(),
}, (table) => ({
  orgSlugIdx: uniqueIndex('org_slug_idx').on(table.organizationId, table.slug),
}));

// Team members
export const teamMembers = pgTable('team_members', {
  teamId: text('team_id').notNull(),
  userId: text('user_id').notNull(),
  role: teamRoleEnum('role').default('MEMBER'),
  joinedAt: timestamp('joined_at').defaultNow(),
}, (table) => ({
  teamUserIdx: uniqueIndex('team_user_idx').on(table.teamId, table.userId),
}));

// Projects table
export const projects = pgTable('projects', {
  id: text('id').primaryKey().$defaultFn(() => crypto.randomUUID()),
  name: text('name'),
  slug: text('slug'),
  description: text('description'),
  isPrivate: boolean('is_private').default(true),
  defaultBranch: text('default_branch').default('main'),
  settings: json('settings'),
  ownerId: text('owner_id'),
  organizationId: text('organization_id'),
  createdAt: timestamp('created_at').defaultNow(),
  updatedAt: timestamp('updated_at').defaultNow(),
}, (table) => ({
  orgSlugIdx: uniqueIndex('org_project_slug_idx').on(table.organizationId, table.slug),
  ownerSlugIdx: uniqueIndex('owner_project_slug_idx').on(table.ownerId, table.slug),
}));

// Project members
export const projectMembers = pgTable('project_members', {
  id: text('id').primaryKey().$defaultFn(() => crypto.randomUUID()),
  projectId: text('project_id'),
  userId: text('user_id'),
  role: text('role').default('member'),
  joinedAt: timestamp('joined_at').defaultNow(),
}, (table) => ({
  projectUserIdx: uniqueIndex('project_user_idx').on(table.projectId, table.userId),
}));

// Branches table
export const branches = pgTable('branches', {
  id: text('id').primaryKey().$defaultFn(() => crypto.randomUUID()),
  name: text('name'),
  projectId: text('project_id'),
  isDefault: boolean('is_default').default(false),
  isProtected: boolean('is_protected').default(false),
  lastCommit: text('last_commit'),
  createdAt: timestamp('created_at').defaultNow(),
  updatedAt: timestamp('updated_at').defaultNow(),
}, (table) => ({
  projectNameIdx: uniqueIndex('project_branch_name_idx').on(table.projectId, table.name),
}));

// Files table
export const files = pgTable('files', {
  id: text('id').primaryKey().$defaultFn(() => crypto.randomUUID()),
  projectId: text('project_id'),
  path: text('path'),
  contentHash: text('content_hash'),
  size: bigint('size', { mode: 'number' }),
  mimeType: text('mime_type'),
  branch: text('branch').default('main'),
  isLocked: boolean('is_locked').default(false),
  lockedBy: text('locked_by'),
  lockedAt: timestamp('locked_at'),
  lastModifiedBy: text('last_modified_by'),
  lastModifiedAt: timestamp('last_modified_at').defaultNow(),
}, (table) => ({
  projectPathBranchIdx: uniqueIndex('project_path_branch_idx').on(table.projectId, table.path, table.branch),
  contentHashIdx: index('content_hash_idx').on(table.contentHash),
  projectPathIdx: index('project_path_idx').on(table.projectId, table.path),
}));

// Commits table
export const commits = pgTable('commits', {
  id: text('id').primaryKey().$defaultFn(() => crypto.randomUUID()),
  projectId: text('project_id').notNull(),
  authorId: text('author_id').notNull(),
  message: text('message').notNull(),
  treeHash: text('tree_hash'),
  parentIds: json('parent_ids'),
  createdAt: timestamp('created_at').defaultNow(),
});

// Commit trees
export const commitTrees = pgTable('commit_trees', {
  id: text('id').primaryKey().$defaultFn(() => crypto.randomUUID()),
  projectId: text('project_id').notNull(),
  commitId: text('commit_id').notNull(),
  files: json('files'),
  createdAt: timestamp('created_at').defaultNow(),
});

// Refs table
export const refs = pgTable('refs', {
  id: text('id').primaryKey().$defaultFn(() => crypto.randomUUID()),
  projectId: text('project_id').notNull(),
  name: text('name').notNull(),
  type: text('type').notNull(),
  commitId: text('commit_id').notNull(),
  createdAt: timestamp('created_at').defaultNow(),
  updatedAt: timestamp('updated_at').defaultNow(),
});

// Tags table
export const tags = pgTable('tags', {
  id: text('id').primaryKey().$defaultFn(() => crypto.randomUUID()),
  projectId: text('project_id').notNull(),
  name: text('name').notNull(),
  message: text('message'),
  commitId: text('commit_id').notNull(),
  taggerId: text('tagger_id').notNull(),
  isAnnotated: boolean('is_annotated').default(false),
  createdAt: timestamp('created_at').defaultNow(),
});

// File versions
export const fileVersions = pgTable('file_versions', {
  id: text('id').primaryKey().$defaultFn(() => crypto.randomUUID()),
  projectId: text('project_id').notNull(),
  path: text('path').notNull(),
  contentHash: text('content_hash').notNull(),
  commitId: text('commit_id').notNull(),
  size: bigint('size', { mode: 'number' }),
  mimeType: text('mime_type'),
  createdAt: timestamp('created_at').defaultNow(),
});

// Relations
export const usersRelations = relations(users, ({ many }) => ({
  accounts: many(accounts),
  sessions: many(sessions),
  ownedOrganizations: many(organizations, { relationName: 'organizationOwner' }),
  organizationMembers: many(organizationMembers),
  ownedProjects: many(projects, { relationName: 'projectOwner' }),
  projectMembers: many(projectMembers),
  teamMembers: many(teamMembers),
  modifiedFiles: many(files, { relationName: 'fileModifier' }),
  lockedFiles: many(files, { relationName: 'fileLocks' }),
}));

export const accountsRelations = relations(accounts, ({ one }) => ({
  user: one(users, {
    fields: [accounts.userId],
    references: [users.id],
  }),
}));

export const sessionsRelations = relations(sessions, ({ one }) => ({
  user: one(users, {
    fields: [sessions.userId],
    references: [users.id],
  }),
}));

export const organizationsRelations = relations(organizations, ({ one, many }) => ({
  owner: one(users, {
    fields: [organizations.ownerId],
    references: [users.id],
    relationName: 'organizationOwner',
  }),
  members: many(organizationMembers),
  projects: many(projects),
  teams: many(teams),
}));

export const organizationMembersRelations = relations(organizationMembers, ({ one }) => ({
  organization: one(organizations, {
    fields: [organizationMembers.organizationId],
    references: [organizations.id],
  }),
  user: one(users, {
    fields: [organizationMembers.userId],
    references: [users.id],
  }),
}));

export const teamsRelations = relations(teams, ({ one, many }) => ({
  organization: one(organizations, {
    fields: [teams.organizationId],
    references: [organizations.id],
  }),
  members: many(teamMembers),
}));

export const teamMembersRelations = relations(teamMembers, ({ one }) => ({
  team: one(teams, {
    fields: [teamMembers.teamId],
    references: [teams.id],
  }),
  user: one(users, {
    fields: [teamMembers.userId],
    references: [users.id],
  }),
}));

export const projectsRelations = relations(projects, ({ one, many }) => ({
  owner: one(users, {
    fields: [projects.ownerId],
    references: [users.id],
    relationName: 'projectOwner',
  }),
  organization: one(organizations, {
    fields: [projects.organizationId],
    references: [organizations.id],
  }),
  members: many(projectMembers),
  files: many(files),
  branches: many(branches),
}));

export const projectMembersRelations = relations(projectMembers, ({ one }) => ({
  project: one(projects, {
    fields: [projectMembers.projectId],
    references: [projects.id],
  }),
  user: one(users, {
    fields: [projectMembers.userId],
    references: [users.id],
  }),
}));

export const branchesRelations = relations(branches, ({ one }) => ({
  project: one(projects, {
    fields: [branches.projectId],
    references: [projects.id],
  }),
}));

export const filesRelations = relations(files, ({ one }) => ({
  project: one(projects, {
    fields: [files.projectId],
    references: [projects.id],
  }),
  lockUser: one(users, {
    fields: [files.lockedBy],
    references: [users.id],
    relationName: 'fileLocks',
  }),
  modifierUser: one(users, {
    fields: [files.lastModifiedBy],
    references: [users.id],
    relationName: 'fileModifier',
  }),
}));

export const commitsRelations = relations(commits, ({ one }) => ({
  project: one(projects, {
    fields: [commits.projectId],
    references: [projects.id],
  }),
  author: one(users, {
    fields: [commits.authorId],
    references: [users.id],
  }),
}));

export const commitTreesRelations = relations(commitTrees, ({ one }) => ({
  project: one(projects, {
    fields: [commitTrees.projectId],
    references: [projects.id],
  }),
}));

export const refsRelations = relations(refs, ({ one }) => ({
  project: one(projects, {
    fields: [refs.projectId],
    references: [projects.id],
  }),
}));

export const tagsRelations = relations(tags, ({ one }) => ({
  project: one(projects, {
    fields: [tags.projectId],
    references: [projects.id],
  }),
  tagger: one(users, {
    fields: [tags.taggerId],
    references: [users.id],
  }),
}));

export const fileVersionsRelations = relations(fileVersions, ({ one }) => ({
  project: one(projects, {
    fields: [fileVersions.projectId],
    references: [projects.id],
  }),
})); 