# awsx

`awsx` is a CLI tool for retrieving temporary credentials via AWS SSO. It simplifies the process of managing multiple AWS SSO configurations and profiles, allowing you to easily switch between accounts and roles.

## Installation

### Homebrew
```bash
brew tap gerdou/awsx
brew install awsx
```

### Binary Download
Get the binary from the [releases page](https://github.com/gerdou/awsx/releases).

### Build from Source
With Go installed, run:
```bash
go install github.com/gerdou/awsx@latest
```

## Usage

### 1. Configuration

Before you can retrieve credentials, add a configuration:

```bash
awsx config [config-name]
```

This is the only way to add a configuration. `awsx`, `awsx select` and `awsx refresh` only use configurations that already exist. If you name one that doesn't exist, they stop and show how to add it, along with the existing configurations and, for `awsx` on its own, the available commands. This way a typo such as `awsx refrsh` never starts setting up a new configuration.

- **config-name**: Optional. Defaults to `default`. You can have multiple SSO configurations (e.g., `work`, `personal`).
- You will be prompted for:
    - **Start URL ID**: The identifier for your AWS SSO Start URL (e.g., if your URL is `https://d-1234567890.awsapps.com/start`, the ID is `d-1234567890`).
    - **SSO Region**: The region where your SSO is configured.
    - **Profile name**: The name of the local AWS profile to update.
    - **Default Region**: The default region for that profile.
    - **Default Account/Role**: (Optional) You can preset a default account and role for the profile.

### 2. Selecting an Account and Role

To browse available accounts and roles in your SSO and update your local AWS credentials:

```bash
awsx select [config-name] [profile-name]
```

- If `config-name` is omitted, the `default` configuration is used. If `profile-name` is omitted, `awsx` prompts you to select from the configuration's profiles.
- `awsx` will open your browser for SSO authentication if needed.
- Once authenticated, you'll see a list of accounts and roles you have access to. Selecting one will update your `~/.aws/credentials` for that profile.

### 3. Refreshing Credentials

If you have already selected an account and role for a profile, you can quickly refresh the temporary credentials without going through the selection process again:

```bash
awsx refresh [config-name] [profile-name]
```

**Tip:** Running `awsx` without any command is an alias for `awsx refresh`.

If the role you last used in an account has been taken away from you, `awsx refresh` prompts you to select another role in that account instead of failing.

### 4. Syncing Profiles with AWS SSO

Every time you run `awsx select` or `awsx refresh`, the profiles of the configuration in use are synced with the accounts you can access in AWS SSO before anything else happens:

- **Lost access**: When you can no longer access an account, every profile whose default account is that account is removed. Their last used account/role info and their sections in `~/.aws/credentials` are removed too.
- **New access**: When you gain access to a new account, a profile is added for it. The profile is named after the account (lowercase, spaces replaced with `-`) and uses the configuration's SSO region as its region. If a profile with that name already exists, the account ID is appended to the name.
- **Renamed accounts**: When an account is renamed in AWS SSO, the account name stored in its profiles is updated.

Each change is logged. Profiles are matched to accounts by account ID, so you can have several profiles for the same account, for example one per region, and they are all kept or removed together. Profiles without a default account are never changed.

If AWS SSO cannot be reached, or returns no accounts, nothing is changed: `awsx` logs a warning and continues with your existing profiles.

To stop an account from being added back, remove its profiles with `awsx config remove --profile`. When the last profile of an account is removed, its account ID is added to the configuration's `ignored_accounts` list, and syncing skips it. Delete the ID from that list to have the account added again.

### 5. Advanced Configuration Management

`awsx` provides several subcommands for managing your saved configurations:

- **View Configuration**:
  ```bash
  awsx config get
  ```
  Prints the current `awsx` configuration in YAML format.

- **Remove Configuration/Profile**:
  ```bash
  awsx config remove [config-name] [--profile profile1,profile2]
  ```
  Removes an entire configuration or specific profiles from it. Accounts left with no profiles are added to `ignored_accounts`, so syncing does not add them back (see [Syncing Profiles with AWS SSO](#4-syncing-profiles-with-aws-sso)).

- **Export/Import**:
  ```bash
  awsx config export -f backup.yaml
  awsx config import -f backup.yaml
  ```
  Useful for moving your configurations between machines.

### 6. Bulk Operations

You can refresh multiple profiles at once:

```bash
awsx refresh default profile1 profile2
# OR refresh all profiles in a config
awsx refresh default all
```

## Files and Locations

- **Configuration Path**: `~/.config/awsx/config`
- **Cache Path**: `~/.config/awsx/cache/` (stores access tokens and last used account/role info)
- **AWS Credentials**: `~/.aws/credentials` (updated by `awsx`)

## License

This project is licensed under the MIT License - see the [LICENSE](LICENSE) file for details.