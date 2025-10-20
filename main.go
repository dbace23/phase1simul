package main

import (
	"encoding/json"
	"errors"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"
	"sync"

	"github.com/julienschmidt/httprouter"
)

// inventory model
type Product struct {
	ID          int     `json:"id"`
	Name        string  `json:"name"`
	Description string  `json:"description"`
	Price       float64 `json:"price"`
}

// mock data
var (
	products = []Product{
		{ID: 1, Name: "shield", Description: "this is a shield", Price: 23.4},
		{ID: 2, Name: "kursi", Description: "this is a kursi", Price: 20.4},
	}
	nextID   = 3
	storeMux sync.RWMutex
)

func main() {
	router := httprouter.New()
	router.GET("/", func(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("OK"))
	})
	router.GET("/products", listProducts)
	router.GET("/products/:id", getProduct)
	router.POST("/products", createProduct)
	router.PUT("/products/:id", updateProduct)
	router.DELETE("/products/:id", deleteProduct)

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080" // local default
	}
	addr := ":" + port
	log.Println("product server on %s", addr)
	log.Fatal(http.ListenAndServe(addr, router))
}

// helper utils folder
func writeJSON(w http.ResponseWriter, code int, data any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	if err := json.NewEncoder(w).Encode(data); err != nil {
		// last-ditch log; headers already sent
		log.Printf("writeJSON encode error: %v", err)
	}
}

func writeError(w http.ResponseWriter, code int, msg string) {
	writeJSON(w, code, map[string]string{"error": msg})
}

func parseID(param string) (int, error) {
	id, err := strconv.Atoi(param)
	if err != nil || id <= 0 {
		return 0, errors.New("invalid id")
	}
	return id, nil
}

func validateItem(it *Product) error {
	if strings.TrimSpace(it.Name) == "" {
		return errors.New("name is required")
	}
	if strings.TrimSpace(it.Description) == "" {
		return errors.New("description is required")
	}
	if it.Price < 0 {
		return errors.New("price cannot be negative")
	}
	return nil
}

func findIndexByID(id int) int {
	for i, v := range products { // replace with DB query in real app
		if v.ID == id {
			return i
		}
	}
	return -1
}

// Handlers
// get products
func listProducts(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
	storeMux.RLock()
	defer storeMux.RUnlock()
	writeJSON(w, http.StatusOK, products) // replace with DB query
}

// get product by id
func getProduct(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
	id, err := parseID(ps.ByName("id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	storeMux.RLock()
	defer storeMux.RUnlock()

	if idx := findIndexByID(id); idx == -1 {
		writeError(w, http.StatusNotFound, "product not found")
		return
	} else {
		writeJSON(w, http.StatusOK, products[idx]) // replace with DB query
	}
}

// post product
func createProduct(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
	var in Product
	dec := json.NewDecoder(r.Body)
	dec.DisallowUnknownFields()
	if err := dec.Decode(&in); err != nil {
		writeError(w, http.StatusBadRequest, "invalid json: "+err.Error())
		return
	}
	if err := validateItem(&in); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	storeMux.Lock()
	defer storeMux.Unlock()

	in.ID = nextID
	nextID++
	products = append(products, in) // replace with INSERT
	writeJSON(w, http.StatusCreated, in)
}

// put product
func updateProduct(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
	id, err := parseID(ps.ByName("id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	var in Product
	dec := json.NewDecoder(r.Body)
	dec.DisallowUnknownFields()
	if err := dec.Decode(&in); err != nil {
		writeError(w, http.StatusBadRequest, "invalid json: "+err.Error())
		return
	}
	if err := validateItem(&in); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	storeMux.Lock()
	defer storeMux.Unlock()

	idx := findIndexByID(id)
	if idx == -1 {
		writeError(w, http.StatusNotFound, "product not found")
		return
	}

	in.ID = id         // path param is the source of truth
	products[idx] = in // replace with UPDATE
	writeJSON(w, http.StatusOK, in)
}

// delete product
func deleteProduct(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
	id, err := parseID(ps.ByName("id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	storeMux.Lock()
	defer storeMux.Unlock()

	idx := findIndexByID(id)
	if idx == -1 {
		writeError(w, http.StatusNotFound, "product not found")
		return
	}

	products = append(products[:idx], products[idx+1:]...)
	writeJSON(w, http.StatusOK, map[string]any{"deleted_id": id})
}
