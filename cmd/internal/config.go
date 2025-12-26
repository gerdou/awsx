package internal

import (
	"errors"
	"fmt"
	"os"
	"path"
	"time"

	ssoTypes "github.com/aws/aws-sdk-go-v2/service/sso/types"
	"github.com/gerdou/awsx/version"
	"gopkg.in/ini.v1"
	"gopkg.in/yaml.v3"
)

type Profile struct {
	Region         string            `yaml:"region"`
	DefaultAccount *UsageInformation `yaml:"default_account,omitempty"`
	Name           string            `yaml:"-"`
}

type Config struct {
	Id        string              `yaml:"Id"`
	Profiles  map[string]*Profile `yaml:"profiles"`
	SsoRegion string              `yaml:"sso_region"`
	Complete  bool                `yaml:"-"`
	Name      string              `yaml:"-"`
}

func (c *Config) GetStartUrl() string {
	return fmt.Sprintf("https://%s.awsapps.com/start", c.Id)
}

type ConfigFile struct {
	Version string             `yaml:"version"`
	Configs map[string]*Config `yaml:"configs"`
}

type ClientInformation struct {
	AccessTokenExpiresAt    time.Time `yaml:"access_token_expires_at"`
	AccessToken             string    `yaml:"access_token"`
	ClientId                string    `yaml:"client_id"`
	ClientSecret            string    `yaml:"client_secret"`
	ClientSecretExpiresAt   time.Time `yaml:"client_secret_expires_at"`
	DeviceCode              string    `yaml:"device_code"`
	VerificationUriComplete string    `yaml:"verification_uri_complete"`
	StartUrl                string    `yaml:"start_url"`
}

type ClientInformationFile struct {
	Version           string                        `yaml:"version"`
	ClientInformation map[string]*ClientInformation `yaml:"client_information"`
}

type UsageInformation struct {
	AccountId   string `yaml:"account_id"`
	AccountName string `yaml:"account_name"`
	Role        string `yaml:"role"`
	Profile     string `yaml:"profile"`
}

type LastUsageInformationFile struct {
	Version              string                                   `yaml:"version"`
	LastUsageInformation map[string]map[string][]UsageInformation `yaml:"last_usage_information"`
}

var home, _ = os.UserHomeDir()
var defaultAwsCredentialsPath = path.Join(home, ".aws")
var defaultAwsCredentialsFileName = "credentials"

var defaultInternalPath = path.Join(home, ".config/awsx")
var defaultConfigFileName = path.Join(defaultInternalPath, "config")

var defaultCachePath = path.Join(defaultInternalPath, "cache")
var defaultClientInformationFileName = path.Join(defaultCachePath, "access-token")
var defaultLastUsageFileName = path.Join(defaultCachePath, "last-usage")

func ReadUsageInformationFile() (*LastUsageInformationFile, error) {
	file, err := os.ReadFile(defaultLastUsageFileName)
	if err != nil {
		return &LastUsageInformationFile{
			Version:              version.Version,
			LastUsageInformation: make(map[string]map[string][]UsageInformation),
		}, err
	}

	lastUsageInformationFile := &LastUsageInformationFile{}
	err = yaml.Unmarshal(file, &lastUsageInformationFile)
	if err != nil {
		return nil, err
	}

	return lastUsageInformationFile, nil
}

func GetUsageInformationForConfig(configName string) (map[string][]UsageInformation, error) {
	usageInformationFile, err := ReadUsageInformationFile()
	if err != nil {
		return nil, nil
	}

	usageInformation, exists := usageInformationFile.LastUsageInformation[configName]
	if !exists {
		return nil, nil
	}

	return usageInformation, nil
}

func SaveUsageInformationForConfig(configName string, information *UsageInformation) error {
	err := os.MkdirAll(defaultCachePath, 0700)
	if err != nil {
		return err
	}

	usageInformationFile, _ := ReadUsageInformationFile()
	usageInformation, exists := usageInformationFile.LastUsageInformation[configName]
	if !exists {
		usageInformation = make(map[string][]UsageInformation)
	}

	usageInformationOfProfile, _ := usageInformation[information.Profile]

	allUsageInformation := append([]UsageInformation{*information}, usageInformationOfProfile...)
	var unique []UsageInformation
	uniqueMap := make(map[UsageInformation]bool)

	for _, value := range allUsageInformation {
		if _, exists := uniqueMap[value]; !exists {
			uniqueMap[value] = true
			unique = append(unique, value)
		}
	}

	usageInformation[information.Profile] = unique
	usageInformationFile.LastUsageInformation[configName] = usageInformation
	content, err := yaml.Marshal(usageInformationFile)

	return os.WriteFile(defaultLastUsageFileName, content, 0700)
}

func ReadClientInformationFile() (*ClientInformationFile, error) {
	file, err := os.ReadFile(defaultClientInformationFileName)
	if err != nil {
		return &ClientInformationFile{
			Version:           version.Version,
			ClientInformation: make(map[string]*ClientInformation),
		}, nil
	}

	clientInformationFile := ClientInformationFile{}
	err = yaml.Unmarshal(file, &clientInformationFile)
	if err != nil {
		return nil, err
	}

	return &clientInformationFile, nil
}

func GetClientInformationForConfig(configName string) (*ClientInformation, error) {
	emptyClientInformation := &ClientInformation{
		AccessTokenExpiresAt:  time.Now().AddDate(-1, 0, 0),
		ClientSecretExpiresAt: time.Now().AddDate(-1, 0, 0),
	}

	clientInformationFile, err := ReadClientInformationFile()
	if err != nil {
		return emptyClientInformation, nil
	}

	clientInformation, exists := clientInformationFile.ClientInformation[configName]
	if !exists {
		return emptyClientInformation, nil
	}

	return clientInformation, nil
}

func SetClientInformationForConfig(configName string, clientInformation *ClientInformation) error {
	err := os.MkdirAll(defaultCachePath, 0700)
	if err != nil {
		return err
	}

	existingClientInformationFile, err := ReadClientInformationFile()
	if err != nil {
		return err
	}

	existingClientInformationFile.ClientInformation[configName] = clientInformation

	content, err := yaml.Marshal(existingClientInformationFile)
	if err != nil {
		return err
	}

	return os.WriteFile(defaultClientInformationFileName, content, 0700)
}

func formatExpiration(roleCredentials *ssoTypes.RoleCredentials) string {
	// Convert the 'Expiration' Unix timestamp to time.Time
	expirationTime := time.UnixMilli(roleCredentials.Expiration).UTC()

	// Format time.Time to string in "2006-01-02T15:04:05Z" format
	expirationString := expirationTime.Format(time.RFC3339)

	return expirationString
}

func WriteAwsConfigFile(profile string, configuration *Config, credentials *ssoTypes.RoleCredentials) error {
	if _, exists := configuration.Profiles[profile]; !exists {
		return errors.New("profile does not exist in the configuration")
	}

	if configuration.Profiles[profile].Region == "" {
		return errors.New("region does not exist in the configuration")
	}

	file, err := os.OpenFile(path.Join(defaultAwsCredentialsPath, defaultAwsCredentialsFileName), os.O_CREATE, 0644)
	if err != nil {
		return err
	}
	defer func(file *os.File) {
		_ = file.Close()
	}(file)

	awsCredentialsFile, err := ini.Load(path.Join(defaultAwsCredentialsPath, defaultAwsCredentialsFileName))
	if err != nil {
		return err
	}

	profileSection := awsCredentialsFile.Section(profile)
	if profileSection == nil {
		profileSection, _ = awsCredentialsFile.NewSection(profile)
		_, _ = profileSection.NewKey("aws_access_key_id", *credentials.AccessKeyId)
		_, _ = profileSection.NewKey("aws_secret_access_key", *credentials.SecretAccessKey)
		_, _ = profileSection.NewKey("aws_session_token", *credentials.SessionToken)
		_, _ = profileSection.NewKey("output", "json")
		_, _ = profileSection.NewKey("region", configuration.Profiles[profile].Region)
		_, _ = profileSection.NewKey("aws_expiration", formatExpiration(credentials))
	} else {
		profileSection.Key("aws_access_key_id").SetValue(*credentials.AccessKeyId)
		profileSection.Key("aws_secret_access_key").SetValue(*credentials.SecretAccessKey)
		profileSection.Key("aws_session_token").SetValue(*credentials.SessionToken)
		profileSection.Key("output").SetValue("json")
		profileSection.Key("region").SetValue(configuration.Profiles[profile].Region)
		profileSection.Key("aws_expiration").SetValue(formatExpiration(credentials))
	}

	err = awsCredentialsFile.SaveTo(path.Join(defaultAwsCredentialsPath, defaultAwsCredentialsFileName))
	if err != nil {
		return err
	}
	return nil
}

func ReadInternalConfig() (map[string]*Config, error) {
	file, err := os.ReadFile(defaultConfigFileName)
	if err != nil {
		return make(map[string]*Config), err
	}

	configFile := ConfigFile{}
	err = yaml.Unmarshal(file, &configFile)
	if err != nil {
		return nil, err
	}

	for configName, config := range configFile.Configs {
		config.Complete = true
		config.Name = configName
		for name, profile := range config.Profiles {
			profile.Name = name
		}
	}

	return configFile.Configs, nil
}

func ExportInternalConfig(exportPath string) error {
	file, err := os.ReadFile(defaultConfigFileName)
	if err != nil {
		return err
	}

	configFile := ConfigFile{}
	err = yaml.Unmarshal(file, &configFile)
	if err != nil {
		return err
	}

	return os.WriteFile(exportPath, file, 0700)
}

func ImportInternalConfig(importPath string) error {
	file, err := os.ReadFile(importPath)
	if err != nil {
		return err
	}

	configFile := ConfigFile{}
	err = yaml.Unmarshal(file, &configFile)
	if err != nil {
		return err
	}

	return WriteInternalConfig(configFile.Configs)
}

func WriteInternalConfig(input map[string]*Config) error {
	err := os.MkdirAll(defaultInternalPath, 0700)
	if err != nil {
		return err
	}

	existingConfigs, _ := ReadInternalConfig()
	if existingConfigs == nil {
		existingConfigs = make(map[string]*Config)
	}

	configs := make(map[string]*Config)
	for key, value := range input {
		if value.Complete {
			configs[key] = value
			continue
		}

		if config, exists := existingConfigs[key]; exists {
			configs[key] = config
		}
	}

	config, err := yaml.Marshal(ConfigFile{
		Version: version.Version,
		Configs: configs,
	})
	if err != nil {
		return err
	}

	return os.WriteFile(defaultConfigFileName, config, 0700)
}

func RemoveInternalConfig(configNames []string) error {
	configs, _ := ReadInternalConfig()

	for _, configName := range configNames {
		if _, ok := configs[configName]; !ok {
			continue
		}

		delete(configs, configName)
	}

	if len(configs) == 0 {
		return os.RemoveAll(defaultInternalPath)
	}

	return WriteInternalConfig(configs)
}

func RemoveProfilesFromConfig(configName string, profileNames []string) error {
	configs, _ := ReadInternalConfig()

	config, ok := configs[configName]
	if !ok {
		return nil
	}

	for _, profileName := range profileNames {
		if _, exists := config.Profiles[profileName]; !exists {
			continue
		}
		delete(config.Profiles, profileName)
	}

	if len(config.Profiles) == 0 {
		delete(configs, configName)
	} else {
		configs[configName] = config
	}

	if len(configs) == 0 {
		return os.RemoveAll(defaultInternalPath)
	}

	return WriteInternalConfig(configs)
}
