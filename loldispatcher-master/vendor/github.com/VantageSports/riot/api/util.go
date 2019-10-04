package api

import (
	"fmt"
	"strings"
)

func joinIDs(ids ...int64) string {
	res := []string{}
	for _, id := range ids {
		res = append(res, fmt.Sprintf("%d", id))
	}
	return strings.Join(res, ",")
}

// GroupStrings takes a list of strings and constructs one or more slices of
// strings with maximum size groupSize.
//
// Example: GroupStrings(3, "a", "b", "c", "d", "e", "f", "g", "h")
//      returns: [["a", "b", "c"], ["d", "e", "f"], ["g", "h"]]
func GroupStrings(groupSize int, strs ...string) [][]string {
	groups := [][]string{}
	currentGroup := []string{}
	for _, str := range strs {
		currentGroup = append(currentGroup, str)
		if len(currentGroup) >= groupSize {
			groups = append(groups, currentGroup)
			currentGroup = []string{}
		}
	}
	if len(currentGroup) > 0 {
		groups = append(groups, currentGroup)
	}
	return groups
}

// Unique returns a deduped slice of the specified strings in unspecified order.
//
// Example: Unique("a", "b", "c", "c", "a", "c")
//      returns: ["a", "b", "c"]
func UniqueStrings(strs ...string) []string {
	set := map[string]bool{}
	for _, str := range strs {
		set[str] = true
	}
	unique := make([]string, len(set))
	i := 0
	for key := range set {
		unique[i] = key
		i++
	}
	return unique
}
