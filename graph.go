// Copyright 2014 Evan Jones. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package vclock

import (
	"fmt"
	"io"
	"strconv"
)

func assert(expression bool) {
	if !expression {
		panic("assertion failed")
	}
}

type node struct {
	value    fmt.Stringer
	pointsTo []*node
}

func (n *node) addEdge(other *node) {
	if other == nil {
		panic("nil node not permitted")
	}
	// TODO: assert(graph.contains(other))
	n.pointsTo = append(n.pointsTo, other)
}

func (n *node) Value() fmt.Stringer {
	return n.value
}

type graph struct {
	nodes map[string]*node
}

func newGraph() *graph {
	return &graph{make(map[string]*node)}
}

func (g *graph) Nodes() []*node {
	nodes := make([]*node, 0, len(g.nodes))
	for _, v := range g.nodes {
		nodes = append(nodes, v)
	}
	return nodes
}

func (g *graph) SetValue(n *node, v fmt.Stringer) {
	assert(g.find(n.value) == n)
	assert(g.find(v) == nil)
	delete(g.nodes, n.value.String())
	n.value = v
	g.nodes[v.String()] = n
	assert(g.find(n.value) == n)
}

func (g *graph) addNode(value fmt.Stringer) *node {
	s := value.String()
	assert(g.nodes[s] == nil)
	n := &node{value, nil}
	g.nodes[s] = n
	return n
}

func (g *graph) findOrAdd(value fmt.Stringer) *node {
	n := g.find(value)
	if n == nil {
		n = g.addNode(value)
	}
	return n
}

func (g *graph) find(value fmt.Stringer) *node {
	return g.nodes[value.String()]
}

func (g *graph) incoming(value fmt.Stringer) []*node {
	in := []*node{}
	target := g.find(value)
	for _, n := range g.nodes {
		for _, pointedTo := range n.pointsTo {
			if pointedTo == target {
				in = append(in, n)
			}
		}
	}
	return in
}

func quote(n *node) string {
	return strconv.Quote(n.value.String())
}

func (g *graph) WriteDot(w io.Writer) {
	fmt.Fprintln(w, "digraph {")
	fmt.Fprintln(w, "  graph [rankdir=LR]")
	fmt.Fprintln(w, "  node [shape=plaintext]")
	for _, n := range g.nodes {
		for _, other := range n.pointsTo {
			fmt.Fprintf(w, "  %s -> %s\n", quote(n), quote(other))
		}
		if len(n.pointsTo) == 0 {
			fmt.Fprintf(w, "  %s\n", quote(n))
		}
	}
	fmt.Fprintln(w, "}")
}

func containsNode(nodes []*node, n *node) bool {
	for _, containedNode := range nodes {
		if containedNode == n {
			return true
		}
	}
	return false
}

func (g *graph) containsCycle() bool {
	unvisited := map[string]*node{}
	for _, n := range g.nodes {
		unvisited[n.value.String()] = n
	}

	for len(unvisited) > 0 {
		var next *node
		for k, n := range unvisited {
			next = n
			delete(unvisited, k)
			break
		}

		// DFS from this node
		stack := []*node{next}
		for len(stack) > 0 {
			// explore children
			visitingChild := false
			for _, child := range stack[len(stack)-1].pointsTo {
				if containsNode(stack, child) {
					// cycle detected!
					return true
				}

				if unvisited[child.value.String()] != nil {
					// visit this child
					delete(unvisited, child.value.String())
					stack = append(stack, child)
					visitingChild = true
					break
				}
			}
			if !visitingChild {
				// no remaining children: pop stack
				stack = stack[:len(stack)-1]
			}
		}
	}
	return false
}

// Partitions clocks relative to clock into 3 sets: before, concurrent, after.
func partitionClocks(clocks []VectorClock, clock VectorClock) ([]VectorClock, []VectorClock, []VectorClock) {
	before := []VectorClock(nil)
	concurrent := []VectorClock(nil)
	after := []VectorClock(nil)

	for _, c := range clocks {
		if c.happensBefore(clock) {
			before = append(before, c)
		} else if clock.happensBefore(c) {
			after = append(after, c)
		} else {
			assert(c.concurrentWith(clock))
			concurrent = append(concurrent, c)
		}
	}
	return before, concurrent, after
}

func clockSetRemove(s []VectorClock, index int) []VectorClock {
	// swap s[index] to the end
	lastIndex := len(s) - 1
	if index < lastIndex {
		s[index], s[lastIndex] = s[lastIndex], s[index]
	} else {
		assert(index == lastIndex)
	}

	// remove the end
	return s[:lastIndex]
}

// Returns the clocks that are the "latest" clocks. For all clocks that are returned are concurrent,
// and are after any of the removed clocks. E.g. if c1 -> c2 then we keep c2.
// Super inefficient: O(n**2)
func latestClocks(clocks []VectorClock) []VectorClock {
	var latest []VectorClock
	for _, c := range clocks {
		addToLatest := true
		for i := 0; i < len(latest); i++ {
			if c.happensBefore(latest[i]) {
				// discard c: it is before latest[i]
				addToLatest = false
				break
			} else if latest[i].happensBefore(c) {
				// discard latest[i]: it is before c
				latest = clockSetRemove(latest, i)
				i -= 1
			} else {
				assert(latest[i].concurrentWith(c))
			}
		}
		if addToLatest {
			latest = append(latest, c)
		}
	}
	return latest
}

// Converts a set of vector clocks to a graph with the minimum number of edges to represents the
// happens-before relationships. We find the edges that are "immediately before" each clock.
// This is effectively a topological sort, but implemented in a horribly inefficient way O(n**3)
func ClocksToGraph(clocks []VectorClock) *graph {
	g := newGraph()

	for _, c := range clocks {
		// partition the clocks on this clock: n-1 comparsons
		before, _, _ := partitionClocks(clocks, c)
		// find the clocks that are later than all the others: worst case (n**2) comparisons
		immediatelyBefore := latestClocks(before)
		// link this clock to all the immediatelyBefore clocks

		n := g.findOrAdd(c)
		for _, previousClock := range immediatelyBefore {
			other := g.findOrAdd(previousClock)
			other.addEdge(n)
		}
	}
	assert(len(g.nodes) == len(clocks))
	return g
}
