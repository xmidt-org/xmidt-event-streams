// SPDX-FileCopyrightText: 2017 Comcast Cable Communications Management, LLC
// SPDX-License-Identifier: Apache-2.0

package install

import (
	"eventstream/internal/metrics"
	"fmt"
	"os"

	"github.com/alecthomas/kong"
	"github.com/goschtalt/goschtalt"
	"github.com/xmidt-org/arrange/arrangehttp"
	"github.com/xmidt-org/candlelight"
	"github.com/xmidt-org/sallust"
	"github.com/xmidt-org/touchstone"
	"github.com/xmidt-org/touchstone/touchhttp"
	"go.uber.org/fx"
	"go.uber.org/fx/fxevent"
	"go.uber.org/zap"

	_ "github.com/goschtalt/goschtalt/pkg/typical"
	_ "github.com/goschtalt/yaml-decoder"
	_ "github.com/goschtalt/yaml-encoder"
)

// These match what goreleaser provides.
var (
	commit  = "undefined"
	version = "undefined"
	date    = "undefined"
	builtBy = "undefined"
)

// EventStream is the main entry point for the program.  It is responsible for
// setting up the dependency injection framework and returning the app object.
func EventStream(args []string, run bool) error {
	app := fx.New(provideAppOptions(args))
	if err := app.Err(); err != nil {
		return err
	}

	if run {
		app.Run()
	}

	return nil
}

func provideAppOptions(args []string) fx.Option {
	var (
		gscfg *goschtalt.Config

		// Capture the dependency tree in case we need to debug something.
		g fx.DotGraph

		// Capture the command line arguments.
		cli *CLI
	)

	opts := fx.Options(
		fx.Supply(cliArgs(args)),
		fx.Populate(&g),
		fx.Populate(&gscfg),
		fx.Populate(&cli),

		fx.WithLogger(func(log *zap.Logger) fxevent.Logger {
			return &fxevent.ZapLogger{Logger: log}
		}),

		fx.Provide(
			provideCLI,
			provideLogger,
			provideConfig,
			provideAuth,

			goschtalt.UnmarshalFunc[sallust.Config]("logger", goschtalt.Optional()),
			goschtalt.UnmarshalFunc[HealthPath]("servers.health.path"),
			goschtalt.UnmarshalFunc[MetricsPath]("servers.metrics.path"),
			goschtalt.UnmarshalFunc[PprofPathPrefix]("servers.pprof.path"),
			goschtalt.UnmarshalFunc[Routes]("routes"),
			goschtalt.UnmarshalFunc[Auth]("auth"),
			goschtalt.UnmarshalFunc[candlelight.Config]("tracing"),
			goschtalt.UnmarshalFunc[touchstone.Config]("prometheus"),
			goschtalt.UnmarshalFunc[touchhttp.Config]("prometheus_handler"),
			goschtalt.UnmarshalFunc[RequestHandler]("request_handler"),
			goschtalt.UnmarshalFunc[FilterManager]("filter_manager"),
			fx.Annotated{
				Name:   "servers.health.config",
				Target: goschtalt.UnmarshalFunc[arrangehttp.ServerConfig]("servers.health.http"),
			},
			fx.Annotated{
				Name:   "servers.metrics.config",
				Target: goschtalt.UnmarshalFunc[arrangehttp.ServerConfig]("servers.metrics.http"),
			},
			fx.Annotated{
				Name:   "servers.pprof.config",
				Target: goschtalt.UnmarshalFunc[arrangehttp.ServerConfig]("servers.pprof.http"),
			},
			fx.Annotated{
				Name:   "servers.primary.config",
				Target: goschtalt.UnmarshalFunc[arrangehttp.ServerConfig]("servers.primary.http"),
			},
			fx.Annotated{
				Name:   "servers.alternate.config",
				Target: goschtalt.UnmarshalFunc[arrangehttp.ServerConfig]("servers.alternate.http"),
			},
		),

		provideCoreEndpoints(),
		provideMetricEndpoint(),
		provideHealthCheck(),
		providePprofEndpoint(),
		touchstone.Provide(),
		touchhttp.Provide(),
		metrics.Provide(),

		arrangehttp.ProvideServer("servers.health"),
		arrangehttp.ProvideServer("servers.metrics"),
		arrangehttp.ProvideServer("servers.pprof"),
		arrangehttp.ProvideServer("servers.primary"),
		arrangehttp.ProvideServer("servers.alternate"),

		FilterModule,
		QueueModule,
		SenderModule,
		HandlerModule,
		EventHandlerModule,
	)

	if cli != nil && cli.Graph != "" {
		_ = os.WriteFile(cli.Graph, []byte(g), 0600)
	}

	return opts
}

// Provides a named type so it's a bit easier to flow through & use in fx.
type cliArgs []string

// CLI is the structure that is used to capture the command line arguments.
type CLI struct {
	Dev     bool     `optional:"" short:"d" help:"Run in development mode."`
	Show    bool     `optional:"" short:"s" help:"Show the configuration and exit."`
	Default string   `optional:""           help:"Output the default configuration file as the specified file."`
	Graph   string   `optional:"" short:"g" help:"Output the dependency graph to the specified file."`
	Files   []string `optional:"" short:"f" help:"Specific configuration files or directories."`
}

// Handle the CLI processing and return the processed input.
func provideCLI(args cliArgs) (*CLI, error) {
	return provideCLIWithOpts(args, false)
}

func provideCLIWithOpts(args cliArgs, testOpts bool) (*CLI, error) {
	var cli CLI

	// Create a no-op option to satisfy the kong.New() call.
	var opt kong.Option = kong.OptionFunc(
		func(*kong.Kong) error {
			return nil
		},
	)

	if testOpts {
		opt = kong.Writers(nil, nil)
	}

	parser, err := kong.New(&cli,
		kong.Name(applicationName),
		kong.Description("The cpe agent for Xmidt service.\n"+
			fmt.Sprintf("\tVersion:  %s\n", version)+
			fmt.Sprintf("\tDate:     %s\n", date)+
			fmt.Sprintf("\tCommit:   %s\n", commit)+
			fmt.Sprintf("\tBuilt By: %s\n", builtBy)+
			"\n"+
			"Configuration files are read in the following order unless the "+
			"-f option is presented:\n"+
			"----------------------------------------------------\n"+
			"\t1.  $(pwd)/xmidt-agent.{properties|yaml|yml}\n"+
			"\t2.  $(pwd)/conf.d/*.{properties|yaml|yml}\n"+
			"\t3.  /etc/xmidt-agent/xmidt-agent.{properties|yaml|yml}\n"+
			"\t4.  /etc/xmidt-agent/xonfig.d/*.{properties|yaml|yml}\n"+
			"\nIf an exact file (1 or 3) is found, the search stops.  If "+
			"a conf.d directory is found, all files in that directory are "+
			"read in lexical order."+
			"\n\nWhen the -f is used, it adds to the list of files or directories "+
			"to read.  The -f option can be used multiple times."+
			"\n\nEnvironment variables are may be used in the configuration "+
			"files.  The environment variables are expanded in the "+
			"configuration files using the standard ${ VAR } syntax."+
			"\n\nIt is suggested to explore the configuration using the -s/--show "+
			"option to help ensure you understand how the client is configured."+
			"",
		),
		kong.UsageOnError(),
		opt,
	)
	if err != nil {
		return nil, err
	}

	if testOpts {
		parser.Exit = func(_ int) { panic("exit") }
	}

	_, err = parser.Parse(args)
	if err != nil {
		parser.FatalIfErrorf(err)
	}

	return &cli, nil
}

type LoggerIn struct {
	fx.In
	CLI *CLI
	Cfg sallust.Config
}

// Create the logger and configure it based on if the program is in
// debug mode or normal mode.
func provideLogger(in LoggerIn) (*zap.AtomicLevel, *zap.Logger, error) {
	if in.CLI.Dev {
		in.Cfg.EncoderConfig.EncodeLevel = "capitalColor"
		in.Cfg.EncoderConfig.EncodeTime = "RFC3339"
		in.Cfg.Level = "DEBUG"
		in.Cfg.Development = true
		in.Cfg.Encoding = "console"
		in.Cfg.OutputPaths = append(in.Cfg.OutputPaths, "stderr")
		in.Cfg.ErrorOutputPaths = append(in.Cfg.ErrorOutputPaths, "stderr")
	}

	zcfg, err := in.Cfg.NewZapConfig()
	if err != nil {
		return nil, nil, err
	}

	logger, err := in.Cfg.Build()

	return &zcfg.Level, logger, err
}
