/* Built : 2012-09-04 11:38:11.577774 +0000 UTC */
//-------------------------------------------------------------------
// Auto generated code, but you are encouraged to modify it ☺
// Manual: http://godag.googlecode.com
//-------------------------------------------------------------------

package main

import (
	"bytes"
	"compress/gzip"
	"flag"
	"fmt"
	"io"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"runtime"
	"strings"
)

//-------------------------------------------------------------------
//  User defined targets: flexible pure go way to control build
//-------------------------------------------------------------------

type Target struct {
	desc  string // description of target
	first func() // this is called prior to building
	last  func() // this is called after building
}

/********************************************************************

 Code inside playground is NOT auto generated, each time you run
 this command:

     gd -gdmk=somefilename.go

  this happens:

  if (somefilename.go already exists) {
      transfer playground from somefilename.go to new version
      of somefilename.go, i.e. you never loose the targets you
      create inside the playground..
  }else{
      write somefilename.go with example targets
  }

********************************************************************/

// PLAYGROUND START

var targets = map[string]*Target{
	"full": &Target{
		desc:  "compile all packages (ignore !modified)",
		first: func() { oldPkgFound = true },
		last:  func() { println("hello world"); os.Exit(0) },
	},
}

// PLAYGROUND STOP

//-------------------------------------------------------------------
// Execute user defined targets
//-------------------------------------------------------------------

func doFirst() {
	args := flag.Args()
	for i := 0; i < len(args); i++ {
		target, ok := targets[args[i]]
		if ok && target.first != nil {
			target.first()
		}
	}
}

func doLast() {
	args := flag.Args()
	for i := 0; i < len(args); i++ {
		target, ok := targets[args[i]]
		if ok && target.last != nil {
			target.last()
		}
	}
}

//-------------------------------------------------------------------
// Simple way to turn print statements on/off
//-------------------------------------------------------------------

type Say bool

func (s Say) Printf(frmt string, args ...interface{}) {
	if bool(s) {
		fmt.Printf(frmt, args...)
	}
}

func (s Say) Println(args ...interface{}) {
	if bool(s) {
		fmt.Println(args...)
	}
}

//-------------------------------------------------------------------
// Initialize variables, flags and so on
//-------------------------------------------------------------------

var (
	compiler    = ""
	linker      = ""
	suffix      = ""
	backend     = ""
	root        = ""
	output      = ""
	match       = ""
	help        = true
	list        = false
	quiet       = false
	external    = false
	clean       = true
	oldPkgFound = false
)

var includeDir = "_obj"

var say = Say(true)

func init() {

	flag.StringVar(&backend, "backend", "gc", "select from [gc,gccgo,express]")
	flag.StringVar(&backend, "B", "gc", "alias for --backend option")
	flag.StringVar(&root, "I", "", "import package directory")
	flag.StringVar(&match, "M", "", "regex to match main package")
	flag.StringVar(&match, "main", "", "regex to match main package")
	flag.StringVar(&output, "o", "", "link main package -> output")
	flag.StringVar(&output, "output", "", "link main package -> output")
	flag.BoolVar(&external, "external", false, "external dependencies")
	flag.BoolVar(&external, "e", false, "external dependencies")
	flag.BoolVar(&quiet, "q", false, "don't print anything but errors")
	flag.BoolVar(&quiet, "quiet", false, "don't print anything but errors")
	flag.BoolVar(&help, "h", false, "help message")
	flag.BoolVar(&help, "help", false, "help message")
	flag.BoolVar(&clean, "clean", false, "delete objects")
	flag.BoolVar(&clean, "c", false, "delete objects")
	flag.BoolVar(&list, "list", false, "list targets for bash autocomplete")

	flag.Usage = func() {
		fmt.Println("\n mk.go - makefile in pure go\n")
		fmt.Println(" usage: go run mk.go [OPTIONS] [TARGET]\n")
		fmt.Println(" options:\n")
		fmt.Println("  -h --help         print this menu and exit")
		fmt.Println("  -B --backend      choose backend [gc,gccgo,express]")
		fmt.Println("  -o --output       link main package -> output")
		fmt.Println("  -M --main         regex to match main package")
		fmt.Println("  -c --clean        delete object files")
		fmt.Println("  -q --quiet        quiet unless errors occur")
		fmt.Println("  -e --external     go install external dependencies")
		fmt.Println("  -I                import package directory\n")

		if len(targets) > 0 {
			fmt.Println(" targets:\n")
			for k, v := range targets {
				fmt.Printf("  %-11s  =>   %s\n", k, v.desc)
			}
			fmt.Println("")
		}
	}
}

func initBackend() {

	switch backend {
	case "gc":
		n := archNum()
		compiler, linker, suffix = n+"g", n+"l", "."+n
	case "gccgo", "gcc":
		compiler, linker, suffix = "gccgo", "gccgo", ".o"
		backend = "gccgo"
	case "express":
		compiler, linker, suffix = "vmgc", "vmld", ".vmo"
	default:
		log.Fatalf("[ERROR] unknown backend: %s\n", backend)
	}

	if backend == "gc" {
		goroot := GOROOT()
		goos := GOOS()
		goarch := GOARCH()
		stub := goos + "_" + goarch
		compiler = filepath.Join(goroot, "pkg", "tool", stub, compiler)
		linker = filepath.Join(goroot, "pkg", "tool", stub, linker)
	}
}

func archNum() (n string) {

	goarch := os.Getenv("GOARCH")
	if goarch == "" {
		goarch = runtime.GOARCH
	}

	switch goarch {
	case "386":
		n = "8"
	case "arm":
		n = "5"
	case "amd64":
		n = "6"
	default:
		log.Fatalf("[ERROR] unknown GOARCH: %s\n", goarch)
	}
	return
}

//-------------------------------------------------------------------
// External dependencies
//-------------------------------------------------------------------

var alien = []string{}

//-------------------------------------------------------------------
// Functions to build/delete project
//-------------------------------------------------------------------

func osify(pkgs []*Package) {

	for j := 0; j < len(pkgs); j++ {

		if pkgs[j].osified {
			break
		}

		pkgs[j].osified = true
		pkgs[j].output = filepath.FromSlash(pkgs[j].output) + suffix
		for i := 0; i < len(pkgs[j].files); i++ {
			pkgs[j].files[i] = filepath.FromSlash(pkgs[j].files[i])
		}
	}
}

func mkdirs(pkgs []*Package) {
	for i := 0; i < len(pkgs); i++ {
		d, _ := filepath.Split(pkgs[i].output)
		if d != "" && !isDir(d) {
			e := os.MkdirAll(d, 0777)
			if e != nil {
				log.Fatalf("[ERROR] %s\n", e)
			}
		}
	}
}

func compile(pkgs []*Package) {

	osify(pkgs)
	mkdirs(pkgs)

	for i := 0; i < len(pkgs); i++ {
		if oldPkgFound || !pkgs[i].up2date() {
			say.Printf("compiling: %s\n", pkgs[i].full)
			pkgs[i].compile()
			oldPkgFound = true
		} else {
			say.Printf("up 2 date: %s\n", pkgs[i].full)
		}
	}
}

func delete(pkgs []*Package) {

	osify(pkgs)

	for j := 0; j < len(pkgs); j++ {
		if isFile(pkgs[j].output) {
			say.Printf("rm: %s\n", pkgs[j].output)
			e := os.Remove(pkgs[j].output)
			if e != nil {
				log.Fatalf("[ERROR] failed to remove: %s\n", pkgs[j].output)
			}
		}
	}

	if emptyDir(includeDir) {
		say.Printf("rm: %s\n", includeDir)
		e := os.RemoveAll(includeDir)
		if e != nil {
			log.Fatalf("[ERROR] failed to remove: %s\n", includeDir)
		}
	}

}

func link(pkgs []*Package) {

	if output == "" {
		return
	}

	var mainPackage *Package
	var mainPkgs = make([]*Package, 0)

	for i := 0; i < len(pkgs); i++ {
		if pkgs[i].name == "main" {
			mainPkgs = append(mainPkgs, pkgs[i])
			mainPackage = pkgs[i]
		}
	}

	switch len(mainPkgs) {
	case 0:
		log.Fatalf("[ERROR] no main package found\n")
	case 1: // do nothing... this is good
	default:
		mainPackage = mainChoice(mainPkgs)
	}

	if !oldPkgFound && isFile(output) {
		pkgLastTs, _ := timestamp(pkgs[len(pkgs)-1].output)
		outputTs, ok := timestamp(output)
		if ok && outputTs > pkgLastTs {
			say.Printf("up 2 date: %s\n", output)
			return
		}
	}

	say.Printf("linking  : %s\n", output)

	argv := make([]string, 0)
	argv = append(argv, linker)

	if backend != "gccgo" {

		argv = append(argv, "-L")
		argv = append(argv, includeDir)

		if root != "" {
			argv = append(argv, "-L")
			argv = append(argv, root)
		}

		// GOPATH
		gopathInc := gopathDirs()
		if len(gopathInc) > 0 {
			for i := 0; i < len(gopathInc); i++ {
				argv = append(argv, "-L")
				argv = append(argv, gopathInc[i])
			}
		}
	}

	argv = append(argv, "-o")
	argv = append(argv, output)

	if backend == "gccgo" {
		for i := 0; i < len(pkgs); i++ {
			argv = append(argv, pkgs[i].output)
		}
		if root != "" {
			filter := func(s string) bool {
				return strings.HasSuffix(s, ".o")
			}
			files := PathWalk(root, filter)
			for i := 0; i < len(files); i++ {
				argv = append(argv, files[i])
			}
		}
	} else {
		argv = append(argv, mainPackage.output)
	}

	run(argv)
}

func mainChoice(pkgs []*Package) *Package {

	var cnt, choice int

	for i := 0; i < len(pkgs); i++ {
		ok, _ := regexp.MatchString(match, pkgs[i].full)
		if ok {
			cnt++
			choice = i
		}
	}

	if cnt == 1 {
		return pkgs[choice]
	}

	fmt.Println("\n More than one main package found\n")

	for i := 0; i < len(pkgs); i++ {
		fmt.Printf(" type %2d  for: %s\n", i, pkgs[i].full)
	}

	fmt.Printf("\n type your choice: ")

	n, e := fmt.Scanf("%d", &choice)

	if e != nil {
		log.Fatalf("%s\n", e)
	}
	if n != 1 {
		log.Fatal("failed to read input\n")
	}

	if choice >= len(pkgs) || choice < 0 {
		log.Fatalf(" bad choice: %d\n", choice)
	}

	fmt.Printf(" chosen main-package: %s\n\n", pkgs[choice].full)

	return pkgs[choice]
}

func goinstall() {

	argv := make([]string, 5)
	argv[0] = "go"
	argv[1] = "get"
	argv[2] = "-u"
	argv[3] = "-a"

	for i := 0; i < len(alien); i++ {
		say.Printf("go get: %s\n", alien[i])
		argv[4] = alien[i]
		run(argv)
	}
}

//-------------------------------------------------------------------
// Utility types and functions
//-------------------------------------------------------------------

func PathWalk(root string, ok func(s string) bool) (files []string) {

	fn := func(p string, d os.FileInfo, e error) error {

		if !d.IsDir() && ok(p) {
			files = append(files, p)
		}

		return e
	}

	filepath.Walk(root, fn)

	return files
}

func emptyDir(pathname string) bool {
	if !isDir(pathname) {
		return false
	}
	fn := func(s string) bool { return true }
	return len(PathWalk(pathname, fn)) == 0
}

func isDir(pathname string) bool {
	fileInfo, err := os.Stat(pathname)
	if err == nil && fileInfo.IsDir() {
		return true
	}
	return false
}

func timestamp(s string) (int64, bool) {
	fileInfo, e := os.Stat(s)
	if e == nil {
		return fileInfo.ModTime().UnixNano(), true
	}
	return 0, false
}

func run(argv []string) {

	cmd := exec.Command(argv[0], argv[1:]...)

	// pass-through
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Stdin = os.Stdin

	err := cmd.Start()

	if err != nil {
		log.Fatalf("[ERROR] %s\n", err)
	}

	err = cmd.Wait()

	if err != nil {
		log.Fatalf("[ERROR] %s\n", err)
	}

}

func isFile(pathname string) bool {
	fileInfo, err := os.Stat(pathname)
	if err == nil {
		if fileInfo.Mode()&os.ModeType != 0 {
			return false
		}
		return true
	}
	return false
}

func quitter(e error) {
	if e != nil {
		log.Fatalf("[ERROR] %s\n", e)
	}
}

func copyGzipStringBuffer(from string, to string, gzipFile bool) {
	buf := bytes.NewBufferString(from)
	copyGzipReader(buf, to, gzipFile)
}

func copyGzipByteBuffer(from []byte, to string, gzipFile bool) {
	buf := bytes.NewBuffer(from)
	copyGzipReader(buf, to, gzipFile)
}

func copyGzip(from, to string, gzipFile bool) {

	var err error
	var fromFile *os.File

	fromFile, err = os.Open(from)
	quitter(err)

	defer fromFile.Close()

	copyGzipReader(fromFile, to, gzipFile)
}

func copyGzipReader(fromReader io.Reader, to string, gzipFile bool) {

	var err error
	var toFile io.WriteCloser

	toFile, err = os.Create(to)
	quitter(err)

	if gzipFile {
		toFile, err = gzip.NewWriterLevel(toFile, gzip.BestCompression)
		quitter(err)
	}

	defer toFile.Close()

	_, err = io.Copy(toFile, fromReader)

	quitter(err)
}

func gopathDirs() (paths []string) {

	var (
		stub   string
		gopath []string
	)

	gopath = GOPATH()

	if len(gopath) > 0 {

		if backend == "gc" {
			stub = GOOS() + "_" + GOARCH()
		} else {
			stub = "gccgo"
		} // should do something for express later perhaps

		for _, gp := range gopath {
			paths = append(paths, filepath.Join(gp, "pkg", stub))
		}
	}

	return
}

func GOPATH() (gp []string) {
	p := os.Getenv("GOPATH")
	if p != "" {
		gp = strings.Split(p, string(os.PathListSeparator))
	}
	return
}

func GOROOT() (r string) {
	r = os.Getenv("GOROOT")
	if r == "" {
		r = runtime.GOROOT()
	}
	return
}

func GOARCH() (a string) {
	a = os.Getenv("GOARCH")
	if a == "" {
		a = runtime.GOARCH
	}
	return
}

func GOOS() (o string) {
	o = os.Getenv("GOOS")
	if o == "" {
		o = runtime.GOOS
	}
	return
}

//-------------------------------------------------------------------
// Package definition
//-------------------------------------------------------------------

type Package struct {
	name, full, output string
	osified            bool
	files              []string
}

func (p *Package) up2date() bool {
	mtime, ok := timestamp(p.output)
	if !ok {
		return false
	}
	for i := 0; i < len(p.files); i++ {
		fmtime, ok := timestamp(p.files[i])
		if !ok {
			log.Fatalf("file missing: %s\n", p.files[i])
		}
		if fmtime > mtime {
			return false
		}
	}
	return true
}

func (p *Package) compile() {

	argv := make([]string, 0)
	argv = append(argv, compiler)
	argv = append(argv, "-I")
	argv = append(argv, includeDir)

	// GOPATH
	gopathInc := gopathDirs()
	if len(gopathInc) > 0 {
		for i := 0; i < len(gopathInc); i++ {
			argv = append(argv, "-I")
			argv = append(argv, gopathInc[i])
		}
	}

	if root != "" {
		argv = append(argv, "-I")
		argv = append(argv, root)
	}
	if backend == "gccgo" {
		argv = append(argv, "-c")
	}
	argv = append(argv, "-o")
	argv = append(argv, p.output)
	argv = append(argv, p.files...)

	run(argv)

}

//-------------------------------------------------------------------
// Package info collected by godag
//-------------------------------------------------------------------

var packages = []*Package{
	&Package{
		name:   "main",
		full:   "main",
		output: "_obj/main",
		files:  []string{"src/main.go"},
	},
}

//-------------------------------------------------------------------
// Main - note flags are parsed before we doFirst, i.e. command line
//        options/arguments can be overridden in doFirst targets
//-------------------------------------------------------------------

func main() {

	flag.Parse()
	initBackend() // gc/gcc/express

	if quiet {
		say = Say(false)
	}

	doFirst()
	defer doLast()

	switch {
	case help:
		flag.Usage()
	case clean:
		delete(packages)
	case external:
		goinstall()
	default:
		compile(packages)
		link(packages)
	}

}
