package main

import (
	"fmt"
	"log"
	"log/slog"
	"os"
	"strings"

	"github.com/Amoolaa/grafana-teams-ldap-sync/sync"
	"github.com/Amoolaa/grafana-teams-ldap-sync/sync/grafana"
	"github.com/knadh/koanf/parsers/yaml"
	"github.com/knadh/koanf/providers/env/v2"
	"github.com/knadh/koanf/providers/file"
	"github.com/knadh/koanf/v2"
	"github.com/urfave/cli/v2"
)

const (
	configFlag  = "config"
	mappingFlag = "mapping"
)

func main() {
	slog.SetDefault(slog.New(slog.NewJSONHandler(os.Stdout, nil)))

	app := &cli.App{
		Name:  "grafana-teams-ldap-sync",
		Usage: "blah blah",
		Commands: []*cli.Command{
			{
				Name:  "server",
				Usage: "run as server",
				Action: func(c *cli.Context) error {
					return nil
				},
			},
		},
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:     configFlag,
				Usage:    "path to config file",
				Value:    "config.yaml",
				Required: true,
			},
			&cli.StringFlag{
				Name:     mappingFlag,
				Usage:    "path to mapping config file",
				Value:    "mapping.yaml",
				Required: true,
			},
		},
		Action: func(c *cli.Context) error {
			s, err := initSyncer(c)
			if err != nil {
				log.Fatalf("failed to initialise config: %v", err)
			}
			s.GrafanaClient = grafana.NewClient(s.Config.Grafana)
			return s.Sync()
		},
	}

	if err := app.Run(os.Args); err != nil {
		log.Fatal(err)
	}
}

func initSyncer(c *cli.Context) (*sync.Syncer, error) {
	s := sync.NewSyncer()

	// parse config
	k := koanf.New(".")
	cfgFile := c.String(configFlag)
	var provider koanf.Provider
	if cfgFile != "" {
		s.Logger.Info("using config file", "path", cfgFile)
		provider = file.Provider(cfgFile)
	} else {
		s.Logger.Error("no config file provided")
		os.Exit(1)
	}

	if err := k.Load(provider, yaml.Parser()); err != nil {
		return nil, fmt.Errorf("failed to load configuration file: %w", err)
	}

	// parse mapping file
	mappingFile := c.String(mappingFlag)
	if cfgFile != "" {
		s.Logger.Info("using mapping file", "path", mappingFile)
		provider = file.Provider(mappingFile)
	} else {
		s.Logger.Error("no mapping file provided")
		os.Exit(1)
	}

	if err := k.Load(provider, yaml.Parser()); err != nil {
		return nil, fmt.Errorf("failed to load mapping file: %w", err)
	}

	envProvider := env.Provider(".", env.Opt{
		TransformFunc: func(k, v string) (string, any) {
			return strings.ReplaceAll(strings.ToLower(k), "_", "."), v
		},
	})

	if err := k.Load(envProvider, nil); err != nil {
		return nil, fmt.Errorf("failed to load environment variables: %w", err)
	}

	cfg := sync.Config{}
	if err := k.UnmarshalWithConf("", &cfg, koanf.UnmarshalConf{Tag: "koanf"}); err != nil {
		return nil, fmt.Errorf("failed to unmarshal configuration: %w", err)
	}

	s.Config = cfg

	if err := sync.ValidateConfig(s.Config); err != nil {
		log.Fatalf("failed to validate config: %v", err)
	}

	return s, nil
}
