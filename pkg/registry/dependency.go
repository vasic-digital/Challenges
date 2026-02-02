package registry

import (
	"fmt"
	"sort"
	"strings"

	"digital.vasic.challenges/pkg/challenge"
)

// topologicalSort orders challenges using Kahn's algorithm.
// It returns an error if a cycle is detected.
func topologicalSort(
	challenges map[challenge.ID]challenge.Challenge,
) ([]challenge.Challenge, error) {
	inDegree := make(map[challenge.ID]int, len(challenges))
	dependents := make(
		map[challenge.ID][]challenge.ID, len(challenges),
	)

	for id, c := range challenges {
		if _, exists := inDegree[id]; !exists {
			inDegree[id] = 0
		}
		for _, dep := range c.Dependencies() {
			inDegree[id]++
			dependents[dep] = append(dependents[dep], id)
		}
	}

	// Seed the queue with zero-degree nodes, sorted for
	// deterministic output.
	var queue []challenge.ID
	for id, degree := range inDegree {
		if degree == 0 {
			queue = append(queue, id)
		}
	}
	sort.Slice(queue, func(i, j int) bool {
		return queue[i] < queue[j]
	})

	ordered := make(
		[]challenge.Challenge, 0, len(challenges),
	)

	for len(queue) > 0 {
		id := queue[0]
		queue = queue[1:]

		if c, exists := challenges[id]; exists {
			ordered = append(ordered, c)
		}

		// Collect and sort neighbours for determinism.
		neighbours := dependents[id]
		sort.Slice(neighbours, func(i, j int) bool {
			return neighbours[i] < neighbours[j]
		})

		for _, dep := range neighbours {
			inDegree[dep]--
			if inDegree[dep] == 0 {
				queue = append(queue, dep)
			}
		}
	}

	if len(ordered) != len(challenges) {
		cycle := detectCycle(challenges)
		return nil, fmt.Errorf(
			"circular dependency detected: %s", cycle,
		)
	}

	return ordered, nil
}

// detectCycle returns a human-readable description of a
// dependency cycle in the challenge graph. It uses iterative
// DFS with three colouring states.
func detectCycle(
	challenges map[challenge.ID]challenge.Challenge,
) string {
	const (
		white = 0 // unvisited
		gray  = 1 // in current path
		black = 2 // finished
	)

	colour := make(map[challenge.ID]int, len(challenges))
	parent := make(
		map[challenge.ID]challenge.ID, len(challenges),
	)

	// Sort IDs for deterministic cycle detection.
	ids := make([]challenge.ID, 0, len(challenges))
	for id := range challenges {
		ids = append(ids, id)
	}
	sort.Slice(ids, func(i, j int) bool {
		return ids[i] < ids[j]
	})

	for _, startID := range ids {
		if colour[startID] != white {
			continue
		}

		type frame struct {
			id    challenge.ID
			deps  []challenge.ID
			index int
		}

		stack := []frame{
			{id: startID, deps: getDeps(challenges, startID)},
		}
		colour[startID] = gray

		for len(stack) > 0 {
			top := &stack[len(stack)-1]

			if top.index >= len(top.deps) {
				colour[top.id] = black
				stack = stack[:len(stack)-1]
				continue
			}

			dep := top.deps[top.index]
			top.index++

			if colour[dep] == gray {
				// Found cycle — reconstruct path.
				path := []string{string(dep)}
				for _, f := range stack {
					path = append(path, string(f.id))
					if f.id == dep {
						break
					}
				}
				return strings.Join(path, " -> ")
			}

			if colour[dep] == white {
				parent[dep] = top.id
				colour[dep] = gray
				stack = append(stack, frame{
					id:   dep,
					deps: getDeps(challenges, dep),
				})
			}
		}
	}

	return "unknown cycle"
}

// getDeps returns the sorted dependency IDs for a challenge.
func getDeps(
	challenges map[challenge.ID]challenge.Challenge,
	id challenge.ID,
) []challenge.ID {
	c, ok := challenges[id]
	if !ok {
		return nil
	}
	deps := c.Dependencies()
	sort.Slice(deps, func(i, j int) bool {
		return deps[i] < deps[j]
	})
	return deps
}
