# Next.js VCS Dashboard File Structure

```
gamedev-vcs-dashboard/
├── src/
│   ├── app/
│   │   ├── globals.css
│   │   ├── layout.tsx                    # Root layout with AuthJS provider
│   │   ├── page.tsx                      # Landing/dashboard overview
│   │   ├── loading.tsx                   # Global loading UI
│   │   ├── error.tsx                     # Global error boundary
│   │   │
│   │   ├── auth/
│   │   │   ├── signin/
│   │   │   │   └── page.tsx              # Sign in page
│   │   │   ├── signup/
│   │   │   │   └── page.tsx              # Sign up page
│   │   │   └── callback/
│   │   │       └── page.tsx              # OAuth callback
│   │   │
│   │   ├── dashboard/
│   │   │   ├── page.tsx                  # Main dashboard (recent activity, projects)
│   │   │   ├── layout.tsx                # Dashboard layout with sidebar
│   │   │   ├── overview/
│   │   │   │   └── page.tsx              # Activity overview, stats
│   │   │   ├── repositories/
│   │   │   │   ├── page.tsx              # All repositories list
│   │   │   │   └── new/
│   │   │   │       └── page.tsx          # Create new repository
│   │   │   └── settings/
│   │   │       ├── page.tsx              # User settings
│   │   │       ├── profile/
│   │   │       │   └── page.tsx          # Profile management
│   │   │       ├── security/
│   │   │       │   └── page.tsx          # Security settings
│   │   │       └── billing/
│   │   │           └── page.tsx          # Billing & usage
│   │   │
│   │   ├── orgs/
│   │   │   ├── page.tsx                  # Organizations list
│   │   │   ├── new/
│   │   │   │   └── page.tsx              # Create organization
│   │   │   └── [orgSlug]/
│   │   │       ├── page.tsx              # Organization overview
│   │   │       ├── layout.tsx            # Org layout with tabs
│   │   │       ├── repositories/
│   │   │       │   ├── page.tsx          # Org repositories
│   │   │       │   └── new/
│   │   │       │       └── page.tsx      # Create org repository
│   │   │       ├── teams/
│   │   │       │   ├── page.tsx          # Teams list
│   │   │       │   ├── new/
│   │   │       │   │   └── page.tsx      # Create team
│   │   │       │   └── [teamSlug]/
│   │   │       │       ├── page.tsx      # Team overview
│   │   │       │       ├── members/
│   │   │       │       │   └── page.tsx  # Team members
│   │   │       │       ├── repositories/
│   │   │       │       │   └── page.tsx  # Team repositories
│   │   │       │       └── settings/
│   │   │       │           └── page.tsx  # Team settings
│   │   │       ├── members/
│   │   │       │   ├── page.tsx          # Organization members
│   │   │       │   └── invitations/
│   │   │       │       └── page.tsx      # Pending invitations
│   │   │       ├── analytics/
│   │   │       │   ├── page.tsx          # Org analytics dashboard
│   │   │       │   ├── productivity/
│   │   │       │   │   └── page.tsx      # Team productivity metrics
│   │   │       │   ├── assets/
│   │   │       │   │   └── page.tsx      # Asset usage analytics
│   │   │       │   └── storage/
│   │   │       │       └── page.tsx      # Storage analytics
│   │   │       └── settings/
│   │   │           ├── page.tsx          # General org settings
│   │   │           ├── billing/
│   │   │           │   └── page.tsx      # Organization billing
│   │   │           ├── security/
│   │   │           │   └── page.tsx      # Security policies
│   │   │           └── webhooks/
│   │   │               └── page.tsx      # Webhook configurations
│   │   │
│   │   ├── [owner]/                      # Dynamic owner (user or org)
│   │   │   └── [repo]/
│   │   │       ├── page.tsx              # Repository home (README, recent activity)
│   │   │       ├── layout.tsx            # Repository layout with tabs
│   │   │       ├── loading.tsx           # Repository loading state
│   │   │       ├── error.tsx             # Repository error boundary
│   │   │       │
│   │   │       ├── tree/
│   │   │       │   ├── page.tsx          # File browser root
│   │   │       │   └── [...path]/
│   │   │       │       └── page.tsx      # File/directory viewer
│   │   │       │
│   │   │       ├── blob/
│   │   │       │   └── [...path]/
│   │   │       │       └── page.tsx      # File content viewer/editor
│   │   │       │
│   │   │       ├── commits/
│   │   │       │   ├── page.tsx          # Commit history
│   │   │       │   └── [commitHash]/
│   │   │       │       └── page.tsx      # Individual commit view
│   │   │       │
│   │   │       ├── branches/
│   │   │       │   ├── page.tsx          # Branches list
│   │   │       │   └── [branchName]/
│   │   │       │       └── page.tsx      # Branch details
│   │   │       │
│   │   │       ├── pulls/                # Future: Pull requests
│   │   │       │   ├── page.tsx          # PR list
│   │   │       │   ├── new/
│   │   │       │   │   └── page.tsx      # Create PR
│   │   │       │   └── [prNumber]/
│   │   │       │       └── page.tsx      # PR details
│   │   │       │
│   │   │       ├── issues/               # Future: Issue tracking
│   │   │       │   ├── page.tsx          # Issues list
│   │   │       │   ├── new/
│   │   │       │   │   └── page.tsx      # Create issue
│   │   │       │   └── [issueNumber]/
│   │   │       │       └── page.tsx      # Issue details
│   │   │       │
│   │   │       ├── analytics/
│   │   │       │   ├── page.tsx          # Repository analytics
│   │   │       │   ├── dependencies/
│   │   │       │   │   └── page.tsx      # Asset dependency graph
│   │   │       │   ├── productivity/
│   │   │       │   │   └── page.tsx      # Team productivity on repo
│   │   │       │   └── insights/
│   │   │       │       └── page.tsx      # Code insights, hot files
│   │   │       │
│   │   │       ├── collaboration/
│   │   │       │   ├── page.tsx          # Live collaboration view
│   │   │       │   ├── presence/
│   │   │       │   │   └── page.tsx      # Team presence
│   │   │       │   └── locks/
│   │   │       │       └── page.tsx      # File locks management
│   │   │       │
│   │   │       └── settings/
│   │   │           ├── page.tsx          # Repository settings
│   │   │           ├── access/
│   │   │           │   └── page.tsx      # Access management
│   │   │           ├── branches/
│   │   │           │   └── page.tsx      # Branch protection
│   │   │           ├── webhooks/
│   │   │           │   └── page.tsx      # Repository webhooks
│   │   │           └── danger/
│   │   │               └── page.tsx      # Delete repository
│   │   │
│   │   └── api/
│   │       ├── auth/
│   │       │   └── [...nextauth]/
│   │       │       └── route.ts          # NextAuth.js API routes
│   │       ├── trpc/
│   │       │   └── [trpc]/
│   │       │       └── route.ts          # tRPC API handler
│   │       └── webhooks/
│   │           └── route.ts              # Webhook endpoints
│   │
│   ├── components/
│   │   ├── ui/                           # shadcn/ui components
│   │   │   ├── button.tsx
│   │   │   ├── input.tsx
│   │   │   ├── card.tsx
│   │   │   ├── avatar.tsx
│   │   │   ├── badge.tsx
│   │   │   ├── dialog.tsx
│   │   │   ├── dropdown-menu.tsx
│   │   │   ├── sheet.tsx
│   │   │   ├── tabs.tsx
│   │   │   ├── table.tsx
│   │   │   ├── scroll-area.tsx
│   │   │   ├── skeleton.tsx
│   │   │   └── toast.tsx
│   │   │
│   │   ├── layout/
│   │   │   ├── header.tsx                # Top navigation
│   │   │   ├── sidebar.tsx               # Dashboard sidebar
│   │   │   ├── footer.tsx                # Footer component
│   │   │   └── breadcrumbs.tsx           # Breadcrumb navigation
│   │   │
│   │   ├── auth/
│   │   │   ├── login-form.tsx            # Login form component
│   │   │   ├── signup-form.tsx           # Signup form component
│   │   │   ├── provider-buttons.tsx      # OAuth provider buttons
│   │   │   └── auth-guard.tsx            # Route protection wrapper
│   │   │
│   │   ├── dashboard/
│   │   │   ├── activity-feed.tsx         # Recent activity component
│   │   │   ├── repository-card.tsx       # Repository preview card
│   │   │   ├── stats-overview.tsx        # Dashboard stats
│   │   │   └── quick-actions.tsx         # Quick action buttons
│   │   │
│   │   ├── repository/
│   │   │   ├── file-tree.tsx             # File browser tree
│   │   │   ├── file-viewer.tsx           # File content display
│   │   │   ├── code-editor.tsx           # In-browser code editor
│   │   │   ├── commit-history.tsx        # Commit timeline
│   │   │   ├── branch-selector.tsx       # Branch dropdown
│   │   │   ├── file-upload.tsx           # Drag & drop file upload
│   │   │   ├── asset-preview.tsx         # UE5 asset preview
│   │   │   └── dependency-graph.tsx      # Asset dependency visualization
│   │   │
│   │   ├── collaboration/
│   │   │   ├── team-presence.tsx         # Live user presence
│   │   │   ├── file-locks.tsx            # Lock status indicators
│   │   │   ├── live-cursors.tsx          # Real-time cursor positions
│   │   │   └── activity-timeline.tsx     # Live activity feed
│   │   │
│   │   ├── analytics/
│   │   │   ├── productivity-chart.tsx    # Team productivity graphs
│   │   │   ├── storage-usage.tsx         # Storage utilization
│   │   │   ├── asset-heatmap.tsx         # Most changed files
│   │   │   └── dependency-tree.tsx       # Interactive dependency tree
│   │   │
│   │   ├── settings/
│   │   │   ├── profile-form.tsx          # User profile editor
│   │   │   ├── security-settings.tsx     # Security configuration
│   │   │   ├── team-management.tsx       # Team member management
│   │   │   └── webhook-form.tsx          # Webhook configuration
│   │   │
│   │   └── shared/
│   │       ├── loading-spinner.tsx       # Loading states
│   │       ├── error-boundary.tsx        # Error handling
│   │       ├── empty-state.tsx           # Empty state illustrations
│   │       ├── search-command.tsx        # Command palette
│   │       ├── theme-toggle.tsx          # Dark/light mode toggle
│   │       └── tooltip.tsx               # Hover tooltips
│   │
│   ├── lib/
│   │   ├── auth.ts                       # NextAuth.js configuration
│   │   ├── trpc.ts                       # tRPC client setup
│   │   ├── utils.ts                      # Utility functions
│   │   ├── validations.ts                # Zod schemas
│   │   ├── constants.ts                  # App constants
│   │   ├── db.ts                         # Database connection
│   │   └── websocket.ts                  # WebSocket client
│   │
│   ├── hooks/
│   │   ├── use-auth.ts                   # Authentication hook
│   │   ├── use-websocket.ts              # WebSocket hook
│   │   ├── use-file-upload.ts            # File upload hook
│   │   ├── use-presence.ts               # Real-time presence
│   │   ├── use-local-storage.ts          # Local storage hook
│   │   └── use-debounce.ts               # Debounced values
│   │
│   ├── stores/
│   │   ├── auth.ts                       # Auth state (Zustand)
│   │   ├── ui.ts                         # UI state
│   │   ├── collaboration.ts              # Collaboration state
│   │   └── settings.ts                   # User preferences
│   │
│   ├── types/
│   │   ├── auth.ts                       # Authentication types
│   │   ├── api.ts                        # API response types
│   │   ├── repository.ts                 # Repository types
│   │   ├── organization.ts               # Organization types
│   │   ├── team.ts                       # Team types
│   │   └── analytics.ts                  # Analytics types
│   │
│   └── styles/
│       ├── globals.css                   # Global styles
│       ├── components.css                # Component-specific styles
│       └── animations.css                # Magic UI animations
│
├── public/
│   ├── icons/                            # App icons
│   ├── images/                           # Static images
│   └── logos/                            # Brand assets
│
├── .env.local                            # Environment variables
├── .env.example                          # Environment template
├── next.config.js                        # Next.js configuration
├── tailwind.config.js                    # Tailwind CSS config
├── tsconfig.json                         # TypeScript configuration
├── components.json                       # shadcn/ui configuration
└── package.json                          # Dependencies
```

## Key Design Decisions

### **1. Hierarchical Structure**
- `[owner]/[repo]` supports both users and orgs seamlessly
- Clear separation between personal, team, and org spaces
- Consistent pattern for settings at each level

### **2. GitHub-Style Navigation**
- Repository tabs (Code, Issues, Pull Requests, Analytics)
- Organization tabs (Repositories, Teams, Analytics, Settings)
- Breadcrumb navigation for deep file paths

### **3. Real-time Features**
- WebSocket integration at component level
- Live presence indicators throughout
- Optimistic UI updates with TanStack Query

### **4. Analytics Integration**
- Dedicated analytics pages at repo and org level
- ClickHouse data visualization components
- Team productivity and asset insights

### **5. VCS-Specific Features**
- File locking UI components
- Asset dependency visualization
- UE5-specific file previews
- Collaboration tools

This structure gives you the GitHub-like experience but with game development superpowers!