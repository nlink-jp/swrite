package cmd

import "github.com/nlink-jp/swrite/internal/slack"

// newClientFunc creates a Slack client from a token and optional cache directory.
// It can be replaced in tests to inject a test client.
var newClientFunc = func(token, cacheDir string) slack.Client {
	c := slack.NewHTTPClient(token)
	if cacheDir != "" {
		c.SetCacheDir(cacheDir)
	}
	return c
}

// SetNewClientFuncForTest replaces the client factory. Call ResetClientFunc in defer.
func SetNewClientFuncForTest(f func(token, cacheDir string) slack.Client) {
	newClientFunc = f
}

// ResetClientFunc restores the default client factory.
func ResetClientFunc() {
	newClientFunc = func(token, cacheDir string) slack.Client {
		c := slack.NewHTTPClient(token)
		if cacheDir != "" {
			c.SetCacheDir(cacheDir)
		}
		return c
	}
}

// newClient returns a Slack client for the given token and cache directory.
func newClient(token, cacheDir string) slack.Client {
	return newClientFunc(token, cacheDir)
}
