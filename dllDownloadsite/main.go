package main

import (
	"fmt"
	"html/template"
	"log"
	"net/http"
)

func main() {
	// Define the file path of the executable file
	filePath := "/path/to/executable/file.exe"

	// Define the route handler for the homepage
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		// Parse the HTML template file
		tmpl, err := template.ParseFiles("index.html")
		if err != nil {
			log.Println("Error parsing template:", err)
			http.Error(w, "Internal server error", http.StatusInternalServerError)
			return
		}

		// Execute the template with data
		data := struct {
			DownloadURL string
		}{
			DownloadURL: "/download",
		}
		err = tmpl.Execute(w, data)
		if err != nil {
			log.Println("Error executing template:", err)
			http.Error(w, "Internal server error", http.StatusInternalServerError)
		}
	})

	// Define the route handler for downloading the file
	http.HandleFunc("/download", func(w http.ResponseWriter, r *http.Request) {
		// Set the appropriate headers
		w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=\"%s\"", "file.exe"))
		w.Header().Set("Content-Type", "application/octet-stream")

		// Serve the file
		http.ServeFile(w, r, filePath)
	})

	// Start the web server
	log.Println("Server listening on http://localhost:7889")
	log.Fatal(http.ListenAndServe(":7889", nil))
}
