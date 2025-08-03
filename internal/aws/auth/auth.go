package auth

import (
	"bufio"
	"context"
	"log/slog"
	"os"
	"strings"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/sts"
	"github.com/livinlefevreloca/canopy/internal/ipc"
)

type AWSConfig struct {
	ipc.AWSConfigData
	SharedConfig *config.SharedConfig
	Config       *aws.Config
}

type SSOLoginError struct {
	Message string
}

func (e *SSOLoginError) Error() string {
	return e.Message
}

func GetAwsConfigFromProfileConfig(profile string, region string) (*AWSConfig, error) {
	ctx := context.Background()

	if profile == "" {
		profile = getAWSProfile()
	}

	slog.Info("Using AWS profile", "profile", profile)

	if region == "" {
		region = getAWSRegion(profile)
	}

	cfg, err := config.LoadDefaultConfig(ctx, config.WithSharedConfigProfile(profile), config.WithDefaultRegion(region))
	if err != nil {
		slog.Error("failed to load config", "error", err)
		return nil, err
	}

	sharedCfg, err := config.LoadSharedConfigProfile(ctx, profile)
	if err != nil {
		slog.Error("failed to load shared config", "error", err)
	}

	if region == "" {
		region = cfg.Region
	}
	slog.Info("Using AWS Region", "region", region)

	creds, err := cfg.Credentials.Retrieve(ctx)
	if err != nil {
		slog.Error("failed to retrieve credentials", "error", err)
		if strings.Contains(err.Error(), "the SSO session has expired or is invalid") {
			msg := "SSO session is expired or invalid"
			slog.Error(msg)
			return nil, &SSOLoginError{
				Message: msg,
			}
		}
		return nil, err
	}

	slog.Info("Using credentials with source", "source", creds.Source)

	accountId, err := getAccountId(ctx, &cfg)
	if err != nil {
		slog.Error("failed to get account ID", "error", err)
	}

	configData := ipc.AWSConfigData{
		Profile:           profile,
		SSORoleName:       sharedCfg.SSORoleName,
		AccountId:         accountId,
		AssumeRoleARN:     "",
		AccessKeyID:       creds.AccessKeyID,
		CredentialsSource: creds.Source,
		Region:            region,
	}

	return &AWSConfig{
		AWSConfigData: configData,
		SharedConfig:  &sharedCfg,
		Config:        &cfg,
	}, nil
}

func GetAwsFromAccessKeys(accessKeyID, secretAccessKey, region string) (*AWSConfig, error) {
	ctx := context.Background()

	if accessKeyID == "" || secretAccessKey == "" {
		return nil, nil // or return an error if you prefer
	}

	slog.Info("Using AWS Access Keys", "AccessKeyID", accessKeyID)

	if region == "" {
		region = getAWSRegion("")
	}

	cfg, err := config.LoadDefaultConfig(ctx,
		config.WithCredentialsProvider(credentials.NewStaticCredentialsProvider(accessKeyID, secretAccessKey, "")),
		config.WithDefaultRegion(region))

	if err != nil {
		slog.Error("failed to load config with access keys", "error", err)
		return nil, err
	}

	creds, err := cfg.Credentials.Retrieve(ctx)
	if err != nil {
		slog.Error("failed to retrieve credentials", "error", err)
		return nil, err
	}

	slog.Info("Using credentials with source", "source", creds.Source)

	accountId, err := getAccountId(ctx, &cfg)
	if err != nil {
		slog.Error("failed to get account ID", "error", err)
	}

	configData := ipc.AWSConfigData{
		Profile:           "",
		SSORoleName:       "",
		AccountId:         accountId,
		AssumeRoleARN:     "",
		AccessKeyID:       accessKeyID,
		CredentialsSource: creds.Source,
		Region:            region,
	}

	return &AWSConfig{
		AWSConfigData: configData,
		Config:        &cfg,
		SharedConfig:  nil, // No shared config for access keys
	}, nil
}

func getAWSProfile() string {
	if profile := os.Getenv("AWS_PROFILE"); profile != "" {
		return profile
	}
	if profile := os.Getenv("AWS_DEFAULT_PROFILE"); profile != "" {
		return profile
	}
	return config.DefaultSharedConfigProfile // "default"
}

func getAccountId(ctx context.Context, cfg *aws.Config) (string, error) {
	client := sts.NewFromConfig(*cfg)
	output, err := client.GetCallerIdentity(ctx, &sts.GetCallerIdentityInput{})
	if err != nil {
		return "", err
	}
	return *output.Account, nil
}

func getAWSRegion(profile string) string {
	if region := os.Getenv("AWS_REGION"); region != "" {
		return region
	}
	if region := os.Getenv("AWS_DEFAULT_REGION"); region != "" {
		return region
	}

	return ""
}

func GetAvailableProfiles() []string {
	var credentialFile, configFile string
	if credentialFile = os.Getenv("AWS_SHARED_CREDENTIALS_FILE"); credentialFile == "" {
		credentialFile = config.DefaultSharedCredentialsFilename()
	}
	if configFile = os.Getenv("AWS_CONFIG_FILE"); configFile == "" {
		configFile = config.DefaultSharedConfigFilename()
	}
	profiles := make(map[string]struct{})
	if credProfiles, err := getProfilesFromFile(credentialFile); err == nil {
		for _, profile := range credProfiles {
			if _, exists := profiles[profile]; !exists {
				profiles[profile] = struct{}{}
			}
		}
	} else {
		slog.Error("Failed to get profiles from credentials file", "error", err)
	}
	if configProfiles, err := getProfilesFromFile(configFile); err == nil {
		for _, profile := range configProfiles {
			if _, exists := profiles[profile]; !exists {
				profiles[profile] = struct{}{}
			}
		}
	} else {
		slog.Error("Failed to get profiles from config file", "error", err)
	}
	profileList := make([]string, 0, len(profiles))
	for profile := range profiles {
		profileList = append(profileList, profile)
	}
	return profileList
}

func getProfilesFromFile(filePath string) ([]string, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	var profiles []string
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if strings.HasPrefix(line, "[profile ") && strings.HasSuffix(line, "]") {
			// Extract profile name from the line
			profile := strings.TrimPrefix(line, "[profile ")
			profile = strings.TrimSuffix(profile, "]")
			if profile != "" {
				profiles = append(profiles, profile)
			}
		}
	}

	if err := scanner.Err(); err != nil {
		return nil, err
	}

	return profiles, nil
}
