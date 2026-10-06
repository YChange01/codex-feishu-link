package control

const (
	FeishuCommandWorkspace            = "workspace"
	FeishuCommandWorkspaceList        = "workspace_list"
	FeishuCommandWorkspaceNew         = "workspace_new"
	FeishuCommandWorkspaceNewDir      = "workspace_new_dir"
	FeishuCommandWorkspaceNewGit      = "workspace_new_git"
	FeishuCommandWorkspaceNewWorktree = "workspace_new_worktree"
	FeishuCommandWorkspaceDetach      = "workspace_detach"
	FeishuCommandAdmin                = "admin"
	FeishuCommandAdminSubcommand      = "admin_subcommand"
	FeishuCommandList                 = "list"
	FeishuCommandStatus               = "status"
	FeishuCommandQuota                = "quota"
	FeishuCommandUse                  = "use"
	FeishuCommandUseAll               = "useall"
	FeishuCommandNew                  = "new"
	FeishuCommandHistory              = "history"
	FeishuCommandPrimary              = "primary"
	FeishuCommandCoworkers            = "coworkers"
	FeishuCommandReview               = "review"
	FeishuCommandSendFile             = "sendfile"
	FeishuCommandFollow               = "follow"
	FeishuCommandDetach               = "detach"
	FeishuCommandStop                 = "stop"
	FeishuCommandCompact              = "compact"
	FeishuCommandSteerAll             = "steerall"
	FeishuCommandMode                 = "mode"
	FeishuCommandAutoWhip             = "autowhip"
	FeishuCommandAutoContinue         = "autocontinue"
	FeishuCommandModel                = "model"
	FeishuCommandReasoning            = "reasoning"
	FeishuCommandAccess               = "access"
	FeishuCommandPlan                 = "plan"
	FeishuCommandVerbose              = "verbose"
	FeishuCommandCodexProfile         = "codex_profile"
	FeishuCommandClaudeProfile        = "claude_profile"
	FeishuCommandOpenCodeProfile      = "opencode_profile"
	FeishuCommandGoal                 = "goal"
	FeishuCommandHelp                 = "help"
	FeishuCommandMenu                 = "menu"
	FeishuCommandDebug                = "debug"
	FeishuCommandCron                 = "cron"
	FeishuCommandMCPOAuth             = "mcp_oauth"
	FeishuCommandUpgrade              = "upgrade"
	FeishuCommandPatch                = "patch"
	FeishuCommandVSCodeMigrate        = "vscode_migrate"
)

type FeishuCommandOption struct {
	Value       string
	Label       string
	Description string
	CommandText string
	MenuKey     string
}

type FeishuCommandDefinition struct {
	ID               string
	GroupID          string
	Title            string
	CanonicalSlash   string
	CanonicalMenuKey string
	ArgumentKind     FeishuCommandArgumentKind
	ArgumentFormHint string
	ArgumentFormNote string
	ArgumentSubmit   string
	Description      string
	Examples         []string
	Options          []FeishuCommandOption
	ShowInHelp       bool
	ShowInMenu       bool
	RecommendedMenu  *FeishuRecommendedMenu
}

type feishuCommandMatch struct {
	alias  string
	action Action
}

type feishuCommandPrefixMatch struct {
	alias string
	kind  ActionKind
}

type feishuCommandDynamicMenuMatch struct {
	prefix        string
	kind          ActionKind
	parseArgument func(string) (string, bool)
}

type feishuCommandSpec struct {
	definition        FeishuCommandDefinition
	textExact         []feishuCommandMatch
	textPrefixes      []feishuCommandPrefixMatch
	menuExact         []feishuCommandMatch
	menuDynamic       []feishuCommandDynamicMenuMatch
	extraActionRoutes []feishuCommandActionRoute
}

type FeishuRecommendedMenu struct {
	Key         string
	Name        string
	Description string
}

var feishuCommandSpecs = composeFeishuCommandSpecs()

func composeFeishuCommandSpecs() []feishuCommandSpec {
	specs := make([]feishuCommandSpec, 0, len(feishuCommandSpecsExecution)+len(feishuCommandSpecsSettings)+len(feishuCommandSpecsWorkspace)+len(feishuCommandSpecsRuntime)+len(feishuCommandSpecsMaintenance))
	specs = append(specs, feishuCommandSpecsExecution...)
	specs = append(specs, feishuCommandSpecsSettings...)
	specs = append(specs, feishuCommandSpecsWorkspace...)
	specs = append(specs, feishuCommandSpecsRuntime...)
	specs = append(specs, feishuCommandSpecsMaintenance...)
	return specs
}
