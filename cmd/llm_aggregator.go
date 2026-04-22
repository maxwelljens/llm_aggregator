package main

import (
	"context"
	"fmt"
	"os"

	tea "github.com/charmbracelet/bubbletea"
	"llm_aggregator/internal/cli"
	"llm_aggregator/internal/config"
	"llm_aggregator/internal/progress"
	"llm_aggregator/internal/runtime"
	"llm_aggregator/internal/tui"
)

var (
	version   string
	buildDate string
)

func main() {
	cli.BuildDate = buildDate
	cli.Version = version

	// Parse command line arguments
	args, err := cli.ParseArgs()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error parsing arguments: %v\n", err)
		os.Exit(1)
	}

	// Get viper instance with config file + environment variables + defaults
	v := config.GetViper()

	// Bind CLI args to viper (highest precedence)
	config.BindCLIArgs(v, args.ToViperMap())

	// Validate API key
	if v.GetString("api_key") == "" {
		fmt.Fprintln(os.Stderr, "Error: OpenAI-compatible API key is required. Set via --api-key, LLM_AGGREGATOR_API_KEY environment variable, or config file.")
		os.Exit(1)
	}

	// Create runtime directly from viper configuration
	rt := config.ViperToRuntime(v, args.FeedsFile, args.Prompt)

	// Run with TUI if requested
	if args.TUI {
		runWithTUI(rt)
	} else {
		runWithoutTUI(rt, args.Verbose)
	}
}

func runWithTUI(rt *runtime.Runtime) {
	// 1. Create the model and pass the runtime to it.
	model := tui.New(rt)

	// 2. Create the tea.Program.
	p := tea.NewProgram(model, tea.WithAltScreen())

	// 3. Create the TUIProgress handler and give it the program instance.
	//    This is the bridge that allows the runtime to send messages to the TUI.
	tp := tui.NewTUIProgress(p)

	// 4. Inject the progress handler into the runtime.
	rt.Progress = tp

	// 5. Run the program. This is a blocking call.
	if _, err := p.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "TUI Error: %v\n", err)
		os.Exit(1)
	}
}

func runWithoutTUI(rt *runtime.Runtime, verbose bool) {
	// Setup logger based on verbose flag and inject it into the runtime
	if verbose {
		rt.Progress = progress.NewSimpleLogger(os.Stdout, true)
	} else {
		// Default is already NoopLogger, but we can be explicit
		rt.Progress = &progress.NoopLogger{}
	}

	// Execute the runtime
	err := rt.Execute(context.Background())
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	// Write output
	if rt.OutputFile != "" {
		if err := rt.WriteOutputToFile(); err != nil {
			fmt.Fprintf(os.Stderr, "Error writing output: %v\n", err)
			os.Exit(1)
		}
	} else {
		if err := rt.WriteOutput(os.Stdout); err != nil {
			fmt.Fprintf(os.Stderr, "Error writing output: %v\n", err)
			os.Exit(1)
		}
	}
}
