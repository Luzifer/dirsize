package main

import (
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"os"
	"path"
	"sort"
	"sync"

	"github.com/Luzifer/rconfig"
)

const (
	BYTE     = 1.0
	KILOBYTE = 1024 * BYTE
	MEGABYTE = 1024 * KILOBYTE
)

var (
	cfg struct {
		Align          bool   `flag:"align,a" default:"true" description:"Align sizes in one column"`
		IgnoreDotFiles bool   `flag:"ignore-dotfiles" default:"false" description:"Do not count directories / files starting with a dot"`
		IgnoreErrors   bool   `flag:"ignore-errors" default:"false" description:"Do not break when encountering errors (results will be incorrect)"`
		Output         string `flag:"output,o" default:"-" description:"Filename to print the list to (or - for stdout)"`
		Sum            bool   `flag:"sum,s" default:"false" description:"Sumarize only instead of printing all directories"`
		VersionAndExit bool   `flag:"version" default:"false" description:"Print version information and exit"`
	}

	dirSizes     map[string]int64
	dirSizesLock sync.Mutex
	wg           sync.WaitGroup

	version = "dev"
)

func init() {
	if err := rconfig.ParseAndValidate(&cfg); err != nil {
		log.Fatalf("Unable to parse CLI parameters: %s", err)
	}

	if cfg.VersionAndExit {
		fmt.Printf("dirsize %s\n", version)
		os.Exit(0)
	}
}

func main() {
	startDir := "."
	if len(rconfig.Args()) > 1 {
		startDir = rconfig.Args()[1]
	}

	var out io.WriteCloser = newNopWCloser(os.Stdout)
	var err error
	if cfg.Output != "-" {
		out, err = os.Create(cfg.Output)
		if err != nil {
			log.Fatalf("Unable to open output file: %s", err)
		}
	}

	dirSizes = make(map[string]int64)

	if _, err = scanDirectory(startDir); err != nil {
		log.Fatalf("Unable to scan directories: %s", err)
	}

	if cfg.Sum {
		fmt.Fprintln(out, fmtMegs(dirSizes[startDir]))
		return
	}

	paths := []string{}
	maxLen := 0
	for k := range dirSizes {
		paths = append(paths, k)
		if len(k) > maxLen {
			maxLen = len(k)
		}
	}
	sort.Sort(sort.Reverse(sort.StringSlice(paths)))

	if !cfg.Align {
		maxLen = 0
	}

	lenStr := fmt.Sprintf("%%-%ds -- %%s\n", maxLen+1)
	for _, p := range paths {
		fmt.Fprintf(out, lenStr, p, fmtMegs(dirSizes[p]))
	}
}

func scanDirectory(dir string) (int64, error) {
	content, err := ioutil.ReadDir(dir)
	if err != nil {
		if cfg.IgnoreErrors {
			return 0, nil
		}
		return 0, err
	}

	var sizeByte int64
	for _, c := range content {
		if cfg.IgnoreDotFiles && c.Name()[0] == '.' {
			continue
		}

		if c.IsDir() {
			ds, err := scanDirectory(path.Join(dir, c.Name()))
			if err != nil {
				return 0, err
			}
			sizeByte += ds
			continue
		}
		sizeByte += c.Size()
	}

	dirSizes[dir] = sizeByte
	return sizeByte, nil
}

func fmtMegs(b int64) string {
	f := "%.2f MB"
	if cfg.Align {
		f = "%8.2f MB"
	}
	return fmt.Sprintf(f, float64(b)/MEGABYTE)
}

type nopWCloser struct {
	io.Writer
}

func newNopWCloser(w io.Writer) io.WriteCloser {
	return nopWCloser{w}
}

func (n nopWCloser) Close() error { return nil }
