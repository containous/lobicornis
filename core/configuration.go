package core

// Configuration task configuration.
type Configuration struct {
	Owner              string        `short:"o" description:"Repository owner. [required]"`
	RepositoryName     string        `long:"repo-name" short:"r" description:"Repository name. [required]"`
	GitHubToken        string        `long:"token" short:"t" description:"GitHub Token. [required]"`
	MinReview          int           `long:"min-review" description:"Minimal number of review."`
	DryRun             bool          `long:"dry-run" description:"Dry run mode."`
	Debug              bool          `long:"debug" description:"Debug mode."`
	SSH                bool          `description:"Use SSH instead HTTPS."`
	DefaultMergeMethod string        `long:"merge-method" description:"Default merge method.(merge|squash|rebase)"`
	MergeMethodPrefix  string        `long:"merge-method-prefix" description:"Use to override default merge method for a PR."`
	LabelMarkers       *LabelMarkers `long:"marker" description:"GitHub Labels."`
}

// LabelMarkers Labels use to control actions.
type LabelMarkers struct {
	NeedHumanMerge  string `long:"need-human-merge" description:"Label use when the bot cannot perform a merge."`
	NeedMerge       string `long:"need-merge" description:"Label use when you want the bot perform a merge."`
	MergeInProgress string `long:"merge-in-progress" description:"Label use when the bot update the PR (merge/rebase)."`
}
