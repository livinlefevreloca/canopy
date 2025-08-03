package ipc

type AWSConfigData struct {
	Profile           string
	SSORoleName       string
	AccountId         string
	AssumeRoleARN     string
	AccessKeyID       string
	CredentialsSource string
	Region            string
}

type AWSAccessKeysData struct {
	AccessKeyID     string
	SecretAccessKey string
	Region          string
}

type ChangeProfileData struct {
	Profile string
}

type AuthenticationData struct {
	Profile string
	Region  string
}

type ReauthenticateSSOData struct {
	Profile string
}

type ErrorData struct {
	Message string
}
