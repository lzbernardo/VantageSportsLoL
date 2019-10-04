package main

import (
	"log"
	"net/http"
	"os"
	"time"

	"github.com/VantageSports/common/credentials/google"
	"github.com/VantageSports/lolvideo/coordinator"
	"github.com/VantageSports/lolvideo/coordinator/handlers"

	"github.com/gorilla/mux"
	"google.golang.org/api/cloudmonitoring/v2beta2"
)

func handleOK(w http.ResponseWriter, r *http.Request) {
	w.Write([]byte("OK"))
}

func main() {
	log.SetFlags(log.LstdFlags | log.Lshortfile)
	log.SetPrefix("coordinator_server_")
	coordinator.InitEnvironment()

	r := mux.NewRouter()
	s := r.PathPrefix("/lolvideo").Subrouter()

	handler := &handlers.LolVideoHandler{
		TimeseriesClient: coordinator.GetGoogleTimeseriesClient(
			google.MustEnvCreds(
				coordinator.GoogProjectID,
				cloudmonitoring.MonitoringScope)),
	}

	s.HandleFunc("/health", handleOK)
	s.HandleFunc("/video_request", handler.HandleVideoRequest())
	s.HandleFunc("/status", handler.HandleStatusRequest())
	s.HandleFunc("/addWorker", handler.HandleAddWorkerRequest())
	s.HandleFunc("/addWorkerSubmit", handler.HandleAddWorkerSubmitRequest())
	s.HandleFunc("/upgradeAmi", handler.HandleUpgradeAmiRequest())
	s.HandleFunc("/upgradeAmiSubmit", handler.HandleUpgradeAmiSubmitRequest())
	s.HandleFunc("/upgradeAmiCallback", handler.HandleUpgradeAmiCallbackRequest())
	s.HandleFunc("/terminate_instance", handler.HandleTerminateInstanceRequest())
	s.HandleFunc("/datagen_request", handler.HandleDatagenRequest())
	s.HandleFunc("/datagen_bootstrap_request", handler.HandleDatagenBootstrapRequest())
	s.HandleFunc("/addDataWorker", handler.HandleAddDataWorkerRequest())
	s.HandleFunc("/addDataWorkerSubmit", handler.HandleAddDataWorkerSubmitRequest())

	// Expose videogenWorker.exe and replay.bat for the worker machine to download
	s.Handle("/static-files/{file}", http.StripPrefix("/lolvideo/static-files",
		http.FileServer(http.Dir(os.Getenv("GOPATH")+"/bin/windows_amd64"))))

	go reportQueueSize(handler.TimeseriesClient, coordinator.GoogProjectID, coordinator.PubsubInputElo, time.Minute)

	log.Println("listening...")
	log.Fatal(http.ListenAndServe(":8080", r))
}
