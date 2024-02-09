package main

import (
	"context"
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

const userContextKey = "userContext"

type todo struct {
	Id          uuid.UUID  `json:"id,omitempty" db:"id"`
	Description string     `json:"description,omitempty" db:"description"`
	CreatedAt   *time.Time `json:"createdAt,omitempty" db:"created_at"`
	CompletedAt *time.Time `json:"completedAt,omitempty" db:"completed_at"`
}

var todos = make([]todo, 0)

func AuthMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		sessionToken := r.Header.Get("token")
		userId, err := getUserIdForToken(sessionToken)
		if err != nil {
			RespondJSON(w, http.StatusUnauthorized, err)
			return
		}

		r = r.WithContext(context.WithValue(r.Context(), userContextKey, userId))

		next.ServeHTTP(w, r)
	})
}

func getUserIdForToken(sessionToken string) (string, error) {
	SQL := `select user_id from session where id = $1 and archived_at is null`
	var userId string
	err := DB.Get(&userId, SQL, sessionToken)
	return userId, err
}

func health(w http.ResponseWriter, req *http.Request) {
	fmt.Fprintf(w, "health\n")
}

//
//// LanguageMiddleWare this middle ware will take the language from header and add it to context
//func LanguageMiddleWare(next http.Handler) http.Handler {
//	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
//		lang := r.Header.Get("language")
//		// check if the language provided is correct if not use english as default
//		userLanguage := i18n.ValidateLanguage(lang)
//		ctx := context.WithValue(r.Context(), LanguageCtx, userLanguage)
//		next.ServeHTTP(w, r.WithContext(ctx))
//		return
//	})
//}

func todoList(w http.ResponseWriter, req *http.Request) {

	//	r = r.WithContext(context.WithValue(r.Context(), userContextKey, userId))
	userId := req.Context().Value(userContextKey).(string)

	// language=SQL
	SQL := "select id, description, created_at, completed_at from todo where archived_at is null and user_id = $1"
	var todos []todo
	err := DB.Select(&todos, SQL, userId)
	if err != nil {
		fmt.Println("error reading todos %+v", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	RespondJSON(w, http.StatusOK, todos)
}

func addTodo(w http.ResponseWriter, req *http.Request) {
	userId := req.Context().Value(userContextKey).(string)

	var body todo
	if err := ParseBody(req.Body, &body); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	SQL := `insert into todo (id, description, created_at, user_id) values ($1, $2, $3, $4)`
	_, err := DB.Queryx(SQL, uuid.New(), body.Description, time.Now(), userId)
	if err != nil {
		fmt.Println("error writing todos %+v", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusCreated)
}

func updateTODO(w http.ResponseWriter, req *http.Request) {
	userId := req.Context().Value(userContextKey).(string)

	var body todo
	if err := ParseBody(req.Body, &body); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	SQL := `update todo set description = $1 where id = $2 and user_id = $3`
	_, err := DB.Queryx(SQL, body.Description, body.Id, userId)
	if err != nil {
		fmt.Println("error writing todos %+v", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusCreated)
}

func completeTODO(w http.ResponseWriter, req *http.Request) {
	userId := req.Context().Value(userContextKey).(string)

	var body todo
	if err := ParseBody(req.Body, &body); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	SQL := `update todo set completed_at = $1 where id = $2 and user_id = $3`
	_, err := DB.Queryx(SQL, time.Now(), body.Id, userId)
	if err != nil {
		fmt.Println("error writing todos %+v", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusCreated)
}

func deleteTODO(w http.ResponseWriter, req *http.Request) {
	userId := req.Context().Value(userContextKey).(string)
	var body todo
	if err := ParseBody(req.Body, &body); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	SQL := `update todo set archived_at = now() where id = $1 and user_id = $2`
	_, err := DB.Queryx(SQL, body.Id, userId)
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
		r.Post("/login", login)
		userRouter(r)
		todoRouters(r)
	})

	fmt.Println("starting server at port 8080")
	http.ListenAndServe(":8080", router)
}

/*
browser -> request -> handler -> response -> browser
browser -> request -> middleware -> handler -> middleware -> response -> browser
*/
type user struct {
	Id          uuid.UUID  `json:"id,omitempty" db:"id"`
	Name        string     `json:"name,omitempty" db:"name"`
	Email       string     `json:"email,omitempty" db:"email"`
	Password    string     `json:"password" db:"password"`
	CreatedAt   *time.Time `json:"createdAt,omitempty" db:"created_at"`
	CompletedAt *time.Time `json:"completedAt,omitempty" db:"completed_at"`
}

func addUser(w http.ResponseWriter, r *http.Request) {
	var body user
	if err := ParseBody(r.Body, &body); err != nil {
		RespondJSON(w, http.StatusBadRequest, err)
		return
	}

	SQL := `insert into users (id, name, email, password) values ($1, $2, $3, $4)`
	_, err := DB.Queryx(SQL, uuid.New(), body.Name, body.Email, body.Password)
	if err != nil {
		RespondJSON(w, http.StatusInternalServerError, err)
		return
	}

	RespondJSON(w, http.StatusCreated, nil)
}

type loginRequest struct {
	Email    string `json:"email,omitempty" db:"email"`
	Password string `json:"password" db:"password"`
}

type response struct {
	Token string `json:"token"`
}

func login(w http.ResponseWriter, r *http.Request) {
	var body loginRequest
	if err := ParseBody(r.Body, &body); err != nil {
		RespondJSON(w, http.StatusBadRequest, err)
		return
	}

	SQL := `select id from users where email = $1 and password = $2`
	var userId string
	err := DB.Get(&userId, SQL, body.Email, body.Password)
	if err != nil {
		RespondJSON(w, http.StatusUnauthorized, err)
		return
	}

	SQL = `insert into session (id, user_id) values ($1, $2)`
	sessionId := uuid.New()
	_, err = DB.Queryx(SQL, sessionId, userId)
	if err != nil {
		RespondJSON(w, http.StatusInternalServerError, err)
		return
	}

	RespondJSON(w, http.StatusCreated, response{
		Token: sessionId.String(),
	})
}

func todoRouters(r chi.Router) chi.Router {
	return r.Route("/todo", func(todoRouter chi.Router) {
		todoRouter.Use(AuthMiddleware)
		todoRouter.Get("/", todoList)
		todoRouter.Post("/", addTodo)
		todoRouter.Route("/{id}", func(todoIdRouter chi.Router) {
			todoIdRouter.Put("/", updateTODO)
			todoIdRouter.Put("/complete", completeTODO)
			todoIdRouter.Delete("/", deleteTODO)
		})
	})
}

func userRouter(r chi.Router) chi.Router {
	return r.Route("/user", func(todoRouter chi.Router) {
		todoRouter.Post("/", addUser)
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
 */

/*
Add User -> register the user in the system
Login -> User can login
---------
* TODO List -> GET 					http://localhost:8080/todo
* Add TODO -> POST 					http://localhost:8080/todo
* Update Description -> PUT 		http://localhost:8080/todo/:id
* Mark todo completed -> PUT 		http://localhost:8080/todo/:id/complete
* Delete -> DELETE 					http://localhost:8080/todo/:id
-------
*/

/*
error reading todos %+v sql: Scan error on column index 2, name "created_at": unsupported Scan, storing driver.Value type <nil> into type *time.Time
*/
