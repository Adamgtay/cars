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

	// Serve static files (images)
	http.Handle("/img/", http.StripPrefix("/img/", http.FileServer(http.Dir("api/img"))))

	// Define HTTP routes
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if err := recover(); err != nil {
				log.Printf("panic: %v", err)
				http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			}
		}()

		tmpl := template.Must(template.New("index").Parse(indexTemplate))
		err := tmpl.Execute(w, data)
		if err != nil {
			handleInternalServerError(w, err)
			return
		}
	})

	http.HandleFunc("/search", func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if err := recover(); err != nil {
				log.Printf("panic: %v", err)
				http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			}
		}()

		query := r.FormValue("q")
		year := r.FormValue("year")
		results := searchCarModels(query, year, data.CarModels, data.Manufacturers)
		tmpl := template.Must(template.New("search").Parse(searchTemplate))
		err := tmpl.Execute(w, results)
		if err != nil {
			handleInternalServerError(w, err)
			return
		}
	})

	http.HandleFunc("/compare", func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if err := recover(); err != nil {
				log.Printf("panic: %v", err)
				http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			}
		}()

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

		tmpl := template.Must(template.New("compare").Parse(compareTemplate))
		err := tmpl.Execute(w, cars)
		if err != nil {
			handleInternalServerError(w, err)
			return
		}
	})

	// Start the server
	log.Println("Starting server on http://localhost:8080...")
	log.Fatal(http.ListenAndServe(":8080", nil))
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

const indexTemplate = `
<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>Car Models</title>
</head>
<body>
    <h1>Car Models</h1>
    <h2>Search Car Models</h2>
    <form action="/search" method="get">
        <input type="text" name="q" placeholder="Search...">
        <input type="text" name="year" placeholder="Year (optional)">
        <button type="submit">Search</button>
    </form>
    <h2>Select Car Models for Comparison</h2>
    <form action="/compare" method="post">
        <select name="carModelID1">
            <option disabled selected>Select first car</option>
            {{range .CarModels}}
            <option value="{{.ID}}">{{.Name}}</option>
            {{end}}
        </select>
        <select name="carModelID2">
            <option disabled selected>Select second car</option>
            {{range .CarModels}}
            <option value="{{.ID}}">{{.Name}}</option>
            {{end}}
        </select>
        <button type="submit">Compare</button>
    </form>
</body>
</html>
`

const searchTemplate = `
<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>Search Results</title>
    <style>
        .car {
            margin-bottom: 20px;
            border: 1px solid #ccc;
            padding: 10px;
            border-radius: 5px;
        }
        .car img {
            max-width: 100%;
            height: auto;
            margin-bottom: 10px;
        }
    </style>
</head>
<body>
    <h1>Search Results</h1>
    {{range .}}
    <div class="car">
        <h2>{{.Name}}</h2>
        <p><strong>Manufacturer:</strong> {{.ManufacturerName}}</p>
        <p><strong>Country:</strong> {{.ManufacturerCountry}}</p>
        <p><strong>Founding Year:</strong> {{.ManufacturerFoundingYear}}</p>
        <img src="/img/{{.Image}}" alt="{{.Name}}">
        <p><strong>Engine:</strong> {{.Specifications.Engine}}</p>
        <p><strong>Horsepower:</strong> {{.Specifications.Horsepower}}</p>
        <p><strong>Transmission:</strong> {{.Specifications.Transmission}}</p>
        <p><strong>Drivetrain:</strong> {{.Specifications.Drivetrain}}</p>
    </div>
    {{end}}
</body>
</html>
`

const compareTemplate = `
<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>Car Comparison</title>
    <style>
        table {
            width: 100%;
            border-collapse: collapse;
        }
        th, td {
            padding: 8px;
            text-align: left;
            border-bottom: 1px solid #ddd;
        }
        th {
            background-color: #f2f2f2;
        }
    </style>
</head>
<body>
    <h1>Car Comparison</h1>
    <table>
        <tr>
            <th>Feature</th>
            {{range .}}
            <th>{{.Name}}</th>
            {{end}}
        </tr>
        <tr>
            <td>Manufacturer</td>
            {{range .}}
            <td>{{.ManufacturerID}}</td>
            {{end}}
        </tr>
        <tr>
            <td>Year</td>
            {{range .}}
            <td>{{.Year}}</td>
            {{end}}
        </tr>
        <tr>
            <td>Engine</td>
            {{range .}}
            <td>{{.Specifications.Engine}}</td>
            {{end}}
        </tr>
        <tr>
            <td>Horsepower</td>
            {{range .}}
            <td>{{.Specifications.Horsepower}}</td>
            {{end}}
        </tr>
        <tr>
            <td>Transmission</td>
            {{range .}}
            <td>{{.Specifications.Transmission}}</td>
            {{end}}
        </tr>
        <tr>
            <td>Drivetrain</td>
            {{range .}}
            <td>{{.Specifications.Drivetrain}}</td>
            {{end}}
        </tr>
    </table>
</body>
</html>
`
