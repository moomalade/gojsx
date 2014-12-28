package main

import (
	"flag"
	"fmt"
	"github.com/go-fsnotify/fsevents"
	"io/ioutil"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

func compileDir(path string) {
	files, err := ioutil.ReadDir(path)
	if err != nil {
		return
	}

	for _, file := range files {
		filePath := filepath.Join(path, file.Name())
		if file.IsDir() {
			compileDir(filePath)
		} else {
			compileFile(filePath)
		}
	}
}

func compileFile(sourcePath string) {

	if filepath.Ext(sourcePath) == ".jsx" {

		targetPath := strings.TrimSuffix(sourcePath, ".jsx") + ".js"

		sourceFileInfo, err := os.Stat(sourcePath)
		if err != nil {
			return
		}

		targetFileInfo, err := os.Stat(targetPath)
		if err != nil {
			return
		}

		if sourceFileInfo.ModTime().After(targetFileInfo.ModTime()) {

			cmdLine := fmt.Sprintf("jsx %s", sourcePath)
			cmd := exec.Command("bash", "-c", cmdLine)

			log.Printf("%s => %s", sourcePath, targetPath)
			//log.Printf("%s", cmdLine )

			cmd.Stderr = os.Stderr

			// pipe stdout to the target file in our process to avoid a file change event
			f, err := os.OpenFile(targetPath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0777)
			if err != nil {
				log.Printf("%s", err)
				return
			}
			defer f.Close()

			cmd.Stdout = f

			err = cmd.Run()

			if err != nil {
				log.Printf("Command Error: %s", err)
				return
			}

		}
	}
}

func main() {

	watchPath := flag.String("d", "", "set directory")
	watch := flag.Bool("w", false, "watch for changes")

	flag.Parse()

	if *watchPath == "" {
		log.Printf("Specify a directory to compile with -d, and watch with -w")
		return
	}

	e := fsevents.EventStream{
		Paths:   []string{*watchPath},
		Latency: time.Duration(1) * time.Second,
		Flags:   fsevents.IgnoreSelf | fsevents.FileEvents,
	}
	e.Start()

	log.Printf("Compiling '%s'", *watchPath)
	compileDir(*watchPath)

	if *watch {
		log.Printf("Watching '%s'", *watchPath)
		for {
			for _, ev := range <-e.Events {

				cwd, err := os.Getwd()
				if err != nil {
					panic(err)
				}

				relPath, err := filepath.Rel(cwd, ev.Path)
				if err != nil {
					log.Printf("%s", err)
					continue
				}

				if (ev.Flags & (fsevents.MustScanSubDirs | fsevents.ItemIsDir)) != 0 {
					compileDir(relPath)
				}

				if (ev.Flags & fsevents.ItemIsFile) != 0 {
					compileFile(relPath)
				}
			}
		}
	}
}
