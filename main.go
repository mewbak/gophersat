package main

import (
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"runtime/debug"
	"sort"
	"strings"

	"github.com/crillab/gophersat/bf"
	"github.com/crillab/gophersat/solver"
)

func main() {
	debug.SetGCPercent(300)
	if len(os.Args) > 2 {
		fmt.Fprintf(os.Stderr, "Syntax : %s [file.cnf|file.bf]\n", os.Args[0])
		os.Exit(1)
	}
	f := os.Stdin
	if len(os.Args) == 2 {
		var err error
		f, err = os.Open(os.Args[1])
		if err != nil {
			fmt.Fprintf(os.Stderr, "Could not open %s: %v\n", os.Args[1], err.Error())
			os.Exit(1)
		}
		defer f.Close()
	}
	if err := parseAndSolve(f); err != nil {
		fmt.Fprintf(os.Stdout, "could not solve problem: %v\n", err)
		os.Exit(1)
	}
}

// tryReader is a reader that read lines from a reader and can be rewinded.
type tryReader struct {
	nextLines io.Reader
	rewinded  bool
	content   []byte
}

func newTryReader(r io.Reader) *tryReader {
	return &tryReader{nextLines: r}
}

func (r *tryReader) rewind() {
	r.rewinded = true
}

func (r *tryReader) Read(p []byte) (n int, err error) {
	if !r.rewinded {
		n, err = r.nextLines.Read(p)
		if err != nil {
			r.content = append(r.content, p[:n]...)
		}
		return n, err
	}
	length := len(r.content)
	if length >= len(p) {
		copy(p, r.content[:len(p)])
		r.content = r.content[len(p):]
		return len(p), nil
	}
	copy(p, r.content)
	r.content = nil
	n, err = r.nextLines.Read(p[:length])
	return n + length, err
}

func parseAndSolve(r io.Reader) error {
	content, err := ioutil.ReadAll(r)
	if err != nil {
		return fmt.Errorf("could not read data: %v", err)
	}
	r2 := strings.NewReader(string(content))
	f, errBF := bf.Parse(r2)
	if errBF == nil {
		return solveBF(f)
	}
	r2 = strings.NewReader(string(content))
	pb, errCNF := solver.ParseCNF(r2)
	if errCNF != nil {
		return fmt.Errorf("could not parse content as DIMACS (%v) nor as boolean formula (%v)", errCNF, errBF)
	}
	return solveCNF(pb)
}

func solveCNF(pb *solver.Problem) error {
	fmt.Printf("c ======================================================================================\n")
	fmt.Printf("c | Number of clauses   : %9d                                                    |\n", len(pb.Clauses))
	fmt.Printf("c | Number of variables : %9d                                                    |\n", pb.NbVars)
	s := solver.New(pb)
	s.Verbose = true
	s.Solve()
	fmt.Printf("c nb conflicts: %d\nc nb restarts: %d\nc nb decisions: %d\n", s.Stats.NbConflicts, s.Stats.NbRestarts, s.Stats.NbDecisions)
	fmt.Printf("c nb unit learned: %d\nc nb binary learned: %d\nc nb learned: %d\n", s.Stats.NbUnitLearned, s.Stats.NbBinaryLearned, s.Stats.NbLearned)
	fmt.Printf("c nb clauses deleted: %d\n", s.Stats.NbDeleted)
	s.OutputModel()
	return nil
}

func solveBF(f bf.Formula) error {
	sat, model, err := bf.Solve(f)
	if err != nil {
		return fmt.Errorf("could not solve formula %q: %v", f, err)
	}
	if !sat {
		fmt.Println("UNSATISFIABLE")
	} else {
		fmt.Println("SATISFIABLE")
		keys := make(sort.StringSlice, 0, len(model))
		for k := range model {
			keys = append(keys, k)
		}
		sort.Sort(keys)
		for _, k := range keys {
			fmt.Printf("%s: %t\n", k, model[k])
		}
	}
	return nil
}
