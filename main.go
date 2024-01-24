package main

import (
	"encoding/json"
	"fmt"
	"github.com/google/uuid"
	"io"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/jmoiron/sqlx"
	_ "github.com/lib/pq"
)

type todo struct {
	Id          uuid.UUID  `json:"id,omitempty" db:"id"`
	Description string     `json:"description,omitempty" db:"description"`
	CreatedAt   *time.Time `json:"createdAt,omitempty" db:"created_at"`
	CompletedAt *time.Time `json:"completedAt,omitempty" db:"completed_at"`
}

var todos = make([]todo, 0)

func health(w http.ResponseWriter, req *http.Request) {
	fmt.Fprintf(w, "health\n")
}

func todoList(w http.ResponseWriter, req *http.Request) {
	// language=SQL
	SQL := "select id, description, created_at, completed_at from todo where archived_at is null"
	var todos []todo
	err := DB.Select(&todos, SQL)
	if err != nil {
		fmt.Println("error reading todos %+v", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	RespondJSON(w, http.StatusOK, todos)
}

func addTodo(w http.ResponseWriter, req *http.Request) {
	var body todo
	if err := ParseBody(req.Body, &body); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	SQL := `insert into todo (id, description, created_at) values ($1, $2, $3)`
	_, err := DB.Queryx(SQL, uuid.New(), body.Description, time.Now())
	if err != nil {
		fmt.Println("error writing todos %+v", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusCreated)
}

func updateTODO(w http.ResponseWriter, req *http.Request) {
	var body todo
	if err := ParseBody(req.Body, &body); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	SQL := `update todo set description = $1 where id = $2`
	_, err := DB.Queryx(SQL, body.Description, body.Id)
	if err != nil {
		fmt.Println("error writing todos %+v", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusCreated)
}

func completeTODO(w http.ResponseWriter, req *http.Request) {
	var body todo
	if err := ParseBody(req.Body, &body); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	SQL := `update todo set completed_at = $1 where id = $2`
	_, err := DB.Queryx(SQL, time.Now(), body.Id)
	if err != nil {
		fmt.Println("error writing todos %+v", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusCreated)
}

func deleteTODO(w http.ResponseWriter, req *http.Request) {
	var body todo
	if err := ParseBody(req.Body, &body); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	SQL := `update todo set archived_at = now() where id = $1`
	_, err := DB.Queryx(SQL, body.Id)
	if err != nil {
		fmt.Println("error writing todos %+v", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusCreated)
}

var DB *sqlx.DB

func main() {

	psqlInfo := fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=disable",
		"localhost", "5432", "local", "local", "todo")

	var err error
	DB, err = sqlx.Open("postgres", psqlInfo)
	if err != nil {
		fmt.Println("Unable to Connect to the Database, ", err)
		return
	}
	err = DB.Ping()
	if err != nil {
		fmt.Println("Ping Panic", err)
		return
	}

	router := chi.NewRouter()
	router.Route("/", func(r chi.Router) {
		r.Get("/health", health)

		//r.Get("/todos", todoList)
		//r.Post("/add-todo", addTodo)
		//r.Put("/update-todo", updateTODO)
		//r.Put("/complete-todo", completeTODO)
		//r.Delete("/delete-todo", deleteTODO)

		todoRouters(r)
	})

	fmt.Println("starting server at port 8080")
	http.ListenAndServe(":8080", router)
}

func todoRouters(r chi.Router) chi.Router {
	return r.Route("/todo", func(todoRouter chi.Router) {
		todoRouter.Get("/", todoList)
		todoRouter.Post("/", addTodo)
		todoRouter.Route("/{id}", func(todoIdRouter chi.Router) {
			todoIdRouter.Put("/", updateTODO)
			todoIdRouter.Put("/complete", completeTODO)
			todoIdRouter.Delete("/", deleteTODO)
		})
	})
}

// RespondJSON sends the rateMetricInterface as a JSON
func RespondJSON(w http.ResponseWriter, statusCode int, body interface{}) {
	w.WriteHeader(statusCode)
	if body != nil {
		if err := EncodeJSONBody(w, body); err != nil {
			fmt.Println(fmt.Errorf("failed to respond JSON with error: %+v", err))
		}
	}
}

// EncodeJSONBody writes the JSON body to response writer
func EncodeJSONBody(resp http.ResponseWriter, data interface{}) error {
	return json.NewEncoder(resp).Encode(data)
}

// ParseBody parses the values from io reader to a given interface
func ParseBody(body io.Reader, out interface{}) error {
	err := json.NewDecoder(body).Decode(out)
	if err != nil {
		return err
	}

	return nil
}

/*
* TODO List -> GET 					http://localhost:8080/todo
* Add TODO -> POST 					http://localhost:8080/todo
* Update Description -> PUT 		http://localhost:8080/todo/:id
* Mark todo completed -> PUT 		http://localhost:8080/todo/:id/complete
* Delete -> DELETE 					http://localhost:8080/todo/:id
 */

/*
error reading todos %+v sql: Scan error on column index 2, name "created_at": unsupported Scan, storing driver.Value type <nil> into type *time.Time
*/
