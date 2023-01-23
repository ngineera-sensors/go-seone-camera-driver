package fspdriver

import "fmt"

func getFullTopicString(relativePath string) string {
	return fmt.Sprintf("/seone/%s%s", SEONE_SN, relativePath)
}
