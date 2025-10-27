package commands

import (
	"os"

	"github.com/spf13/cobra"
)

// NewCompletionCommand creates the completion command for shell completions
func NewCompletionCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "completion [bash|zsh|fish|powershell]",
		Short: "Generate shell completion script",
		Long: `Generate shell completion script for Conduit CLI.

To load completions:

Bash:

  $ source <(conduit completion bash)

  # To load completions for each session, execute once:
  # Linux:
  $ conduit completion bash > /etc/bash_completion.d/conduit
  # macOS:
  $ conduit completion bash > $(brew --prefix)/etc/bash_completion.d/conduit

Zsh:

  # If shell completion is not already enabled in your environment,
  # you will need to enable it. You can execute the following once:

  $ echo "autoload -U compinit; compinit" >> ~/.zshrc

  # To load completions for each session, execute once:
  $ conduit completion zsh > "${fpath[1]}/_conduit"

  # You will need to start a new shell for this setup to take effect.

Fish:

  $ conduit completion fish | source

  # To load completions for each session, execute once:
  $ conduit completion fish > ~/.config/fish/completions/conduit.fish

PowerShell:

  PS> conduit completion powershell | Out-String | Invoke-Expression

  # To load completions for every new session, run:
  PS> conduit completion powershell > conduit.ps1
  # and source this file from your PowerShell profile.
`,
		DisableFlagsInUseLine: true,
		ValidArgs:             []string{"bash", "zsh", "fish", "powershell"},
		Args:                  cobra.MatchAll(cobra.ExactArgs(1), cobra.OnlyValidArgs),
		RunE: func(cmd *cobra.Command, args []string) error {
			shell := args[0]
			root := cmd.Root()

			switch shell {
			case "bash":
				return root.GenBashCompletion(os.Stdout)
			case "zsh":
				return root.GenZshCompletion(os.Stdout)
			case "fish":
				return root.GenFishCompletion(os.Stdout, true)
			case "powershell":
				return root.GenPowerShellCompletionWithDesc(os.Stdout)
			}
			return nil
		},
	}

	return cmd
}
