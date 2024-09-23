package config

import (
	"os"
	"strings"

	"github.com/jeremywohl/flatten"
	"github.com/metal-toolbox/bioscfg/internal/controllers/kind"
	"github.com/metal-toolbox/bioscfg/internal/store/fleetdb"
	"github.com/metal-toolbox/rivets/events"
	"github.com/mitchellh/mapstructure"
	"github.com/pkg/errors"
	"github.com/spf13/viper"
)

var (
	ErrConfig = errors.New("configuration error")
)

type Configuration struct {
	FacilityCode string
	LogLevel     string
	Endpoints    Endpoints
	Controller   kind.Controller
	Dryrun       bool
	Concurrency  int
}

type Endpoints struct {
	// NatsOptions defines the NATs events broker configuration parameters.
	Nats events.NatsOptions `mapstructure:"nats"`

	// FleetDBConfig defines the fleetdb client configuration parameters
	FleetDB fleetdb.Config `mapstructure:"fleetdb"`
}

func Load(cfgFilePath, loglevel string) (*Configuration, error) {
	v := viper.New()
	cfg := &Configuration{}

	err := cfg.envBindVars(v)
	if err != nil {
		return nil, err
	}

	v.SetConfigType("yaml")
	v.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	v.AutomaticEnv()

	err = readInFile(v, cfg, cfgFilePath)
	if err != nil {
		return nil, err
	}

	if loglevel != "" {
		cfg.LogLevel = loglevel
	}

	err = cfg.validate()
	return cfg, err
}

// Reads in the cfgFile when available and overrides from environment variables.
func readInFile(v *viper.Viper, cfg *Configuration, path string) error {
	if cfg == nil {
		return ErrConfig
	}

	if path != "" {
		fh, err := os.Open(path)
		if err != nil {
			return errors.Wrap(ErrConfig, err.Error())
		}

		if err = v.ReadConfig(fh); err != nil {
			return errors.Wrap(ErrConfig, "ReadConfig error:"+err.Error())
		}
	} else {
		v.AddConfigPath(".")
		v.SetConfigName("config")
		err := v.ReadInConfig()
		if err != nil {
			return err
		}
	}

	err := v.Unmarshal(cfg)
	if err != nil {
		return err
	}

	return nil
}

func (cfg *Configuration) validate() error {
	if cfg == nil {
		return ErrConfig
	}

	if cfg.FacilityCode == "" {
		return errors.Wrap(ErrConfig, "no facility code")
	}

	if cfg.LogLevel == "" {
		cfg.LogLevel = "info"
	}

	if cfg.Concurrency == 0 {
		cfg.Concurrency = 1
	}

	return nil
}

// envBindVars binds environment variables to the struct
// without a configuration file being unmarshalled,
// this is a workaround for a viper bug,
//
// This can be replaced by the solution in https://github.com/spf13/viper/pull/1429
// once that PR is merged.
func (cfg *Configuration) envBindVars(v *viper.Viper) error {
	envKeysMap := map[string]interface{}{}
	if err := mapstructure.Decode(cfg, &envKeysMap); err != nil {
		return err
	}

	// Flatten nested conf map
	flat, err := flatten.Flatten(envKeysMap, "", flatten.DotStyle)
	if err != nil {
		return errors.Wrap(err, "Unable to flatten config")
	}

	for k := range flat {
		if err := v.BindEnv(k); err != nil {
			return errors.Wrap(ErrConfig, "env var bind error: "+err.Error())
		}
	}

	return nil
}
