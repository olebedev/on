package main

import (
	"bytes"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"text/template"

	"github.com/codegangsta/cli"
	"gopkg.in/fsnotify.v1"
)

func main() {
	app := cli.NewApp()
	app.Usage = "cli for fsnotify"
	app.UsageText = `on [options] [path]

   Path could be mix of folders and filepaths, default is '.'.
   Regular usecase is watch the file's changes and execute some
   bash script or command line tool. It could be done in whis way:

   on | xargs -n1 -I{} <program>`

	app.Author = "olebedev <ole6edev@gmail.com>"
	app.Version = "0.1.0"
	app.Flags = []cli.Flag{
		cli.StringFlag{
			Name:  "template,t",
			Usage: "output template to render received event, see: https://godoc.org/gopkg.in/fsnotify.v1#Event",
			Value: "{{.Name}}",
		},
		cli.IntFlag{
			Name:  "mask,m",
			Usage: "event's bitwise mask, see: https://godoc.org/gopkg.in/fsnotify.v1#Op",
			Value: 15,
		},
	}
	app.Action = func(c *cli.Context) {
		t, err := template.New("output").Parse(c.String("template"))
		if err != nil {
			log.Fatal(err)
		}
		id := fsnotify.Op(c.Int("mask"))

		watcher, err := fsnotify.NewWatcher()
		if err != nil {
			log.Fatal(err)
		}
		defer watcher.Close()

		done := make(chan struct{})

		go func() {
			for {
				select {
				case event := <-watcher.Events:
					// if bitwise mask covers the event
					if event.Op&id == event.Op {
						fmt.Println(render(t, event))
					}
				case err := <-watcher.Errors:
					fmt.Fprintln(os.Stderr, err)
				}
			}
		}()

		args := c.Args()
		if len(args) == 0 {
			args = []string{"."}
		}

		for _, arg := range args {
			if err := addPath(watcher, arg); err != nil {
				fmt.Fprintln(os.Stderr, err)
				os.Exit(1)
			}
		}
		<-done
	}

	app.Run(os.Args)
}

var events = map[string]fsnotify.Op{
	"create": fsnotify.Create,
	"write":  fsnotify.Write,
	"remove": fsnotify.Remove,
	"rename": fsnotify.Rename,
}

func render(t *template.Template, e fsnotify.Event) string {
	var doc bytes.Buffer
	t.Execute(&doc, e)
	return doc.String()
}

func addPath(w *fsnotify.Watcher, name string) error {
	f, err := os.Stat(name)
	if err != nil {
		return err
	}

	if !f.IsDir() {
		return w.Add(name)
	}

	return filepath.Walk(name, func(p string, f os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if f.IsDir() {
			return w.Add(p)
		}
		return nil
	})
}
