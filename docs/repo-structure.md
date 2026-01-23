# Repository Structure

<details>
  <summary>Some info</summary>
so this project has a kinda wierd file structure mostly cause it was my first go project and I didn't quite understand how projects repo is set, and now I am too lazy to fix it.
</details>

### The entry point is the `main.go` which is just a wrapper for command package or module which is in cmd folder

```bash
gls --tree -d 3
├── .github/
│   └── workflows/ -> all the worflows will be added in this folder
│       └── build.yaml -> builds the bin and makes a relase
├── cmd/
│   └── commands/ -> command package, handels args
│       ├── commands.go -> handel args
│       ├── showInfo.go -> -i, --info flag Show detailed info about cached item(s)
│       ├── showList.go -> -l, --list          Show all cached files
│       ├── showStats.go -> -s, --stats         Show cache statistics
│       ├── showThemes.go -> -t, --themes        Previews theme
│       ├── showUsage.go  -> -h, --help          Show this help message
│       └── version.go  -> -v, --version        Show version information
├── docs/
│   ├── configuration/
│   │   ├── condig.md -> documentaion on config
│   │   └── config.toml -> default config
│   └── repo-structure.md -> this file
├── internal/
│   ├── config/
│   │   ├── config.go -> manges config related operations like loading and writing if missing
│   │   └── exportConfig.go -> not yet added but can be used to create backup or use new config from net
│   ├── helpers/ -> helpers package, responsible for core logic kinda like backend of this project
│   │   ├── helpers.go -> core logic of vanish like file deltion, recover, cache cleaning and more
│   │   ├── helpers_test.go -> tests for helpers.go
│   │   ├── index.go -> manages indexing so that info and list operations can be done
│   │   ├── logging.go -> creates log duh
│   │   ├── symlink.go -> handels symlink deltion
│   │   └── terminal.go -> checks for terminal size and other stuff
│   ├── tui/ -> manages tui
│   │   ├── headless.go -> no ui direct operation, exist cause to perform automation was asked by @zloylinux in #3
│   │   ├── tui-helper.go -> helper for tui
│   │   └── tui.go -> tui in bubble tea
│   └── types/ -> all common types
│       └── types.go -> types duhh
├── .gitignore -> ignore this
├── LICENSE -> IMP license pls dont violate
├── Makefile -> makes project <mind blown>
├── PKGBUILD -> builds project <mind blown>
├── README.md -> what,why,preview and other stuff
├── TODO -> cause my ass lazy
├── go.mod -> packages used
├── go.sum -> specific shit
├── install.sh -> common stuff to isntall
├── main.go -> main stuff
├── test-build.sh -> check if builds or not
└── vx -> bin
```

<details>
  <summary>See more</summary>
  gls stands for git ls its a simple cli i made which ignores .gitignore files and folder from ls
</details>
