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
}
type dropCmd struct {
	All bool `cli:"all,a"`
}

func (c replaceCmd) Run(args []string) error {
	if len(args) == 0 {
		return errors.New("no target module specified.")
	}

	gomod := "./go.mod"

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

	var newPath string
	if len(args) == 2 {
		abs, err := filepath.Abs(args[1])
		if err != nil {
			return fmt.Errorf("invalid path: %v", err)
		}
		newPath = abs
	} else {
		wd, err := os.Getwd()
		if err != nil {
			return fmt.Errorf("cant get working directory!?: %v", err)
		}

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
			return errors.New("could not find common path.")
		}

		newPathCompo := append(wdPathCompos[:wi], oldModCompos[oi:]...)
		newPath = strings.Join(newPathCompo, string(filepath.Separator))
	}

	println(tgtMod.Path, "=>", newPath)

	err = modFile.AddReplace(tgtMod.Path, "", newPath, "")
	if err != nil {
		return fmt.Errorf("failed to add replace: %v", err)
	}

	data, err = modFile.Format()
	if err != nil {
		return fmt.Errorf("failed to format: %v", err)
	}

	err = ioutil.WriteFile(gomod, data, os.ModePerm)
	if err != nil {
		return fmt.Errorf("failed to write to: %v", err)
	}

	return nil
}

func (c dropCmd) Run(args []string) error {
	if len(args) == 0 && !c.All {
		return errors.New("no target module specified.")
	}

	gomod := "./go.mod"

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

	for _, r := range modFile.Replace {
		if len(args) >= 1 && strings.Contains(r.Old.Path, args[0]) || c.All {
			println("drop " + r.New.Path)

			modFile.DropReplace(r.Old.Path, "")
			if err != nil {
				return fmt.Errorf("failed to drop replace: %v", err)
			}

			data, err = modFile.Format()
			if err != nil {
				return fmt.Errorf("failed to format: %v", err)
			}

			err = ioutil.WriteFile(gomod, data, os.ModePerm)
			if err != nil {
				return fmt.Errorf("failed to write to: %v", err)
			}
		}
	}

	return nil
}

func main() {
	app := gli.NewWith(&globalCmd{})
	app.Name = "gomodrepl"
	app.Desc = "replace go.mod by guessed path"
	app.Version = Version
	app.Usage = `gomodrepl replace MODULE_NAME {MODULE_PATH}`
	app.Copyright = "(C) 2020 Shuhei Kubota"
	app.Run(os.Args)

}
