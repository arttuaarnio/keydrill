// Package drills holds the shell course: topics, and the drills inside them.
package drills

import (
	"fmt"
	"strings"

	"github.com/arttuaarnio/keydrill/internal/sandbox"
)

// Topic is one group of commands, taught with a briefing and a run of
// drills that get harder. The last drill is the exam.
type Topic struct {
	ID       string // stable key for saved progress
	Title    string
	Commands string // what it teaches, as shown in the topic list
	Summary  string // one line under the briefing title
	Briefing []Usage
	Note     string  // closing line of the briefing
	Drills   []Drill // empty while the topic is still being written
}

// Usage is one line of a briefing: an example and what it does.
type Usage struct{ Example, Does string }

// Drill is one task, checked by where the player ends up.
//
// Task, Start, StartDir, GoalDir and Solution may contain placeholders like
// {a}. Fill swaps each one for a name from Names, so every play of a drill
// uses different names.
type Drill struct {
	ID       string // stable key for saved progress, unique within its topic
	Title    string
	Task     string
	Start    sandbox.Files
	StartDir string   // where the player starts, like "~/work"; empty means ~
	GoalDir  string   // the folder to end in
	Solution []string // a reference solution, shown by hint; its length is par
	Names    map[string][]string
}

// Fill returns the drill with every placeholder replaced by one of its
// names. pick(n) chooses which of the n names to use.
func (d Drill) Fill(pick func(n int) int) Drill {
	var pairs []string
	for name, options := range d.Names {
		pairs = append(pairs, "{"+name+"}", options[pick(len(options))])
	}
	r := strings.NewReplacer(pairs...)

	d.Task, d.StartDir, d.GoalDir = r.Replace(d.Task), r.Replace(d.StartDir), r.Replace(d.GoalDir)
	start := sandbox.Files{}
	for path, content := range d.Start {
		start[r.Replace(path)] = r.Replace(content)
	}
	d.Start = start
	solution := make([]string, len(d.Solution))
	for i, c := range d.Solution {
		solution[i] = r.Replace(c)
	}
	d.Solution = solution
	return d
}

// Par is the number of commands a good solution needs.
func (d Drill) Par() int { return len(d.Solution) }

// Sandbox creates a fresh sandbox for a filled drill, standing in StartDir.
func (d Drill) Sandbox() (*sandbox.Sandbox, error) {
	s, err := sandbox.New(d.Start)
	if err != nil || d.StartDir == "" {
		return s, err
	}
	if _, err := s.Run("cd " + d.StartDir); err != nil || s.Where() != d.StartDir {
		s.Close()
		return nil, fmt.Errorf("drill %s: can't start in %s: %v", d.ID, d.StartDir, err)
	}
	return s, nil
}

// Solved reports whether the player has reached the goal.
func (d Drill) Solved(s *sandbox.Sandbox) bool { return s.Where() == d.GoalDir }

// Shell is the course, easiest topic first.
var Shell = []Topic{
	movingAround,
	{ID: "making-things", Title: "Making things", Commands: "mkdir touch"},
	{ID: "mv", Title: "Moving and renaming", Commands: "mv"},
	{ID: "cp", Title: "Copying", Commands: "cp"},
	{ID: "rm", Title: "Deleting", Commands: "rm rmdir"},
	{ID: "wildcards", Title: "Wildcards", Commands: "* ? [ ] { }"},
	{ID: "redirects", Title: "Reading and redirecting", Commands: "cat head tail wc > >>"},
	{ID: "grep", Title: "Searching text", Commands: "grep"},
	{ID: "find", Title: "Finding files", Commands: "find"},
	{ID: "pipes", Title: "Pipes", Commands: "| sort uniq cut"},
	{ID: "xargs", Title: "xargs", Commands: "xargs"},
}
