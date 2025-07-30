package gateway

import (
	"fmt"
	"strings"

	"github.com/AndreZiviani/lgtmp-query-gateway/internal/config"
)

func findMostSpecificPrefix(path string, destinations map[string]config.Destination) (config.Destination, string, error) {
	var (
		mostSpecificDestination config.Destination
		mostSpecificPrefix      string
	)

	for prefix, destination := range destinations {
		// Check if the prefix matches the start of the path
		if strings.HasPrefix(path, prefix) {
			// Update if this prefix is more specific (longer) than the current one
			if len(prefix) > len(mostSpecificPrefix) {
				mostSpecificPrefix = prefix
				mostSpecificDestination = destination
			}
		}
	}

	if mostSpecificPrefix == "" {
		return config.Destination{}, "", fmt.Errorf("no matching prefix found for path: %s", path)
	}

	return mostSpecificDestination, mostSpecificPrefix, nil
}
