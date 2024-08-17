package main

import (
	"github.com/gorilla/websocket"
	"go.uber.org/zap"
	"io"
	"net/http"
	"os"
	"path"
	"server/bird"
	"server/click_race"
	"server/core"
	"server/demo"
	"server/match"
	"server/moles"
	"server/race"
	"server/rps"
	"server/shooter"
	"slices"
)

func main() {
	if !slices.Contains(os.Args, "--verbose") {
		core.Logger = core.Logger.WithOptions(zap.IncreaseLevel(zap.InfoLevel))
	}

	// We have to pass in the minigame prototypes because Go doesn't allow circular package
	// dependencies :/ This is ugly, but it works.
	hub := core.NewHub(map[string]core.MinigamePrototype{
		demo.ProtoSp.Name:        demo.ProtoSp,
		demo.Proto1v1.Name:       demo.Proto1v1,
		demo.Proto2v2.Name:       demo.Proto2v2,
		match.Prototype.Name:     match.Prototype,
		moles.Prototype.Name:     moles.Prototype,
		click_race.ProtoSp.Name:  click_race.ProtoSp,
		click_race.Proto1v1.Name: click_race.Proto1v1,
		rps.Prototype.Name:       rps.Prototype,
		shooter.Proto1v1.Name:    shooter.Proto1v1,
		shooter.Proto2v2.Name:    shooter.Proto2v2,
		shooter.Proto3v3.Name:    shooter.Proto3v3,
		race.ProtoSp.Name:        race.ProtoSp,
		race.Proto1v1.Name:       race.Proto1v1,
		race.Proto2v2.Name:       race.Proto2v2,
		race.Proto3v3.Name:       race.Proto3v3,
		bird.ProtoSp.Name:        bird.ProtoSp,
	})

	upgrader := websocket.Upgrader{CheckOrigin: func(req *http.Request) bool {
		// We have to allow all origins because we are receiving connections from random
		// players' devices.
		return true
	}}

	wsFunc := func(writer http.ResponseWriter, request *http.Request) {
		l := core.Logger.With(zap.String("from", request.RemoteAddr))

		l.Debug("new connection on /ws")

		// Try to upgrade the HTTP connection to a WebSocket connection.
		ws, err := upgrader.Upgrade(writer, request, nil)

		if err != nil {
			l.Error("failed to upgrade to WebSocket connection", zap.Error(err))

			return
		}

		l.Debug("successfully upgraded connection to WebSocket")

		hub.AddConnection(ws)
	}

	http.HandleFunc("/ws", wsFunc)

	args := os.Args[1:]

	dirPathIndex := slices.Index(args, "--directory") + 1

	// The index will be zero if slices.Index returned -1.
	if dirPathIndex != 0 {
		if dirPathIndex >= len(args) {
			core.Logger.Fatal("--directory should be followed by a valid directory path")
		}

		// Create a file server so we can serve files from the provided directory.
		fileServer := http.FileServer(http.Dir(args[dirPathIndex]))

		http.HandleFunc("/", func(writer http.ResponseWriter, request *http.Request) {
			h := writer.Header()

			// Godot requires us to add these headers.
			h.Add("Cross-Origin-Embedder-Policy", "require-corp")
			h.Add("Cross-Origin-Opener-Policy", "same-origin")

			fileServer.ServeHTTP(writer, request)
		})
	} else {
		http.HandleFunc("/", func(writer http.ResponseWriter, request *http.Request) {
			_, _ = io.WriteString(writer, "No directory path was given at startup. "+
				"Restart the server with `--directory x` and then reload the page.")
		})
	}

	hub.Start()

	var err error

	certDirPathIndex := slices.Index(args, "--cert") + 1

	if certDirPathIndex != 0 {
		if certDirPathIndex >= len(args) {
			core.Logger.Fatal("--cert should be followed by a valid directory path")
		}

		certDirPath := args[certDirPathIndex]

		crtPath := path.Join(certDirPath, "cert.crt")
		keyPath := path.Join(certDirPath, "cert.key")
		core.Logger.Info("started server on port 443")
		err = http.ListenAndServeTLS(":443", crtPath, keyPath, nil)
	} else {
		core.Logger.Info("started server on port 8080")
		err = http.ListenAndServe(":8080", nil)
	}

	core.Logger.Panic("http server exited", zap.Error(err))
}
