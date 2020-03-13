package middleware

import (
	"fmt"
	"strings"

	"github.com/maruel/panicparse/stack"
	"github.com/mgutz/ansi"
)

// StackPalette defines the color used.
//
// An empty object StackPalette{} can be used to disable coloring.
type StackPalette struct {
	EOLReset string

	// Routine header.
	RoutineFirst string // The first routine printed.
	Routine      string // Following routines.
	CreatedBy    string

	// Call line.
	Package            string
	SrcFile            string
	FuncStdLib         string
	FuncStdLibExported string
	FuncMain           string
	FuncOther          string
	FuncOtherExported  string
	Arguments          string
}

// CalcLengths returns the maximum length of the source lines and package names.
func CalcLengths(buckets []*stack.Bucket, fullPath bool) (int, int) {
	srcLen := 0
	pkgLen := 0
	for _, bucket := range buckets {
		for _, line := range bucket.Signature.Stack.Calls {
			l := 0
			if fullPath {
				l = len(line.FullSrcLine())
			} else {
				l = len(line.SrcLine())
			}
			if l > srcLen {
				srcLen = l
			}
			l = len(line.Func.PkgName())
			if l > pkgLen {
				pkgLen = l
			}
		}
	}
	return srcLen, pkgLen
}

// functionColor returns the color to be used for the function name based on
// the type of package the function is in.
func (p *StackPalette) functionColor(line *stack.Call) string {
	if line.IsStdlib {
		if line.Func.IsExported() {
			return p.FuncStdLibExported
		}
		return p.FuncStdLib
	} else if line.IsPkgMain() {
		return p.FuncMain
	} else if line.Func.IsExported() {
		return p.FuncOtherExported
	}
	return p.FuncOther
}

// routineColor returns the color for the header of the goroutines bucket.
func (p *StackPalette) routineColor(bucket *stack.Bucket, multipleBuckets bool) string {
	if bucket.First && multipleBuckets {
		return p.RoutineFirst
	}
	return p.Routine
}

// BucketHeader prints the header of a goroutine signature.
func (p *StackPalette) BucketHeader(bucket *stack.Bucket, fullPath, multipleBuckets bool) string {
	extra := ""
	if s := bucket.SleepString(); s != "" {
		extra += " [" + s + "]"
	}
	if bucket.Locked {
		extra += " [locked]"
	}
	if c := bucket.CreatedByString(fullPath); c != "" {
		extra += p.CreatedBy + " [Created by " + c + "]"
	}
	return fmt.Sprintf(
		"%s%d: %s%s%s\n",
		p.routineColor(bucket, multipleBuckets), len(bucket.IDs),
		bucket.State, extra,
		p.EOLReset)
}

// callLine prints one stack line.
func (p *StackPalette) callLine(line *stack.Call, srcLen, pkgLen int, fullPath bool) string {
	src := ""
	if fullPath {
		src = line.FullSrcLine()
	} else {
		src = line.SrcLine()
	}
	return fmt.Sprintf(
		"    %s%-*s %s%-*s %s%s%s(%s)%s",
		p.Package, pkgLen, line.Func.PkgName(),
		p.SrcFile, srcLen, src,
		p.functionColor(line), line.Func.Name(),
		p.Arguments, &line.Args,
		p.EOLReset)
}

// StackLines prints one complete stack trace, without the header.
func (p *StackPalette) StackLines(signature *stack.Signature, srcLen, pkgLen int, fullPath bool) string {
	var out []string
	for i := range signature.Stack.Calls {
		call := &signature.Stack.Calls[i]
		switch call.Func.Raw {
		case "runtime/debug.Stack",
			"github.com/moisespsena-go/tracederror.New",
			"github.com/moisespsena-go/tracederror.Wrap",
			"github.com/moisespsena-go/tracederror.Traced",
			"github.com/moisespsena-go/tracederror.TracedWrap":
			continue
		}
		out = append(out, p.callLine(call, srcLen, pkgLen, fullPath))
	}
	if signature.Stack.Elided {
		out = append(out, "    (...)")
	}
	return strings.Join(out, "\n") + "\n"
}

// resetFG is similar to ansi.Reset except that it doesn't reset the
// background color, only the foreground color and the style.
//
// That much for the "ansi" abstraction layer...
const resetFG = ansi.DefaultFG + "\033[m"

// defaultPalette is the default recommended palette.
var defaultStackPalette = StackPalette{
	EOLReset:           resetFG,
	RoutineFirst:       ansi.ColorCode("magenta+b"),
	CreatedBy:          ansi.LightBlack,
	Package:            ansi.ColorCode("default+b"),
	SrcFile:            resetFG,
	FuncStdLib:         ansi.Green,
	FuncStdLibExported: ansi.ColorCode("green+b"),
	FuncMain:           ansi.ColorCode("yellow+b"),
	FuncOther:          ansi.Red,
	FuncOtherExported:  ansi.ColorCode("red+b"),
	Arguments:          resetFG,
}
