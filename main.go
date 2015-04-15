package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path"
	"path/filepath"
	"strings"
	"time"

	"github.com/edgard/goutil"
	"github.com/kolo/xmlrpc"
)

var config struct {
	ExpireHours int      `json:"expirehours"`
	RPCURL      string   `json:"rpcurl"`
	FakeBaseDir string   `json:"fakebasedir"`
	RealBaseDir string   `json:"realbasedir"`
	CheckDirs   []string `json:"checkdirs"`
}

func removeExpired(client *xmlrpc.Client) {
	var torrents []string
	err := client.Call("download_list", []interface{}{"", "complete"}, &torrents)
	if err != nil {
		fmt.Println(err)
		return
	}

	for _, torrent := range torrents {
		var timestamp int64
		err := client.Call("d.timestamp.finished", torrent, &timestamp)
		if err != nil {
			fmt.Println(err)
			return
		}
		delta := time.Since(time.Unix(timestamp, 0))
		if int(delta.Hours()) > config.ExpireHours {
			fmt.Printf("Expired: %s | %.0f hours\n", torrent, delta.Hours())
			var torrentPath string
			client.Call("d.get_base_path", torrent, &torrentPath)
			client.Call("d.delete_tied", torrent, nil)
			client.Call("d.erase", torrent, nil)
			err := os.RemoveAll(torrentPath)
			if err != nil {
				fmt.Println(err)
			}
		}
	}
}

func removeOrphans(client *xmlrpc.Client) {
	var torrents []string
	err := client.Call("download_list", []interface{}{"", "complete"}, &torrents)
	if err != nil {
		fmt.Println(err)
		return
	}

	var torrentPathlist []string
	for _, torrent := range torrents {
		var torrentPath string
		err := client.Call("d.get_base_path", torrent, &torrentPath)
		if err != nil {
			fmt.Println(err)
			return
		}
		torrentPathlist = append(torrentPathlist, torrentPath)
	}

	var dirPathlist []string
	for _, dir := range config.CheckDirs {
		dirPath, err := filepath.Glob(path.Join(dir, "*"))
		if err != nil {
			fmt.Println(err)
			return
		}
		dirPathlist = append(dirPathlist, dirPath...)
	}

	for _, dirPath := range dirPathlist {
		if !goutil.StringInSlice(strings.Replace(dirPath, config.RealBaseDir, config.FakeBaseDir, 1), torrentPathlist) {
			fmt.Printf("Orphaned: %s\n", dirPath)
			err := os.RemoveAll(dirPath)
			if err != nil {
				fmt.Println(err)
			}
		}
	}
}

func main() {
	f, err := os.Open("config.json")
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
	defer f.Close()
	err = json.NewDecoder(f).Decode(&config)
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	client, err := xmlrpc.NewClient(config.RPCURL, nil)
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
	defer client.Close()

	removeExpired(client)
	removeOrphans(client)
}
