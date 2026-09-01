package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"

	"github.com/kimjooyoon/gooo-measurement-boundary-projector/internal/projector"
)

func main() {
	if len(os.Args) < 2 {
		fatalf("usage: measurement-boundary-projector compile|collect|evaluate|report|conformance|inventory|run")
	}
	var err error
	switch os.Args[1] {
	case "compile":
		err = compile(os.Args[2:])
	case "collect":
		err = collect(os.Args[2:])
	case "evaluate":
		err = evaluate(os.Args[2:])
	case "report":
		err = report(os.Args[2:])
	case "conformance":
		err = conformance(os.Args[2:])
	case "inventory":
		err = inventory(os.Args[2:])
	case "run":
		err = run(os.Args[2:])
	default:
		err = fmt.Errorf("unknown command %q", os.Args[1])
	}
	if err != nil {
		fatalf("%v", err)
	}
}

func compile(args []string) error {
	flags := flag.NewFlagSet("compile", flag.ContinueOnError)
	source := flags.String("source", "", "input .gooo file")
	out := flags.String("out", "", "caller-owned output directory")
	if err := flags.Parse(args); err != nil {
		return err
	}
	if *source == "" || *out == "" {
		return fmt.Errorf("source and out are required")
	}
	_, err := projector.Compile(*source, *out)
	return err
}

func collect(args []string) error {
	flags := flag.NewFlagSet("collect", flag.ContinueOnError)
	irPath := flags.String("ir", "", "semantic-ir.json")
	fixture := flags.String("fixture", "", "deterministic fixture")
	out := flags.String("out", "", "caller-owned output directory")
	if err := flags.Parse(args); err != nil {
		return err
	}
	if *irPath == "" || *fixture == "" || *out == "" {
		return fmt.Errorf("ir, fixture, and out are required")
	}
	var ir projector.SemanticIR
	if err := projector.LoadJSON(*irPath, &ir); err != nil {
		return err
	}
	_, _, err := projector.CollectFixture(ir, *fixture, *out)
	return err
}

func evaluate(args []string) error {
	flags := flag.NewFlagSet("evaluate", flag.ContinueOnError)
	irPath := flags.String("ir", "", "semantic-ir.json")
	collectionPath := flags.String("collection", "", "collection.json")
	out := flags.String("out", "", "evaluation.json")
	if err := flags.Parse(args); err != nil {
		return err
	}
	if *irPath == "" || *collectionPath == "" || *out == "" {
		return fmt.Errorf("ir, collection, and out are required")
	}
	var ir projector.SemanticIR
	if err := projector.LoadJSON(*irPath, &ir); err != nil {
		return err
	}
	var collection projector.Collection
	if err := projector.LoadJSON(*collectionPath, &collection); err != nil {
		return err
	}
	_, err := projector.Evaluate(ir, collection, *out)
	return err
}

func report(args []string) error {
	flags := flag.NewFlagSet("report", flag.ContinueOnError)
	evaluationPath := flags.String("evaluation", "", "evaluation.json")
	out := flags.String("out", "", "human-report.md")
	if err := flags.Parse(args); err != nil {
		return err
	}
	if *evaluationPath == "" || *out == "" {
		return fmt.Errorf("evaluation and out are required")
	}
	var evaluation projector.Evaluation
	if err := projector.LoadJSON(*evaluationPath, &evaluation); err != nil {
		return err
	}
	return projector.WriteText(*out, projector.RenderHumanReport(evaluation))
}

func conformance(args []string) error {
	flags := flag.NewFlagSet("conformance", flag.ContinueOnError)
	source := flags.String("source", "examples/measurement-boundary.gooo", "input .gooo file")
	corpus := flags.String("corpus", "fixtures/corpus.json", "canonical fixture corpus")
	out := flags.String("out", "", "caller-owned output directory")
	if err := flags.Parse(args); err != nil {
		return err
	}
	if *out == "" {
		return fmt.Errorf("out is required")
	}
	_, err := projector.RunConformance(*source, *corpus, *out)
	return err
}

func inventory(args []string) error {
	flags := flag.NewFlagSet("inventory", flag.ContinueOnError)
	root := flags.String("root", ".", "repository root")
	generated := flags.String("generated-root", "", "caller-owned generated output root")
	out := flags.String("out", "", "inventory.json")
	if err := flags.Parse(args); err != nil {
		return err
	}
	if *out == "" {
		return fmt.Errorf("out is required")
	}
	value, err := projector.InventoryFor(*root, *generated)
	if err != nil {
		return err
	}
	return projector.WriteJSON(*out, value)
}

func run(args []string) error {
	flags := flag.NewFlagSet("run", flag.ContinueOnError)
	source := flags.String("source", "", "input .gooo file")
	fixture := flags.String("fixture", "", "deterministic fixture")
	out := flags.String("out", "", "caller-owned output directory")
	if err := flags.Parse(args); err != nil {
		return err
	}
	if *source == "" || *fixture == "" || *out == "" {
		return fmt.Errorf("source, fixture, and out are required")
	}
	if _, err := projector.Compile(*source, filepath.Join(*out, "compile")); err != nil {
		return err
	}
	var ir projector.SemanticIR
	if err := projector.LoadJSON(filepath.Join(*out, "compile", "semantic-ir.json"), &ir); err != nil {
		return err
	}
	collection, _, err := projector.CollectFixture(ir, *fixture, filepath.Join(*out, "collection"))
	if err != nil {
		return err
	}
	evaluation, err := projector.Evaluate(ir, collection, filepath.Join(*out, "evaluation.json"))
	if err != nil {
		return err
	}
	return projector.WriteText(filepath.Join(*out, "human-report.md"), projector.RenderHumanReport(evaluation))
}

func fatalf(format string, args ...any) {
	fmt.Fprintf(os.Stderr, format+"\n", args...)
	os.Exit(1)
}
