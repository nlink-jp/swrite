package cmd

import "github.com/nlink-jp/swrite/internal/slack"

// newClientFunc creates a Slack client from a token.
// It can be replaced in tests to inject a test client.
var newClientFunc = func(token string) slack.Client {
	return slack.NewHTTPClient(token)
}

// SetNewClientFuncForTest replaces the client factory. Call ResetClientFunc in defer.
func SetNewClientFuncForTest(f func(token string) slack.Client) {
	newClientFunc = f
}

// ResetClientFunc restores the default client factory.
func ResetClientFunc() {
	newClientFunc = func(token string) slack.Client {
		return slack.NewHTTPClient(token)
	}
}

// newClient returns a Slack client for the given token.
func newClient(token string) slack.Client {
	return newClientFunc(token)
}
