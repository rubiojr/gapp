package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"runtime"
	"time"

	_ "embed"

	"github.com/rubiojr/gapp/pkg/glance"
	webview "github.com/webview/webview_go"
)

//go:embed glance.example.yml
var config []byte

func main() {
	var host = "127.0.1.1"
	// 127.0.1.1 is a no-no in macOS
	if runtime.GOOS == "darwin" {
		host = "localhost"
	}
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
	w.SetSize(1024, 600, webview.HintNone)
	w.Navigate(fmt.Sprintf("http://%s:%d", host, port))
	time.Sleep(time.Second)

	w.Run()
}
