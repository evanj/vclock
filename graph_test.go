// Copyright 2014 Evan Jones. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package vclock

import (
	"fmt"
	"testing"
)

func TestLinearClocksToGraph(t *testing.T) {
	c1 := VectorClockWithValues(1)
	c2 := VectorClockWithValues(2)
	g := ClocksToGraph([]VectorClock{c2, c1})

	if len(g.incoming(c2)) != 1 {
		t.Error("err")
	}
	if len(g.incoming(c1)) != 0 {
		t.Error("err")
	}
}

// (c1 c4) -> c2 -> c3
// and c4 -> c2
var c1 = VectorClockWithValues(0, 1)
var c2 = VectorClockWithValues(1, 2)
var c3 = VectorClockWithValues(1, 3)
var c4 = VectorClockWithValues(1, 0)
var c5 = VectorClockWithValues(2, 2)

func setEquals(s1 []VectorClock, s2 []VectorClock) bool {
	contained := append([]VectorClock(nil), s2...)
	for _, v1 := range s1 {
		found := false
		for i, v2 := range contained {
			if v1.Equals(v2) {
				// remove v2 from contained
				lastIndex := len(contained) - 1
				contained[lastIndex], contained[i] = contained[i], contained[lastIndex]
				contained = contained[:lastIndex]
				found = true
				break
			}
		}
		if !found {
			return false
		}
	}
	return len(contained) == 0
}

func nodesToClocks(nodes []*node) []VectorClock {
	var out []VectorClock
	for _, n := range nodes {
		out = append(out, n.value.(VectorClock))
	}
	return out
}

func assertClocksEqual(s1 []VectorClock, s2 []VectorClock) {
	if !setEquals(s1, s2) {
		panic(fmt.Sprintf("clock sets not equal %v != %v", s1, s2))
	}
}

func TestClocksToGraph(t *testing.T) {
	g := ClocksToGraph([]VectorClock{c1, c2, c3, c4, c5})

	assertClocksEqual(nodesToClocks(g.incoming(c2)), []VectorClock{c4, c1})
	assertClocksEqual(nodesToClocks(g.find(c2).pointsTo), []VectorClock{c3, c5})
}

func TestPartitionClocks(t *testing.T) {
	clocks := []VectorClock{c1, c2, c3, c4, c5}

	before, concurrent, after := partitionClocks(clocks, c2)
	if !setEquals(before, []VectorClock{c1, c4}) {
		t.Error("wrong before", before, []VectorClock{c1, c4})
	}
	if !setEquals(concurrent, []VectorClock{c2}) {
		t.Error("wrong concurrent", concurrent)
	}
	if !setEquals(after, []VectorClock{c3, c5}) {
		t.Error("wrong after", after)
	}
}

func permute(l []VectorClock) [][]VectorClock {
	if len(l) <= 1 {
		return [][]VectorClock{l}
	}
	out := [][]VectorClock{}

	lastIndex := len(l) - 1
	for i := lastIndex; i >= 0; i-- {
		// swap index i and lastIndex
		l[lastIndex], l[i] = l[i], l[lastIndex]

		// produce every sub-permutation and append l[lastIndex]
		for _, sub := range permute(l[:lastIndex]) {
			permutation := make([]VectorClock, len(sub)+1)
			copy(permutation, sub)
			permutation[len(sub)] = l[lastIndex]
			out = append(out, permutation)
		}

		// swap back
		l[lastIndex], l[i] = l[i], l[lastIndex]
	}
	return out
}

func TestLatestClocks(t *testing.T) {
	latest := latestClocks([]VectorClock{c1, c2, c3, c4, c5})
	assertClocksEqual(latest, []VectorClock{c3, c5})

	// Example: sequence of (0 (1a 1b) (2c 2d)) (3e 3f) 4; we are examining 3e
	// 0 is before 3e: keep
	// 1a 1b are before 3e and conncurrent with 0: keep all
	// 2c, 2d is before 3e; AFTER 1a and 1b: discard these. concurrent with 0
	// 4 happens after 3: discard
	// final before set: (0, 2c, 2d); concurrent set: (3e 3f)
	op0 := VectorClockWithValues(0, 0, 1)
	op1a := VectorClockWithValues(1, 0, 0)
	op1b := VectorClockWithValues(0, 1, 0)
	op2c := VectorClockWithValues(2, 1, 0)
	op2d := VectorClockWithValues(1, 2, 0)
	op3e := VectorClockWithValues(3, 2, 1)
	op3f := VectorClockWithValues(2, 3, 1)
	op4 := VectorClockWithValues(4, 3, 1)
	clocks := []VectorClock{op0, op1a, op1b, op2c, op2d, op3e, op3f, op4}

	// ensure this algorithm is order independent
	for _, clocksPermutation := range permute(clocks) {
		before, concurrent, after := partitionClocks(clocksPermutation, op3e)
		assertClocksEqual(before, []VectorClock{op0, op1a, op1b, op2c, op2d})
		assertClocksEqual(concurrent, []VectorClock{op3e, op3f})
		assertClocksEqual(after, []VectorClock{op4})

		before = latestClocks(before)
		assertClocksEqual(before, []VectorClock{op0, op2c, op2d})
	}
}

type stringerString string

func (s stringerString) String() string {
	return string(s)
}

func TestContainsCycle(t *testing.T) {
	g := newGraph()
	n1 := g.addNode(stringerString("a"))
	n2 := g.addNode(stringerString("b"))
	if g.containsCycle() {
		t.Error("no cycle", g)
	}
	n1.addEdge(n2)
	if g.containsCycle() {
		t.Error("no cycle", g)
	}
	n2.addEdge(n1)
	if !g.containsCycle() {
		t.Error("does contain cycle", g)
	}
}
