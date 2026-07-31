package console

import "testing"

func TestTwoColumnAcceptsLongValues(t *testing.T) {
	NewOutput().TwoColumn(
		"Environment",
		"/a/very/long/project/path/that/exceeds/the/default/console/detail/width/.env",
	)
}
