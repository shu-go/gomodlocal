package main

import (
	"errors"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/shu-go/gli"
	"golang.org/x/mod/modfile"
	"golang.org/x/mod/module"
)

var Version string

func init() {
	if Version == "" {
		Version = "dev-" + time.Now().Format("20060102")
	}
}

const gomod = "./go.mod"

type globalCmd struct {
	Replace replaceCmd `cli:"replace,r"`
	Drop    dropCmd    `cli:"drop,d"`
}

type replaceCmd struct {
	_ struct{} `help:"add a replace directive"`

	Abs   bool `cli:"absolute,abs" help:"local path is absolute (default: relative)"`
	Force bool `cli:"force" help:"do not check 'require' path in my go.mod vs 'module' path in go.mod of required module"`
}

type dropCmd struct {
	_ struct{} `help:"drop replace directive(s)"`

	All bool `cli:"all,a" help:"drop all replace directives"`
}

func (c replaceCmd) Run(args []string) error {
	if len(args) == 0 {
		return errors.New("no target module specified")
	}

	// read my go.mod

	f, err := os.Open(gomod)
	if err != nil {
		return fmt.Errorf("go.mod not found: %v", err)
	}

	data, err := ioutil.ReadAll(f)
	if err != nil {
		return fmt.Errorf("failed to open go.mod: %v", err)
	}
	f.Close()
	f = nil

	modFile, err := modfile.Parse(gomod, data, nil)
	if err != nil {
		return fmt.Errorf("failed to parse go.mod: %v", err)
	}

	var tgtMod *module.Version
	for _, r := range modFile.Require {
		if strings.Contains(r.Mod.Path, args[0]) {
			tgtMod = &r.Mod
			break
		}
	}

	if tgtMod == nil {
		return fmt.Errorf("no module found for `%v`", args[0])
	}

	// find local mod file path (localPath) starting from wd

	wd, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("can not get working directory!?: %v", err)
	}

	var localPath string
	if len(args) == 2 {
		abs, err := filepath.Abs(args[1])
		if err != nil {
			return fmt.Errorf("invalid path: %v", err)
		}
		localPath = abs
	} else {
		// reqPathComponents == [github.com shu-go gli]
		// wdPathComponents == [C: github.com shu-go gomodlocal]
		//    ^ where you want to replace with localpath
		reqPathComponents := strings.Split(tgtMod.Path, "/")
		wdPathComponents := strings.Split(wd, string(filepath.Separator))

		var ri, wi int
	loop:
		for ri = len(reqPathComponents) - 1; ri >= 0; ri-- {
			// reqPathComponents[ri]: gli -> shu-go -> github.com

			for wi = len(wdPathComponents) - 1; wi >= 0; wi-- {
				// ri == 2: gli vs each in [gomodlocal(miss) shu-go(miss) github.com(miss) C:(miss)]
				// ri == 1: shu-go vs each in [gomodlocal(miss) shu-go(HIT!!)]
				if reqPathComponents[ri] == wdPathComponents[wi] {
					break loop
				}
			}
		}
		if ri == -1 || wi == -1 {
			return errors.New("can not find common path")
		}

		// localPathCompos == [C: github.com shu-go] + [gli]
		// localPath == C:\github.com\shu-go\gli
		localPathCompos := append(wdPathComponents[:wi], reqPathComponents[ri:]...)
		localPath = strings.Join(localPathCompos, string(filepath.Separator))
	}

	if !c.Abs {
		// wd == C:\github.com\shu-go\gomodlocal
		// localPath == ..\gli
		localPath, err = filepath.Rel(wd, localPath)
		if err != nil {
			return fmt.Errorf("can not get relative path: %v", err)
		}
		if !strings.HasPrefix(localPath, ".") {
			// "hoge" -> "./hoge"

			//localPath = filepath.Join(".", localPath) => still "hoge"
			localPath = "." + string(filepath.Separator) + localPath //=> "./hoge"
		}
	}

	// check a go.mod of requred module

	_, err = os.Stat(localPath)
	if err != nil {
		return fmt.Errorf("local pkg not found: %v", err)
	}

	destgomod := filepath.Join(localPath, gomod)

	df, err := os.Open(destgomod)
	if err != nil {
		return fmt.Errorf("dest go.mod not found: %v", err)
	}

	data, err = ioutil.ReadAll(df)
	if err != nil {
		return fmt.Errorf("failed to open go.mod: %v", err)
	}
	df.Close()
	df = nil

	destModFile, err := modfile.Parse(destgomod, data, nil)
	if err != nil {
		return fmt.Errorf("failed to parse go.mod: %v", err)
	}

	if !c.Force {
		if destModFile.Module.Mod.Path != tgtMod.Path {
			return fmt.Errorf("dest mod(%v) in %v is not %v", destModFile.Module.Mod.Path, localPath, tgtMod.Path)
		}
	}

	// add a replace directive

	println(tgtMod.Path, "=>", localPath)

	err = modFile.AddReplace(tgtMod.Path, "", localPath, "")
	if err != nil {
		return fmt.Errorf("failed to add replace: %v", err)
	}

	modFile.Cleanup()

	data, err = modFile.Format()
	if err != nil {
		return fmt.Errorf("failed to format: %v", err)
	}

	// write to my go.mod

	err = ioutil.WriteFile(gomod, data, os.ModePerm)
	if err != nil {
		return fmt.Errorf("failed to write to: %v", err)
	}

	return nil
}

func (c dropCmd) Run(args []string) error {
	if len(args) == 0 && !c.All {
		return errors.New("no target module specified")
	}

	f, err := os.Open(gomod)
	if err != nil {
		return fmt.Errorf("go.mod not found: %v", err)
	}

	data, err := ioutil.ReadAll(f)
	if err != nil {
		return fmt.Errorf("failed to open go.mod: %v", err)
	}

	modFile, err := modfile.Parse(gomod, data, nil)
	if err != nil {
		return fmt.Errorf("failed to parse go.mod: %v", err)
	}

	var changed bool
	for _, r := range modFile.Replace {
		if len(args) >= 1 && strings.Contains(r.Old.Path, args[0]) || c.All {
			changed = true
			println("drop " + r.New.Path)

			modFile.DropReplace(r.Old.Path, "")
			if err != nil {
				return fmt.Errorf("failed to drop replace: %v", err)
			}
		}
	}

	if changed {
		data, err = modFile.Format()
		if err != nil {
			return fmt.Errorf("failed to format: %v", err)
		}

		modFile.Cleanup()

		err = ioutil.WriteFile(gomod, data, os.ModePerm)
		if err != nil {
			return fmt.Errorf("failed to write to: %v", err)
		}
	}

	return nil
}

func main() {
	app := gli.NewWith(&globalCmd{})
	app.Name = "gomodlocal"
	app.Desc = "replace go.mod by guessed local path"
	app.Version = Version
	app.Usage = `gomodlocal replace MODULE_NAME {MODULE_PATH}`
	app.Copyright = "(C) 2020 Shuhei Kubota"
	app.Run(os.Args)
}
