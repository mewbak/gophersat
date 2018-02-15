package main

import (
	"flag"
	"fmt"
	"os"
	"runtime/debug"
	"sort"
	"strings"

	"github.com/crillab/gophersat/bf"
	"github.com/crillab/gophersat/solver"
)

func main() {
	// defer profile.Start().Stop()
	debug.SetGCPercent(300)
	var (
		verbose bool
		count   bool
	)
	flag.BoolVar(&verbose, "verbose", false, "sets verbose mode on")
	flag.BoolVar(&count, "count", false, "rather than solving the problem, counts the number of models it accepts")
	flag.Parse()
	if len(flag.Args()) != 1 {
		fmt.Fprintf(os.Stderr, "Syntax : %s [options] (file.cnf|file.bf|file.opb)\n", os.Args[0])
		flag.PrintDefaults()
		os.Exit(1)
	}
	path := flag.Args()[0]
	fmt.Printf("c solving %s\n", path)
	if strings.HasSuffix(path, ".bf") {
		if err := parseAndSolveBF(path); err != nil {
			fmt.Fprintf(os.Stderr, "could not parse formula: %v\n", err)
			os.Exit(1)
		}
	} else {
		if pb, printFn, err := parse(flag.Args()[0]); err != nil {
			fmt.Fprintf(os.Stderr, "could not parse problem: %v\n", err)
			os.Exit(1)
		} else if count {
			countModels(pb, verbose)
		} else {
			solve(pb, verbose, printFn)
		}
	}
}

func countModels(pb *solver.Problem, verbose bool) {
	s := solver.New(pb)
	if verbose {
		fmt.Printf("c ======================================================================================\n")
		fmt.Printf("c | Number of clauses   : %9d                                                    |\n", len(pb.Clauses))
		fmt.Printf("c | Number of variables : %9d                                                    |\n", pb.NbVars)
		s.Verbose = true
	}
	models := make(chan solver.ModelMap)
	go s.Enumerate(models, nil)
	nb := 0
	for range models {
		nb++
		if verbose {
			fmt.Printf("c %d models found\n", nb)
		}
	}
	fmt.Println(nb)
}

func solve(pb *solver.Problem, verbose bool, printFn func(chan solver.Result)) {
	s := solver.New(pb)
	if verbose {
		fmt.Printf("c ======================================================================================\n")
		fmt.Printf("c | Number of clauses   : %9d                                                    |\n", len(pb.Clauses))
		fmt.Printf("c | Number of variables : %9d                                                    |\n", pb.NbVars)
		s.Verbose = true
	}
	results := make(chan solver.Result)
	go s.Optimal(results, nil)
	printFn(results)
	if verbose {
		fmt.Printf("c nb conflicts: %d\nc nb restarts: %d\nc nb decisions: %d\n", s.Stats.NbConflicts, s.Stats.NbRestarts, s.Stats.NbDecisions)
		fmt.Printf("c nb unit learned: %d\nc nb binary learned: %d\nc nb learned: %d\n", s.Stats.NbUnitLearned, s.Stats.NbBinaryLearned, s.Stats.NbLearned)
		fmt.Printf("c nb clauses deleted: %d\n", s.Stats.NbDeleted)
	}
}

func parseAndSolveBF(path string) error {
	f, err := os.Open(path)
	if err != nil {
		return fmt.Errorf("could not open %q: %v", path, err)
	}
	defer f.Close()
	form, err := bf.Parse(f)
	if err != nil {
		return fmt.Errorf("could not parse formula in %q: %v", path, err)
	}
	solveBF(form)
	return nil
}

func parse(path string) (pb *solver.Problem, printFn func(chan solver.Result), err error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, nil, fmt.Errorf("could not open %q: %v", path, err)
	}
	defer f.Close()
	if strings.HasSuffix(path, ".bf") {
		_, err := bf.Parse(f)
		if err != nil {
			return nil, nil, fmt.Errorf("could not parse %q: %v", path, err)
		}
		panic("not yet implemented")
	}
	if strings.HasSuffix(path, ".cnf") {
		pb, err := solver.ParseCNF(f)
		if err != nil {
			return nil, nil, fmt.Errorf("could not parse DIMACS file %q: %v", path, err)
		}
		return pb, printDecisionResults, nil
	}
	if strings.HasSuffix(path, ".opb") {
		pb, err := solver.ParseOPB(f)
		if err != nil {
			return nil, nil, fmt.Errorf("could not parse OPB file %q: %v", path, err)
		}
		return pb, printOptimizationResults, nil
	}
	return nil, nil, fmt.Errorf("invalid file format for %q", path)
}

func solveBF(f bf.Formula) {
	if model := bf.Solve(f); model == nil {
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
}

// prints the result to a SAT decision problem in the competition format.
func printDecisionResults(results chan solver.Result) {
	var res solver.Result
	for res = range results {
	}
	switch res.Status {
	case solver.Unsat:
		fmt.Println("s UNSATISFIABLE")
	case solver.Sat:
		fmt.Println("s SATISFIABLE")
		fmt.Printf("v ")
		for i := 1; i <= len(res.Model); i++ {
			val := i
			if !res.Model[i] {
				val = -i
			}
			fmt.Printf("%d ", val)
		}
		fmt.Println("0")
	default:
		fmt.Println("s UNKNOWN")
	}
}

// prints the result to a PB optimization problem in the competition format.
func printOptimizationResults(results chan solver.Result) {
	var res solver.Result
	for res = range results {
		if res.Status == solver.Sat {
			fmt.Printf("o %d\n", res.Weight)
		}
	}
	switch res.Status {
	case solver.Unsat:
		fmt.Println("s UNSATISFIABLE")
	case solver.Sat:
		fmt.Println("s OPTIMUM FOUND")
		fmt.Printf("v ")
		for i := 1; i <= len(res.Model); i++ {
			var val string
			if !res.Model[i] {
				val = fmt.Sprintf("-x%d", i)
			} else {
				val = fmt.Sprintf("x%d", i)
			}
			fmt.Printf("%s ", val)
		}
		fmt.Println()
	default:
		fmt.Println("s UNKNOWN")
	}
}
