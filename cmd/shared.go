package cmd

import (
	"errors"
	"fmt"
	"log"
	"os"
	"slices"
	"strings"

	"github.com/aws/aws-sdk-go-v2/service/sso"
	"github.com/aws/aws-sdk-go-v2/service/ssooidc"
	"github.com/gerdou/awsx/cmd/internal"
	"github.com/gerdou/awsx/utilities"
	"github.com/spf13/cobra"
)

func processInputArgsForSelectAndRefresh(cmd *cobra.Command, args []string) (configName string, configs map[string]*internal.Config, profileNames []string, err error) {
	configName = "default"

	if len(args) >= 1 {
		configName = args[0]
	}

	if len(args) >= 2 {
		profileNames = args[1:]
	}

	configs, err = internal.ReadInternalConfig()
	if err != nil && !errors.Is(err, os.ErrNotExist) {
		return "", nil, nil, err
	}

	// Configs are only ever created by the config command, so a mistyped name never starts onboarding
	if _, exists := configs[configName]; !exists {
		cmd.SilenceUsage = true
		return "", nil, nil, unknownConfigError(cmd, configName, len(args) >= 1, configs)
	}

	return configName, configs, profileNames, nil
}

// unknownConfigError explains that configName is not a config, how to add it, and what may have been meant instead
func unknownConfigError(cmd *cobra.Command, configName string, named bool, configs map[string]*internal.Config) error {
	typedAtRoot := named && !cmd.HasParent()

	var message strings.Builder
	switch {
	case typedAtRoot:
		fmt.Fprintf(&message, "%q is neither a config nor a command\n", configName)
	case named:
		fmt.Fprintf(&message, "there is no config named %q\n", configName)
	default:
		fmt.Fprintf(&message, "there is no config named %q, which is used when you don't name a config\n", configName)
	}

	fmt.Fprintf(&message, "\nTo add a config named %q, run:\n  awsx config %s\n", configName, configName)

	if typedAtRoot {
		if suggestions := cmd.SuggestionsFor(configName); len(suggestions) > 0 {
			message.WriteString("\nDid you mean this command?\n")
			for _, suggestion := range suggestions {
				fmt.Fprintf(&message, "  awsx %s\n", suggestion)
			}
		}
		message.WriteString("\nIf you misspelled it, these are the available configs and commands.\n")
	}

	configNames := utilities.Keys(configs)
	slices.Sort(configNames)
	if len(configNames) == 0 {
		message.WriteString("\nThere are no configs yet.\n")
	} else {
		message.WriteString("\nAvailable configs:\n")
		for _, name := range configNames {
			fmt.Fprintf(&message, "  %s\n", name)
		}
	}

	if typedAtRoot {
		message.WriteString("\nAvailable commands:\n")
		for _, command := range cmd.Commands() {
			if command.IsAvailableCommand() || command.Name() == "help" {
				fmt.Fprintf(&message, "  %-*s %s\n", command.NamePadding(), command.Name(), command.Short)
			}
		}
	}

	return errors.New(strings.TrimRight(message.String(), "\n"))
}

func actionWithUnspecifiedProfiles(config *internal.Config, oidcApi *ssooidc.Client, ssoApi *sso.Client, action func(*internal.Config, *internal.Profile, *ssooidc.Client, *sso.Client) error) error {
	var selectedProfiles []*internal.Profile
	if len(config.Profiles) > 1 {
		prompt := internal.Prompter{}
		profiles := utilities.Keys(config.Profiles)
		slices.Sort(profiles)
		indexes, err := prompt.MultiSelect("Select the profile", profiles, nil)
		if err != nil {
			return err
		}

		for _, index := range indexes {
			selectedProfiles = append(selectedProfiles, config.Profiles[profiles[index]])
		}
	} else {
		for _, p := range config.Profiles {
			selectedProfiles = []*internal.Profile{p}
		}
	}

	if len(selectedProfiles) == 0 {
		return errors.New("no profile selected")
	}

	var errs []error
	for _, profile := range selectedProfiles {
		if profile == nil {
			return errors.New(fmt.Sprintf("profile is nil for config \"%s\"", config.Name))
		}

		if profile.Name == "" {
			return errors.New(fmt.Sprintf("profile name is empty for config \"%s\"", config.Name))
		}

		if profile.Region == "" {
			return errors.New("no region is set for this profile")
		}

		err := action(config, profile, oidcApi, ssoApi)
		if err != nil {
			errs = append(errs, errors.New(fmt.Sprintf("%s: %v", profile.Name, err)))
		}
	}

	return errors.Join(errs...)
}

func syncProfiles(config *internal.Config, oidcApi *ssooidc.Client, ssoApi *sso.Client) {
	if err := internal.SyncProfiles(config, oidcApi, ssoApi); err != nil {
		log.Printf("Could not sync profiles of %s config with AWS SSO, continuing with the existing profiles: %v", config.Name, err)
	}
}

func actionWithSpecifiedProfiles(config *internal.Config, profileNames []string, oidcApi *ssooidc.Client, ssoApi *sso.Client, action func(*internal.Config, *internal.Profile, *ssooidc.Client, *sso.Client) error) error {
	var validProfileNames []string
	var errs []error
	for _, profile := range config.Profiles {
		if slices.Contains(profileNames, profile.Name) {
			validProfileNames = append(validProfileNames, profile.Name)
			err := action(config, profile, oidcApi, ssoApi)
			if err != nil {
				errs = append(errs, errors.New(fmt.Sprintf("%s: %v", profile.Name, err)))
			}
		}
	}

	for _, profileName := range profileNames {
		if !slices.Contains(validProfileNames, profileName) {
			log.Printf("Profile %s does not exist in %s config", profileName, config.Name)
		}
	}

	return errors.Join(errs...)
}
