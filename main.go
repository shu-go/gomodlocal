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

type globalCmd struct {
	Replace replaceCmd `cli:"replace,r"`
	Drop    dropCmd    `cli:"drop,d"`
}

type replaceCmd struct {
	Abs bool `cli:"absolute,abs"`
}
type dropCmd struct {
	All bool `cli:"all,a"`
}

func (c replaceCmd) Run(args []string) error {
	if len(args) == 0 {
		return errors.New("no target module specified")
	}

	gomod := "go.mod"
	mygomod := filepath.Join(".", gomod)

	f, err := os.Open(mygomod)
	if err != nil {
		return fmt.Errorf("go.mod not found: %v", err)
	}

	data, err := ioutil.ReadAll(f)
	if err != nil {
		return fmt.Errorf("failed to open go.mod: %v", err)
	}
	f.Close()
	f = nil

	modFile, err := modfile.Parse(mygomod, data, nil)
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

	wd, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("can not get working directory!?: %v", err)
	}

	var newPath string
	if len(args) == 2 {
		abs, err := filepath.Abs(args[1])
		if err != nil {
			return fmt.Errorf("invalid path: %v", err)
		}
		newPath = abs
	} else {
		oldModCompos := strings.Split(tgtMod.Path, "/")
		wdPathCompos := strings.Split(wd, string(filepath.Separator))

		var oi, wi int
	loop:
		for oi = len(oldModCompos) - 1; oi >= 0; oi-- {
			for wi = len(wdPathCompos) - 1; wi >= 0; wi-- {
				if oldModCompos[oi] == wdPathCompos[wi] {
					break loop
				}
			}
		}
		if oi == -1 || wi == -1 {
			return errors.New("can not find common path")
		}

		newPathCompo := append(wdPathCompos[:wi], oldModCompos[oi:]...)
		newPath = strings.Join(newPathCompo, string(filepath.Separator))
	}

	if !c.Abs {
		newPath, err = filepath.Rel(wd, newPath)
		if err != nil {
			return fmt.Errorf("can not get relative path: %v", err)
		}
	}

	_, err = os.Stat(newPath)
	if err != nil {
		return fmt.Errorf("local pkg not found: %v", err)
	}

	// check desg go.mod

	destgomod := filepath.Join(newPath, gomod)

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

	if destModFile.Module.Mod.Path != tgtMod.Path {
		return fmt.Errorf("dest mod(%v) in %v is not %v", destModFile.Module.Mod.Path, newPath, tgtMod.Path)
	}

	println(tgtMod.Path, "=>", newPath)

	err = modFile.AddReplace(tgtMod.Path, "", newPath, "")
	if err != nil {
		return fmt.Errorf("failed to add replace: %v", err)
	}

	modFile.Cleanup()

	data, err = modFile.Format()
	if err != nil {
		return fmt.Errorf("failed to format: %v", err)
	}

	err = ioutil.WriteFile(mygomod, data, os.ModePerm)
	if err != nil {
		return fmt.Errorf("failed to write to: %v", err)
	}

	return nil
}

func (c dropCmd) Run(args []string) error {
	if len(args) == 0 && !c.All {
		return errors.New("no target module specified")
	}

	gomod := "./go.mod"
	mygomod := filepath.Join(".", gomod)

	f, err := os.Open(mygomod)
	if err != nil {
		return fmt.Errorf("go.mod not found: %v", err)
	}

	data, err := ioutil.ReadAll(f)
	if err != nil {
		return fmt.Errorf("failed to open go.mod: %v", err)
	}

	modFile, err := modfile.Parse(mygomod, data, nil)
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

		err = ioutil.WriteFile(mygomod, data, os.ModePerm)
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
