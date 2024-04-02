package main

import (
	"encoding/json"
	"html/template"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

// Manufacturer represents a car manufacturer.
type Manufacturer struct {
	ID           int    `json:"id"`
	Name         string `json:"name"`
	Country      string `json:"country"`
	FoundingYear int    `json:"foundingYear"`
}

// Category represents a car category.
type Category struct {
	ID   int    `json:"id"`
	Name string `json:"name"`
}

// Specifications represents car specifications.
type Specifications struct {
	Engine       string `json:"engine"`
	Horsepower   int    `json:"horsepower"`
	Transmission string `json:"transmission"`
	Drivetrain   string `json:"drivetrain"`
}

// CarModel represents a car model.
type CarModel struct {
	ID                       int            `json:"id"`
	Name                     string         `json:"name"`
	ManufacturerID           int            `json:"manufacturerId"`
	CategoryID               int            `json:"categoryId"`
	Year                     int            `json:"year"`
	Specifications           Specifications `json:"specifications"`
	Image                    string         `json:"image"`
	ManufacturerName         string         `json:"manufacturerName,omitempty"`
	ManufacturerCountry      string         `json:"manufacturerCountry,omitempty"`
	ManufacturerFoundingYear int            `json:"manufacturerFoundingYear,omitempty"`
}

// Data represents all data from the JSON file.
type Data struct {
	Manufacturers []Manufacturer `json:"manufacturers"`
	Categories    []Category     `json:"categories"`
	CarModels     []CarModel     `json:"carModels"`
}

func loadJSONFile(fileName string, v interface{}) error {
	filePath := filepath.Join("api", fileName)
	file, err := os.Open(filePath)
	if err != nil {
		return err
	}
	defer file.Close()

	return json.NewDecoder(file).Decode(v)
}

func main() {
	// Load data from JSON file
	var data Data
	err := loadJSONFile("data.json", &data)
	if err != nil {
		log.Fatalf("Error loading JSON file: %v", err)
	}

	// Parse HTML templates
	indexTemplate := parseTemplate("index.html")
	searchTemplate := parseTemplate("search.html")
	compareTemplate := parseTemplate("compare.html")

	// Serve static files (images)
	http.Handle("/img/", http.StripPrefix("/img/", http.FileServer(http.Dir("static/img"))))

	// Define HTTP routes
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		// Handler function using indexTemplate
		renderTemplate(w, indexTemplate, data)
	})

	http.HandleFunc("/search", func(w http.ResponseWriter, r *http.Request) {
		// Handler function using searchTemplate
		query := r.FormValue("q")
		year := r.FormValue("year")
		results := searchCarModels(query, year, data.CarModels, data.Manufacturers)
		renderTemplate(w, searchTemplate, results)
	})

	http.HandleFunc("/compare", func(w http.ResponseWriter, r *http.Request) {
		// Handler function using compareTemplate
		if r.Method != http.MethodPost {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}

		carModelID1 := r.FormValue("carModelID1")
		carModelID2 := r.FormValue("carModelID2")

		if carModelID1 == "" || carModelID2 == "" {
			http.Error(w, "Please select exactly two car models to compare", http.StatusBadRequest)
			return
		}

		cars := make([]CarModel, 2)
		for i, idStr := range []string{carModelID1, carModelID2} {
			id, err := strconv.Atoi(idStr)
			if err != nil {
				http.Error(w, "Invalid car model ID", http.StatusBadRequest)
				return
			}
			var found bool
			for _, car := range data.CarModels {
				if car.ID == id {
					cars[i] = car
					found = true
					break
				}
			}
			if !found {
				http.Error(w, "Invalid car model ID", http.StatusBadRequest)
				return
			}
		}

		renderTemplate(w, compareTemplate, cars)
	})

	// Start the server
	log.Println("Starting server on http://localhost:8080...")
	log.Fatal(http.ListenAndServe(":8080", nil))
}

func parseTemplate(filename string) *template.Template {
	tmpl, err := template.ParseFiles(filename)
	if err != nil {
		log.Fatalf("Error parsing template %s: %v", filename, err)
	}
	return tmpl
}

func renderTemplate(w http.ResponseWriter, tmpl *template.Template, data interface{}) {
	err := tmpl.Execute(w, data)
	if err != nil {
		handleInternalServerError(w, err)
		return
	}
}

func handleInternalServerError(w http.ResponseWriter, err error) {
	log.Printf("Internal server error: %v", err)
	http.Error(w, "Internal server error", http.StatusInternalServerError)
}

func searchCarModels(query, year string, carModels []CarModel, manufacturers []Manufacturer) []CarModel {
	var results []CarModel
	query = strings.ToLower(query)
	for _, carModel := range carModels {
		if strings.Contains(strings.ToLower(carModel.Name), query) &&
			(year == "" || strconv.Itoa(carModel.Year) == year) {
			// Find manufacturer information
			var manufacturerInfo Manufacturer
			for _, manufacturer := range manufacturers {
				if manufacturer.ID == carModel.ManufacturerID {
					manufacturerInfo = manufacturer
					break
				}
			}
			// Include manufacturer information in the results
			carModelWithManufacturer := carModel
			carModelWithManufacturer.ManufacturerName = manufacturerInfo.Name
			carModelWithManufacturer.ManufacturerCountry = manufacturerInfo.Country
			carModelWithManufacturer.ManufacturerFoundingYear = manufacturerInfo.FoundingYear
			results = append(results, carModelWithManufacturer)
		}
	}
	return results
}
