package autolink_test

import (
	"testing"

	"github.com/mattermost/mattermost-plugin-autolink/server/autolink"
)

var phabTests = []linkTest{
	{
		"Url replacement at end",
		autolink.Autolink{
			Pattern:           "(?P<task>\\bT\\d+)",
			Template:          "https://phabricator.wikimedia.org/$task",
			LookupUrlTemplate: "https://phabricator.wikimedia.org/$task",
		},
		"I like T12312",
		"Welcome s [MM-12345](https://mattermost.atlassian.net/browse/MM-12345)",
	},
}

func TestPhabricator(t *testing.T) {
	testLinks(t, phabTests...)
}
