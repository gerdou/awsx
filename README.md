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

The first time you run `awsx`, it will guide you through setting up a configuration. You can also manually trigger this with:

```bash
awsx config [config-name]
```

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

- If `config-name` or `profile-name` are omitted, `awsx` will prompt you to select from your saved configurations/profiles.
- `awsx` will open your browser for SSO authentication if needed.
- Once authenticated, you'll see a list of accounts and roles you have access to. Selecting one will update your `~/.aws/credentials` for that profile.

### 3. Refreshing Credentials

If you have already selected an account and role for a profile, you can quickly refresh the temporary credentials without going through the selection process again:

```bash
awsx refresh [config-name] [profile-name]
```

**Tip:** Running `awsx` without any command is an alias for `awsx refresh`.

### 4. Advanced Configuration Management

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
  Removes an entire configuration or specific profiles from it.

- **Export/Import**:
  ```bash
  awsx config export -f backup.yaml
  awsx config import -f backup.yaml
  ```
  Useful for moving your configurations between machines.

### 5. Bulk Operations

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