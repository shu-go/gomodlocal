replace go.mod by guessed local path

[![Go Report Card](https://goreportcard.com/badge/github.com/shu-go/gomodlocal)](https://goreportcard.com/report/github.com/shu-go/gomodlocal)
![MIT License](https://img.shields.io/badge/License-MIT-blue)


It replaces a module to point to your local path.

# Usage

If your local directory structure is like here (you are building `your-tool`):

## Replace

```
github.com/
  - YOU/
    - your-tool/
      - main.go  (uses github.com/YOU/your-lib1)
      - go.mod
      - go.sum
    - your-lib1/
      - ...
  - ANOTHER/
    - ano-libZ/
      - ...
```

You may have a `go.mod` like:

```
module github.com/YOU/your-tool

go 1.15

require (
	github.com/YOU/your-lib1 v0.0.0-20201126235959-0ab12c34def5
	github.com/ANOTHER/ano-libZ v0.1.0-20201126235959-1bc23d45efg6
)
```

:arrow_down::arrow_down::arrow_down:

```
gomodlocal replace your-lib1
```

:arrow_down::arrow_down::arrow_down:

```
module github.com/YOU/your-tool

go 1.15

require (
	github.com/YOU/your-lib1 v0.0.0-20201126235959-0ab12c34def5
	github.com/ANOTHER/ano-libZ v0.1.0-20201126235959-1bc23d45efg6
)

replace github.com/YOU/your-lib1 => ..\your-lib1
```

Now you can debug your-lib1 freely.


:arrow_down::arrow_down::arrow_down:And more...:arrow_down::arrow_down::arrow_down:

```
gomodlocal replace ano-libZ
```

:arrow_down::arrow_down::arrow_down:

```
module github.com/YOU/your-tool

go 1.15

require (
	github.com/YOU/your-lib1 v0.0.0-20201126235959-0ab12c34def5
	github.com/ANOTHER/ano-libZ v0.1.0-20201126235959-1bc23d45efg6
)

replace github.com/YOU/your-lib1 => ..\your-lib1

replace github.com/ANOTHER/ano-libZ => ..\..\ano-libZ
```

## Drop

:arrow_down::arrow_down::arrow_down:

```
gomodlocal drop --all
(gomodlocal drop your-lib1 && gomodlocal drop ano-libZ)
```

:arrow_down::arrow_down::arrow_down:

```
module github.com/YOU/your-tool

go 1.15

require (
	github.com/YOU/your-lib1 v0.0.0-20201126235959-0ab12c34def5
	github.com/ANOTHER/ano-libZ v0.1.0-20201126235959-1bc23d45efg6
)
```

It's back.

# More help

```
gomodlocal help
gomodlocal help subcommand
```
