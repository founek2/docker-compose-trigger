package main

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/julienschmidt/httprouter"
)

func getServicePath(service_name string) string {
	path, err := os.Getwd()
	if err != nil {
		log.Fatal(err)
	}

	service_path := filepath.Join(path, service_name)

	log.Println("Using directory", service_path)

	return service_path
}

func pullImages(service_path string) {
	var cmd = exec.Command("docker-compose", "pull")
	cmd.Dir = service_path
	err := cmd.Run()
	if err != nil {
		log.Fatal(err)
	}
}

func Index(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
	fmt.Fprint(w, "Simply docker-compose-trigger API\n")
}

func PullAndRestartService(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
	var service_name = ps.ByName("service_name")
	var service_path = getServicePath(service_name)

	log.Println("Pulling images...")
	pullImages(service_path)

	log.Printf("Killing service '%s'\n", service_name)

	cmd := exec.Command("docker-compose", "down")
	cmd.Dir = service_path
	err := cmd.Run()
	if err != nil {
		log.Fatal(err)
	}

	log.Printf("Starting service '%s'\n", service_name)
	cmd = exec.Command("docker-compose", "up")
	cmd.Dir = service_path
	err = cmd.Run()
	if err != nil {
		log.Fatal(err)
	}

	log.Println("Done")
	w.Write([]byte("Success\n"))
}

func PullAndRestartApp(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
	var service_name = ps.ByName("service_name")
	var app_name = ps.ByName("app_name")

	var service_path = getServicePath(service_name)

	log.Println("Pulling images...")

	var cmd = exec.Command("docker-compose", "pull", app_name)
	cmd.Dir = service_path
	err := cmd.Run()
	if err != nil {
		log.Fatal(err)
	}

	log.Printf("Killing app '%s/%s'\n", service_name, app_name)

	cmd = exec.Command("docker-compose", "down", app_name)
	cmd.Dir = service_path
	err = cmd.Run()
	if err != nil {
		log.Fatal(err)
	}

	log.Printf("Starting app '%s/%s'\n", service_name, app_name)
	cmd = exec.Command("docker-compose", "up", "-d")
	cmd.Dir = service_path
	err = cmd.Run()
	if err != nil {
		log.Fatal(err)
	}

	log.Println("Done")
	w.Write([]byte("Success\n"))
}

func BasicAuth(h httprouter.Handle, apiToken string) httprouter.Handle {
	return func(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
		// Get the Basic Authentication credentials
		if r.Header.Get("X-API-Key") != apiToken {
			w.WriteHeader(403)
			w.Write([]byte("Invalid API-Key"))
			return
		}

		h(w, r, ps)
	}
}

func main() {
	router := httprouter.New()

	router.POST("/trigger/:service_name", BasicAuth(PullAndRestartService, os.Getenv("API_KEY")))
	router.POST("/trigger/:service_name/:app_name", BasicAuth(PullAndRestartApp, os.Getenv("API_KEY")))

	var address = os.Getenv("ADDRESS")
	if address == "" {
		log.Fatal("You have to specify env ADDRESS: ex. :8080")
	}
	if os.Getenv("API_KEY") == "" {
		log.Fatal("You have to specify env API_KEY: ex. 12ds23kj23")
	}

	log.Fatal(http.ListenAndServe(address, router))
}
