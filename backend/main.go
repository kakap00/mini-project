package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"net/http"

	"github.com/gorilla/mux"
	_ "github.com/lib/pq"
)

const (
	host     = "localhost"
	port     = 5432
	user     = "postgres"
	password = "tigermollie"
	dbname   = "test1"
)

var db *sql.DB

type Movie struct {
	ID   int    `json:"id"`
	ISBN int    `json:"isbn"`
	NAME string `json:"name"`
}

func getMovies(w http.ResponseWriter, r *http.Request) { //get all movies
	fmt.Println("Handling /movies request")
	w.Header().Set("Content-Type", "application/json")
	rows, err := db.Query("select * from movies;")
	if err != nil {
		fmt.Println("Error executing query:", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	fmt.Println("Query executed successfully, processing rows...")

	var movies []Movie
	for rows.Next() {
		var movie Movie
		err := rows.Scan(&movie.ID, &movie.ISBN, &movie.NAME)
		if err != nil {
			fmt.Println("Error scanning row:", err)
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		movies = append(movies, movie)
	}

	fmt.Println("Returning movie data...")
	json.NewEncoder(w).Encode(movies)
}

func createMovie(w http.ResponseWriter, r *http.Request) {
	fmt.Println("Handling /movies request for creation of movie")
	w.Header().Set("Content-Type", "application/json")
	var movie Movie
	err := json.NewDecoder(r.Body).Decode(&movie)

	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	_, err = db.Exec("insert into movies (isbn, name) values ($1, $2)", movie.ISBN, movie.NAME)

	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusCreated)
	fmt.Fprintf(w, "User added successfully")
}

func receiveRust(w http.ResponseWriter, r *http.Request) {
	fmt.Println("receiving direct query from rust middleware...")
	w.Header().Set("Content-Type", "application/json")
	var query string
	err := json.NewDecoder(r.Body).Decode(&query) // Expecting raw SQL query in JSON format
	if err != nil {
		fmt.Println("Error decoding query:", err)
		http.Error(w, "Invalid query format", http.StatusBadRequest)
		return
	}

	fmt.Println("Executing query:", query)

	rows, err := db.Query(query)
	if err != nil {
		log.Fatalf("Error executing query: %v", err)
	}
	defer rows.Close() // Ensure rows are closed when done

	// Iterate through the rows and print the output
	columns, err := rows.Columns()
	if err != nil {
		log.Fatalf("Error fetching columns: %v", err)
	}

	var results []map[string]interface{}

	for rows.Next() {
		// Create a slice of interface{} to hold column values
		values := make([]interface{}, len(columns))
		valuePointers := make([]interface{}, len(columns))

		for i := range values {
			valuePointers[i] = &values[i]
		}

		// Scan the row into the value pointers
		if err := rows.Scan(valuePointers...); err != nil {
			log.Fatalf("Error scanning row: %v", err)
		}

		// Convert the row to a map for better display
		result := make(map[string]interface{})
		for i, col := range columns {
			result[col] = values[i]
		}

		results = append(results, result)

		fmt.Printf("Row: %v\n", result) // Print row data
	}

	if err := rows.Err(); err != nil {
		log.Fatalf("Error during row iteration: %v", err)
	}

	reponsedata, err := json.Marshal(results)
	if err != nil {
		log.Fatalf("error marshalling %v", err)
	}

	fmt.Println(string(reponsedata))

	w.WriteHeader(http.StatusOK)
	w.Header().Set("Content-Type", "application/json")
	w.Write(reponsedata) // this is sending my return value of postgres query back up to rust

	//converting of output into JSON format and then sending that back up to rust
}

func main() {

	psqlInfo := fmt.Sprintf("host=%s port=%d user=%s password=%s dbname=%s sslmode=disable", host, port, user, password, dbname)
	var err error
	db, err = sql.Open("postgres", psqlInfo) //Open method only validates all our info
	if err != nil {
		panic(err)
	}

	defer db.Close()

	err = db.Ping() //PING CONNECTS
	if err != nil {
		log.Fatal("DIDNT CONNECT!!")
	}

	log.Println("connected to db!")

	r := mux.NewRouter() //connection instance

	r.HandleFunc("/movies", getMovies).Methods("GET") //function for getting all movies
	r.HandleFunc("/movies", createMovie).Methods("POST")
	r.HandleFunc("/receive_from_rust", receiveRust).Methods("POST") //takes direct query from rust via post

	fmt.Printf("starting server at port 8000...")
	log.Fatal(http.ListenAndServe(":8000", r))

}
