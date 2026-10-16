package main

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/julienschmidt/httprouter"
)

func getServicePath(service_name string) (string, error) {
	if service_name == "" || service_name == "." || service_name == ".." ||
		filepath.IsAbs(service_name) || filepath.Clean(service_name) != service_name ||
		strings.ContainsAny(service_name, `/\`) {
		return "", fmt.Errorf("invalid service name %q", service_name)
	}

	path, err := os.Getwd()
	if err != nil {
		return "", fmt.Errorf("get working directory: %w", err)
	}

	service_path := filepath.Join(path, service_name)
	base_path, err := filepath.EvalSymlinks(path)
	if err != nil {
		return "", fmt.Errorf("resolve working directory: %w", err)
	}
	service_path, err = filepath.EvalSymlinks(service_path)
	if err != nil {
		return "", fmt.Errorf("resolve service directory: %w", err)
	}
	relative_path, err := filepath.Rel(base_path, service_path)
	if err != nil {
		return "", fmt.Errorf("check service directory: %w", err)
	}
	if relative_path == ".." || strings.HasPrefix(relative_path, ".."+string(filepath.Separator)) {
		return "", fmt.Errorf("service directory %q is outside the working directory", service_name)
	}

	log.Println("Using directory", service_path)

	return service_path, nil
}

func pullImages(service_path string) {
	var cmd = exec.Command("docker", "compose", "pull")
	cmd.Dir = service_path
	err := cmd.Run()
	if err != nil {
		log.Println("Failed to pull images ", err)
		return
	}
}

func Index(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
	fmt.Fprint(w, "Simply docker compose-trigger API\n")
}

func PullAndRestartService(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
	var service_name = ps.ByName("service_name")
	service_path, err := getServicePath(service_name)
	if err != nil {
		log.Printf("Invalid service directory: %v", err)
		http.Error(w, "Invalid service name", http.StatusBadRequest)
		return
	}

	log.Println("Pulling images...")
	pullImages(service_path)

	log.Printf("Starting service '%s'\n", service_name)
	cmd := exec.Command("docker", "compose", "up", "-d")
	cmd.Dir = service_path
	err = cmd.Run()
	if err != nil {
		log.Println("Failed to start ", err)
		return
	}

	log.Println("Done")
	w.Write([]byte("Success\n"))
}

func PullAndRestartApp(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
	var service_name = ps.ByName("service_name")
	var app_name = ps.ByName("app_name")

	service_path, err := getServicePath(service_name)
	if err != nil {
		log.Printf("Invalid service directory: %v", err)
		http.Error(w, "Invalid service name", http.StatusBadRequest)
		return
	}

	log.Println("Pulling image...")

	var cmd = exec.Command("docker", "compose", "pull", app_name)
	cmd.Dir = service_path
	err = cmd.Run()
	if err != nil {
		log.Printf("Failed to pull image for app %s %s\n", app_name, err)
		return
	}

	log.Printf("Updating app '%s/%s'\n", service_name, app_name)
	cmd = exec.Command("docker", "compose", "up", "-d")
	cmd.Dir = service_path
	err = cmd.Run()
	if err != nil {
		log.Println("Failed to start app ", app_name)
		return
	}

	log.Println("Done")
	w.Write([]byte("Success\n"))
}

func BasicAuth(h httprouter.Handle, apiToken string) httprouter.Handle {
	return func(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
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

	router.GET("/", Index)
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
