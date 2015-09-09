package main

import (
	"flag"
	"log"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"

	"github.com/qiniu/api.v6/conf"
	"github.com/qiniu/api.v6/io"
	"github.com/qiniu/api.v6/rs"
	"gopkg.in/gcfg.v1"
)

type Config struct {
	Qiniu struct {
		UpHost    string `gcfg:"uphost"`
		AccessKey string `gcfg:"accesskey"`
		SecretKey string `gcfg:"secretkey"`
		Bucket    string
		KeyPrefix string `gcfg:"keyprefix"`
	}
	Local struct {
		SyncDir string `gcfg:"syncdir"`
	}
}

func genUptoken(bucketName string) string {
	putPolicy := rs.PutPolicy{
		Scope: bucketName,
	}
	//putPolicy.SaveKey = key
	return putPolicy.Token(nil)
}

func uploadFile(bucket, key, filename string) error {
	uptoken := genUptoken(bucket + ":" + key) // in order to rewrite exists file

	var ret io.PutRet
	var extra = &io.PutExtra{}
	return io.PutFile(nil, &ret, uptoken, key, filename, extra)
}

func syncDir(bucket, keyPrefix, dir string) int {
	keyPrefix = strings.TrimPrefix(keyPrefix, "/")
	errCount := 0
	wg := &sync.WaitGroup{}
	filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if info.IsDir() {
			return nil
		}
		rel, _ := filepath.Rel(dir, path)
		key := filepath.Join(keyPrefix, rel)
		wg.Add(1)
		go func() {
			log.Printf("Upload %v ...", strconv.Quote(key))
			if err := uploadFile(bucket, key, path); err != nil {
				errCount += 1
				log.Printf("Failed %v, %v", strconv.Quote(path), err)
			}
			log.Printf("Done %v", strconv.Quote(key))
			wg.Done()
		}()
		return nil
	})
	wg.Wait()
	return errCount
}

func main() {
	cfgFile := flag.String("c", "conf.ini", "config file")
	flag.Parse()

	var cfg Config
	if err := gcfg.ReadFileInto(&cfg, *cfgFile); err != nil {
		log.Fatal(err)
	}
	conf.ACCESS_KEY = cfg.Qiniu.AccessKey
	conf.SECRET_KEY = cfg.Qiniu.SecretKey
	conf.UP_HOST = cfg.Qiniu.UpHost
	log.Printf("Use upload host: %v", conf.UP_HOST)

	syncDir(cfg.Qiniu.Bucket, cfg.Qiniu.KeyPrefix, cfg.Local.SyncDir)
}
