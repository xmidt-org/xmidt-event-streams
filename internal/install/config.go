// SPDX-FileCopyrightText: 2017 Comcast Cable Communications Management, LLC
// SPDX-License-Identifier: Apache-2.0

package install

import (
	"fmt"
	"os"
	"time"

	"github.com/xmidt-org/xmidt-event-streams/filter"
	"github.com/xmidt-org/xmidt-event-streams/internal/auth"
	"github.com/xmidt-org/xmidt-event-streams/sender"

	"github.com/goschtalt/goschtalt"
	"github.com/xmidt-org/arrange/arrangehttp"
	"github.com/xmidt-org/arrange/arrangepprof"
	"github.com/xmidt-org/candlelight"
	"github.com/xmidt-org/sallust"
	"github.com/xmidt-org/touchstone"
	"github.com/xmidt-org/touchstone/touchhttp"
	"gopkg.in/dealancer/validate.v2"
)

const applicationName = "eventStream"

// Config is the top level configuration for the notus service.  Everything
// is contained in this structure or it will intentially cause a failure.
type Config struct {
	Logging           sallust.Config
	Tracing           candlelight.Config
	Prometheus        touchstone.Config
	PrometheusHandler touchhttp.Config
	Servers           Servers
	Routes            Routes
	RequestHandler    RequestHandler
	Auth              Auth
	FilterManager     filter.FilterManagerConfig
}

type Servers struct {
	Health    HealthServer
	Metrics   MetricsServer
	Pprof     PprofServer
	Primary   PrimaryServer
	Alternate PrimaryServer
}

type HealthServer struct {
	HTTP arrangehttp.ServerConfig
	Path HealthPath `validate:"empty=false"`
}

type HealthPath string

type MetricsServer struct {
	HTTP arrangehttp.ServerConfig
	Path MetricsPath `validate:"empty=false"`
}

type MetricsPath string

type PrimaryServer struct {
	HTTP arrangehttp.ServerConfig
}

type PprofServer struct {
	HTTP arrangehttp.ServerConfig
	Path PprofPathPrefix
}

type PprofPathPrefix string

type Routes struct {
	Events Route
}

type Route struct {
	Path   string `validate:"empty=false"`
	Server string `validate:"one_of=primary,alternate"`
}

type Auth struct {
	IsBasic bool
	Basic   auth.Basic
	Jwt     auth.Jwt
}

type RequestHandler struct {
	MaxOutstanding int32
}

type Filters struct {
	FilterConfig []filter.FilterConfig
}

// Collect and process the configuration files and env vars and
// produce a configuration object.
func provideConfig(cli *CLI) (*goschtalt.Config, error) {
	gs, err := goschtalt.New(
		goschtalt.StdCfgLayout(applicationName, cli.Files...),
		goschtalt.ConfigIs("two_words"),
		goschtalt.DefaultUnmarshalOptions(
			goschtalt.WithValidator(
				goschtalt.ValidatorFunc(validate.Validate),
			),
		),

		// Seed the program with the default, built-in configuration.
		// Mark this as a default so it is ordered correctly.
		goschtalt.AddValue("built-in", goschtalt.Root, defaultConfig,
			goschtalt.AsDefault()),
	)
	if err != nil {
		return nil, err
	}

	if cli.Show {
		// handleCLIShow handles the -s/--show option where the configuration is
		// shown, then the program is exited.
		//
		// Exit with success because if the configuration is broken it will be
		// very hard to debug where the problem originates.  This way you can
		// see the configuration and then run the service with the same
		// configuration to see the error.

		fmt.Fprintln(os.Stdout, gs.Explain().String())

		out, err := gs.Marshal()
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
		} else {
			fmt.Fprintln(os.Stdout, "## Final Configuration\n---\n"+string(out))
		}

		os.Exit(0)
	}

	var tmp Config
	err = gs.Unmarshal(goschtalt.Root, &tmp)
	if err != nil {
		fmt.Fprintln(os.Stderr, "There is a critical error in the configuration.")
		fmt.Fprintln(os.Stderr, "Run with -s/--show to see the configuration.")
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)

		// Exit here to prevent a very difficult to debug error from occurring.
		os.Exit(-1)
	}

	return gs, nil
}

// -----------------------------------------------------------------------------
// Keep the default configuration at the bottom of the file so it is easy to
// see what the default configuration is.
// -----------------------------------------------------------------------------
var defaultConfig = Config{
	Servers: Servers{
		Health: HealthServer{
			HTTP: arrangehttp.ServerConfig{
				Network: "tcp",
				Address: ":8088",
			},
			Path: HealthPath("/"),
		},
		Metrics: MetricsServer{
			HTTP: arrangehttp.ServerConfig{
				Network: "tcp",
				Address: "127.0.0.1:9361",
			},
			Path: MetricsPath("/metrics"),
		},
		Pprof: PprofServer{
			HTTP: arrangehttp.ServerConfig{
				Network: "tcp",
				Address: "127.0.0.1:9999",
			},
			Path: arrangepprof.DefaultPathPrefix,
		},
		Primary: PrimaryServer{
			HTTP: arrangehttp.ServerConfig{
				Network: "tcp",
				Address: "127.0.0.1:8080",
			},
		},
		Alternate: PrimaryServer{
			HTTP: arrangehttp.ServerConfig{
				Network: "tcp",
				Address: "127.0.0.1:9090",
			},
		},
	},
	Routes: Routes{
		Events: Route{
			Path:   "/api/events",
			Server: "primary",
		},
	},
	Prometheus: touchstone.Config{
		DefaultNamespace: "anemoi",
		DefaultSubsystem: "eventstream",
	},
	PrometheusHandler: touchhttp.Config{
		InstrumentMetricHandler: true,
		MaxRequestsInFlight:     5,
		Timeout:                 5 * time.Second,
	},

	Tracing: candlelight.Config{
		ApplicationName: applicationName,
	},

	Auth: Auth{
		IsBasic: true,
		// dummy default values for testing & local development
		Basic: auth.Basic{
			// nolint:gosec
			Username: "eventstream",
			// nolint:gosec
			Password: "eventstream-password",
		},
		Jwt: auth.Jwt{
			PublicKeyUrl:        "https://sat-stg.codebig2.net/v2/sign-keys/available",
			KeyRefreshInterval:  1,
			ServiceCapabilities: []string{"eventstream:write"},
		},
	},
	FilterManager: filter.FilterManagerConfig{
		DeliveryRetries:   3,
		DefaultQueueSize:  1000,
		DefaultBatchSize:  100,
		DefaultMaxWorkers: 1000,
		Filters: []filter.FilterConfig{
			{
				Stream: filter.Stream{
					StreamName: "comcast-cl.device-status.local",
					ConfigItems: []filter.ConfigItem{
						{Key: sender.Endpoint, Value: "http://localhost:4567"},
						{Key: sender.Region, Value: "local"},
						{Key: sender.AccessKey, Value: "accessKey"},
						{Key: sender.SecretKey, Value: "secretKey"},
						{Key: sender.SessionToken, Value: "sessionToken"},
						{Key: sender.Version, Value: "1.0"},
					},
				},
				Events:        []string{"device-status.*"},
				DestType:      "kinesis",
				StreamVersion: "1.0",
				MaxWorkers:    100,
				QueueSize:     100,
				BatchSize:     1,
			},
		},
	},
}
