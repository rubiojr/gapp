# Glance Embedding

A Go package for embedding [Glance](https://github.com/glanceapp/glance) in your Go applications.

## Examples

Embeds the Glance configuration and builds a binary that runs the Glance dashboard in a webview.

```Go
package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"time"

	_ "embed"

	"github.com/rubiojr/gapp/pkg/glance"
	webview "github.com/rubiojr/webview_go"
)

//go:embed glance.example.yml
var config []byte

func main() {
	var host = "127.0.0.1"
	var port = uint16(65529)

	opts := []glance.Option{
		glance.WithServerPort(port),
		glance.WithLogger(log.New(os.Stdout, "", log.LstdFlags)),
		glance.WithHost(host),
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	err := glance.ServeApp(ctx, config, opts...)
	if err != nil {
		panic(err)
	}

	w := webview.New(false)
	defer w.Destroy()

	w.SetTitle("Glance Dashboard")
	w.SetSize(1024, 600, webview.HintMin)
	w.Navigate(fmt.Sprintf("http://%s:%d", host, port))
	time.Sleep(time.Second)

	w.Run()
}
```

## Building the example

CGO_ENABLED=1 is required to build this package.

### macOS

No specific requirements, other than the Go toolchain.

### Linux

webkit2gtk4.0 development libraries are required to build this package.

- Fedora: webkit2gtk4.1-devel
- Ubuntu: libwebkit2gtk-4.0-dev

See https://github.com/webview/webview_go.


## Credits

The following projects were used to build this package:

- [Glance](https://github.com/glanceapp/glance)
- [webview](https://github.com/webview/webview_go)

Glance source code has been slightly modified to work with this package. The modified source code is available in the [internal/glance](/internal/glance) directory.
