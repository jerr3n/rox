package util

import "slices"

func IsIn(target string, searchable ...string) bool {
	return slices.Contains(searchable, target)
}
