package configuration

import (
	"net/url"
	"os"
	"strings"
	"time"

	"github.com/jeremywohl/flatten"
	"github.com/metal-toolbox/bioscfg/internal/model"
	"github.com/mitchellh/mapstructure"
	"github.com/pkg/errors"
	"github.com/spf13/viper"
)

var (
	// NATs streaming configuration
	defaultNatsConnectTimeout = 100 * time.Millisecond
)

// NatsConfig holds NATS specific configuration
type NatsConfig struct {
	NatsURL        string
	CredsFile      string
	KVReplicas     int
	ConnectTimeout time.Duration
}

func newNatsConfig() *NatsConfig {
	return &NatsConfig{
		ConnectTimeout: defaultNatsConnectTimeout,
	}
}

// Configuration holds application configuration read from a YAML or set by env variables.
// nolint:govet // prefer readability over field alignment optimization for this case.
type Configuration struct {
	// LogLevel is the app verbose logging level.
	// one of - info, debug, trace
	LogLevel string `mapstructure:"log_level"`

	// Concurrency is the number of concurrent tasks that can be running at once.
	Concurrency int `mapstructure:"concurrency"`

	// FacilityCode limits this service to events in a facility.
	FacilityCode string `mapstructure:"facility_code"`

	// FleetDBOptions defines the fleetdb client configuration parameters
	FleetDBOptions *FleetDBOptions `mapstructure:"fleetdb"`

	// NatsConfig defines the NATs events broker configuration parameters.
	NatsConfig *NatsConfig `mapstructure:"nats"`

	EnableProfiling bool `mapstructure:"enable_profiling"`
}

// New creates an empty configuration struct.
func New() *Configuration {
	config := &Configuration{}

	// these are initialized here so viper can read in configuration from env vars
	// once https://github.com/spf13/viper/pull/1429 is merged, this can go.
	config.FleetDBOptions = &FleetDBOptions{}
	config.NatsConfig = newNatsConfig()

	return config
}

func (c *Configuration) AsLogFields() []any {
	return []any{
		"logLevel", c.LogLevel,
		"concurrency", c.Concurrency,
		"facilityCode", c.FacilityCode,
		"disableOAuth", c.FleetDBOptions.DisableOAuth,
		"fleetDBUrl", c.FleetDBOptions.Endpoint,
		"natsURL", c.NatsConfig.NatsURL,
		"enableProfiling", c.EnableProfiling,
	}
}

func (c *Configuration) LoadArgs(args *model.Args) {
	c.LogLevel = args.LogLevel
	c.EnableProfiling = args.EnableProfiling
	c.FacilityCode = args.FacilityCode
}

// FleetDBOptions defines configuration for the fleetdb client.
// https://github.com/metal-toolbox/fleetdb
type FleetDBOptions struct {
	Endpoint             string   `mapstructure:"endpoint"`
	OidcIssuerEndpoint   string   `mapstructure:"oidc_issuer_endpoint"`
	OidcAudienceEndpoint string   `mapstructure:"oidc_audience_endpoint"`
	OidcClientSecret     string   `mapstructure:"oidc_client_secret"`
	OidcClientID         string   `mapstructure:"oidc_client_id"`
	OidcClientScopes     []string `mapstructure:"oidc_client_scopes"`
	DisableOAuth         bool     `mapstructure:"disable_oauth"`
}

// Load the application configuration
// Reads in the configFile when available and overrides from environment variables.
func Load(args *model.Args) (*Configuration, error) {
	viperConfig := viper.New()
	viperConfig.SetConfigType("yaml")
	viperConfig.SetEnvPrefix(model.AppName)
	viperConfig.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	viperConfig.AutomaticEnv()

	if args.ConfigFile != "" {
		fh, err := os.Open(args.ConfigFile)
		if err != nil {
			return nil, errors.Wrap(model.ErrConfig, err.Error())
		}

		if err = viperConfig.ReadConfig(fh); err != nil {
			return nil, errors.Wrap(model.ErrConfig, "ReadConfig error: "+err.Error())
		}
	}

	config := New()
	config.LoadArgs(args)

	if err := config.envBindVars(viperConfig); err != nil {
		return nil, errors.Wrap(model.ErrConfig, "env var bind error: "+err.Error())
	}

	if err := viperConfig.Unmarshal(config); err != nil {
		return nil, errors.Wrap(model.ErrConfig, "Unmarshal error: "+err.Error())
	}

	config.envVarAppOverrides(viperConfig)

	if err := config.envVarNatsOverrides(viperConfig); err != nil {
		return nil, errors.Wrap(model.ErrConfig, "nats env overrides error: "+err.Error())
	}

	if err := config.envVarFleetDBOverrides(viperConfig); err != nil {
		return nil, errors.Wrap(model.ErrConfig, "fleetdb env overrides error: "+err.Error())
	}

	return config, nil
}

func (c *Configuration) envVarAppOverrides(viperConfig *viper.Viper) {
	logLevel := viperConfig.GetString("log.level")
	if logLevel != "" {
		c.LogLevel = logLevel
	}
}

// envBindVars binds environment variables to the struct
// without a configuration file being unmarshalled,
// this is a workaround for a viper bug,
//
// This can be replaced by the solution in https://github.com/spf13/viper/pull/1429
// once that PR is merged.
func (c *Configuration) envBindVars(viperConfig *viper.Viper) error {
	envKeysMap := map[string]interface{}{}
	if err := mapstructure.Decode(c, &envKeysMap); err != nil {
		return err
	}

	// Flatten nested conf map
	flat, err := flatten.Flatten(envKeysMap, "", flatten.DotStyle)
	if err != nil {
		return errors.Wrap(err, "Unable to flatten configuration")
	}

	for k := range flat {
		if err := viperConfig.BindEnv(k); err != nil {
			return errors.Wrap(model.ErrConfig, "env var bind error: "+err.Error())
		}
	}

	return nil
}

// nolint:gocyclo // nats env configuration load is cyclomatic
func (c *Configuration) envVarNatsOverrides(viperConfig *viper.Viper) error {
	if c.NatsConfig == nil {
		c.NatsConfig = newNatsConfig()
	}

	if viperConfig.GetString("nats.url") != "" {
		c.NatsConfig.NatsURL = viperConfig.GetString("nats.url")
	}

	if c.NatsConfig.NatsURL == "" {
		return errors.New("missing parameter: nats.url")
	}

	if viperConfig.GetString("nats.creds.file") != "" {
		c.NatsConfig.CredsFile = viperConfig.GetString("nats.creds.file")
	}

	if viperConfig.GetDuration("nats.connect.timeout") != 0 {
		c.NatsConfig.ConnectTimeout = viperConfig.GetDuration("nats.connect.timeout")
	}

	if viperConfig.GetInt("nats.kv.replicas") != 0 {
		c.NatsConfig.KVReplicas = viperConfig.GetInt("nats.kv.replicas")
	}

	return nil
}

// nolint:gocyclo // parameter validation is cyclomatic
func (c *Configuration) envVarFleetDBOverrides(viperConfig *viper.Viper) error {
	if c.FleetDBOptions == nil {
		c.FleetDBOptions = &FleetDBOptions{}
	}

	if viperConfig.GetString("fleetdb.endpoint") != "" {
		c.FleetDBOptions.Endpoint = viperConfig.GetString("fleetdb.endpoint")
	}

	// Validate endpoint
	_, err := url.Parse(c.FleetDBOptions.Endpoint)
	if err != nil {
		return errors.New("fleetdb endpoint URL error: " + err.Error())
	}

	if viperConfig.GetString("fleetdb.disable.oauth") != "" {
		c.FleetDBOptions.DisableOAuth = viperConfig.GetBool("fleetdb.disable.oauth")
	}

	if c.FleetDBOptions.DisableOAuth {
		return nil
	}

	if viperConfig.GetString("fleetdb.oidc.issuer.endpoint") != "" {
		c.FleetDBOptions.OidcIssuerEndpoint = viperConfig.GetString("fleetdb.oidc.issuer.endpoint")
	}

	if c.FleetDBOptions.OidcIssuerEndpoint == "" {
		return errors.New("fleetdb oidc.issuer.endpoint not defined")
	}

	if viperConfig.GetString("fleetdb.oidc.audience.endpoint") != "" {
		c.FleetDBOptions.OidcAudienceEndpoint = viperConfig.GetString("fleetdb.oidc.audience.endpoint")
	}

	if c.FleetDBOptions.OidcAudienceEndpoint == "" {
		return errors.New("fleetdb oidc.audience.endpoint not defined")
	}

	if viperConfig.GetString("fleetdb.oidc.client.secret") != "" {
		c.FleetDBOptions.OidcClientSecret = viperConfig.GetString("fleetdb.oidc.client.secret")
	}

	if c.FleetDBOptions.OidcClientSecret == "" {
		return errors.New("fleetdb.oidc.client.secret not defined")
	}

	if viperConfig.GetString("fleetdb.oidc.client.id") != "" {
		c.FleetDBOptions.OidcClientID = viperConfig.GetString("fleetdb.oidc.client.id")
	}

	if c.FleetDBOptions.OidcClientID == "" {
		return errors.New("fleetdb.oidc.client.id not defined")
	}

	if viperConfig.GetString("fleetdb.oidc.client.scopes") != "" {
		c.FleetDBOptions.OidcClientScopes = viperConfig.GetStringSlice("fleetdb.oidc.client.scopes")
	}

	if len(c.FleetDBOptions.OidcClientScopes) == 0 {
		return errors.New("fleetdb oidc.client.scopes not defined")
	}

	return nil
}
