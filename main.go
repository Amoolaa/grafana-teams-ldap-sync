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
	flag "github.com/spf13/pflag"
)

const (
	configFlag = "config"
)

func main() {
	slog.SetDefault(slog.New(slog.NewJSONHandler(os.Stdout, nil)))

	flags := flag.NewFlagSet("config", flag.ContinueOnError)
	initFlags(flags)

	flags.Parse(os.Args[1:])
	s, err := initConfig(flags)
	if err != nil {
		log.Fatalf("failed to initialise config: %v", err)
	}

	if err := sync.ValidateConfig(s.Config); err != nil {
		log.Fatalf("failed to validate config: %v", err)
	}

	g := grafana.NewClient(s.Config.Grafana)
	s.GrafanaClient = g

	if s.Config.Sync.Enabled {
		cron, err := s.InitCron()
		if err != nil {
			log.Fatalf("failed to initialise cron job: %v", err)
		}
		defer func() { _ = cron.Shutdown() }()
	}

	if err = s.Start(); err != nil {
		s.Logger.Error("error serving traffic", "error", err)
		os.Exit(1)
	}
}

func initFlags(f *flag.FlagSet) {
	f.String(configFlag, "", "path to config file")
	f.Usage = func() {
		fmt.Println(f.FlagUsages())
		os.Exit(0)
	}
}

func initConfig(flags *flag.FlagSet) (*sync.Syncer, error) {
	s := sync.NewSyncer()

	cfgFile, _ := flags.GetString(configFlag)
	var provider koanf.Provider
	if cfgFile != "" {
		s.Logger.Info("using config file", "path", cfgFile)
		provider = file.Provider(cfgFile)
	} else {
		s.Logger.Error("no config file provided")
		os.Exit(1)
	}

	k := koanf.New(".")

	if err := k.Load(provider, yaml.Parser()); err != nil {
		return nil, fmt.Errorf("failed to load configuration file: %w", err)
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
	return s, nil
}
