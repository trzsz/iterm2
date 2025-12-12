# iTerm2

### Go library for automating iTerm2 Scripts

Forked from [marwan-at-work/iterm2](https://github.com/marwan-at-work/iterm2).

### Install

go get github.com/trzsz/iterm2

### Usage

```go
package main

import (
	"log"

	"github.com/trzsz/iterm2"
)

func main() {
	app, err := iterm2.NewApp("MyCoolPlugin")
	if err != nil {
		log.Fatal(err)
	}
	defer func() { _ = app.Close() }()
	// use app to create or list windows, tabs, and sessions and send various commands to the terminal.
}
```

### How do I actually run the script?

- Since you will be using this library in a "main" program, you can literally just run the Go program through "go run" or install your program/binary globally through "go install" and then run it from any terminal.

- A nicer way to run the script is to "register" the plugin with iTerm2 so you can run it from iTerm's command pallette (cmd+shift+o). This means you won't need a terminal tab open or to remember what the plugin name is. See the following section on how to do that:

- Ensure you enable the Python API: https://iterm2.com/python-api-auth.html

### Progress

This is currently a work in progress and it is a subset of what the iTerm2 WebSocket protocol provides.

I don't intend to implement all of it as I am mainly implementing only the parts that I need for daily work.

If you'd like to add more features, feel free to open an issue and a PR.
