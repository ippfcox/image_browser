package main

import (
	"io/fs"
	"log"
	"path"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/rjeczalik/notify"
)

type imageInfo struct {
	ID      int // 数组索引
	Name    string
	Path    string
	ModTime time.Time
}

func isImage(info fs.FileInfo) bool {
	ext := strings.ToLower(filepath.Ext(info.Name()))
	for _, imageExt := range supportedExts {
		if ext == imageExt {
			return true
		}
	}
	return false
}

func loadImages(absdir string) []imageInfo {
	imageInfos := make([]imageInfo, 0)

	filepath.Walk(dir, func(path string, info fs.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if !info.IsDir() && isImage(info) {
			imageInfos = append(imageInfos, imageInfo{
				ID:      0,
				Name:    info.Name(),
				Path:    path,
				ModTime: info.ModTime(),
			})
		}
		return nil
	})
	sort.Slice(imageInfos, func(i, j int) bool {
		return imageInfos[i].ModTime.After(imageInfos[j].ModTime)
	})
	for id := range imageInfos {
		imageInfos[id].ID = id
	}
	return imageInfos
}

func watchImages(absdir string) {
	imageInfos = loadImages(absdir)

	c := make(chan notify.EventInfo)
	notify.Watch(path.Join(absdir, "..."), c, notify.All)
	go func() {
		for e := range c {
			log.Printf("File change: %s, %s", e.Path(), e.Event().String())
			imageInfos = loadImages(absdir)
		}
	}()
}
