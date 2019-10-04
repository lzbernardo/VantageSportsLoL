// download_match downloads a single match's replay files to a temp directory.
// if the remote flag is set to a remote directory, the replay files are saved
// there.
//
// Example of downloading a match from replay.gg:
// $ go build
// $ ./download_match -creds prod_creds.json -dest gs://vsp-esports/lol/replay/matches/ -key <encryption_key> -match 12345 -platform NA1 -projectid vs-main -server http://replay.gg:8080

// TODO(Cameron): Future improvements.
// It would be nice if the current_game.json it constructed automatically added
// the participants, gameMode, gameStartTime, gameType, mapId, and queueType,
// since downstream processes rely on that.

package main

import (
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"strings"

	"cloud.google.com/go/storage"

	"github.com/VantageSports/common/credentials/google"
	"github.com/VantageSports/common/files"
	"github.com/VantageSports/lolobserver"
	"github.com/VantageSports/riot/api"
)

var (
	matchID      = flag.Int64("match", -1, "id of match to download")
	platform     = flag.String("platform", "NA1", "platform ID (if no server specified)")
	key          = flag.String("key", "", "encryption key")
	server       = flag.String("server", "", "server to download replay from")
	destDir      = flag.String("dest", os.TempDir(), "directory to save files")
	projectID    = flag.String("projectid", "", "project id for remote directory")
	projectCreds = flag.String("creds", "", "credentials for remote directory")
)

func main() {
	flag.Parse()

	if *server == "" || *platform == "" || *matchID < 0 || *destDir == "" {
		flag.Usage()
		log.Fatalln("server, platform, dest, and matchID required")
	}
	if !strings.HasPrefix(*server, "http") {
		log.Println("WARN: server should start with http or https. This probably won't work...")
	}

	tmpdir, err := ioutil.TempDir("", "loldownload")
	if err != nil {
		log.Fatalln(err)
	}
	defer os.RemoveAll(tmpdir)

	fc := mustFilesClient()
	dir := fmt.Sprintf("%s/%d-%s", strings.TrimSuffix(*destDir, "/"), *matchID, *platform)
	saver, err := lolobserver.NewFileSaver(fc, dir)
	if err != nil {
		log.Fatalln(err)
	}
	defer saver.Close()

	log.Println("Attempting to examine previously downloaded match files...")
	existing, err := fc.List(dir)
	if err != nil {
		log.Printf("Warning, unable to list files in %s. Starting from scratch\n", *destDir)
	}

	ms := &lolobserver.MatchUsers{
		CurrentGame: api.CurrentGameInfo{
			GameID:     *matchID,
			PlatformID: *platform,
			Observers: api.Observer{
				EncryptionKey: *key,
			},
		},
	}

	ds, err := lolobserver.NewReplaySaveState(ms, existing)
	if err != nil {
		log.Fatalln(err)
	}
	log.Println("Starting download...")
	if err = ds.Save(*server, saver); err != nil {
		log.Println("error:", err)
	} else {
		log.Println("success")
	}
}

func mustFilesClient() *files.Client {
	fc, err := files.InitClient()
	if err != nil {
		log.Fatalln(err)
	}

	if *projectCreds != "" && (*destDir)[0:3] == "gs:" {
		creds, err := google.File(*projectCreds, *projectID, storage.ScopeReadWrite)
		if err != nil {
			log.Fatalln(err)
		}

		gcs, err := files.NewGCSProvider(creds)
		if err != nil {
			log.Fatalln(err)
		}

		if err = fc.Register((*destDir)[0:5], gcs, gcs); err != nil {
			log.Fatalln(err)
		}
	}

	return fc
}
