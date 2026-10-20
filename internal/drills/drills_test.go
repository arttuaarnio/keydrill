package drills

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

var placeholder = regexp.MustCompile(`\{[a-z0-9]+\}`)

// shells lists the sh-style shells installed here, since players run drills
// in their own.
func shells(t *testing.T) []string {
	var found []string
	for _, sh := range []string{"/bin/sh", "/bin/bash", "/bin/zsh", "/bin/dash"} {
		if _, err := os.Stat(sh); err == nil {
			found = append(found, sh)
		}
	}
	if len(found) == 0 {
		t.Skip("no sh-style shell installed")
	}
	return found
}

func TestDrillData(t *testing.T) {
	topicIDs := map[string]bool{}
	for _, topic := range Shell {
		if topicIDs[topic.ID] {
			t.Errorf("duplicate topic ID %q", topic.ID)
		}
		topicIDs[topic.ID] = true
		if len(topic.Drills) > 0 && (topic.Summary == "" || len(topic.Briefing) == 0) {
			t.Errorf("topic %s has drills but no briefing", topic.ID)
		}

		drillIDs := map[string]bool{}
		for _, d := range topic.Drills {
			if drillIDs[d.ID] {
				t.Errorf("duplicate drill ID %s/%s", topic.ID, d.ID)
			}
			drillIDs[d.ID] = true

			// Two placeholders sharing a name could merge folders.
			seen := map[string]string{}
			for ph, names := range d.Names {
				for _, n := range names {
					if other, ok := seen[n]; ok {
						t.Errorf("%s/%s: %q is in both {%s} and {%s}", topic.ID, d.ID, n, other, ph)
					}
					seen[n] = ph
				}
			}
		}
	}
}

// Every reference solution must solve its drill in exactly par commands,
// with every name, in every shell, and no drill may start out solved.
func TestEveryDrillIsSolvable(t *testing.T) {
	for _, topic := range Shell {
		for _, d := range topic.Drills {
			variants := 1
			for _, names := range d.Names {
				variants = max(variants, len(names))
			}
			for i := range variants {
				filled := d.Fill(func(n int) int { return i % n })
				checkFilled(t, topic.ID+"/"+d.ID, filled)
				for _, sh := range shells(t) {
					t.Run(fmt.Sprintf("%s/%s/%d/%s", topic.ID, d.ID, i, filepath.Base(sh)), func(t *testing.T) {
						t.Setenv("SHELL", sh)
						solve(t, filled)
					})
				}
			}
		}
	}
}

func checkFilled(t *testing.T, name string, d Drill) {
	t.Helper()
	parts := append([]string{d.Task, d.StartDir, d.GoalDir}, d.Solution...)
	for path, content := range d.Start {
		parts = append(parts, path, content)
	}
	if left := placeholder.FindString(strings.Join(parts, "\n")); left != "" {
		t.Errorf("%s: %s has no names", name, left)
	}
}

func solve(t *testing.T, d Drill) {
	t.Helper()
	s, err := d.Sandbox()
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()
	if d.Solved(s) {
		t.Fatal("solved before any command")
	}
	for i, c := range d.Solution {
		res, err := s.Run(c)
		if err != nil || res.ExitCode != 0 {
			t.Fatalf("%q failed: %v %s", c, err, res.Output)
		}
		if last := i == len(d.Solution)-1; d.Solved(s) != last {
			t.Fatalf("after %q: solved = %v, want %v (in %s)", c, !last, last, s.Where())
		}
	}
}

func TestFillUsesTheSameNameEverywhere(t *testing.T) {
	d := Drill{
		Task:     "Get to {a}/{b}.",
		Start:    map[string]string{"{a}/{b}/x.txt": "{a}"},
		GoalDir:  "~/{a}/{b}",
		Solution: []string{"cd {a}/{b}"},
		Names:    map[string][]string{"a": {"one", "two"}, "b": {"three", "four"}},
	}
	got := d.Fill(func(n int) int { return n - 1 })
	if got.Task != "Get to two/four." || got.GoalDir != "~/two/four" || got.Solution[0] != "cd two/four" ||
		got.Start["two/four/x.txt"] != "two" {
		t.Errorf("Fill = %+v", got)
	}
	if d.Solution[0] != "cd {a}/{b}" {
		t.Error("Fill changed the original drill")
	}
}
